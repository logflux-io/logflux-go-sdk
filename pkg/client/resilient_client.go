package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/crypto"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/discovery"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/handshake"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/queue"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/retry"
)

// DropReason tracks why entries were dropped.
type DropReason string

const (
	DropQueueOverflow  DropReason = "queue_overflow"
	DropNetworkError   DropReason = "network_error"
	DropSendError      DropReason = "send_error"
	DropRateLimited    DropReason = "ratelimit_backoff"
	DropQuotaExceeded  DropReason = "quota_exceeded"
	DropBeforeSend     DropReason = "before_send"
	DropValidation     DropReason = "validation_error"
)

// ClientStats holds runtime statistics.
type ClientStats struct {
	EntriesSent    int64
	EntriesDropped int64
	EntriesQueued  int64
	QueueSize      int64
	QueueCapacity  int64
	DropReasons    map[DropReason]int64
	LastSendError  string
	LastSendTime   time.Time
	HandshakeOK    bool
}

// ResilientClientConfig holds configuration for the resilient client.
type ResilientClientConfig struct {
	Node              string
	APIKey            string
	Environment       string
	LogGroup          string
	CustomEndpointURL string

	QueueSize     int
	FlushInterval time.Duration
	BatchSize     int

	RetryConfig retry.Config

	HTTPTimeout       time.Duration
	FailsafeMode      bool
	WorkerCount       int
	EnableCompression bool
	ResilientMode     bool
	BeforeSend        BeforeSendFunc
}

func DefaultResilientClientConfig() ResilientClientConfig {
	return ResilientClientConfig{
		QueueSize:         1000,
		FlushInterval:     5 * time.Second,
		BatchSize:         100,
		RetryConfig:       retry.DefaultConfig(),
		HTTPTimeout:       30 * time.Second,
		FailsafeMode:      true,
		WorkerCount:       2,
		EnableCompression: true,
		ResilientMode:     false,
	}
}

func BasicClientConfig() ResilientClientConfig {
	return DefaultResilientClientConfig()
}

func FaultTolerantClientConfig() ResilientClientConfig {
	c := DefaultResilientClientConfig()
	c.QueueSize = 2000
	c.FlushInterval = 3 * time.Second
	c.RetryConfig = retry.ResilientConfig()
	c.HTTPTimeout = 45 * time.Second
	c.WorkerCount = 3
	c.ResilientMode = true
	return c
}

// ResilientClient is an async client with queue, retry, rate-limit awareness, and loss tracking.
type ResilientClient struct {
	config     ResilientClientConfig
	encryptor  *crypto.Encryptor
	endpoints  *discovery.EndpointInfo
	httpClient *http.Client
	queue      *queue.Queue
	retryer    *retry.Retryer
	keyUUID    string
	limits     *handshake.HandshakeLimits

	serverPublicKeyPEM   string
	serverKeyFingerprint string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics (atomic for lock-free fast path)
	totalSent    atomic.Int64
	totalQueued  atomic.Int64

	// Guarded by mu
	mu              sync.RWMutex
	totalDropped    int64
	dropReasons     map[DropReason]int64
	lastSendError   string
	lastSendTime    time.Time
	handshakeOK     bool

	// Rate limit state
	rateLimitMu        sync.RWMutex
	rateLimitLimit     int
	rateLimitRemaining int
	rateLimitReset     int64
	rateLimitPauseUntil time.Time

	// Quota state — per-category blocked
	quotaMu      sync.RWMutex
	quotaBlocked map[string]bool

	closed atomic.Bool
}

