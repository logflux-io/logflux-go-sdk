package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewLogEntry(t *testing.T) {
	entry := NewLogEntry("Test message", "test")

	if entry.Payload != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", entry.Payload)
	}

	if entry.LogLevel != LevelInfo {
		t.Errorf("Expected default level %d, got %d", LevelInfo, entry.LogLevel)
	}

	if entry.EntryType != TypeLog {
		t.Errorf("Expected default type %d, got %d", TypeLog, entry.EntryType)
	}

	if entry.Metadata == nil {
		t.Error("Expected labels map to be initialized")
	}

	if entry.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestLogEntryBuilderPattern(t *testing.T) {
	timestamp := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	entry := NewLogEntry("Test message", "test").
		WithLogLevel(LevelError).
		WithEntryType(999). // Any value becomes TypeLog in minimal SDK
		WithSource("test-source").
		WithTimestamp(timestamp).
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2")

	if entry.LogLevel != LevelError {
		t.Errorf("Expected level %d, got %d", LevelError, entry.LogLevel)
	}

	if entry.EntryType != TypeLog {
		t.Errorf("Expected type %d, got %d", TypeLog, entry.EntryType)
	}

	if entry.Source != "test-source" {
		t.Errorf("Expected source 'test-source', got %s", entry.Source)
	}

	if entry.Timestamp != timestamp.UTC().Format(time.RFC3339) {
		t.Errorf("Expected timestamp %s, got %s", timestamp.UTC().Format(time.RFC3339), entry.Timestamp)
	}

	if entry.Metadata["key1"] != "value1" {
		t.Errorf("Expected metadata key1='value1', got %s", entry.Metadata["key1"])
	}

	if entry.Metadata["key2"] != "value2" {
		t.Errorf("Expected metadata key2='value2', got %s", entry.Metadata["key2"])
	}
}

func TestAllLogLevels(t *testing.T) {
	levels := []struct {
		name  string
		level int
	}{
		{"Emergency", LevelEmergency},
		{"Alert", LevelAlert},
		{"Critical", LevelCritical},
		{"Error", LevelError},
		{"Warning", LevelWarning},
		{"Notice", LevelNotice},
		{"Info", LevelInfo},
		{"Debug", LevelDebug},
	}

	for _, test := range levels {
		t.Run(test.name, func(t *testing.T) {
			entry := NewLogEntry("Test", "test").WithLogLevel(test.level)
			if entry.LogLevel != test.level {
				t.Errorf("Expected level %d, got %d", test.level, entry.LogLevel)
			}
		})
	}
}

func TestEntryType(t *testing.T) {
	// In minimal SDK, all entries are TypeLog
	entry := NewLogEntry("Test", "test").WithEntryType(999) // Invalid type
	if entry.EntryType != TypeLog {
		t.Errorf("Expected TypeLog (%d), got %d", TypeLog, entry.EntryType)
	}
}

func TestWithAllMetadata(t *testing.T) {
	metadata := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	entry := NewLogEntry("Test", "test").WithAllMetadata(metadata)

	for key, expectedValue := range metadata {
		if entry.Metadata[key] != expectedValue {
			t.Errorf("Expected metadata %s='%s', got %s", key, expectedValue, entry.Metadata[key])
		}
	}
}

func TestLogEntryWithPayloadType(t *testing.T) {
	entry := NewLogEntry("Test", "test").WithPayloadType(PayloadTypeGenericJSON)

	if entry.PayloadType != string(PayloadTypeGenericJSON) {
		t.Errorf("Expected payload type %s, got %s", PayloadTypeGenericJSON, entry.PayloadType)
	}
}

func TestLogEntryJSONSerialization(t *testing.T) {
	entry := NewLogEntry("Test message", "test").
		WithLogLevel(LevelError).
		WithMetadata("key", "value")

	jsonData, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal entry: %v", err)
	}

	var unmarshaled LogEntry
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal entry: %v", err)
	}

	if unmarshaled.Payload != entry.Payload {
		t.Errorf("Expected payload %s, got %s", entry.Payload, unmarshaled.Payload)
	}

	if unmarshaled.LogLevel != entry.LogLevel {
		t.Errorf("Expected level %d, got %d", entry.LogLevel, unmarshaled.LogLevel)
	}

	if unmarshaled.Metadata["key"] != entry.Metadata["key"] {
		t.Errorf("Expected metadata key='%s', got %s", entry.Metadata["key"], unmarshaled.Metadata["key"])
	}
}

func TestIsValidJSON(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect bool
	}{
		{"Valid JSON object", `{"valid": "json"}`, true},
		{"Valid JSON array", `[1, 2, 3]`, true},
		{"Valid JSON string", `"simple string"`, true},
		{"Valid JSON number", `42`, true},
		{"Valid JSON boolean", `true`, true},
		{"Valid JSON null", `null`, true},
		{"Invalid JSON", `{malformed: json}`, false},
		{"Incomplete JSON", `{"incomplete": }`, false},
		{"Plain text", `plain text`, false},
		{"Empty string", ``, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsValidJSON(tc.input)
			if result != tc.expect {
				t.Errorf("Expected %v for input %s, got %v", tc.expect, tc.input, result)
			}
		})
	}
}

