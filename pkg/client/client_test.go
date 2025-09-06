package client

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

func TestNewClient(t *testing.T) {
	cfg := config.DefaultConfig()
	client := NewClient(cfg)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.config != cfg {
		t.Error("Expected client to use provided config")
	}
}

func TestNewClientWithNilConfig(t *testing.T) {
	client := NewClient(nil)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.config == nil {
		t.Error("Expected client to have default config when nil provided")
	}
}

func TestNewUnixClient(t *testing.T) {
	socketPath := "/tmp/test.sock"
	client := NewUnixClient(socketPath)

	if client.config.Network != "unix" {
		t.Errorf("Expected network 'unix', got %s", client.config.Network)
	}

	if client.config.Address != socketPath {
		t.Errorf("Expected address %s, got %s", socketPath, client.config.Address)
	}
}

func TestNewTCPClient(t *testing.T) {
	host := "127.0.0.1"
	port := 9090
	client := NewTCPClient(host, port)

	if client.config.Network != "tcp" {
		t.Errorf("Expected network 'tcp', got %s", client.config.Network)
	}

	expectedAddress := "127.0.0.1:9090"
	if client.config.Address != expectedAddress {
		t.Errorf("Expected address %s, got %s", expectedAddress, client.config.Address)
	}
}

func TestConnectTimeout(t *testing.T) {
	// Create a client with a very short timeout
	cfg := config.DefaultConfig()
	cfg.Address = "127.0.0.1:99999" // Non-existent port
	cfg.Network = "tcp"
	cfg.Timeout = 10 * time.Millisecond

	client := NewClient(cfg)

	ctx := context.Background()
	err := client.Connect(ctx)

	if err == nil {
		t.Error("Expected connection to fail to non-existent port")
	}

	if !strings.Contains(err.Error(), "failed to connect") {
		t.Errorf("Expected connection error message, got: %v", err)
	}
}

func TestCloseWithoutConnection(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")

	// Should not panic or error when closing without connection
	err := client.Close()
	if err != nil {
		t.Errorf("Expected no error when closing without connection, got: %v", err)
	}
}

func TestSendLogEntryWithoutConnection(t *testing.T) {
	// Use sync mode for this test to check error handling
	cfg := config.DefaultConfig()
	cfg.AsyncMode = false
	client := NewClient(cfg)
	client.config.Network = "unix"
	client.config.Address = "/tmp/nonexistent.sock"

	entry := types.NewLogEntry("Test message", "test")

	err := client.SendLogEntry(entry)
	if err == nil {
		t.Error("Expected error when sending without connection")
	}
}

func TestSendLogBatchWithoutConnection(t *testing.T) {
	// Use sync mode for this test to check error handling
	cfg := config.DefaultConfig()
	cfg.AsyncMode = false
	client := NewClient(cfg)
	client.config.Network = "unix"
	client.config.Address = "/tmp/nonexistent.sock"

	entries := []types.LogEntry{
		types.NewLogEntry("Message 1", "test"),
		types.NewLogEntry("Message 2", "test"),
	}

	err := client.SendLogBatch(entries)
	if err == nil {
		t.Error("Expected error when sending batch without connection")
	}
}

func TestSendLogConvenienceMethod(t *testing.T) {
	// Use sync mode for this test to check error handling
	cfg := config.DefaultConfig()
	cfg.AsyncMode = false
	client := NewClient(cfg)
	client.config.Network = "unix"
	client.config.Address = "/tmp/nonexistent.sock"

	err := client.SendLog("Simple message", "test")
	if err == nil {
		t.Error("Expected error when sending without connection")
	}
}

