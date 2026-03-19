package adapters

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

// LogrusLevel represents logrus log levels
type LogrusLevel uint32

const (
	// LogrusPanicLevel level, highest level of severity
	LogrusPanicLevel LogrusLevel = iota
	// LogrusFatalLevel level
	LogrusFatalLevel
	// LogrusErrorLevel level
	LogrusErrorLevel
	// LogrusWarnLevel level
	LogrusWarnLevel
	// LogrusInfoLevel level
	LogrusInfoLevel
	// LogrusDebugLevel level
	LogrusDebugLevel
	// LogrusTraceLevel level
	LogrusTraceLevel
)

// LogrusEntry represents a logrus log entry
type LogrusEntry struct {
	logger *LogrusLogger
	Data   map[string]interface{}
	Time   time.Time
	Level  LogrusLevel
}

// LogrusLogger provides a drop-in replacement for logrus.Logger
type LogrusLogger struct {
	client LoggerInterface
	level  LogrusLevel
}

// NewLogrusLogger creates a new logrus logger adapter
func NewLogrusLogger(client LoggerInterface) *LogrusLogger {
	return &LogrusLogger{
		client: client,
		level:  LogrusInfoLevel,
	}
}

// NewLogrusLoggerFromEnv creates a logrus logger adapter from environment variables
func NewLogrusLoggerFromEnv(node string) (*LogrusLogger, error) {
	resilientClient, err := client.NewResilientClientFromEnvWithHandshake(node)
	if err != nil {
		return nil, err
	}

	return NewLogrusLogger(resilientClient), nil
}

// mapLogrusLevel converts logrus levels to LogFlux levels
func mapLogrusLevel(level LogrusLevel) int {
	switch level {
	case LogrusTraceLevel, LogrusDebugLevel:
		return models.LogLevelDebug
	case LogrusInfoLevel:
		return models.LogLevelInfo
	case LogrusWarnLevel:
		return models.LogLevelWarning
	case LogrusErrorLevel:
		return models.LogLevelError
	case LogrusFatalLevel, LogrusPanicLevel:
		return models.LogLevelCritical
	default:
		return models.LogLevelInfo
	}
}

// formatMessage formats a message with optional fields
func (l *LogrusLogger) formatMessage(message string, fields map[string]interface{}) string {
	if len(fields) == 0 {
		return message
	}

	var parts []string
	for key, value := range fields {
		parts = append(parts, key+"="+formatValue(value))
	}

	if message != "" {
		return message + " " + strings.Join(parts, " ")
	}
	return strings.Join(parts, " ")
}

// formatValue formats a field value as string
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// WithField adds a field to the logger
func (l *LogrusLogger) WithField(key string, value interface{}) *LogrusEntry {
	return &LogrusEntry{
		logger: l,
		Data:   map[string]interface{}{key: value},
		Time:   time.Now(),
		Level:  l.level,
	}
}

// WithFields adds multiple fields to the logger
func (l *LogrusLogger) WithFields(fields map[string]interface{}) *LogrusEntry {
	return &LogrusEntry{
		logger: l,
		Data:   fields,
		Time:   time.Now(),
		Level:  l.level,
	}
}

// WithError adds an error field to the logger
func (l *LogrusLogger) WithError(err error) *LogrusEntry {
	if err == nil {
		return l.WithField("error", "<nil>")
	}
	return l.WithField("error", err.Error())
}

// WithTime adds a time field to the logger
func (l *LogrusLogger) WithTime(t time.Time) *LogrusEntry {
	entry := &LogrusEntry{
		logger: l,
		Data:   make(map[string]interface{}),
		Time:   t,
		Level:  l.level,
	}
	return entry
}

// SetLevel sets the logging level
func (l *LogrusLogger) SetLevel(level LogrusLevel) {
	l.level = level
}

// GetLevel returns the current logging level
func (l *LogrusLogger) GetLevel() LogrusLevel {
	return l.level
}

// IsLevelEnabled checks if a level is enabled
func (l *LogrusLogger) IsLevelEnabled(level LogrusLevel) bool {
	return level <= l.level
}

// Log methods for LogrusLogger
func (l *LogrusLogger) Trace(args ...interface{}) {
	if l.IsLevelEnabled(LogrusTraceLevel) {
		message := l.formatMessage(sprint(args...), nil)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusTraceLevel))
	}
}

func (l *LogrusLogger) Debug(args ...interface{}) {
	if l.IsLevelEnabled(LogrusDebugLevel) {
		message := l.formatMessage(sprint(args...), nil)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusDebugLevel))
	}
}

func (l *LogrusLogger) Info(args ...interface{}) {
	if l.IsLevelEnabled(LogrusInfoLevel) {
		message := l.formatMessage(sprint(args...), nil)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusInfoLevel))
	}
}

