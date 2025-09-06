package logrus

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// Hook implements logrus.Hook to send logs to LogFlux.
// It integrates with the popular Logrus structured logging library.
type Hook struct {
	client *client.BatchClient
	source string
}

// NewHook creates a new LogFlux logrus hook.
// Uses batch client for better performance with high-volume logging.
func NewHook(client *client.BatchClient, source string) *Hook {
	if source == "" {
		source = "logrus"
	}
	return &Hook{
		client: client,
		source: source,
	}
}

// Levels returns the log levels that this hook handles.
// Currently handles all levels.
func (h *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire processes a logrus entry and sends it to LogFlux.
func (h *Hook) Fire(entry *logrus.Entry) error {
	// Convert logrus level to LogFlux level
	logLevel := convertLevel(entry.Level)

	// Create LogFlux entry
	logEntry := types.NewLogEntry(entry.Message, h.source).
		WithLogLevel(logLevel)

	// Add logrus fields as metadata
	for key, value := range entry.Data {
		if str, ok := value.(string); ok {
			logEntry = logEntry.WithMetadata(key, str)
		} else {
			// Convert non-string values to string safely
			logEntry = logEntry.WithMetadata(key, formatValue(value))
		}
	}

	return h.client.SendLogEntry(logEntry)
}

// convertLevel converts logrus.Level to LogFlux log level
func convertLevel(level logrus.Level) int {
	switch level {
	case logrus.PanicLevel:
		return types.LevelEmergency
	case logrus.FatalLevel:
		return types.LevelAlert
	case logrus.ErrorLevel:
		return types.LevelError
	case logrus.WarnLevel:
		return types.LevelWarning
	case logrus.InfoLevel:
		return types.LevelInfo
	case logrus.DebugLevel:
		return types.LevelDebug
	case logrus.TraceLevel:
		return types.LevelDebug
	default:
		return types.LevelInfo
	}
}

// formatValue safely converts any value to string representation
func formatValue(value interface{}) string {
	if value == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", value)
}
