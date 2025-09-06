package slog

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

func TestNewHandler(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	handler := NewHandler(batchClient, "slog-test")

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.client != batchClient {
		t.Error("Expected handler to use provided client")
	}
	if handler.source != "slog-test" {
		t.Errorf("Expected source 'slog-test', got %s", handler.source)
	}
}

func TestNewHandlerWithEmptySource(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	handler := NewHandler(batchClient, "")

	if handler.source != "slog" {
		t.Errorf("Expected default source 'slog', got %s", handler.source)
	}
}

func TestHandlerEnabled(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	handler := NewHandler(batchClient, "test")

	ctx := context.Background()

	// Should be enabled for all levels
	levels := []slog.Level{
		slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError,
	}

	for _, level := range levels {
		if !handler.Enabled(ctx, level) {
			t.Errorf("Expected level %v to be enabled", level)
		}
	}
}

func TestConvertLevel(t *testing.T) {
	testCases := []struct {
		slogLevel     slog.Level
		expectedLevel int
	}{
		{slog.LevelError, types.LevelError},
		{slog.LevelWarn, types.LevelWarning},
		{slog.LevelInfo, types.LevelInfo},
		{slog.LevelDebug, types.LevelDebug},
		{slog.Level(-10), types.LevelDebug}, // Very low level
	}

	for _, tc := range testCases {
		result := convertLevel(tc.slogLevel)
		if result != tc.expectedLevel {
			t.Errorf("Expected level %d for %v, got %d", tc.expectedLevel, tc.slogLevel, result)
		}
	}
}

func TestHandlerHandle(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	handler := NewHandler(batchClient, "slog-test")

	ctx := context.Background()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test log message", 0)

	// Add some attributes
	record.Add("key1", "value1", "key2", 42)

	// Handle should not return error (even though connection will fail)
	err := handler.Handle(ctx, record)
	if err != nil {
		t.Errorf("Expected no error from Handle, got: %v", err)
	}
}

func TestHandlerWithAttrs(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	handler := NewHandler(batchClient, "test")

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}

	newHandler := handler.WithAttrs(attrs)
	if newHandler == handler {
		t.Error("Expected WithAttrs to return a new handler instance")
	}

	// Check that attributes were added
	slogHandler, ok := newHandler.(*Handler)
	if !ok {
		t.Fatal("Expected returned handler to be *Handler type")
	}

	if len(slogHandler.attrs) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(slogHandler.attrs))
	}
}

func TestHandlerWithGroup(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	handler := NewHandler(batchClient, "test")

	newHandler := handler.WithGroup("group1")
	if newHandler == handler {
		t.Error("Expected WithGroup to return a new handler instance")
	}

	// Check that group was added to source
	slogHandler, ok := newHandler.(*Handler)
	if !ok {
		t.Fatal("Expected returned handler to be *Handler type")
	}

	expected := "test.group1"
	if slogHandler.source != expected {
		t.Errorf("Expected source %s, got %s", expected, slogHandler.source)
	}
}