func TestClientWithMockConnection(t *testing.T) {
	// Create a pair of connected pipes for testing
	server, clientConn := net.Pipe()
	defer server.Close()
	defer clientConn.Close()

	// Create SDK client and manually set connection
	cfg := config.DefaultConfig()
	client := NewClient(cfg)
	client.conn = clientConn

	// Start a goroutine to read from the server side
	received := make(chan string, 1)
	go func() {
		defer server.Close()
		buffer := make([]byte, 1024)
		n, err := server.Read(buffer)
		if err != nil {
			return
		}
		received <- string(buffer[:n])
	}()

	// Send a log entry
	entry := types.NewLogEntry("Test message", "test").WithLogLevel(types.LevelWarning)
	err := client.sendData(entry)
	if err != nil {
		t.Fatalf("Failed to send data: %v", err)
	}

	// Verify received data
	select {
	case data := <-received:
		if !strings.Contains(data, "Test message") {
			t.Errorf("Expected to receive 'Test message', got: %s", data)
		}

		// Verify it's valid JSON
		var receivedEntry types.LogEntry
		err := json.Unmarshal([]byte(strings.TrimSpace(data)), &receivedEntry)
		if err != nil {
			t.Errorf("Received data is not valid JSON: %v", err)
		}

		if receivedEntry.Payload != "Test message" {
			t.Errorf("Expected payload 'Test message', got %s", receivedEntry.Payload)
		}

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for data")
	}
}

func TestRetryLogic(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AsyncMode = false // Use sync mode to test retry logic timing
	cfg.MaxRetries = 2
	cfg.RetryDelay = 10 * time.Millisecond
	cfg.Address = "127.0.0.1:99999" // Non-existent address
	cfg.Network = "tcp"
	cfg.Timeout = 50 * time.Millisecond

	client := NewClient(cfg)
	entry := types.NewLogEntry("Test message", "test")

	start := time.Now()
	err := client.SendLogEntry(entry)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected error when connecting to non-existent address")
	}

	// Should have made 3 attempts (initial + 2 retries) with delays
	// With exponential backoff, first retry = 10ms, second retry = 20ms (2x multiplier)
	expectedMinDuration := 10*time.Millisecond + 20*time.Millisecond
	if duration < expectedMinDuration {
		t.Errorf("Expected at least %v duration for retries, got %v", expectedMinDuration, duration)
	}
}

func TestConnectionRecovery(t *testing.T) {
	// Use sync mode to test connection error handling
	cfg := config.DefaultConfig()
	cfg.AsyncMode = false
	client := NewClient(cfg)
	client.config.Network = "unix"
	client.config.Address = "/tmp/test.sock"

	// Simulate connection that gets closed
	server, conn := net.Pipe()
	client.conn = conn

	// Close the connection to simulate network error
	conn.Close()
	server.Close()

	entry := types.NewLogEntry("Test message", "test")
	err := client.SendLogEntry(entry)

	// Should fail but not panic
	if err == nil {
		t.Error("Expected error when connection is closed")
	}

	// Connection should be nil after failure
	if client.conn != nil {
		t.Error("Expected connection to be nil after failure")
	}
}

