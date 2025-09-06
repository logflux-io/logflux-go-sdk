//go:build integration
// +build integration

// Package integration contains integration tests for the LogFlux Go SDK.
// These tests require a running LogFlux agent and validate real-world functionality.
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// Integration tests require a running logflux-agent
// Run with: go test -tags=integration -v ./test/integration/
//
// Default agent socket: /tmp/logflux-agent.sock
// Override with: LOGFLUX_SOCKET=/path/to/socket

func getAgentSocket() string {
	socket := os.Getenv("LOGFLUX_SOCKET")
	if socket == "" {
		socket = "/tmp/logflux-agent.sock"
	}
	return socket
}

func TestBasicConnectivity(t *testing.T) {
	socket := getAgentSocket()
	t.Logf("Testing connectivity to %s", socket)

	// Check if socket exists
	if _, err := os.Stat(socket); os.IsNotExist(err) {
		t.Skipf("LogFlux agent socket not found at %s - skipping integration test", socket)
	}

	client := client.NewUnixClient(socket)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to agent: %v", err)
	}
	defer client.Close()

	// Test ping
	resp, err := client.Ping()
	if err != nil {
		t.Fatalf("Failed to ping agent: %v", err)
	}

	if resp.Status != "pong" {
		t.Errorf("Expected pong response, got: %s", resp.Status)
	}

	t.Logf("Successfully connected and pinged agent")
}

func TestLogTransmission(t *testing.T) {
	socket := getAgentSocket()

	if _, err := os.Stat(socket); os.IsNotExist(err) {
		t.Skipf("LogFlux agent socket not found at %s", socket)
	}

	client := client.NewUnixClient(socket)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test different log scenarios
	testCases := []struct {
		name    string
		entry   types.LogEntry
		wantErr bool
	}{
		{
			name: "basic_text_log",
			entry: types.NewLogEntry("Integration test message", "go-sdk-integration").
				WithLogLevel(types.LevelInfo),
			wantErr: false,
		},
		{
			name: "json_payload",
			entry: types.NewLogEntry(`{"event":"test","data":{"key":"value"}}`, "go-sdk-integration").
				WithLogLevel(types.LevelInfo).
				WithMetadata("content_type", "json"),
			wantErr: false,
		},
		{
			name: "high_priority_log",
			entry: types.NewLogEntry("Critical system event", "go-sdk-integration").
				WithLogLevel(types.LevelCritical).
				WithMetadata("system", "integration_test").
				WithMetadata("priority", "high"),
			wantErr: false,
		},
		{
			name: "unicode_content",
			entry: types.NewLogEntry("Unicode test: ñáéíóú 中文 русский", "go-sdk-integration").
				WithLogLevel(types.LevelInfo).
				WithMetadata("encoding", "utf8"),
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := client.SendLogEntry(tc.entry)

			if tc.wantErr && err == nil {
				t.Error("Expected error, but got none")
			} else if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err == nil {
				t.Logf("Successfully sent %s", tc.name)
			}
		})
	}
}

func TestBatchOperations(t *testing.T) {
	socket := getAgentSocket()

	if _, err := os.Stat(socket); os.IsNotExist(err) {
		t.Skipf("LogFlux agent socket not found at %s", socket)
	}

	unixClient := client.NewUnixClient(socket)

	// Configure small batch for testing
	batchConfig := config.DefaultBatchConfig()
	batchConfig.MaxBatchSize = 3
	batchConfig.AutoFlush = true
	batchConfig.FlushInterval = 100 * time.Millisecond

	batchClient := client.NewBatchClient(unixClient, batchConfig)
	defer batchClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := batchClient.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect batch client: %v", err)
	}

	// Send batch that will trigger auto-flush
	entriesCount := 7
	for i := 1; i <= entriesCount; i++ {
		entry := types.NewLogEntry(fmt.Sprintf("Batch integration test #%d", i), "go-sdk-batch-integration").
			WithLogLevel(types.LevelInfo).
			WithMetadata("batch_test", "true").
			WithMetadata("entry_number", fmt.Sprintf("%d", i))

		err := batchClient.SendLogEntry(entry)
		if err != nil {
			t.Fatalf("Failed to send batch entry %d: %v", i, err)
		}
	}

	// Wait for auto-flush to complete
	time.Sleep(200 * time.Millisecond)

	// Manual flush for any remaining
	if err := batchClient.Flush(); err != nil {
		t.Fatalf("Failed to flush batch: %v", err)
	}

	// Check final stats
	stats := batchClient.GetStats()
	if stats.PendingEntries != 0 {
		t.Errorf("Expected 0 pending entries after flush, got %d", stats.PendingEntries)
	}

	t.Logf("Successfully sent %d entries via batch client", entriesCount)
	t.Logf("Batch stats: MaxSize=%d, AutoFlush=%v", stats.MaxBatchSize, stats.AutoFlush)
}

func TestAllLogLevels(t *testing.T) {
	socket := getAgentSocket()

	if _, err := os.Stat(socket); os.IsNotExist(err) {
		t.Skipf("LogFlux agent socket not found at %s", socket)
	}

	client := client.NewUnixClient(socket)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test all log levels
	levels := []struct {
		level int
		name  string
	}{
		{types.LevelEmergency, "Emergency"},
		{types.LevelAlert, "Alert"},
		{types.LevelCritical, "Critical"},
		{types.LevelError, "Error"},
		{types.LevelWarning, "Warning"},
		{types.LevelNotice, "Notice"},
		{types.LevelInfo, "Info"},
		{types.LevelDebug, "Debug"},
	}

	for _, l := range levels {
		entry := types.NewLogEntry(
			fmt.Sprintf("Integration test for %s level", l.name),
			"go-sdk-levels-integration").
			WithLogLevel(l.level).
			WithMetadata("level_name", l.name).
			WithMetadata("level_value", fmt.Sprintf("%d", l.level))

		err := client.SendLogEntry(entry)
		if err != nil {
			t.Fatalf("Failed to send %s level log: %v", l.name, err)
		}
	}

	t.Logf("Successfully sent all %d log levels", len(levels))
}

func TestPerformanceBaseline(t *testing.T) {
	socket := getAgentSocket()

	if _, err := os.Stat(socket); os.IsNotExist(err) {
		t.Skipf("LogFlux agent socket not found at %s", socket)
	}

	client := client.NewUnixClient(socket)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Performance test parameters
	messageCount := 100
	entry := types.NewLogEntry("Performance test message", "go-sdk-perf-integration").
		WithLogLevel(types.LevelInfo).
		WithMetadata("test_type", "performance")

	// Measure individual send performance
	start := time.Now()

	for i := 0; i < messageCount; i++ {
		testEntry := entry.WithMetadata("message_id", fmt.Sprintf("%d", i))

		if err := client.SendLogEntry(testEntry); err != nil {
			t.Fatalf("Failed to send performance test entry %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	msgsPerSecond := float64(messageCount) / duration.Seconds()

	t.Logf("Performance baseline: %d messages in %v", messageCount, duration)
	t.Logf("Throughput: %.1f messages/second", msgsPerSecond)
	t.Logf("Average latency: %v per message", duration/time.Duration(messageCount))

	// Basic performance expectations (these are quite lenient)
	if msgsPerSecond < 10 {
		t.Errorf("Performance below expectations: %.1f msg/sec (expected >10)", msgsPerSecond)
	}

	if duration/time.Duration(messageCount) > 100*time.Millisecond {
		t.Errorf("Average latency too high: %v per message (expected <100ms)",
			duration/time.Duration(messageCount))
	}
}
