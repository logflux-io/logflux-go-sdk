package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/crypto"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/sdkversion"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/discovery"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/handshake"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

var userAgent = "logflux-go-sdk/" + sdkversion.Version

// Response body size limits to prevent unbounded reads.
const (
	maxErrorResponseSize     = 1 << 20 // 1 MiB for error response bodies
	maxDiscoveryResponseSize = 64 << 10 // 64 KiB for handshake/discovery responses
)

// BeforeSendFunc is called before each entry is sent. Return nil to drop the entry.
type BeforeSendFunc func(entry *models.LogEntry) *models.LogEntry

// ClientConfig holds configuration for the sync client.
type ClientConfig struct {
	APIKey            string
	Node              string
	Environment       string
	LogGroup          string
	EnableCompression bool
	HTTPTimeout       time.Duration
	DiscoveryTimeout  time.Duration
	CustomEndpointURL string
	BeforeSend        BeforeSendFunc
}

// Client is a synchronous LogFlux client (blocks until HTTP response).
// Suitable for scripts, CLI tools, Lambda, and testing.
type Client struct {
	node              string
	apiKey            string
	environment       string
	logGroup          string
	encryptor         *crypto.Encryptor
	httpClient        *http.Client
	keyUUID           string
	enableCompression bool
	beforeSend        BeforeSendFunc
	limits            *handshake.HandshakeLimits

	serverPublicKeyPEM   string
	serverKeyFingerprint string

	rateLimitLimit     int
	rateLimitRemaining int
	rateLimitReset     int64

	discoveryClient *discovery.DiscoveryClient
	endpoints       *discovery.EndpointInfo
}

func NewClient(apiKey, node string) (*Client, error) {
	cfg := ClientConfig{
		APIKey:            apiKey,
		Node:              node,
		HTTPTimeout:       30 * time.Second,
		DiscoveryTimeout:  10 * time.Second,
		EnableCompression: true,
	}
	return NewClientWithConfig(cfg)
}

func NewClientWithCustomEndpoint(apiKey, customEndpointURL, node string) (*Client, error) {
	cfg := ClientConfig{
		APIKey:            apiKey,
		Node:              node,
		CustomEndpointURL: customEndpointURL,
		HTTPTimeout:       30 * time.Second,
		EnableCompression: true,
	}
	return NewClientWithConfig(cfg)
}

func NewClientWithConfig(cfg ClientConfig) (*Client, error) {
	if err := config.ValidateAPIKey(cfg.APIKey); err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	if httpClient.Timeout == 0 {
		httpClient.Timeout = 30 * time.Second
	}

	var discoveryClient *discovery.DiscoveryClient
	var endpoints *discovery.EndpointInfo

	if cfg.CustomEndpointURL != "" {
		discoveryClient = discovery.NewDiscoveryClient(discovery.DiscoveryConfig{
			APIKey:     cfg.APIKey,
			Timeout:    10 * time.Second,
			HTTPClient: httpClient,
		})
		endpoints = discoveryClient.SetCustomEndpoint(cfg.CustomEndpointURL)
	} else {
		discoveryTimeout := cfg.DiscoveryTimeout
		if discoveryTimeout == 0 {
			discoveryTimeout = 10 * time.Second
		}
		discoveryClient = discovery.NewDiscoveryClient(discovery.DiscoveryConfig{
			APIKey:     cfg.APIKey,
			Timeout:    discoveryTimeout,
			HTTPClient: httpClient,
		})
		ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
		defer cancel()
		var err error
		endpoints, err = discoveryClient.DiscoverEndpoints(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("endpoint discovery failed: %w", err)
		}
	}

	nodeName := cfg.Node
	if nodeName == "" {
		if hostname, err := os.Hostname(); err == nil {
			nodeName = hostname
		} else {
			nodeName = "unknown"
		}
	}

	handshakeURL := endpoints.GetHandshakeURL()
	handshakeResult, err := handshake.PerformHandshakeWithURL(handshakeURL, cfg.APIKey, httpClient)
	if err != nil {
		if errors.Is(err, handshake.ErrIngestorUnavailable) {
			return nil, fmt.Errorf("cannot connect to %s: %v", endpoints.BaseURL, err)
		}
		return nil, fmt.Errorf("handshake failed: %w", err)
	}

	enc := crypto.NewEncryptor(handshakeResult.AESKey)
	// Zero source key material after the encryptor has its own copy
	for i := range handshakeResult.AESKey {
		handshakeResult.AESKey[i] = 0
	}

	return &Client{
		node:                 nodeName,
		apiKey:               cfg.APIKey,
		environment:          cfg.Environment,
		logGroup:             cfg.LogGroup,
		encryptor:            enc,
		httpClient:           httpClient,
		keyUUID:              handshakeResult.KeyUUID,
		enableCompression:    cfg.EnableCompression,
		beforeSend:           cfg.BeforeSend,
		limits:               handshakeResult.Limits,
		serverPublicKeyPEM:   handshakeResult.ServerPublicKeyPEM,
		serverKeyFingerprint: handshakeResult.ServerKeyFingerprint,
		discoveryClient:      discoveryClient,
		endpoints:            endpoints,
	}, nil
}