func (l *LogrusLogger) Warn(args ...interface{}) {
	if l.IsLevelEnabled(LogrusWarnLevel) {
		message := l.formatMessage(sprint(args...), nil)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusWarnLevel))
	}
}

func (l *LogrusLogger) Warning(args ...interface{}) {
	l.Warn(args...)
}

func (l *LogrusLogger) Error(args ...interface{}) {
	if l.IsLevelEnabled(LogrusErrorLevel) {
		message := l.formatMessage(sprint(args...), nil)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusErrorLevel))
	}
}

func (l *LogrusLogger) Fatal(args ...interface{}) {
	if l.IsLevelEnabled(LogrusFatalLevel) {
		message := l.formatMessage(sprint(args...), nil)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusFatalLevel))
	}
	os.Exit(1)
}

func (l *LogrusLogger) Panic(args ...interface{}) {
	message := l.formatMessage(sprint(args...), nil)
	if l.IsLevelEnabled(LogrusPanicLevel) {
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusPanicLevel))
	}
	panic(message)
}

// Formatted log methods for LogrusLogger
func (l *LogrusLogger) Tracef(format string, args ...interface{}) {
	if l.IsLevelEnabled(LogrusTraceLevel) {
		message := sprintf(format, args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusTraceLevel))
	}
}

func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	if l.IsLevelEnabled(LogrusDebugLevel) {
		message := sprintf(format, args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusDebugLevel))
	}
}

func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	if l.IsLevelEnabled(LogrusInfoLevel) {
		message := sprintf(format, args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusInfoLevel))
	}
}

func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	if l.IsLevelEnabled(LogrusWarnLevel) {
		message := sprintf(format, args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusWarnLevel))
	}
}

func (l *LogrusLogger) Warningf(format string, args ...interface{}) {
	l.Warnf(format, args...)
}

func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	if l.IsLevelEnabled(LogrusErrorLevel) {
		message := sprintf(format, args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusErrorLevel))
	}
}

func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	if l.IsLevelEnabled(LogrusFatalLevel) {
		message := sprintf(format, args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusFatalLevel))
	}
	os.Exit(1)
}

func (l *LogrusLogger) Panicf(format string, args ...interface{}) {
	message := sprintf(format, args...)
	if l.IsLevelEnabled(LogrusPanicLevel) {
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusPanicLevel))
	}
	panic(message)
}

// Line log methods for LogrusLogger
func (l *LogrusLogger) Traceln(args ...interface{}) {
	if l.IsLevelEnabled(LogrusTraceLevel) {
		message := sprintln(args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusTraceLevel))
	}
}

func (l *LogrusLogger) Debugln(args ...interface{}) {
	if l.IsLevelEnabled(LogrusDebugLevel) {
		message := sprintln(args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusDebugLevel))
	}
}

func (l *LogrusLogger) Infoln(args ...interface{}) {
	if l.IsLevelEnabled(LogrusInfoLevel) {
		message := sprintln(args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusInfoLevel))
	}
}

func (l *LogrusLogger) Warnln(args ...interface{}) {
	if l.IsLevelEnabled(LogrusWarnLevel) {
		message := sprintln(args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusWarnLevel))
	}
}

func (l *LogrusLogger) Warningln(args ...interface{}) {
	l.Warnln(args...)
}

func (l *LogrusLogger) Errorln(args ...interface{}) {
	if l.IsLevelEnabled(LogrusErrorLevel) {
		message := sprintln(args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusErrorLevel))
	}
}

func (l *LogrusLogger) Fatalln(args ...interface{}) {
	if l.IsLevelEnabled(LogrusFatalLevel) {
		message := sprintln(args...)
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusFatalLevel))
	}
	os.Exit(1)
}

func (l *LogrusLogger) Panicln(args ...interface{}) {
	message := sprintln(args...)
	if l.IsLevelEnabled(LogrusPanicLevel) {
		_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), mapLogrusLevel(LogrusPanicLevel))
	}
	panic(message)
}

// LogrusEntry methods
func (e *LogrusEntry) WithField(key string, value interface{}) *LogrusEntry {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	e.Data[key] = value
	return e
}

func (e *LogrusEntry) WithFields(fields map[string]interface{}) *LogrusEntry {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	for k, v := range fields {
		e.Data[k] = v
	}
	return e
}

func (e *LogrusEntry) WithError(err error) *LogrusEntry {
	if err == nil {
		return e.WithField("error", "<nil>")
	}
	return e.WithField("error", err.Error())
}

func (e *LogrusEntry) WithTime(t time.Time) *LogrusEntry {
	e.Time = t
	return e
}

