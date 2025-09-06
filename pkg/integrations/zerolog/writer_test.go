package zerolog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

func TestNewWriter(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "zerolog-test")

	if writer == nil {
		t.Fatal("Expected non-nil writer")
	}
	if writer.client != batchClient {
		t.Error("Expected writer to use provided client")
	}
	if writer.source != "zerolog-test" {
		t.Errorf("Expected source 'zerolog-test', got %s", writer.source)
	}
}

func TestNewWriterWithEmptySource(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "")

	if writer.source != "zerolog" {
		t.Errorf("Expected default source 'zerolog', got %s", writer.source)
	}
}

func TestConvertZerologLevel(t *testing.T) {
	testCases := []struct {
		name          string
		zerologLevel  string
		expectedLevel int
	}{
		{"panic", "panic", types.LevelEmergency},
		{"fatal", "fatal", types.LevelAlert},
		{"error", "error", types.LevelError},
		{"warn", "warn", types.LevelWarning},
		{"info", "info", types.LevelInfo},
		{"debug", "debug", types.LevelDebug},
		{"trace", "trace", types.LevelDebug},
		{"unknown", "unknown", types.LevelInfo},
		{"empty", "", types.LevelInfo},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertLevel(tc.zerologLevel)
			if result != tc.expectedLevel {
				t.Errorf("Expected level %d for %s, got %d", tc.expectedLevel, tc.zerologLevel, result)
			}
		})
	}
}

func TestWriterWrite(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "zerolog-test")

	// Test valid JSON log entry
	jsonLog := `{"level":"info","msg":"test message","key1":"value1","key2":42}`
	n, err := writer.Write([]byte(jsonLog))

	if err != nil {
		t.Errorf("Expected no error from Write, got: %v", err)
	}
	if n != len(jsonLog) {
		t.Errorf("Expected bytes written %d, got %d", len(jsonLog), n)
	}
}

func TestWriterWriteInvalidJSON(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "zerolog-test")

	// Test invalid JSON - should still process as plain text
	invalidJSON := `{invalid json`
	n, err := writer.Write([]byte(invalidJSON))

	if err != nil {
		t.Errorf("Expected no error from Write with invalid JSON, got: %v", err)
	}
	if n != len(invalidJSON) {
		t.Errorf("Expected bytes written %d, got %d", len(invalidJSON), n)
	}
}

func TestWriterWriteEmptyMessage(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "zerolog-test")

	// Test empty message
	n, err := writer.Write([]byte(""))

	if err != nil {
		t.Errorf("Expected no error from Write with empty message, got: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected 0 bytes written, got %d", n)
	}
}

func TestWriterWriteWithoutMsg(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "zerolog-test")

	// Test JSON without msg field
	jsonLog := `{"level":"error","key1":"value1","timestamp":"2023-01-01T12:00:00Z"}`
	n, err := writer.Write([]byte(jsonLog))

	if err != nil {
		t.Errorf("Expected no error from Write without msg field, got: %v", err)
	}
	if n != len(jsonLog) {
		t.Errorf("Expected bytes written %d, got %d", len(jsonLog), n)
	}
}

func TestWriterMultiWriter(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "zerolog-test")

	// Create a string builder as second writer
	var sb strings.Builder
	multiWriter := writer.MultiWriter(&sb)

	if multiWriter == nil {
		t.Fatal("Expected non-nil multi writer")
	}

	// Test writing to multi writer
	testLog := `{"level":"info","msg":"test"}`
	n, err := multiWriter.Write([]byte(testLog))

	if err != nil {
		t.Errorf("Expected no error from MultiWriter.Write, got: %v", err)
	}
	if n != len(testLog) {
		t.Errorf("Expected bytes written %d, got %d", len(testLog), n)
	}

	// Check that it was written to the string builder too
	if sb.String() != testLog {
		t.Errorf("Expected string builder to contain %s, got %s", testLog, sb.String())
	}
}

func TestFormatValueVariousTypes(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	writer := NewWriter(batchClient, "test")

	// Test various value types to improve formatValue coverage
	testCases := []struct {
		input    interface{}
		expected string
	}{
		{"string_value", "string_value"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
		{nil, "<nil>"},
		{[]int{1, 2, 3}, "[1 2 3]"},
		{map[string]int{"key": 123}, "map[key:123]"},
	}

	for _, tc := range testCases {
		// Create a JSON log with the test value
		testLog := fmt.Sprintf(`{"level":"info","message":"test","field":%v}`, jsonValue(tc.input))

		n, err := writer.Write([]byte(testLog))
		if err != nil {
			t.Errorf("Expected no error writing log with %T value, got: %v", tc.input, err)
		}
		if n != len(testLog) {
			t.Errorf("Expected %d bytes written, got %d", len(testLog), n)
		}
	}
}

// Helper function to convert values to JSON representation for test
func jsonValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, val)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", val)
	}
}