func NewClientFromEnv(node string) (*Client, error) {
	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return NewClientWithConfig(ClientConfig{
		APIKey:            cfg.APIKey,
		Node:              node,
		Environment:       cfg.Environment,
		LogGroup:          cfg.LogGroup,
		EnableCompression: cfg.EnableCompression,
		DiscoveryTimeout:  10 * time.Second,
		HTTPTimeout: func() time.Duration {
			if cfg.HTTPTimeout > 0 {
				return time.Duration(cfg.HTTPTimeout) * time.Second
			}
			return 30 * time.Second
		}(),
	})
}

// LogMessage represents a log message for batch operations.
type LogMessage struct {
	Message      string
	Timestamp    time.Time
	Level        int
	EntryType    int
	PayloadType  int
	Labels       map[string]string
	SearchTokens []string
}

// --- Send methods ---

func (c *Client) SendLog(message string) error {
	return c.SendLogWithLevel(message, models.LogLevelInfo)
}

func (c *Client) SendLogWithLevel(message string, level int) error {
	return c.SendLogWithTimestampAndLevel(message, time.Now(), level)
}

func (c *Client) SendLogWithTimestamp(message string, timestamp time.Time) error {
	return c.SendLogWithTimestampAndLevel(message, timestamp, models.LogLevelInfo)
}

func (c *Client) SendLogWithTimestampAndLevel(message string, timestamp time.Time, level int) error {
	return c.SendEntry(models.LogEntry{
		Message:   message,
		Timestamp: timestamp,
		Level:     level,
		EntryType: models.EntryTypeLog,
		Node:      c.node,
	})
}

func (c *Client) SendLogWithTimestampLevelAndLabels(message string, timestamp time.Time, level int, labels map[string]string) error {
	return c.SendEntry(models.LogEntry{
		Message:   message,
		Timestamp: timestamp,
		Level:     level,
		EntryType: models.EntryTypeLog,
		Node:      c.node,
		Labels:    labels,
	})
}

func (c *Client) SendLogWithTimestampLevelTypeAndLabels(message string, timestamp time.Time, level, entryType int, labels map[string]string) error {
	if entryType == 0 {
		entryType = models.EntryTypeLog
	}
	return c.SendEntry(models.LogEntry{
		Message:   message,
		Timestamp: timestamp,
		Level:     level,
		EntryType: entryType,
		Node:      c.node,
		Labels:    labels,
	})
}

// SendEntry sends a single entry using multipart/mixed.
func (c *Client) SendEntry(entry models.LogEntry) error {
	if err := validateEntry(&entry); err != nil {
		return err
	}
	if entry.Node == "" {
		entry.Node = c.node
	}
	if c.beforeSend != nil {
		result := c.beforeSend(&entry)
		if result == nil {
			return nil // dropped by before_send
		}
		entry = *result
	}

	body, contentType, err := c.buildMultipartBody([]models.LogEntry{entry})
	if err != nil {
		return err
	}
	return c.doIngest(body, contentType)
}

