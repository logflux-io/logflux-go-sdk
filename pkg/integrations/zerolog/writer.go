package zerolog

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/rs/zerolog"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// Writer implements io.Writer to capture zerolog JSON output.
// It integrates with zerolog by parsing JSON log entries and sending to LogFlux.
type Writer struct {
	client *client.BatchClient
	source string
}

// NewWriter creates a new LogFlux zerolog writer.
// Use with zerolog logger output to capture and forward logs to LogFlux.
func NewWriter(client *client.BatchClient, source string) *Writer {
	if source == "" {
		source = "zerolog"
	}
	return &Writer{
		client: client,
		source: source,
	}
}

// Write implements io.Writer interface.
// Parses zerolog JSON output and converts to LogFlux entries.
func (w *Writer) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	if message == "" {
		return len(p), nil
	}

	// Parse the JSON log entry
	var logData map[string]interface{}
	if err := json.Unmarshal([]byte(message), &logData); err != nil {
		// If not valid JSON, treat as plain text
		entry := types.NewLogEntry(message, w.source)
		if sendErr := w.client.SendLogEntry(entry); sendErr != nil { //nolint:staticcheck // Empty branch required for io.Writer interface compliance
			// Intentionally empty - io.Writer interface must not return errors for log failures
		}
		return len(p), nil
	}

	// Extract standard zerolog fields
	logMessage := extractString(logData, zerolog.MessageFieldName, "")
	logLevel := convertLevel(extractString(logData, zerolog.LevelFieldName, "info"))
	timestamp := extractString(logData, zerolog.TimestampFieldName, "")

	// Create LogFlux entry
	entry := types.NewLogEntry(logMessage, w.source).
		WithLogLevel(logLevel)

	// Set timestamp if available
	if timestamp != "" {
		entry = entry.WithTimestampString(timestamp)
	}

	// Add remaining fields as metadata
	for key, value := range logData {
		// Skip standard fields
		if key == zerolog.MessageFieldName ||
			key == zerolog.LevelFieldName ||
			key == zerolog.TimestampFieldName {
			continue
		}

		// Convert value to string
		if str, ok := value.(string); ok {
			entry = entry.WithMetadata(key, str)
		} else {
			entry = entry.WithMetadata(key, formatValue(value))
		}
	}

	// Send to LogFlux - errors are silently ignored to maintain io.Writer contract
	if sendErr := w.client.SendLogEntry(entry); sendErr != nil { //nolint:staticcheck // Empty branch required for io.Writer interface compliance
		// Intentionally empty - io.Writer interface must not return errors for log failures
	}

	return len(p), nil
}

// MultiWriter creates an io.Writer that duplicates writes to both LogFlux and another writer.
// Useful for sending logs to LogFlux while maintaining existing output (e.g., stdout).
func (w *Writer) MultiWriter(other io.Writer) io.Writer {
	return io.MultiWriter(w, other)
}

// extractString safely extracts a string value from the log data map
func extractString(data map[string]interface{}, key, defaultValue string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// convertLevel converts zerolog level string to LogFlux log level
func convertLevel(level string) int {
	switch strings.ToLower(level) {
	case "trace":
		return types.LevelDebug
	case "debug":
		return types.LevelDebug
	case "info":
		return types.LevelInfo
	case "warn":
		return types.LevelWarning
	case "error":
		return types.LevelError
	case "fatal":
		return types.LevelAlert
	case "panic":
		return types.LevelEmergency
	default:
		return types.LevelInfo
	}
}

// formatValue safely converts any value to string representation
func formatValue(value interface{}) string {
	if value == nil {
		return "<nil>"
	}

	// Handle common types efficiently
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		// For all other types, marshal to JSON
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return "<invalid>"
	}
}
