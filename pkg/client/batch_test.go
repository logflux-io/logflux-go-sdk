package client

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

func TestNewBatchClient(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	batchConfig := config.DefaultBatchConfig()

	bc := NewBatchClient(client, batchConfig)

	if bc == nil {
		t.Fatal("Expected BatchClient, got nil")
	}
	if bc.client != client {
		t.Error("BatchClient should wrap the provided client")
	}
	if bc.config != batchConfig {
		t.Error("BatchClient should use the provided config")
	}
}

func TestNewBatchClientWithNilConfig(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")

	bc := NewBatchClient(client, nil)

	if bc == nil {
		t.Fatal("Expected BatchClient, got nil")
	}
	if bc.config == nil {
		t.Error("BatchClient should create default config when nil is passed")
	}
}

func TestNewBatchClientPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when client is nil")
		}
	}()

	NewBatchClient(nil, nil)
}

func TestNewBatchUnixClient(t *testing.T) {
	batchConfig := config.DefaultBatchConfig()

	bc := NewBatchUnixClient("/tmp/test.sock", batchConfig)

	if bc == nil {
		t.Fatal("Expected BatchClient, got nil")
	}
	if bc.config != batchConfig {
		t.Error("BatchClient should use the provided config")
	}
}

func TestNewBatchTCPClient(t *testing.T) {
	batchConfig := config.DefaultBatchConfig()

	bc := NewBatchTCPClient("localhost", 8080, batchConfig)

	if bc == nil {
		t.Fatal("Expected BatchClient, got nil")
	}
	if bc.config != batchConfig {
		t.Error("BatchClient should use the provided config")
	}
}

func TestBatchClientConnect(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	bc := NewBatchClient(client, nil)

	// This will fail to connect, but we're testing the delegation
	ctx := context.Background()
	err := bc.Connect(ctx)

	// Should return an error (connection failed) but not panic
	if err == nil {
		t.Error("Expected connection error for non-existent socket")
	}
}

func TestBatchClientClose(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	bc := NewBatchClient(client, nil)

	// Add some entries to batch
	entry := types.NewLogEntry("test message", "test")
	_ = bc.SendLogEntry(entry) // Error expected due to no connection

	err := bc.Close()

	// Should not return error even without connection
	if err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}

	// Should be marked as stopped
	bc.mu.Lock()
	stopped := bc.stopped
	bc.mu.Unlock()

	if !stopped {
		t.Error("BatchClient should be marked as stopped after Close()")
	}
}

func TestBatchClientSendLog(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	bc := NewBatchClient(client, nil)

	err := bc.SendLog("test message", "test source")

	// Should not return error (retry logic will handle connection failure)
	if err != nil {
		t.Errorf("Expected no immediate error, got: %v", err)
	}

	// Check that entry was added to batch
	bc.mu.Lock()
	batchSize := len(bc.batch)
	bc.mu.Unlock()

	if batchSize != 1 {
		t.Errorf("Expected 1 entry in batch, got %d", batchSize)
	}
}

func TestBatchClientSendLogEntry(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	bc := NewBatchClient(client, nil)

	entry := types.NewLogEntry("test message", "test source")
	err := bc.SendLogEntry(entry)

	if err != nil {
		t.Errorf("Expected no immediate error, got: %v", err)
	}

	// Check that entry was added to batch
	bc.mu.Lock()
	batchSize := len(bc.batch)
	bc.mu.Unlock()

	if batchSize != 1 {
		t.Errorf("Expected 1 entry in batch, got %d", batchSize)
	}
}

func TestBatchClientSendLogEntryWhenStopped(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	bc := NewBatchClient(client, nil)

	// Stop the client
	bc.Close()

	entry := types.NewLogEntry("test message", "test source")
	err := bc.SendLogEntry(entry)

	// Should attempt direct send when stopped (will fail due to no connection)
	if err == nil {
		t.Error("Expected error when sending to stopped client without connection")
	}

	// Batch should remain empty
	bc.mu.Lock()
	batchSize := len(bc.batch)
	bc.mu.Unlock()

	if batchSize != 0 {
		t.Errorf("Expected 0 entries in batch when stopped, got %d", batchSize)
	}
}

func TestBatchFlushLogic(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	batchConfig := config.DefaultBatchConfig()
	batchConfig.MaxBatchSize = 3  // Small batch size for testing
	batchConfig.AutoFlush = false // Disable auto-flush for this test

	bc := NewBatchClient(client, batchConfig)

	// Add entries up to batch size - 1
	for i := 0; i < 2; i++ {
		entry := types.NewLogEntry("test message", "test source")
		_ = bc.SendLogEntry(entry) // Error expected due to no connection
	}

	// Check batch size
	bc.mu.Lock()
	batchSize := len(bc.batch)
	bc.mu.Unlock()

	if batchSize != 2 {
		t.Errorf("Expected 2 entries in batch, got %d", batchSize)
	}

	// Add one more entry to trigger flush
	entry := types.NewLogEntry("final message", "test source")
	_ = bc.SendLogEntry(entry) // This should trigger flush, error expected

	// Give a moment for async operations
	time.Sleep(10 * time.Millisecond)

	// Batch should be cleared after flush attempt (even if it fails)
	bc.mu.Lock()
	batchSize = len(bc.batch)
	bc.mu.Unlock()

	if batchSize != 0 {
		t.Errorf("Expected 0 entries in batch after flush, got %d", batchSize)
	}
}