// NewResilientClientWithHandshake creates a resilient client with auto key negotiation.
func NewResilientClientWithHandshake(cfg ResilientClientConfig) (*ResilientClient, error) {
	if err := config.ValidateAPIKey(cfg.APIKey); err != nil {
		return nil, err
	}
	applyDefaults(&cfg)

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}

	var endpoints *discovery.EndpointInfo
	if cfg.CustomEndpointURL != "" {
		dc := discovery.NewDiscoveryClient(discovery.DiscoveryConfig{
			APIKey: cfg.APIKey, Timeout: 10 * time.Second, HTTPClient: httpClient,
		})
		endpoints = dc.SetCustomEndpoint(cfg.CustomEndpointURL)
	} else {
		dc := discovery.NewDiscoveryClient(discovery.DiscoveryConfig{
			APIKey: cfg.APIKey, Timeout: 10 * time.Second, HTTPClient: httpClient,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var err error
		endpoints, err = dc.DiscoverEndpoints(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("endpoint discovery failed: %w", err)
		}
	}

	handshakeResult, err := handshake.PerformHandshakeWithURL(
		endpoints.GetHandshakeURL(), cfg.APIKey, httpClient,
	)
	if err != nil {
		return nil, fmt.Errorf("handshake failed: %w", err)
	}

	enc := crypto.NewEncryptor(handshakeResult.AESKey)
	// Zero source key material after the encryptor has its own copy
	for i := range handshakeResult.AESKey {
		handshakeResult.AESKey[i] = 0
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &ResilientClient{
		config:               cfg,
		encryptor:            enc,
		httpClient:           httpClient,
		queue:                queue.NewQueue(cfg.QueueSize),
		retryer:              retry.NewRetryer(cfg.RetryConfig),
		keyUUID:              handshakeResult.KeyUUID,
		limits:               handshakeResult.Limits,
		serverPublicKeyPEM:   handshakeResult.ServerPublicKeyPEM,
		serverKeyFingerprint: handshakeResult.ServerKeyFingerprint,
		ctx:                  ctx,
		cancel:               cancel,
		endpoints:            endpoints,
		dropReasons:          make(map[DropReason]int64),
		quotaBlocked:         make(map[string]bool),
		handshakeOK:          true,
	}

	if cfg.ResilientMode && endpoints != nil {
		c.retryer.SetHealthCheckURL(endpoints.GetHealthURL())
		c.retryer.EnableResilientMode(true)
	}

	c.startWorkers()
	return c, nil
}

// NewResilientClientFromEnvWithHandshake creates a resilient client from env vars.
func NewResilientClientFromEnvWithHandshake(node string) (*ResilientClient, error) {
	envCfg, err := config.LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}
	cfg := DefaultResilientClientConfig()
	cfg.Node = node
	cfg.APIKey = envCfg.APIKey
	cfg.Environment = envCfg.Environment
	cfg.LogGroup = envCfg.LogGroup
	if envCfg.QueueSize > 0 {
		cfg.QueueSize = envCfg.QueueSize
	}
	if envCfg.FlushInterval > 0 {
		cfg.FlushInterval = time.Duration(envCfg.FlushInterval) * time.Second
	}
	if envCfg.BatchSize > 0 {
		cfg.BatchSize = envCfg.BatchSize
	}
	if envCfg.MaxRetries > 0 {
		cfg.RetryConfig.MaxRetries = envCfg.MaxRetries
	}
	if envCfg.InitialDelay > 0 {
		cfg.RetryConfig.InitialDelay = time.Duration(envCfg.InitialDelay) * time.Millisecond
	}
	if envCfg.MaxDelay > 0 {
		cfg.RetryConfig.MaxDelay = time.Duration(envCfg.MaxDelay) * time.Second
	}
	if envCfg.BackoffFactor > 0 {
		cfg.RetryConfig.BackoffFactor = envCfg.BackoffFactor
	}
	if envCfg.HTTPTimeout > 0 {
		cfg.HTTPTimeout = time.Duration(envCfg.HTTPTimeout) * time.Second
	}
	if envCfg.WorkerCount > 0 {
		cfg.WorkerCount = envCfg.WorkerCount
	}
	cfg.FailsafeMode = envCfg.FailsafeMode
	cfg.EnableCompression = envCfg.EnableCompression
	return NewResilientClientWithHandshake(cfg)
}

func applyDefaults(cfg *ResilientClientConfig) {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1000
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 2
	}
	if cfg.Node == "" {
		if hostname, err := os.Hostname(); err == nil {
			cfg.Node = hostname
		} else {
			cfg.Node = "unknown"
		}
	}
}

// --- Send methods ---

func (c *ResilientClient) SendLog(message string) error {
	return c.SendLogWithLevel(message, models.LogLevelInfo)
}

func (c *ResilientClient) SendLogWithLevel(message string, level int) error {
	return c.SendLogWithTimestampAndLevel(message, time.Now(), level)
}

