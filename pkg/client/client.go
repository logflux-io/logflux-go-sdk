package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// Client is a lightweight client for communicating with LogFlux agent local server.
// It supports both Unix socket and TCP connections with automatic retry logic.
// Supports both synchronous and asynchronous sending modes with circuit breaker protection.
type Client struct {
	config         *config.Config
	conn           net.Conn
	asyncChan      chan asyncRequest
	stopChan       chan struct{}
	circuitBreaker *circuitBreaker
	mu             sync.RWMutex
	asyncWorker    sync.WaitGroup
}

// asyncRequest represents an async send request
type asyncRequest struct {
	data     interface{}
	respChan chan error // Channel to send result back (nil for fire-and-forget)
}

// circuitBreakerState represents the state of the circuit breaker
type circuitBreakerState int32

const (
	circuitClosed circuitBreakerState = iota
	circuitOpen
	circuitHalfOpen
)

// circuitBreaker implements circuit breaker pattern to prevent cascading failures
type circuitBreaker struct {
	config          *config.Config
	lastFailureTime int64 // atomic: unix nanoseconds of last failure
	state           int32 // atomic: circuitBreakerState
	failureCount    int32 // atomic: consecutive failure count
}

// NewClient creates a new SDK client with the given configuration.
// If cfg is nil, uses default configuration with Unix socket transport.
func NewClient(cfg *config.Config) *Client {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	client := &Client{
		config: cfg,
		circuitBreaker: &circuitBreaker{
			state:  int32(circuitClosed),
			config: cfg,
		},
	}

	// Initialize async mode if enabled
	if cfg.AsyncMode {
		client.asyncChan = make(chan asyncRequest, cfg.ChannelBuffer)
		client.stopChan = make(chan struct{})
		client.startAsyncWorker()
	}

	return client
}

// NewUnixClient creates a client configured for Unix socket communication.
// If socketPath is empty, uses the default socket path from configuration.
func NewUnixClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = config.DefaultSocketPath
	}

	cfg := config.DefaultConfig()
	cfg.Network = "unix"
	cfg.Address = socketPath
	return NewClient(cfg)
}

// NewTCPClient creates a client configured for TCP communication.
// If host is empty, defaults to "localhost". If port is invalid, defaults to 8080.
// SharedSecret must be set manually for TCP authentication.
func NewTCPClient(host string, port int) *Client {
	if host == "" {
		host = "localhost"
	}
	if port <= 0 || port > 65535 {
		port = 8080 // Default port
	}

	cfg := config.DefaultConfig()
	cfg.Network = "tcp"
	cfg.Address = fmt.Sprintf("%s:%d", host, port)
	return NewClient(cfg)
}

// Connect establishes connection to the agent local server.
// Uses the provided context for timeout and cancellation.
func (c *Client) Connect(ctx context.Context) error {
	var err error

	// Set deadline if timeout is configured
	if c.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.Timeout)
		defer cancel()
	}

	// Establish connection based on network type
	dialer := &net.Dialer{}
	c.conn, err = dialer.DialContext(ctx, c.config.Network, c.config.Address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s://%s: %w", c.config.Network, c.config.Address, err)
	}

	return nil
}

// Close closes the connection to the agent and stops async workers
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop async worker if running
	if c.config.AsyncMode && c.stopChan != nil {
		close(c.stopChan)
		c.asyncWorker.Wait() // Wait for worker to finish
		close(c.asyncChan)
		c.stopChan = nil
		c.asyncChan = nil
	}

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendLog sends a single log message to the agent.
// Creates a LogEntry with the provided message and source, using default values
// for other fields. Requires message and source as per API specification.
func (c *Client) SendLog(message, source string) error {
	entry := types.NewLogEntry(message, source)
	return c.SendLogEntry(entry)
}

// SendLogEntry sends a log entry to the agent.
// Sets timestamp if not already provided and uses retry logic for reliability.
// Uses async mode if configured, otherwise sends synchronously.
func (c *Client) SendLogEntry(entry types.LogEntry) error {
	// Set timestamp if not provided
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	if c.config.AsyncMode {
		return c.sendAsync(entry)
	}
	return c.sendWithRetry(entry)
}

