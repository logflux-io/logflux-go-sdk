package config

import (
	"math/rand"
	"time"
)

// Default configuration constants
const (
	// Network defaults
	DefaultNetwork    = "unix"
	DefaultSocketPath = "/tmp/logflux-agent.sock"

	// Timeout defaults
	DefaultTimeout         = 10 * time.Second
	DefaultRetryDelay      = 100 * time.Millisecond // Reduced for tests
	DefaultMaxRetryDelay   = 5 * time.Second        // Reduced for tests
	DefaultRetryMultiplier = 2.0
	DefaultJitterPercent   = 0.1

	// Batch defaults
	DefaultBatchSize     = 10
	DefaultMaxBatchSize  = 10
	DefaultFlushInterval = 5 * time.Second

	// Retry defaults
	DefaultMaxRetries = 3

	// Async defaults
	DefaultAsyncMode     = true
	DefaultChannelBuffer = 1000

	// Circuit breaker defaults
	DefaultCircuitBreakerThreshold = 5                // Failures before opening
	DefaultCircuitBreakerTimeout   = 30 * time.Second // How long to stay open

	// Batch size limits (from API spec)
	MinBatchSize = 1
	MaxBatchSize = 100
)

// Config holds configuration for the SDK client
type Config struct {
	// Connection settings
	Network       string        // "unix" or "tcp"
	Address       string        // Socket path for unix, host:port for tcp
	SharedSecret  string        // Optional shared secret for authentication
	Timeout       time.Duration // Connection timeout
	FlushInterval time.Duration // Time to wait before sending partial batch
	BatchSize     int           // Number of messages to batch before sending

	// Retry settings with exponential backoff
	MaxRetries      int           // Maximum retry attempts
	RetryDelay      time.Duration // Initial delay between retries
	MaxRetryDelay   time.Duration // Maximum delay between retries
	RetryMultiplier float64       // Backoff multiplier (e.g., 2.0 for doubling)
	JitterPercent   float64       // Jitter as percentage (0.0-1.0)

	// Async settings
	AsyncMode     bool // Enable async/non-blocking mode
	ChannelBuffer int  // Buffer size for async channel

	// Circuit breaker settings
	CircuitBreakerThreshold int           // Consecutive failures before opening circuit
	CircuitBreakerTimeout   time.Duration // How long to keep circuit open
}

// BatchConfig holds configuration for batch processing
type BatchConfig struct {
	MaxBatchSize  int           // Maximum entries per batch
	FlushInterval time.Duration // Time to wait before sending partial batch
	AutoFlush     bool          // Automatically flush batches
}

// DefaultConfig returns a default configuration for Unix socket connection
func DefaultConfig() *Config {
	return &Config{
		Network:                 DefaultNetwork,
		Address:                 DefaultSocketPath,
		Timeout:                 DefaultTimeout,
		SharedSecret:            "",
		BatchSize:               DefaultBatchSize,
		FlushInterval:           DefaultFlushInterval,
		MaxRetries:              DefaultMaxRetries,
		RetryDelay:              DefaultRetryDelay,
		MaxRetryDelay:           DefaultMaxRetryDelay,
		RetryMultiplier:         DefaultRetryMultiplier,
		JitterPercent:           DefaultJitterPercent,
		AsyncMode:               DefaultAsyncMode,
		ChannelBuffer:           DefaultChannelBuffer,
		CircuitBreakerThreshold: DefaultCircuitBreakerThreshold,
		CircuitBreakerTimeout:   DefaultCircuitBreakerTimeout,
	}
}

// DefaultBatchConfig returns default batch configuration
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		MaxBatchSize:  DefaultMaxBatchSize,
		FlushInterval: DefaultFlushInterval,
		AutoFlush:     true,
	}
}

// CalculateBackoffDelay calculates the next retry delay using exponential backoff with jitter
func (c *Config) CalculateBackoffDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return c.RetryDelay
	}

	// Calculate exponential backoff: delay * multiplier^attempt
	delay := float64(c.RetryDelay)
	for i := 0; i < attempt; i++ {
		delay *= c.RetryMultiplier
	}

	// Cap at maximum delay
	if maxDelay := float64(c.MaxRetryDelay); delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter: ±(delay * jitterPercent)
	if c.JitterPercent > 0 {
		jitter := delay * c.JitterPercent
		// Random value between -jitter and +jitter
		jitterAmount := (rand.Float64()*2 - 1) * jitter
		delay += jitterAmount
	}

	// Ensure we don't go below the initial delay
	if finalDelay := time.Duration(delay); finalDelay < c.RetryDelay {
		return c.RetryDelay
	}

	return time.Duration(delay)
}