// SendLogBatch sends multiple entries in a single multipart/mixed request.
func (c *Client) SendLogBatch(messages []LogMessage) error {
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
			Node:         c.node,
			Labels:       msg.Labels,
			SearchTokens: msg.SearchTokens,
		}
		if err := validateEntry(&entry); err != nil {
			return err
		}
		if c.beforeSend != nil {
			result := c.beforeSend(&entry)
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

	body, contentType, err := c.buildMultipartBody(entries)
	if err != nil {
		return err
	}
	return c.doIngest(body, contentType)
}

// buildMultipartBody creates a multipart/mixed request body.
func (c *Client) buildMultipartBody(entries []models.LogEntry) (*bytes.Buffer, string, error) {
	b := &multipartBuilder{
		encryptor:         c.encryptor,
		keyUUID:           c.keyUUID,
		enableCompression: c.enableCompression,
	}
	return b.build(entries)
}

func (c *Client) doIngest(body *bytes.Buffer, contentType string) error {
	req, err := http.NewRequest("POST", c.endpoints.GetIngestURL(), body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: cannot connect to %s: %v", handshake.ErrIngestorUnavailable, c.endpoints.BaseURL, err)
	}
	defer resp.Body.Close()

	c.updateRateLimitInfo(resp)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseSize))
		return parseErrorResponse(resp.StatusCode, respBody, resp.Header)
	}

	return nil
}

func parseErrorResponse(statusCode int, body []byte, headers http.Header) error {
	var errResp models.ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Code != "" {
		msg := fmt.Sprintf("ingest failed [%s]: %s", errResp.Error.Code, errResp.Error.Message)
		if errResp.Error.Details != "" {
			msg += " - " + errResp.Error.Details
		}
		if statusCode == http.StatusTooManyRequests {
			if ra := headers.Get("Retry-After"); ra != "" {
				msg += fmt.Sprintf(" (retry after %s seconds)", ra)
			}
		}
		return errors.New(msg)
	}
	return fmt.Errorf("ingest failed with status %d: %s", statusCode, string(body))
}

// --- Convenience methods ---

func (c *Client) Debug(message string) error   { return c.SendLogWithLevel(message, models.LogLevelDebug) }
func (c *Client) Info(message string) error    { return c.SendLogWithLevel(message, models.LogLevelInfo) }
func (c *Client) Warn(message string) error    { return c.SendLogWithLevel(message, models.LogLevelWarning) }
func (c *Client) Error(message string) error   { return c.SendLogWithLevel(message, models.LogLevelError) }
func (c *Client) Fatal(message string) error   { return c.SendLogWithLevel(message, models.LogLevelCritical) }

func (c *Client) Emergency(message string) error { return c.SendLogWithLevel(message, models.LogLevelEmergency) }
func (c *Client) Alert(message string) error     { return c.SendLogWithLevel(message, models.LogLevelAlert) }
func (c *Client) Critical(message string) error  { return c.SendLogWithLevel(message, models.LogLevelCritical) }
func (c *Client) Warning(message string) error   { return c.SendLogWithLevel(message, models.LogLevelWarning) }
func (c *Client) Notice(message string) error    { return c.SendLogWithLevel(message, models.LogLevelNotice) }

func (c *Client) SendLogWithLabels(message string, labels map[string]string) error {
	return c.SendLogWithTimestampLevelAndLabels(message, time.Now(), models.LogLevelInfo, labels)
}

func (c *Client) SendLogWithLevelAndLabels(message string, level int, labels map[string]string) error {
	return c.SendLogWithTimestampLevelAndLabels(message, time.Now(), level, labels)
}

func (c *Client) SendLogWithEntryType(message string, level, entryType int) error {
	return c.SendLogWithTimestampLevelTypeAndLabels(message, time.Now(), level, entryType, nil)
}