func (c *ResilientClient) SendLogWithTimestamp(message string, timestamp time.Time) error {
	return c.SendLogWithTimestampAndLevel(message, timestamp, models.LogLevelInfo)
}

func (c *ResilientClient) SendLogWithTimestampAndLevel(message string, timestamp time.Time, level int) error {
	return c.SendLogWithTimestampLevelAndLabels(message, timestamp, level, nil)
}

func (c *ResilientClient) SendLogWithTimestampLevelAndLabels(message string, timestamp time.Time, level int, labels map[string]string) error {
	return c.enqueueEntry(message, timestamp, level, models.EntryTypeLog, 0, labels, nil)
}

func (c *ResilientClient) SendLogWithEntryType(message string, level, entryType int) error {
	return c.enqueueEntry(message, time.Now(), level, entryType, 0, nil, nil)
}

func (c *ResilientClient) SendLogWithLabels(message string, labels map[string]string) error {
	return c.SendLogWithTimestampLevelAndLabels(message, time.Now(), models.LogLevelInfo, labels)
}

func (c *ResilientClient) SendLogWithLevelAndLabels(message string, level int, labels map[string]string) error {
	return c.SendLogWithTimestampLevelAndLabels(message, time.Now(), level, labels)
}

func (c *ResilientClient) enqueueEntry(message string, timestamp time.Time, level, entryType, payloadType int, labels map[string]string, searchTokens []string) error {
	if c.closed.Load() {
		if c.config.FailsafeMode {
			return nil
		}
		return fmt.Errorf("client is closed")
	}

	if entryType == 0 {
		entryType = models.EntryTypeLog
	}

	entry := models.LogEntry{
		Message:      message,
		Timestamp:    timestamp,
		Level:        level,
		EntryType:    entryType,
		PayloadType:  payloadType,
		Node:         c.config.Node,
		Labels:       labels,
		SearchTokens: searchTokens,
	}

	if err := validateEntry(&entry); err != nil {
		c.recordDrop(DropValidation, 1)
		if c.config.FailsafeMode {
			return nil
		}
		return err
	}

	if c.config.BeforeSend != nil {
		result := c.config.BeforeSend(&entry)
		if result == nil {
			c.recordDrop(DropBeforeSend, 1)
			return nil
		}
		entry = *result
	}

	// Check quota
	category := models.EntryTypeCategory(entry.EntryType)
	c.quotaMu.RLock()
	blocked := c.quotaBlocked[category]
	c.quotaMu.RUnlock()
	if blocked {
		c.recordDrop(DropQuotaExceeded, 1)
		if c.config.FailsafeMode {
			return nil
		}
		return fmt.Errorf("quota exceeded for category: %s", category)
	}

	qEntry := queue.LogEntry{
		ID:           generateID(),
		Message:      entry.Message,
		Timestamp:    entry.Timestamp,
		Level:        entry.Level,
		EntryType:    entry.EntryType,
		PayloadType:  entry.PayloadType,
		Node:         entry.Node,
		Labels:       entry.Labels,
		SearchTokens: entry.SearchTokens,
		CreatedAt:    time.Now(),
	}

	if c.queue.Enqueue(qEntry) {
		c.totalQueued.Add(1)
		return nil
	}

	// Queue full
	c.recordDrop(DropQueueOverflow, 1)
	if c.config.FailsafeMode {
		return nil
	}
	return fmt.Errorf("queue is full, entry dropped")
}