// SendLogBatch sends multiple log entries as a batch.
// Sets timestamps for entries that don't have them and uses retry logic.
// Uses async mode if configured, otherwise sends synchronously.
func (c *Client) SendLogBatch(entries []types.LogEntry) error {
	// Set timestamps if not provided
	for i := range entries {
		if entries[i].Timestamp == "" {
			entries[i].Timestamp = time.Now().UTC().Format(time.RFC3339)
		}
	}

	batch := types.LogBatch{
		Version: types.DefaultProtocolVersion,
		Entries: entries,
	}

	if c.config.AsyncMode {
		return c.sendAsync(batch)
	}
	return c.sendWithRetry(batch)
}

// sendWithRetry sends data with exponential backoff retry logic and circuit breaker protection
func (c *Client) sendWithRetry(data interface{}) error {
	// Check circuit breaker first
	if err := c.circuitBreaker.canExecute(); err != nil {
		return err
	}

	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Use exponential backoff with jitter
			delay := c.config.CalculateBackoffDelay(attempt)
			time.Sleep(delay)
		}

		// Ensure we have a connection
		if c.conn == nil {
			ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
			if err := c.Connect(ctx); err != nil {
				cancel()
				lastErr = err
				continue
			}
			cancel()
		}

		// Send the data
		if err := c.sendData(data); err != nil {
			lastErr = err
			// Close connection on error to force reconnect
			_ = c.Close() // Ignore close error during retry
			c.conn = nil
			continue
		}

		// Success - notify circuit breaker
		c.circuitBreaker.onSuccess()
		return nil
	}

	// All retries failed - notify circuit breaker
	c.circuitBreaker.onFailure()
	return fmt.Errorf("failed to send after %d attempts: %w", c.config.MaxRetries+1, lastErr)
}

// sendData sends JSON data over the connection
func (c *Client) sendData(data interface{}) error {
	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Add newline for line-based protocol
	jsonData = append(jsonData, '\n')

	// Set write timeout if configured
	if c.config.Timeout > 0 {
		if writeErr := c.conn.SetWriteDeadline(time.Now().Add(c.config.Timeout)); writeErr != nil {
			return fmt.Errorf("failed to set write deadline: %w", writeErr)
		}
	}

	// Send data
	_, err = c.conn.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// Ping sends a ping request to the agent for health checking.
// Returns a PongResponse on success or an error if the ping fails.
func (c *Client) Ping() (*types.PongResponse, error) {
	ping := types.NewPingRequest()

	if err := c.sendWithRetry(ping); err != nil {
		return nil, fmt.Errorf("failed to send ping: %w", err)
	}

	// Ping is fire-and-forget - assumes success if no send error
	return &types.PongResponse{Status: "pong"}, nil
}

// Authenticate sends an authentication request for TCP connections.
// Only required for TCP connections. Returns an AuthResponse on success.
func (c *Client) Authenticate() (*types.AuthResponse, error) {
	if c.config.Network != "tcp" {
		return nil, fmt.Errorf("authentication only required for TCP connections")
	}

	if c.config.SharedSecret == "" {
		return nil, fmt.Errorf("shared secret required for TCP authentication")
	}

	authReq := types.NewAuthRequest(c.config.SharedSecret)

	if err := c.sendWithRetry(authReq); err != nil {
		return nil, fmt.Errorf("failed to send auth request: %w", err)
	}

	// Authentication is fire-and-forget - assumes success if no send error
	return &types.AuthResponse{
		Status:  "success",
		Message: "Authentication successful",
	}, nil
}

// startAsyncWorker starts the background goroutine for async sending
func (c *Client) startAsyncWorker() {
	c.asyncWorker.Add(1)
	go func() {
		defer c.asyncWorker.Done()
		for {
			select {
			case req := <-c.asyncChan:
				err := c.sendWithRetry(req.data)
				if req.respChan != nil {
					req.respChan <- err
					close(req.respChan)
				}
			case <-c.stopChan:
				// Drain remaining requests
				for {
					select {
					case req := <-c.asyncChan:
						err := fmt.Errorf("client shutting down")
						if req.respChan != nil {
							req.respChan <- err
							close(req.respChan)
						}
					default:
						return
					}
				}
			}
		}
	}()
}

// sendAsync sends data asynchronously via the worker goroutine
func (c *Client) sendAsync(data interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.asyncChan == nil || c.stopChan == nil {
		return fmt.Errorf("async mode not initialized")
	}

	req := asyncRequest{
		data:     data,
		respChan: nil, // Fire-and-forget
	}

	select {
	case c.asyncChan <- req:
		return nil // Successfully queued
	default:
		return fmt.Errorf("async channel full, dropping log entry")
	}
}

// SendAsyncWithResponse sends data asynchronously and returns a channel for the response
// This allows callers to optionally wait for the send result
func (c *Client) SendAsyncWithResponse(data interface{}) <-chan error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	respChan := make(chan error, 1)

	if c.asyncChan == nil || c.stopChan == nil {
		respChan <- fmt.Errorf("async mode not initialized")
		close(respChan)
		return respChan
	}

	req := asyncRequest{
		data:     data,
		respChan: respChan,
	}

	select {
	case c.asyncChan <- req:
		return respChan
	default:
		respChan <- fmt.Errorf("async channel full, dropping log entry")
		close(respChan)
		return respChan
	}
}

