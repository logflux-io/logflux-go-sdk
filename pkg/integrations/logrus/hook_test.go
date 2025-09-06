package logrus

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

func TestNewHook(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	hook := NewHook(batchClient, "test-source")

	if hook == nil {
		t.Fatal("Expected non-nil hook")
	}
	if hook.client != batchClient {
		t.Error("Expected hook to use provided client")
	}
	if hook.source != "test-source" {
		t.Errorf("Expected source 'test-source', got %s", hook.source)
	}
}

func TestNewHookWithEmptySource(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	hook := NewHook(batchClient, "")

	if hook.source != "logrus" {
		t.Errorf("Expected default source 'logrus', got %s", hook.source)
	}
}

func TestHookLevels(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	hook := NewHook(batchClient, "test")

	levels := hook.Levels()
	expectedLevels := []logrus.Level{
		logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel,
		logrus.WarnLevel, logrus.InfoLevel, logrus.DebugLevel, logrus.TraceLevel,
	}

	if len(levels) != len(expectedLevels) {
		t.Errorf("Expected %d levels, got %d", len(expectedLevels), len(levels))
	}

	for i, expected := range expectedLevels {
		if levels[i] != expected {
			t.Errorf("Expected level %v at index %d, got %v", expected, i, levels[i])
		}
	}
}

func TestConvertLogrusLevel(t *testing.T) {
	testCases := []struct {
		logrusLevel   logrus.Level
		expectedLevel int
	}{
		{logrus.PanicLevel, types.LevelEmergency},
		{logrus.FatalLevel, types.LevelAlert},
		{logrus.ErrorLevel, types.LevelError},
		{logrus.WarnLevel, types.LevelWarning},
		{logrus.InfoLevel, types.LevelInfo},
		{logrus.DebugLevel, types.LevelDebug},
		{logrus.TraceLevel, types.LevelDebug},
	}

	for _, tc := range testCases {
		result := convertLevel(tc.logrusLevel)
		if result != tc.expectedLevel {
			t.Errorf("Expected level %d for %v, got %d", tc.expectedLevel, tc.logrusLevel, result)
		}
	}
}

func TestFormatValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "test", "test"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"nil", nil, "<nil>"},
		{"struct", struct{ Name string }{"test"}, "{test}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestHookFire(t *testing.T) {
	batchClient := client.NewBatchUnixClient("/tmp/test.sock", config.DefaultBatchConfig())
	hook := NewHook(batchClient, "logrus-test")

	// Create logrus entry
	entry := &logrus.Entry{
		Message: "Test log message",
		Level:   logrus.InfoLevel,
		Data: logrus.Fields{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		},
	}

	// Fire should not return error (even though connection will fail)
	err := hook.Fire(entry)
	if err != nil {
		t.Errorf("Expected no error from Fire, got: %v", err)
	}
}