func TestBatchClientFlush(t *testing.T) {
	// Use sync mode for the underlying client to test error handling
	cfg := config.DefaultConfig()
	cfg.AsyncMode = false
	client := NewClient(cfg)
	client.config.Network = "unix"
	client.config.Address = "/tmp/test.sock"
	bc := NewBatchClient(client, nil)

	// Add some entries
	for i := 0; i < 3; i++ {
		entry := types.NewLogEntry("test message", "test source")
		_ = bc.SendLogEntry(entry) // Error expected due to no connection
	}

	// Manually flush
	err := bc.Flush()

	// Will return error due to no connection, but should clear batch
	if err == nil {
		t.Error("Expected flush error due to no connection")
	}

	// Check that batch was cleared
	bc.mu.Lock()
	batchSize := len(bc.batch)
	bc.mu.Unlock()

	if batchSize != 0 {
		t.Errorf("Expected 0 entries in batch after flush, got %d", batchSize)
	}
}

func TestBatchClientGetStats(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	batchConfig := config.DefaultBatchConfig()
	batchConfig.MaxBatchSize = 100
	batchConfig.FlushInterval = 5 * time.Second
	batchConfig.AutoFlush = true

	bc := NewBatchClient(client, batchConfig)

	// Add some entries
	for i := 0; i < 5; i++ {
		entry := types.NewLogEntry("test message", "test source")
		_ = bc.SendLogEntry(entry) // Error expected due to no connection
	}

	stats := bc.GetStats()

	if stats.PendingEntries != 5 {
		t.Errorf("Expected 5 pending entries, got %d", stats.PendingEntries)
	}
	if stats.MaxBatchSize != 100 {
		t.Errorf("Expected MaxBatchSize 100, got %d", stats.MaxBatchSize)
	}
	if stats.FlushInterval != 5*time.Second {
		t.Errorf("Expected FlushInterval 5s, got %v", stats.FlushInterval)
	}
	if !stats.AutoFlush {
		t.Error("Expected AutoFlush to be true")
	}
}

func TestBatchClientAutoFlushTimer(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	batchConfig := config.DefaultBatchConfig()
	batchConfig.MaxBatchSize = 100                     // Large batch size so flush is timer-driven
	batchConfig.FlushInterval = 100 * time.Millisecond // Short interval for testing
	batchConfig.AutoFlush = true

	bc := NewBatchClient(client, batchConfig)

	// Add an entry
	entry := types.NewLogEntry("test message", "test source")
	_ = bc.SendLogEntry(entry) // Error expected due to no connection

	// Verify entry is in batch
	bc.mu.Lock()
	batchSize := len(bc.batch)
	bc.mu.Unlock()

	if batchSize != 1 {
		t.Fatalf("Expected 1 entry in batch, got %d", batchSize)
	}

	// Wait for auto-flush timer
	time.Sleep(200 * time.Millisecond)

	// Batch should be flushed by timer
	bc.mu.Lock()
	batchSize = len(bc.batch)
	bc.mu.Unlock()

	if batchSize != 0 {
		t.Errorf("Expected 0 entries in batch after auto-flush, got %d", batchSize)
	}
}

func TestBatchClientConcurrency(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	batchConfig := config.DefaultBatchConfig()
	batchConfig.MaxBatchSize = 100 // Larger than total entries to prevent auto-flush
	batchConfig.AutoFlush = false  // Disable auto-flush for predictable testing

	bc := NewBatchClient(client, batchConfig)

	const numGoroutines = 10
	const entriesPerGoroutine = 5
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < entriesPerGoroutine; j++ {
				entry := types.NewLogEntry("concurrent message", "test source")
				_ = bc.SendLogEntry(entry) // Error expected due to no connection
			}
		}(i)
	}

	wg.Wait()

	// Check final batch size
	bc.mu.Lock()
	batchSize := len(bc.batch)
	bc.mu.Unlock()

	expectedEntries := numGoroutines * entriesPerGoroutine
	if batchSize != expectedEntries {
		t.Errorf("Expected %d entries in batch, got %d", expectedEntries, batchSize)
	}
}

func TestBatchClientPing(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	bc := NewBatchClient(client, nil)

	// Should delegate to underlying client
	_, err := bc.Ping()

	// Will fail due to connection failure, but should not panic
	if err == nil {
		t.Error("Expected ping error due to connection failure")
	}
}

func TestBatchClientAuthenticate(t *testing.T) {
	client := NewTCPClient("localhost", 8080)
	bc := NewBatchClient(client, nil)

	// Should delegate to underlying client - this will fail but not panic
	_, err := bc.Authenticate()

	// Will fail due to no connection, but should not panic
	if err == nil {
		t.Error("Expected auth error due to no connection")
	}
}

func TestBatchClientTimerCleanup(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")
	batchConfig := config.DefaultBatchConfig()
	batchConfig.FlushInterval = 1 * time.Second
	batchConfig.AutoFlush = true

	bc := NewBatchClient(client, batchConfig)

	// Add entry to start timer
	entry := types.NewLogEntry("test message", "test source")
	_ = bc.SendLogEntry(entry) // Error expected due to no connection

	// Verify timer exists
	bc.mu.Lock()
	hasTimer := bc.timer != nil
	bc.mu.Unlock()

	if !hasTimer {
		t.Error("Expected timer to be created")
	}

	// Close should stop timer
	bc.Close()

	// Timer should be stopped (but pointer may still exist)
	bc.mu.Lock()
	stopped := bc.stopped
	bc.mu.Unlock()

	if !stopped {
		t.Error("Expected client to be stopped after Close()")
	}
}