// Log methods for LogrusEntry
func (e *LogrusEntry) Trace(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusTraceLevel) {
		message := e.logger.formatMessage(sprint(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusTraceLevel))
	}
}

func (e *LogrusEntry) Debug(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusDebugLevel) {
		message := e.logger.formatMessage(sprint(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusDebugLevel))
	}
}

func (e *LogrusEntry) Info(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusInfoLevel) {
		message := e.logger.formatMessage(sprint(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusInfoLevel))
	}
}

func (e *LogrusEntry) Warn(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusWarnLevel) {
		message := e.logger.formatMessage(sprint(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusWarnLevel))
	}
}

func (e *LogrusEntry) Warning(args ...interface{}) {
	e.Warn(args...)
}

func (e *LogrusEntry) Error(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusErrorLevel) {
		message := e.logger.formatMessage(sprint(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusErrorLevel))
	}
}

func (e *LogrusEntry) Fatal(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusFatalLevel) {
		message := e.logger.formatMessage(sprint(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusFatalLevel))
	}
	os.Exit(1)
}

func (e *LogrusEntry) Panic(args ...interface{}) {
	message := e.logger.formatMessage(sprint(args...), e.Data)
	if e.logger.IsLevelEnabled(LogrusPanicLevel) {
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusPanicLevel))
	}
	panic(message)
}

// Formatted log methods for LogrusEntry
func (e *LogrusEntry) Tracef(format string, args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusTraceLevel) {
		message := e.logger.formatMessage(sprintf(format, args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusTraceLevel))
	}
}

func (e *LogrusEntry) Debugf(format string, args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusDebugLevel) {
		message := e.logger.formatMessage(sprintf(format, args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusDebugLevel))
	}
}

func (e *LogrusEntry) Infof(format string, args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusInfoLevel) {
		message := e.logger.formatMessage(sprintf(format, args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusInfoLevel))
	}
}

func (e *LogrusEntry) Warnf(format string, args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusWarnLevel) {
		message := e.logger.formatMessage(sprintf(format, args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusWarnLevel))
	}
}

func (e *LogrusEntry) Warningf(format string, args ...interface{}) {
	e.Warnf(format, args...)
}

func (e *LogrusEntry) Errorf(format string, args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusErrorLevel) {
		message := e.logger.formatMessage(sprintf(format, args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusErrorLevel))
	}
}

func (e *LogrusEntry) Fatalf(format string, args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusFatalLevel) {
		message := e.logger.formatMessage(sprintf(format, args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusFatalLevel))
	}
	os.Exit(1)
}

func (e *LogrusEntry) Panicf(format string, args ...interface{}) {
	message := e.logger.formatMessage(sprintf(format, args...), e.Data)
	if e.logger.IsLevelEnabled(LogrusPanicLevel) {
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusPanicLevel))
	}
	panic(message)
}

// Line log methods for LogrusEntry
func (e *LogrusEntry) Traceln(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusTraceLevel) {
		message := e.logger.formatMessage(sprintln(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusTraceLevel))
	}
}

func (e *LogrusEntry) Debugln(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusDebugLevel) {
		message := e.logger.formatMessage(sprintln(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusDebugLevel))
	}
}

func (e *LogrusEntry) Infoln(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusInfoLevel) {
		message := e.logger.formatMessage(sprintln(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusInfoLevel))
	}
}

func (e *LogrusEntry) Warnln(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusWarnLevel) {
		message := e.logger.formatMessage(sprintln(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusWarnLevel))
	}
}

func (e *LogrusEntry) Warningln(args ...interface{}) {
	e.Warnln(args...)
}

func (e *LogrusEntry) Errorln(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusErrorLevel) {
		message := e.logger.formatMessage(sprintln(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusErrorLevel))
	}
}

func (e *LogrusEntry) Fatalln(args ...interface{}) {
	if e.logger.IsLevelEnabled(LogrusFatalLevel) {
		message := e.logger.formatMessage(sprintln(args...), e.Data)
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusFatalLevel))
	}
	os.Exit(1)
}

func (e *LogrusEntry) Panicln(args ...interface{}) {
	message := e.logger.formatMessage(sprintln(args...), e.Data)
	if e.logger.IsLevelEnabled(LogrusPanicLevel) {
		_ = e.logger.client.SendLogWithTimestampAndLevel(message, e.Time, mapLogrusLevel(LogrusPanicLevel))
	}
	panic(message)
}

// Helper functions to match fmt package behavior
func sprint(args ...interface{}) string {
	return fmt.Sprint(args...)
}

func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func sprintln(args ...interface{}) string {
	return fmt.Sprintln(args...)
}

// Close closes the underlying client
func (l *LogrusLogger) Close() error {
	return l.client.Close()
}