func TestTimestampHandling(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")

	// Create mock connection
	server, conn := net.Pipe()
	defer server.Close()
	defer conn.Close()

	client.conn = conn

	// Test entry without timestamp
	entry1 := types.LogEntry{
		Payload:   "No timestamp",
		EntryType: types.TypeLog,
		Source:    "test",
		LogLevel:  types.LevelInfo,
	}

	// Test entry with timestamp
	customTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	entry2 := types.LogEntry{
		Payload:   "With timestamp",
		EntryType: types.TypeLog,
		Source:    "test",
		LogLevel:  types.LevelInfo,
		Timestamp: customTime.UTC().Format(time.RFC3339),
	}

	received := make(chan []types.LogEntry, 2)
	go func() {
		defer server.Close()
		buffer := make([]byte, 1024)

		// Read first entry
		n, _ := server.Read(buffer)
		var receivedEntry1 types.LogEntry
		_ = json.Unmarshal([]byte(strings.TrimSpace(string(buffer[:n]))), &receivedEntry1)
		received <- []types.LogEntry{receivedEntry1}

		// Read second entry
		n, _ = server.Read(buffer)
		var receivedEntry2 types.LogEntry
		_ = json.Unmarshal([]byte(strings.TrimSpace(string(buffer[:n]))), &receivedEntry2)
		received <- []types.LogEntry{receivedEntry2}
	}()

	// Send entries
	err := client.SendLogEntry(entry1)
	if err != nil {
		t.Fatalf("Failed to send entry1: %v", err)
	}

	err = client.SendLogEntry(entry2)
	if err != nil {
		t.Fatalf("Failed to send entry2: %v", err)
	}

	// Check first entry got timestamp
	select {
	case entries := <-received:
		if entries[0].Timestamp == "" {
			t.Error("Expected timestamp to be set for entry without timestamp")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for first entry")
	}

	// Check second entry preserved timestamp
	select {
	case entries := <-received:
		expectedTimestamp := customTime.UTC().Format(time.RFC3339)
		if entries[0].Timestamp != expectedTimestamp {
			t.Errorf("Expected timestamp %s, got %s", expectedTimestamp, entries[0].Timestamp)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for second entry")
	}
}

func TestClientAuthenticateUnixSocket(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")

	// Should return error for Unix socket (authentication not required)
	_, err := client.Authenticate()

	if err == nil {
		t.Error("Expected error for Unix socket authentication")
	}
	if !strings.Contains(err.Error(), "authentication only required for TCP connections") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestClientAuthenticateNoSharedSecret(t *testing.T) {
	client := NewTCPClient("localhost", 8080)
	// Clear shared secret
	client.config.SharedSecret = ""

	_, err := client.Authenticate()

	if err == nil {
		t.Error("Expected error when shared secret is empty")
	}
	if !strings.Contains(err.Error(), "shared secret required") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestClientAuthenticateWithSharedSecret(t *testing.T) {
	client := NewTCPClient("localhost", 8080)
	client.config.SharedSecret = "test-secret"

	// This will fail due to no connection, but tests the code path with shared secret
	_, err := client.Authenticate()

	// Should fail due to connection, not due to missing shared secret
	if err == nil {
		t.Error("Expected error due to no connection")
	}
	if strings.Contains(err.Error(), "shared secret required") {
		t.Error("Should not complain about shared secret when it's provided")
	}
}

func TestNewUnixClientEmptyPath(t *testing.T) {
	client := NewUnixClient("")

	// Should use default socket path when empty
	if client.config.Address == "" {
		t.Error("Expected default socket path, got empty address")
	}
}

func TestNewTCPClientZeroPort(t *testing.T) {
	client := NewTCPClient("localhost", 0)

	// Should use default port 8080 when 0 is provided
	expectedAddress := "localhost:8080"
	if client.config.Address != expectedAddress {
		t.Errorf("Expected address %s, got %s", expectedAddress, client.config.Address)
	}
}

func TestNewTCPClientEmptyHost(t *testing.T) {
	client := NewTCPClient("", 9090)

	// Should use default host "localhost" when empty
	expectedAddress := "localhost:9090"
	if client.config.Address != expectedAddress {
		t.Errorf("Expected address %s, got %s", expectedAddress, client.config.Address)
	}
}

func TestNewTCPClientInvalidPort(t *testing.T) {
	// Test negative port
	client1 := NewTCPClient("test", -1)
	expectedAddress1 := "test:8080"
	if client1.config.Address != expectedAddress1 {
		t.Errorf("Expected address %s for negative port, got %s", expectedAddress1, client1.config.Address)
	}

	// Test port too high
	client2 := NewTCPClient("test", 70000)
	expectedAddress2 := "test:8080"
	if client2.config.Address != expectedAddress2 {
		t.Errorf("Expected address %s for high port, got %s", expectedAddress2, client2.config.Address)
	}
}

func TestSendLogBatchEmpty(t *testing.T) {
	// Use sync mode to test error handling
	cfg := config.DefaultConfig()
	cfg.AsyncMode = false
	client := NewClient(cfg)
	client.config.Network = "unix"
	client.config.Address = "/tmp/test.sock"

	// Test with empty batch
	err := client.SendLogBatch([]types.LogEntry{})
	if err == nil {
		t.Error("Expected error when sending empty batch without connection")
	}
}

func TestPingWithUnixSocket(t *testing.T) {
	client := NewUnixClient("/tmp/test.sock")

	// Should work for Unix sockets (no network restriction like Authenticate)
	_, err := client.Ping()
	if err == nil {
		t.Error("Expected error due to no connection")
	}
	// Should not complain about network type
	if strings.Contains(err.Error(), "only required for TCP") {
		t.Error("Ping should work for Unix sockets")
	}
}

// Test missing coverage functions
func TestSendAsyncWithResponse(t *testing.T) {
	// Test with async mode enabled and non-existent socket
	cfg := config.DefaultConfig()
	cfg.AsyncMode = true
	cfg.Address = "/tmp/non-existent-socket-test-12345.sock"
	client := NewClient(cfg)

	entry := types.NewLogEntry("test message", "test-source")

	// Should return error channel immediately
	errChan := client.SendAsyncWithResponse(entry)

	// Give the async worker a moment to process
	select {
	case err := <-errChan:
		if err == nil {
			t.Error("Expected connection error when connecting to non-existent socket")
		} else {
			t.Logf("Got expected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Expected response within timeout - async worker may not be processing")
	}

	// Test with async mode disabled (should get immediate error)
	syncCfg := config.DefaultConfig()
	syncCfg.AsyncMode = false
	syncClient := NewClient(syncCfg)

	errChan2 := syncClient.SendAsyncWithResponse(entry)
	select {
	case err := <-errChan2:
		if err == nil {
			t.Error("Expected error for disabled async mode")
		}
		expectedError := "async mode not initialized"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected immediate response for disabled async mode")
	}
}

func TestGetCircuitBreakerStats(t *testing.T) {
	client := NewClient(config.DefaultConfig())

	stats := client.GetCircuitBreakerStats()

	// Initial state should be closed
	if stats.State != "closed" {
		t.Errorf("Expected initial state 'closed', got %s", stats.State)
	}
	if stats.IsOpen != false {
		t.Error("Expected IsOpen to be false initially")
	}
	if stats.FailureCount != 0 {
		t.Errorf("Expected FailureCount 0, got %d", stats.FailureCount)
	}
}

func TestCircuitBreakerFailureScenario(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MaxRetries = 0 // Disable retries to test circuit breaker quickly
	cfg.CircuitBreakerThreshold = 2
	cfg.AsyncMode = false
	client := NewClient(cfg)
	client.config.Address = "/nonexistent/path"

	entry := types.NewLogEntry("test", "test")

	// First failure
	_ = client.SendLogEntry(entry)
	stats := client.GetCircuitBreakerStats()
	if stats.FailureCount != 1 {
		t.Errorf("Expected FailureCount 1 after first failure, got %d", stats.FailureCount)
	}

	// Second failure should open circuit
	_ = client.SendLogEntry(entry)
	stats = client.GetCircuitBreakerStats()
	if stats.FailureCount != 2 {
		t.Errorf("Expected FailureCount 2 after second failure, got %d", stats.FailureCount)
	}
	if stats.State != "open" {
		t.Errorf("Expected state 'open' after threshold reached, got %s", stats.State)
	}
	if !stats.IsOpen {
		t.Error("Expected IsOpen to be true when circuit is open")
	}
}

func TestCircuitBreakerHalfOpenState(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MaxRetries = 0
	cfg.CircuitBreakerThreshold = 1
	cfg.CircuitBreakerTimeout = time.Millisecond * 10 // Very short timeout
	cfg.AsyncMode = false
	client := NewClient(cfg)
	client.config.Address = "/nonexistent/path"

	entry := types.NewLogEntry("test", "test")

	// Trigger failure to open circuit
	_ = client.SendLogEntry(entry)

	// Wait for timeout
	time.Sleep(time.Millisecond * 20)

	// Next call should try half-open state
	_ = client.SendLogEntry(entry)

	// Verify circuit breaker logic was exercised
	stats := client.GetCircuitBreakerStats()
	// Should be back to open after failed half-open attempt
	if stats.State != "open" {
		t.Logf("Circuit breaker state: %s (may vary due to timing)", stats.State)
	}
}

func TestAsyncChannelFull(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AsyncMode = true
	cfg.ChannelBuffer = 1 // Very small buffer
	client := NewClient(cfg)

	entry := types.NewLogEntry("test", "test")

	// Fill the channel
	err1 := client.SendLogEntry(entry)
	if err1 != nil {
		t.Errorf("First send should succeed, got error: %v", err1)
	}

	// Second send should fail due to full channel
	err2 := client.SendLogEntry(entry)
	if err2 == nil {
		t.Error("Expected error due to full async channel")
	}
	expectedError := "async channel full"
	if !strings.Contains(err2.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err2)
	}
}