// SendLogBatch sends multiple entries directly (bypasses queue).
func (c *ResilientClient) SendLogBatch(messages []LogMessage) error {
	if c.closed.Load() {
		if c.config.FailsafeMode {
			return nil
		}
		return fmt.Errorf("client is closed")
	}
	if len(messages) == 0 {
		return nil
	}
	if len(messages) > 1000 {
		return fmt.Errorf("batch size exceeds maximum of 1000 entries")
	}

	var entries []models.LogEntry
	for _, msg := range messages {
		et := msg.EntryType
		if et == 0 {
			et = models.EntryTypeLog
		}
		entry := models.LogEntry{
			Message:      msg.Message,
			Timestamp:    msg.Timestamp,
			Level:        msg.Level,
			EntryType:    et,
			PayloadType:  msg.PayloadType,
			Node:         c.config.Node,
			Labels:       msg.Labels,
			SearchTokens: msg.SearchTokens,
		}
		if err := validateEntry(&entry); err != nil {
			return err
		}
		if c.config.BeforeSend != nil {
			result := c.config.BeforeSend(&entry)
			if result == nil {
				continue
			}
			entry = *result
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil
	}

	err := c.retryer.Retry(c.ctx, func() error {
		return c.sendMultipart(entries)
	})
	if err != nil {
		c.recordDrop(DropSendError, int64(len(entries)))
		c.recordError(err)
		if c.config.FailsafeMode {
			return nil
		}
		return err
	}

	c.totalSent.Add(int64(len(entries)))
	c.mu.Lock()
	c.lastSendTime = time.Now()
	c.mu.Unlock()
	return nil
}

// --- Background workers ---

func (c *ResilientClient) startWorkers() {
	for i := 0; i < c.config.WorkerCount; i++ {
		c.wg.Add(1)
		go c.worker()
	}
}

func (c *ResilientClient) worker() {
	defer c.wg.Done()
	for {
		// Block until at least one entry is available
		entry := c.queue.DequeueWithContext(c.ctx)
		if entry == nil {
			return
		}

		// Rate limit pre-flight
		c.rateLimitMu.RLock()
		pauseUntil := c.rateLimitPauseUntil
		c.rateLimitMu.RUnlock()
		if time.Now().Before(pauseUntil) {
			// Re-enqueue if possible, otherwise drop
			if !c.queue.Enqueue(*entry) {
				c.recordDrop(DropRateLimited, 1)
			}
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(time.Until(pauseUntil)):
			}
			continue
		}

		// Build a batch: start with the entry we already dequeued
		entries := []models.LogEntry{{
			Message:      entry.Message,
			Timestamp:    entry.Timestamp,
			Level:        entry.Level,
			EntryType:    entry.EntryType,
			PayloadType:  entry.PayloadType,
			Node:         entry.Node,
			Labels:       entry.Labels,
			SearchTokens: entry.SearchTokens,
		}}

		// Drain more entries up to batch size
		if c.config.BatchSize > 1 {
			extra := c.queue.DequeueBatch(c.config.BatchSize - 1)
			for _, e := range extra {
				entries = append(entries, models.LogEntry{
					Message:      e.Message,
					Timestamp:    e.Timestamp,
					Level:        e.Level,
					EntryType:    e.EntryType,
					PayloadType:  e.PayloadType,
					Node:         e.Node,
					Labels:       e.Labels,
					SearchTokens: e.SearchTokens,
				})
			}
		}

		count := int64(len(entries))
		err := c.retryer.Retry(c.ctx, func() error {
			return c.sendMultipart(entries)
		})

		if err != nil {
			c.handleSendError(err, count)
		} else {
			c.totalSent.Add(count)
			c.mu.Lock()
			c.lastSendTime = time.Now()
			c.mu.Unlock()
		}
	}
}

// sendMultipart builds and sends a multipart/mixed request.
func (c *ResilientClient) sendMultipart(entries []models.LogEntry) error {
	body, contentType, err := c.buildMultipartBody(entries)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(c.ctx, "POST", c.endpoints.GetIngestURL(), body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	c.updateRateLimitInfo(resp)

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 60 // default
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if v, err := strconv.Atoi(ra); err == nil {
				retryAfter = v
			}
		}
		c.rateLimitMu.Lock()
		c.rateLimitPauseUntil = time.Now().Add(time.Duration(retryAfter) * time.Second)
		c.rateLimitMu.Unlock()

		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseSize))
		return retry.NewHTTPErrorFromResponse(resp, string(body))
	}

	if resp.StatusCode == http.StatusInsufficientStorage { // 507
		// Block the affected category
		if len(entries) > 0 {
			category := models.EntryTypeCategory(entries[0].EntryType)
			c.quotaMu.Lock()
			c.quotaBlocked[category] = true
			c.quotaMu.Unlock()
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseSize))
		return retry.NewHTTPErrorFromResponse(resp, string(respBody))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseSize))
		return retry.NewHTTPErrorFromResponse(resp, string(respBody))
	}

	return nil
}

