package zap

import (
	"math"
	"strconv"

	"go.uber.org/zap/zapcore"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// Core implements zapcore.Core to send logs to LogFlux.
// It integrates with Uber's high-performance Zap logging library.
type Core struct {
	client        *client.BatchClient
	source        string
	contextFields []zapcore.Field // Preserved fields from With() calls
	level         zapcore.Level
}

// NewCore creates a new LogFlux Zap core.
// Uses batch client for optimal performance with Zap's high-throughput logging.
func NewCore(client *client.BatchClient, source string, level zapcore.Level) zapcore.Core {
	if source == "" {
		source = "zap"
	}
	return &Core{
		client:        client,
		source:        source,
		level:         level,
		contextFields: nil,
	}
}

// Enabled returns true if the given level is enabled.
func (c *Core) Enabled(level zapcore.Level) bool {
	return level >= c.level
}

// With adds structured context to the core.
// Returns a new core with the provided fields preserved for future log entries.
func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	// Combine existing context fields with new fields
	combinedFields := make([]zapcore.Field, len(c.contextFields)+len(fields))
	copy(combinedFields, c.contextFields)
	copy(combinedFields[len(c.contextFields):], fields)

	return &Core{
		client:        c.client,
		source:        c.source,
		level:         c.level,
		contextFields: combinedFields,
	}
}

// Check determines whether the supplied entry should be written.
func (c *Core) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write serializes the entry and fields and sends to LogFlux.
func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Convert Zap level to LogFlux level
	logLevel := convertLevel(entry.Level)

	// Create LogFlux entry
	logEntry := types.NewLogEntry(entry.Message, c.source).
		WithLogLevel(logLevel)

	// Add caller information if available
	if entry.Caller.Defined {
		logEntry = logEntry.WithMetadata("caller", entry.Caller.String())
	}

	// Add stack trace if available
	if entry.Stack != "" {
		logEntry = logEntry.WithMetadata("stack", entry.Stack)
	}

	// Add logger name if available
	if entry.LoggerName != "" {
		logEntry = logEntry.WithMetadata("logger", entry.LoggerName)
	}

	// Convert context fields (from With() calls) to metadata first
	for _, field := range c.contextFields {
		logEntry = logEntry.WithMetadata(field.Key, fieldToString(field))
	}

	// Convert current log entry fields to metadata
	for _, field := range fields {
		logEntry = logEntry.WithMetadata(field.Key, fieldToString(field))
	}

	return c.client.SendLogEntry(logEntry)
}

// Sync flushes buffered logs (delegates to batch client flush).
func (c *Core) Sync() error {
	return c.client.Flush()
}

// convertLevel converts zapcore.Level to LogFlux log level
func convertLevel(level zapcore.Level) int {
	switch level {
	case zapcore.DebugLevel:
		return types.LevelDebug
	case zapcore.InfoLevel:
		return types.LevelInfo
	case zapcore.WarnLevel:
		return types.LevelWarning
	case zapcore.ErrorLevel:
		return types.LevelError
	case zapcore.DPanicLevel:
		return types.LevelCritical
	case zapcore.PanicLevel:
		return types.LevelAlert
	case zapcore.FatalLevel:
		return types.LevelEmergency
	default:
		return types.LevelInfo
	}
}

// fieldToString converts a Zap field to string representation
func fieldToString(field zapcore.Field) string {
	switch field.Type {
	case zapcore.StringType:
		return field.String
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return strconv.FormatInt(field.Integer, 10)
	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return strconv.FormatUint(uint64(field.Integer), 10)
	case zapcore.Float64Type:
		return strconv.FormatFloat(math.Float64frombits(uint64(field.Integer)), 'f', -1, 64)
	case zapcore.Float32Type:
		return strconv.FormatFloat(float64(math.Float32frombits(uint32(field.Integer))), 'f', -1, 32)
	case zapcore.BoolType:
		return strconv.FormatBool(field.Integer == 1)
	default:
		return field.String
	}
}