func (c *Client) Close() error {
	if c.encryptor != nil {
		c.encryptor.Close()
	}
	return nil
}
func (c *Client) SetTimeout(timeout time.Duration)               { c.httpClient.Timeout = timeout }
func (c *Client) GetNodeName() string                            { return c.node }
func (c *Client) GetAPIKeyMasked() string                        { return maskAPIKey(c.apiKey) }
func (c *Client) GetServerPublicKeyFingerprint() string          { return c.serverKeyFingerprint }
func (c *Client) GetServerPublicKeyPEM() string                  { return c.serverPublicKeyPEM }
func (c *Client) EnableCompressionMode(enable bool)              { c.enableCompression = enable }
func (c *Client) IsCompressionEnabled() bool                     { return c.enableCompression }

func (c *Client) updateRateLimitInfo(resp *http.Response) {
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

func (c *Client) GetRateLimitInfo() (limit, remaining int, resetTime int64) {
	return c.rateLimitLimit, c.rateLimitRemaining, c.rateLimitReset
}

func (c *Client) GetVersion() (map[string]interface{}, error) {
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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseSize))
		return nil, fmt.Errorf("version request failed: %d: %s", resp.StatusCode, string(body))
	}
	var version map[string]interface{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxDiscoveryResponseSize)).Decode(&version); err != nil {
		return nil, err
	}
	return version, nil
}

func (c *Client) HealthCheck() error {
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

// maskAPIKey returns a masked version of the API key showing only the last 4 characters.
// Example: "eu-lf_abc123xyz" -> "eu-lf_****xyz"
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	idx := strings.Index(key, "lf_")
	if idx >= 0 {
		prefix := key[:idx+3] // e.g. "eu-lf_"
		return prefix + "****" + key[len(key)-4:]
	}
	return "****" + key[len(key)-4:]
}

// --- Validation ---

func validateEntry(entry *models.LogEntry) error {
	if entry.Message == "" {
		return fmt.Errorf("message cannot be empty")
	}
	if len(entry.Message) > 1<<20 { // 1 MiB
		return fmt.Errorf("message exceeds 1 MiB limit")
	}
	if !entry.Timestamp.IsZero() {
		if err := validateTimestamp(entry.Timestamp); err != nil {
			return err
		}
	}
	if entry.Level != 0 {
		if err := validateLogLevel(entry.Level); err != nil {
			return err
		}
	}
	if entry.EntryType != 0 {
		if err := validateEntryType(entry.EntryType); err != nil {
			return err
		}
	}
	return validateLabels(entry.Labels)
}

func validateTimestamp(ts time.Time) error {
	now := time.Now().UTC()
	if ts.After(now.Add(1 * time.Minute)) {
		return fmt.Errorf("timestamp cannot be more than 1 minute in the future")
	}
	if ts.Before(now.AddDate(-1, 0, 0)) {
		return fmt.Errorf("timestamp cannot be older than 1 year")
	}
	return nil
}

func validateLogLevel(level int) error {
	if level < 1 || level > 8 {
		return fmt.Errorf("loglevel must be between 1 and 8")
	}
	return nil
}

func validateEntryType(entryType int) error {
	if entryType == 0 {
		return nil
	}
	if entryType < 1 || entryType > 7 {
		return fmt.Errorf("entry_type must be between 1 and 7")
	}
	return nil
}

func validateLabels(labels map[string]string) error {
	if labels == nil {
		return nil
	}
	if len(labels) > 20 {
		return fmt.Errorf("labels exceed maximum of 20")
	}
	disallowed := map[string]struct{}{
		"customer": {}, "application": {}, "node": {}, "timestamp": {}, "loglevel": {}, "key_uuid": {},
	}
	for k, v := range labels {
		if len(k) == 0 {
			return fmt.Errorf("label key cannot be empty")
		}
		if len(k) > 64 {
			return fmt.Errorf("label key '%s' exceeds 64 characters", k)
		}
		if len(v) > 256 {
			return fmt.Errorf("label '%s' value exceeds 256 characters", k)
		}
		if _, found := disallowed[strings.ToLower(k)]; found {
			return fmt.Errorf("label key '%s' is disallowed", k)
		}
	}
	return nil
}