// buildMultipartBody is shared with the sync Client via delegation.
func (c *ResilientClient) buildMultipartBody(entries []models.LogEntry) (*bytes.Buffer, string, error) {
	// Read encryptor and keyUUID under lock to avoid races with RenewSession
	c.mu.RLock()
	enc := c.encryptor
	keyID := c.keyUUID
	c.mu.RUnlock()

	builder := &multipartBuilder{
		encryptor:         enc,
		keyUUID:           keyID,
		enableCompression: c.config.EnableCompression,
	}
	return builder.build(entries)
}

func (c *ResilientClient) handleSendError(err error, count int64) {
	if httpErr, ok := err.(*retry.HTTPError); ok {
		if httpErr.IsQuotaExceeded() {
			c.recordDrop(DropQuotaExceeded, count)
		} else if httpErr.IsRateLimited() {
			c.recordDrop(DropRateLimited, count)
		} else {
			c.recordDrop(DropSendError, count)
		}
	} else {
		c.recordDrop(DropNetworkError, count)
	}
	c.recordError(err)
}

func (c *ResilientClient) recordDrop(reason DropReason, count int64) {
	c.mu.Lock()
	c.totalDropped += count
	c.dropReasons[reason] += count
	c.mu.Unlock()
}

func (c *ResilientClient) recordError(err error) {
	c.mu.Lock()
	c.lastSendError = err.Error()
	c.mu.Unlock()
}

func (c *ResilientClient) updateRateLimitInfo(resp *http.Response) {
	c.rateLimitMu.Lock()
	defer c.rateLimitMu.Unlock()
	if v := resp.Header.Get("X-RateLimit-Limit"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.rateLimitLimit = val
		}
	}
	if v := resp.Header.Get("X-RateLimit-Remaining"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.rateLimitRemaining = val
		}
	}
	if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.rateLimitReset = val
		}
	}
}

// --- Convenience methods ---

func (c *ResilientClient) Debug(message string) error     { return c.SendLogWithLevel(message, models.LogLevelDebug) }
func (c *ResilientClient) Info(message string) error      { return c.SendLogWithLevel(message, models.LogLevelInfo) }
func (c *ResilientClient) Warn(message string) error      { return c.SendLogWithLevel(message, models.LogLevelWarning) }
func (c *ResilientClient) Warning(message string) error   { return c.SendLogWithLevel(message, models.LogLevelWarning) }
func (c *ResilientClient) Error(message string) error     { return c.SendLogWithLevel(message, models.LogLevelError) }
func (c *ResilientClient) Fatal(message string) error     { return c.SendLogWithLevel(message, models.LogLevelCritical) }
func (c *ResilientClient) Emergency(message string) error { return c.SendLogWithLevel(message, models.LogLevelEmergency) }
func (c *ResilientClient) Alert(message string) error     { return c.SendLogWithLevel(message, models.LogLevelAlert) }
func (c *ResilientClient) Critical(message string) error  { return c.SendLogWithLevel(message, models.LogLevelCritical) }
func (c *ResilientClient) Notice(message string) error    { return c.SendLogWithLevel(message, models.LogLevelNotice) }

func (c *ResilientClient) DebugWithLabels(message string, labels map[string]string) error   { return c.SendLogWithLevelAndLabels(message, models.LogLevelDebug, labels) }
func (c *ResilientClient) InfoWithLabels(message string, labels map[string]string) error    { return c.SendLogWithLevelAndLabels(message, models.LogLevelInfo, labels) }
func (c *ResilientClient) WarnWithLabels(message string, labels map[string]string) error    { return c.SendLogWithLevelAndLabels(message, models.LogLevelWarning, labels) }
func (c *ResilientClient) ErrorWithLabels(message string, labels map[string]string) error   { return c.SendLogWithLevelAndLabels(message, models.LogLevelError, labels) }
func (c *ResilientClient) FatalWithLabels(message string, labels map[string]string) error   { return c.SendLogWithLevelAndLabels(message, models.LogLevelCritical, labels) }

// --- Stats, lifecycle ---