// canExecute checks if the circuit breaker allows execution
func (cb *circuitBreaker) canExecute() error {
	state := circuitBreakerState(atomic.LoadInt32(&cb.state))

	switch state {
	case circuitClosed:
		return nil
	case circuitOpen:
		// Check if timeout has elapsed
		lastFailure := atomic.LoadInt64(&cb.lastFailureTime)
		if time.Since(time.Unix(0, lastFailure)) >= cb.config.CircuitBreakerTimeout {
			// Try to transition to half-open
			if atomic.CompareAndSwapInt32(&cb.state, int32(circuitOpen), int32(circuitHalfOpen)) {
				return nil
			}
		}
		return fmt.Errorf("circuit breaker is open")
	case circuitHalfOpen:
		return nil
	default:
		return nil
	}
}

// onSuccess records a successful operation
func (cb *circuitBreaker) onSuccess() {
	state := circuitBreakerState(atomic.LoadInt32(&cb.state))

	if state == circuitHalfOpen {
		// Successful call in half-open state, close the circuit
		atomic.StoreInt32(&cb.state, int32(circuitClosed))
		atomic.StoreInt32(&cb.failureCount, 0)
	} else if state == circuitClosed {
		// Reset failure count on success
		atomic.StoreInt32(&cb.failureCount, 0)
	}
}

// onFailure records a failed operation
func (cb *circuitBreaker) onFailure() {
	failures := atomic.AddInt32(&cb.failureCount, 1)
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())

	state := circuitBreakerState(atomic.LoadInt32(&cb.state))

	if state == circuitHalfOpen {
		// Failure in half-open state, go back to open
		atomic.StoreInt32(&cb.state, int32(circuitOpen))
	} else if state == circuitClosed && failures >= int32(cb.config.CircuitBreakerThreshold) {
		// Too many failures in closed state, open the circuit
		atomic.StoreInt32(&cb.state, int32(circuitOpen))
	}
}

// CircuitBreakerStats represents circuit breaker status information
type CircuitBreakerStats struct {
	State        string `json:"state"`
	FailureCount int32  `json:"failure_count"`
	IsOpen       bool   `json:"is_open"`
}

// GetCircuitBreakerStats returns the current circuit breaker status
func (c *Client) GetCircuitBreakerStats() CircuitBreakerStats {
	state := circuitBreakerState(atomic.LoadInt32(&c.circuitBreaker.state))
	failureCount := atomic.LoadInt32(&c.circuitBreaker.failureCount)

	var stateStr string
	var isOpen bool

	switch state {
	case circuitClosed:
		stateStr = "closed"
		isOpen = false
	case circuitOpen:
		stateStr = "open"
		isOpen = true
	case circuitHalfOpen:
		stateStr = "half-open"
		isOpen = false
	default:
		stateStr = "unknown"
		isOpen = false
	}

	return CircuitBreakerStats{
		State:        stateStr,
		FailureCount: failureCount,
		IsOpen:       isOpen,
	}
}
