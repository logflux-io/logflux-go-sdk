package zap

import (
	"testing"

	"go.uber.org/zap/zapcore"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

func TestNewCore(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	core := NewCore(batchClient, "zap-test", zapcore.InfoLevel)

	if core == nil {
		t.Fatal("Expected non-nil core")
	}

	// Cast to our Core type to access fields
	zapCore, ok := core.(*Core)
	if !ok {
		t.Fatal("Expected core to be *Core type")
	}

	if zapCore.client != batchClient {
		t.Error("Expected core to use provided client")
	}
	if zapCore.source != "zap-test" {
		t.Errorf("Expected source 'zap-test', got %s", zapCore.source)
	}
	if zapCore.level != zapcore.InfoLevel {
		t.Errorf("Expected level %v, got %v", zapcore.InfoLevel, zapCore.level)
	}
}

func TestNewCoreWithEmptySource(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	core := NewCore(batchClient, "", zapcore.InfoLevel)

	zapCore := core.(*Core)
	if zapCore.source != "zap" {
		t.Errorf("Expected default source 'zap', got %s", zapCore.source)
	}
}

func TestCoreEnabled(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	core := NewCore(batchClient, "test", zapcore.WarnLevel)

	testCases := []struct {
		level    zapcore.Level
		expected bool
	}{
		{zapcore.DebugLevel, false},
		{zapcore.InfoLevel, false},
		{zapcore.WarnLevel, true},
		{zapcore.ErrorLevel, true},
		{zapcore.FatalLevel, true},
	}

	for _, tc := range testCases {
		result := core.Enabled(tc.level)
		if result != tc.expected {
			t.Errorf("Expected %v for level %v, got %v", tc.expected, tc.level, result)
		}
	}
}

func TestConvertZapLevel(t *testing.T) {
	testCases := []struct {
		zapLevel      zapcore.Level
		expectedLevel int
	}{
		{zapcore.FatalLevel, types.LevelEmergency},
		{zapcore.ErrorLevel, types.LevelError},
		{zapcore.WarnLevel, types.LevelWarning},
		{zapcore.InfoLevel, types.LevelInfo},
		{zapcore.DebugLevel, types.LevelDebug},
		{zapcore.Level(-10), types.LevelInfo}, // Very low level defaults to Info
	}

	for _, tc := range testCases {
		result := convertLevel(tc.zapLevel)
		if result != tc.expectedLevel {
			t.Errorf("Expected level %d for %v, got %d", tc.expectedLevel, tc.zapLevel, result)
		}
	}
}

func TestCoreWith(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	core := NewCore(batchClient, "test", zapcore.InfoLevel)

	fields := []zapcore.Field{
		{Key: "key1", String: "value1"},
		{Key: "key2", Integer: 42},
	}

	newCore := core.With(fields)
	if newCore == core {
		t.Error("Expected With to return a new core instance")
	}

	// Check that context fields were added
	zapCore, ok := newCore.(*Core)
	if !ok {
		t.Fatal("Expected returned core to be *Core type")
	}

	if len(zapCore.contextFields) != 2 {
		t.Errorf("Expected 2 context fields, got %d", len(zapCore.contextFields))
	}
}

func TestCoreCheck(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	core := NewCore(batchClient, "test", zapcore.InfoLevel)

	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "test message",
	}

	checkedEntry := core.Check(entry, nil)
	if checkedEntry == nil {
		t.Error("Expected non-nil checked entry for enabled level")
	}

	// Test disabled level
	disabledEntry := zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "debug message",
	}
	checkedEntry = core.Check(disabledEntry, nil)
	if checkedEntry != nil {
		t.Error("Expected nil checked entry for disabled level")
	}
}

func TestCoreWrite(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	core := NewCore(batchClient, "zap-test", zapcore.InfoLevel)

	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "Test log message",
	}

	fields := []zapcore.Field{
		{Key: "string_field", String: "string_value"},
		{Key: "int_field", Integer: 42},
		{Key: "bool_field", Integer: 1}, // Bool as integer
	}

	// Write should not return error (even though connection will fail)
	err := core.Write(entry, fields)
	if err != nil {
		t.Errorf("Expected no error from Write, got: %v", err)
	}
}

func TestCoreSync(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	core := NewCore(batchClient, "test", zapcore.InfoLevel)

	// Sync should not return error (delegates to batch client)
	err := core.Sync()
	if err != nil {
		t.Errorf("Expected no error from Sync, got: %v", err)
	}
}

func TestFieldToString(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	core := NewCore(batchClient, "test", zapcore.InfoLevel)

	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "test with various field types",
	}

	// Test various field types to improve fieldToString coverage
	fields := []zapcore.Field{
		{Key: "string", String: "test_string", Type: zapcore.StringType},
		{Key: "int", Integer: 123, Type: zapcore.Int64Type},
		{Key: "float", String: "1.23", Type: zapcore.Float64Type},
		{Key: "bool_true", Integer: 1, Type: zapcore.BoolType},
		{Key: "bool_false", Integer: 0, Type: zapcore.BoolType},
		{Key: "duration", Integer: int64(5000000), Type: zapcore.DurationType}, // 5ms in nanoseconds
		{Key: "time", Integer: 1640995200, Type: zapcore.TimeType},             // Unix timestamp
		{Key: "object", String: `{"nested":"value"}`, Type: zapcore.ReflectType},
		{Key: "unknown", String: "fallback", Type: zapcore.FieldType(99)}, // Unknown type
	}

	// This will exercise the fieldToString function
	err := core.Write(entry, fields)
	if err != nil {
		t.Errorf("Expected no error from Write with various field types, got: %v", err)
	}
}
