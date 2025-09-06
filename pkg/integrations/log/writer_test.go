package log

import (
	"testing"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
)

func TestNewWriter(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "log-test")

	if writer == nil {
		t.Fatal("Expected non-nil writer")
	}
	if writer.client != batchClient {
		t.Error("Expected writer to use provided client")
	}
	if writer.source != "log-test" {
		t.Errorf("Expected source 'log-test', got %s", writer.source)
	}
}

func TestNewWriterWithEmptySource(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "")

	if writer.source != "log" {
		t.Errorf("Expected default source 'log', got %s", writer.source)
	}
}

func TestWriterWrite(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "log-test")

	message := "Test log message"
	n, err := writer.Write([]byte(message))

	if err != nil {
		t.Errorf("Expected no error from Write, got: %v", err)
	}
	if n != len(message) {
		t.Errorf("Expected bytes written %d, got %d", len(message), n)
	}
}

func TestWriterWriteEmpty(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "log-test")

	n, err := writer.Write([]byte(""))

	if err != nil {
		t.Errorf("Expected no error from Write with empty message, got: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected 0 bytes written, got %d", n)
	}
}

func TestWriterWriteWithNewlines(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "log-test")

	message := "Test log message\n"
	n, err := writer.Write([]byte(message))

	if err != nil {
		t.Errorf("Expected no error from Write, got: %v", err)
	}
	if n != len(message) {
		t.Errorf("Expected bytes written %d, got %d", len(message), n)
	}
}

func TestWriterWriteMultiline(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "log-test")

	message := "Line 1\nLine 2\nLine 3"
	n, err := writer.Write([]byte(message))

	if err != nil {
		t.Errorf("Expected no error from Write, got: %v", err)
	}
	if n != len(message) {
		t.Errorf("Expected bytes written %d, got %d", len(message), n)
	}
}

// MultiWriter test removed as function is not implemented yet