func TestAutoDetectPayloadType(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected PayloadType
	}{
		{"Simple text message", "Simple text message", PayloadTypeGeneric},
		{"JSON object", `{"level": "info", "message": "test"}`, PayloadTypeGenericJSON},
		{"JSON array", `[1, 2, 3]`, PayloadTypeGenericJSON},
		{"Malformed JSON", `{"malformed": json`, PayloadTypeGeneric},
		{"Empty string", "", PayloadTypeGeneric},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AutoDetectPayloadType(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s for input %s, got %s", tc.expected, tc.input, result)
			}
		})
	}
}

func TestAutoDetection(t *testing.T) {
	// Test JSON auto-detection in NewLogEntry
	jsonEntry := NewLogEntry(`{"test": "value"}`, "test")
	if jsonEntry.PayloadType != string(PayloadTypeGenericJSON) {
		t.Errorf("Expected payload type %s for JSON, got %s", PayloadTypeGenericJSON, jsonEntry.PayloadType)
	}

	// Test plain text auto-detection in NewLogEntry
	textEntry := NewLogEntry("plain text", "test")
	if textEntry.PayloadType != string(PayloadTypeGeneric) {
		t.Errorf("Expected payload type %s for plain text, got %s", PayloadTypeGeneric, textEntry.PayloadType)
	}
}

func TestWithPayloadType(t *testing.T) {
	// Test manual override of auto-detected type
	entry := NewLogEntry("Test", "test").WithPayloadType(PayloadTypeGenericJSON)
	if entry.PayloadType != string(PayloadTypeGenericJSON) {
		t.Errorf("Expected payload type %s, got %s", PayloadTypeGenericJSON, entry.PayloadType)
	}
}

func TestLogBatch(t *testing.T) {
	entries := []LogEntry{
		NewLogEntry("Message 1", "test"),
		NewLogEntry("Message 2", "test"),
	}

	batch := LogBatch{Entries: entries}

	if len(batch.Entries) != 2 {
		t.Errorf("Expected 2 entries in batch, got %d", len(batch.Entries))
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("Failed to marshal batch: %v", err)
	}

	var unmarshaled LogBatch
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal batch: %v", err)
	}

	if len(unmarshaled.Entries) != 2 {
		t.Errorf("Expected 2 entries after unmarshal, got %d", len(unmarshaled.Entries))
	}
}

func TestLabelOverwrite(t *testing.T) {
	entry := NewLogEntry("Test", "test").
		WithMetadata("key", "value1").
		WithMetadata("key", "value2") // Should overwrite

	if entry.Metadata["key"] != "value2" {
		t.Errorf("Expected metadata key='value2', got %s", entry.Metadata["key"])
	}
}

func TestEmptyAndLongMessages(t *testing.T) {
	// Empty message
	emptyEntry := NewLogEntry("", "test")
	if emptyEntry.Payload != "" {
		t.Errorf("Expected empty payload, got %s", emptyEntry.Payload)
	}

	// Very long message
	longMessage := string(make([]byte, 10000))
	longEntry := NewLogEntry(longMessage, "test")
	if len(longEntry.Payload) != 10000 {
		t.Errorf("Expected payload length 10000, got %d", len(longEntry.Payload))
	}
}

func TestUnicodeMessages(t *testing.T) {
	unicodeMessage := "Hello 世界 🌍"
	entry := NewLogEntry(unicodeMessage, "test")

	if entry.Payload != unicodeMessage {
		t.Errorf("Expected unicode message %s, got %s", unicodeMessage, entry.Payload)
	}

	// Test JSON serialization with unicode
	jsonData, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal unicode entry: %v", err)
	}

	var unmarshaled LogEntry
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal unicode entry: %v", err)
	}

	if unmarshaled.Payload != unicodeMessage {
		t.Errorf("Expected unicode message %s after unmarshal, got %s", unicodeMessage, unmarshaled.Payload)
	}
}

func TestPingRequest(t *testing.T) {
	ping := NewPingRequest()
	if ping.Action != "ping" {
		t.Errorf("Expected action 'ping', got %s", ping.Action)
	}
}

func TestAuthRequest(t *testing.T) {
	secret := "test-secret"
	auth := NewAuthRequest(secret)
	if auth.Action != "authenticate" {
		t.Errorf("Expected action 'authenticate', got %s", auth.Action)
	}
	if auth.SharedSecret != secret {
		t.Errorf("Expected shared secret %s, got %s", secret, auth.SharedSecret)
	}
}

func TestWithTimestampString(t *testing.T) {
	customTimestamp := "2023-01-01T12:00:00Z"
	entry := NewLogEntry("Test", "test").WithTimestampString(customTimestamp)

	if entry.Timestamp != customTimestamp {
		t.Errorf("Expected timestamp %s, got %s", customTimestamp, entry.Timestamp)
	}
}