func (c *ResilientClient) GetStats() ClientStats {
	c.mu.RLock()
	reasons := make(map[DropReason]int64, len(c.dropReasons))
	for k, v := range c.dropReasons {
		reasons[k] = v
	}
	stats := ClientStats{
		EntriesSent:    c.totalSent.Load(),
		EntriesDropped: c.totalDropped,
		EntriesQueued:  c.totalQueued.Load(),
		QueueSize:      int64(c.queue.Size()),
		QueueCapacity:  int64(c.config.QueueSize),
		DropReasons:    reasons,
		LastSendError:  c.lastSendError,
		LastSendTime:   c.lastSendTime,
		HandshakeOK:    c.handshakeOK,
	}
	c.mu.RUnlock()
	return stats
}

func (c *ResilientClient) GetRateLimitInfo() (limit, remaining int, resetTime time.Time) {
	c.rateLimitMu.RLock()
	defer c.rateLimitMu.RUnlock()
	return c.rateLimitLimit, c.rateLimitRemaining, time.Unix(c.rateLimitReset, 0)
}

func (c *ResilientClient) Flush(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if c.queue.IsEmpty() {
			return nil
		}
		select {
		case <-timer.C:
			return fmt.Errorf("flush timeout: queue still has %d entries", c.queue.Size())
		case <-ticker.C:
			continue
		}
	}
}

func (c *ResilientClient) Close() error {
	if c.closed.Load() {
		return nil
	}
	c.closed.Store(true)

	// Flush with 10s default timeout
	_ = c.Flush(10 * time.Second)

	c.cancel()
	c.queue.Close()
	c.wg.Wait()

	// Zero key material
	c.mu.RLock()
	enc := c.encryptor
	c.mu.RUnlock()
	if enc != nil {
		enc.Close()
	}

	return nil
}

func (c *ResilientClient) GetNodeName() string               { return c.config.Node }
func (c *ResilientClient) GetAPIKeyMasked() string            { return maskAPIKey(c.config.APIKey) }
func (c *ResilientClient) GetServerPublicKeyFingerprint() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverKeyFingerprint
}
func (c *ResilientClient) GetServerPublicKeyPEM() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverPublicKeyPEM
}
func (c *ResilientClient) SetTimeout(timeout time.Duration) { c.httpClient.Timeout = timeout }
func (c *ResilientClient) EnableCompressionMode(enable bool) { c.config.EnableCompression = enable }
func (c *ResilientClient) IsCompressionEnabled() bool        { return c.config.EnableCompression }

func (c *ResilientClient) EnableResilientMode(enabled bool) {
	c.config.ResilientMode = enabled
	c.retryer.EnableResilientMode(enabled)
	if enabled && c.endpoints != nil {
		c.retryer.SetHealthCheckURL(c.endpoints.GetHealthURL())
	}
}

func (c *ResilientClient) IsResilientModeEnabled() bool { return c.config.ResilientMode }

func (c *ResilientClient) RenewSession() error {
	handshakeResult, err := handshake.PerformHandshakeWithURL(
		c.endpoints.GetHandshakeURL(), c.config.APIKey, c.httpClient,
	)
	if err != nil {
		return fmt.Errorf("session renewal failed: %w", err)
	}
	newEncryptor := crypto.NewEncryptor(handshakeResult.AESKey)
	// Zero source key material
	for i := range handshakeResult.AESKey {
		handshakeResult.AESKey[i] = 0
	}
	c.mu.Lock()
	oldEncryptor := c.encryptor
	c.encryptor = newEncryptor
	c.keyUUID = handshakeResult.KeyUUID
	c.serverPublicKeyPEM = handshakeResult.ServerPublicKeyPEM
	c.serverKeyFingerprint = handshakeResult.ServerKeyFingerprint
	c.limits = handshakeResult.Limits
	c.mu.Unlock()
	// Zero old key material
	if oldEncryptor != nil {
		oldEncryptor.Close()
	}
	return nil
}

func (c *ResilientClient) HealthCheck() error {
	req, err := http.NewRequest("GET", c.endpoints.GetHealthURL(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %d", resp.StatusCode)
	}
	return nil
}

func (c *ResilientClient) GetVersion() (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", c.endpoints.GetVersionURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseSize))
		return nil, fmt.Errorf("version request failed: %d: %s", resp.StatusCode, string(respBody))
	}
	var version map[string]interface{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxDiscoveryResponseSize)).Decode(&version); err != nil {
		return nil, err
	}
	return version, nil
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
