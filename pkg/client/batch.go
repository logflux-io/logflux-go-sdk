package client

import (
	"context"
	"sync"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// BatchClient wraps the basic client with automatic batching functionality.
// It collects log entries and sends them in batches to improve performance.
// Supports automatic flushing based on batch size or time intervals.
type BatchClient struct {
	client  *Client
	config  *config.BatchConfig
	timer   *time.Timer
	batch   []types.LogEntry
	mu      sync.Mutex
	stopped bool
}

// NewBatchClient creates a new batch client with the given configuration.
// If batchConfig is nil, uses default batch configuration.
// Panics if client is nil.
func NewBatchClient(client *Client, batchConfig *config.BatchConfig) *BatchClient {
	if client == nil {
		panic("client cannot be nil")
	}
	if batchConfig == nil {
		batchConfig = config.DefaultBatchConfig()
	}

	bc := &BatchClient{
		client: client,
		config: batchConfig,
		batch:  make([]types.LogEntry, 0, batchConfig.MaxBatchSize),
	}

	// Start auto-flush timer if enabled
	if batchConfig.AutoFlush && batchConfig.FlushInterval > 0 {
		bc.startFlushTimer()
	}

	return bc
}

// NewBatchUnixClient creates a batch client for Unix socket communication.
// Creates an underlying Unix client and wraps it with batching functionality.
func NewBatchUnixClient(socketPath string, batchConfig *config.BatchConfig) *BatchClient {
	client := NewUnixClient(socketPath)
	return NewBatchClient(client, batchConfig)
}

// NewBatchTCPClient creates a batch client for TCP communication.
// Creates an underlying TCP client and wraps it with batching functionality.
func NewBatchTCPClient(host string, port int, batchConfig *config.BatchConfig) *BatchClient {
	client := NewTCPClient(host, port)
	return NewBatchClient(client, batchConfig)
}

// Connect establishes connection to the agent
func (bc *BatchClient) Connect(ctx context.Context) error {
	return bc.client.Connect(ctx)
}

// Close closes the connection and flushes any remaining entries
func (bc *BatchClient) Close() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.stopped = true

	// Stop timer
	if bc.timer != nil {
		bc.timer.Stop()
	}

	// Flush remaining entries
	if len(bc.batch) > 0 {
		_ = bc.flushBatchLocked() // nolint:errcheck // Ignore error during close
	}

	return bc.client.Close()
}

// SendLog adds a log message to the batch.
// Creates a LogEntry with the provided message and source and adds it to the batch.
// Requires message and source as per API specification.
func (bc *BatchClient) SendLog(message, source string) error {
	entry := types.NewLogEntry(message, source)
	return bc.SendLogEntry(entry)
}

// SendLogEntry adds a log entry to the batch.
// If the batch reaches maximum size, automatically flushes it.
// If the client is stopped, sends the entry directly.
func (bc *BatchClient) SendLogEntry(entry types.LogEntry) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.stopped {
		return bc.client.SendLogEntry(entry) // Send directly if stopped
	}

	// Add to batch
	bc.batch = append(bc.batch, entry)

	// Check if batch is full
	if len(bc.batch) >= bc.config.MaxBatchSize {
		return bc.flushBatchLocked()
	}

	return nil
}

// Flush manually flushes the current batch.
// Sends all pending entries to the agent immediately.
func (bc *BatchClient) Flush() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	return bc.flushBatchLocked()
}

// flushBatchLocked flushes the current batch (must be called with lock held)
func (bc *BatchClient) flushBatchLocked() error {
	if len(bc.batch) == 0 {
		return nil
	}

	// Create a copy of the batch to avoid race conditions with async worker
	batchCopy := make([]types.LogEntry, len(bc.batch))
	copy(batchCopy, bc.batch)

	// Clear the batch regardless of error (avoid infinite retry loops)
	bc.batch = bc.batch[:0]

	// Restart timer
	if bc.config.AutoFlush && !bc.stopped {
		bc.startFlushTimerLocked()
	}

	// Send the batch copy (after releasing lock state)
	return bc.client.SendLogBatch(batchCopy)
}

// startFlushTimer starts the auto-flush timer
func (bc *BatchClient) startFlushTimer() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.startFlushTimerLocked()
}

// startFlushTimerLocked starts the timer (must be called with lock held)
func (bc *BatchClient) startFlushTimerLocked() {
	if bc.timer != nil {
		bc.timer.Stop()
	}

	bc.timer = time.AfterFunc(bc.config.FlushInterval, func() {
		bc.mu.Lock()
		defer bc.mu.Unlock()

		if !bc.stopped && len(bc.batch) > 0 {
			_ = bc.flushBatchLocked() // nolint:errcheck // Ignore error in timer callback
		}
	})
}

// GetStats returns batch client statistics.
// Provides information about pending entries, batch configuration, and flush settings.
func (bc *BatchClient) GetStats() BatchStats {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	return BatchStats{
		PendingEntries: len(bc.batch),
		MaxBatchSize:   bc.config.MaxBatchSize,
		FlushInterval:  bc.config.FlushInterval,
		AutoFlush:      bc.config.AutoFlush,
	}
}

// BatchStats represents batch client statistics.
// Contains information about the current state and configuration of the batch client.
type BatchStats struct {
	PendingEntries int           `json:"pending_entries"`
	MaxBatchSize   int           `json:"max_batch_size"`
	FlushInterval  time.Duration `json:"flush_interval"`
	AutoFlush      bool          `json:"auto_flush"`
}

// Ping delegates to the underlying client for health checking.
// Sends a ping request directly without batching.
func (bc *BatchClient) Ping() (*types.PongResponse, error) {
	return bc.client.Ping()
}

// Authenticate delegates to the underlying client for TCP authentication.
// Only required for TCP connections. Sends auth request directly without batching.
func (bc *BatchClient) Authenticate() (*types.AuthResponse, error) {
	return bc.client.Authenticate()
}