func TestWithVersion(t *testing.T) {
	customVersion := "2.0"
	entry := NewLogEntry("Test", "test").WithVersion(customVersion)

	if entry.Version != customVersion {
		t.Errorf("Expected version %s, got %s", customVersion, entry.Version)
	}
}

func TestWithSource(t *testing.T) {
	// Test empty source
	entry1 := NewLogEntry("Test", "").WithSource("new-source")
	if entry1.Source != "new-source" {
		t.Errorf("Expected source 'new-source', got %s", entry1.Source)
	}

	// Test non-empty source override
	entry2 := NewLogEntry("Test", "old-source").WithSource("new-source")
	if entry2.Source != "new-source" {
		t.Errorf("Expected source 'new-source', got %s", entry2.Source)
	}
}

func TestWithLogLevel(t *testing.T) {
	// Test with 0 level (invalid, should default to LevelInfo)
	entry1 := NewLogEntry("Test", "test").WithLogLevel(0)
	if entry1.LogLevel != LevelInfo {
		t.Errorf("Expected log level %d (default), got %d", LevelInfo, entry1.LogLevel)
	}

	// Test with negative level (invalid, should default to LevelInfo)
	entry2 := NewLogEntry("Test", "test").WithLogLevel(-1)
	if entry2.LogLevel != LevelInfo {
		t.Errorf("Expected log level %d (default), got %d", LevelInfo, entry2.LogLevel)
	}

	// Test with high level (invalid, should default to LevelInfo)
	entry3 := NewLogEntry("Test", "test").WithLogLevel(999)
	if entry3.LogLevel != LevelInfo {
		t.Errorf("Expected log level %d (default), got %d", LevelInfo, entry3.LogLevel)
	}

	// Test with valid levels
	entry4 := NewLogEntry("Test", "test").WithLogLevel(LevelError)
	if entry4.LogLevel != LevelError {
		t.Errorf("Expected log level %d, got %d", LevelError, entry4.LogLevel)
	}
}

func TestWithMetadataEdgeCases(t *testing.T) {
	// Test nil metadata map initialization
	entry := LogEntry{
		Payload:   "Test",
		EntryType: TypeLog,
		Source:    "test",
		LogLevel:  LevelInfo,
		Metadata:  nil, // Start with nil
	}

	// This should initialize the map
	entry = entry.WithMetadata("key", "value")

	if entry.Metadata == nil {
		t.Error("Expected metadata map to be initialized")
	}
	if entry.Metadata["key"] != "value" {
		t.Errorf("Expected metadata key='value', got %s", entry.Metadata["key"])
	}
}

func TestWithAllMetadataEdgeCases(t *testing.T) {
	// Test with nil input
	entry1 := NewLogEntry("Test", "test").WithAllMetadata(nil)
	if entry1.Metadata == nil {
		t.Error("Expected metadata map to remain initialized")
	}

	// Test with empty input
	entry2 := NewLogEntry("Test", "test").WithAllMetadata(map[string]string{})
	if entry2.Metadata == nil {
		t.Error("Expected metadata map to remain initialized")
	}

	// Test overwriting existing metadata
	entry3 := NewLogEntry("Test", "test").
		WithMetadata("existing", "old").
		WithAllMetadata(map[string]string{"existing": "new", "added": "value"})

	if entry3.Metadata["existing"] != "new" {
		t.Errorf("Expected metadata existing='new', got %s", entry3.Metadata["existing"])
	}
	if entry3.Metadata["added"] != "value" {
		t.Errorf("Expected metadata added='value', got %s", entry3.Metadata["added"])
	}
}

func TestNewLogEntryEdgeCases(t *testing.T) {
	// Test with empty strings
	entry1 := NewLogEntry("", "")
	if entry1.Payload != "" {
		t.Errorf("Expected empty payload, got %s", entry1.Payload)
	}
	if entry1.Source != "unknown" { // Default source when empty
		t.Errorf("Expected source 'unknown', got %s", entry1.Source)
	}

	// Test auto-detection with edge cases
	entry2 := NewLogEntry("{", "test") // Invalid JSON
	if entry2.PayloadType != string(PayloadTypeGeneric) {
		t.Errorf("Expected payload type %s for invalid JSON, got %s", PayloadTypeGeneric, entry2.PayloadType)
	}
}

func TestNewAuthRequestEmpty(t *testing.T) {
	// Test with empty shared secret should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when shared secret is empty")
		}
	}()

	NewAuthRequest("") // Should panic
}

func TestWithSourceEdgeCases(t *testing.T) {
	// Test with empty source - should default to "unknown"
	entry := LogEntry{
		Payload:   "Test",
		EntryType: TypeLog,
		Source:    "", // Already empty
		LogLevel:  LevelInfo,
	}

	entry = entry.WithSource("") // Set to empty again, should become "unknown"
	if entry.Source != "unknown" {
		t.Errorf("Expected source 'unknown', got %s", entry.Source)
	}
}
