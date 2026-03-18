package adapters

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

// ZerologLevel represents zerolog log levels
type ZerologLevel int8

const (
	// ZerologDebugLevel defines debug log level
	ZerologDebugLevel ZerologLevel = iota
	// ZerologInfoLevel defines info log level
	ZerologInfoLevel
	// ZerologWarnLevel defines warn log level
	ZerologWarnLevel
	// ZerologErrorLevel defines error log level
	ZerologErrorLevel
	// ZerologFatalLevel defines fatal log level
	ZerologFatalLevel
	// ZerologPanicLevel defines panic log level
	ZerologPanicLevel
	// ZerologNoLevel defines an absent level
	ZerologNoLevel
	// ZerologDisabled disables the logger
	ZerologDisabled
	// ZerologTraceLevel defines trace log level
	ZerologTraceLevel = -1
)

// ZerologLogger provides a drop-in replacement for zerolog.Logger
type ZerologLogger struct {
	client LoggerInterface
	level  ZerologLevel
	fields map[string]interface{}
}

// ZerologEvent represents a log event in zerolog
type ZerologEvent struct {
	logger *ZerologLogger
	level  ZerologLevel
	fields map[string]interface{}
}

// NewZerologLogger creates a new zerolog logger adapter
func NewZerologLogger(client LoggerInterface) *ZerologLogger {
	return &ZerologLogger{
		client: client,
		level:  ZerologInfoLevel,
		fields: make(map[string]interface{}),
	}
}

// NewZerologLoggerFromEnv creates a zerolog logger adapter from environment variables
func NewZerologLoggerFromEnv(node string) (*ZerologLogger, error) {
	resilientClient, err := client.NewResilientClientFromEnvWithHandshake(node)
	if err != nil {
		return nil, err
	}

	return NewZerologLogger(resilientClient), nil
}

// mapZerologLevel converts zerolog levels to LogFlux levels
func mapZerologLevel(level ZerologLevel) int {
	switch level {
	case ZerologTraceLevel, ZerologDebugLevel:
		return models.LogLevelDebug
	case ZerologInfoLevel:
		return models.LogLevelInfo
	case ZerologWarnLevel:
		return models.LogLevelWarning
	case ZerologErrorLevel:
		return models.LogLevelError
	case ZerologFatalLevel, ZerologPanicLevel:
		return models.LogLevelCritical
	default:
		return models.LogLevelInfo
	}
}

// formatZerologMessage formats a message with structured fields
func (l *ZerologLogger) formatZerologMessage(message string, fields map[string]interface{}) string {
	if len(fields) == 0 {
		return message
	}

	var parts []string
	for key, value := range fields {
		parts = append(parts, key+"="+formatZerologValue(value))
	}

	if message != "" {
		return message + " " + strings.Join(parts, " ")
	}
	return strings.Join(parts, " ")
}

// formatZerologValue formats a field value as string
func formatZerologValue(value interface{}) string {
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
	case time.Time:
		return v.Format(time.RFC3339)
	case time.Duration:
		return v.String()
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Logger configuration methods
func (l *ZerologLogger) Level(level ZerologLevel) *ZerologLogger {
	newLogger := &ZerologLogger{
		client: l.client,
		level:  level,
		fields: make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

func (l *ZerologLogger) Sample(s interface{}) *ZerologLogger {
	// For simplicity, we'll ignore sampling and return a copy
	return l.With()
}

func (l *ZerologLogger) Hook(h interface{}) *ZerologLogger {
	// For simplicity, we'll ignore hooks and return a copy
	return l.With()
}

func (l *ZerologLogger) Output(w io.Writer) *ZerologLogger {
	// LogFlux handles output, so we ignore this and return a copy
	return l.With()
}

func (l *ZerologLogger) With() *ZerologLogger {
	newLogger := &ZerologLogger{
		client: l.client,
		level:  l.level,
		fields: make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// Context methods for adding fields
func (l *ZerologLogger) Str(key, val string) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Strs(key string, vals []string) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = strings.Join(vals, ",")
	return newLogger
}

func (l *ZerologLogger) Int(key string, val int) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Int64(key string, val int64) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Uint(key string, val uint) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Uint64(key string, val uint64) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Float32(key string, val float32) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Float64(key string, val float64) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Bool(key string, val bool) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Time(key string, val time.Time) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Dur(key string, val time.Duration) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Bytes(key string, val []byte) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Hex(key string, val []byte) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = fmt.Sprintf("%x", val)
	return newLogger
}

func (l *ZerologLogger) Interface(key string, val interface{}) *ZerologLogger {
	newLogger := l.With()
	newLogger.fields[key] = val
	return newLogger
}

func (l *ZerologLogger) Err(err error) *ZerologLogger {
	if err == nil {
		return l
	}
	newLogger := l.With()
	newLogger.fields["error"] = err.Error()
	return newLogger
}

func (l *ZerologLogger) AnErr(key string, err error) *ZerologLogger {
	if err == nil {
		return l
	}
	newLogger := l.With()
	newLogger.fields[key] = err.Error()
	return newLogger
}

func (l *ZerologLogger) Errs(key string, errs []error) *ZerologLogger {
	if len(errs) == 0 {
		return l
	}
	newLogger := l.With()
	var errStrs []string
	for _, err := range errs {
		errStrs = append(errStrs, err.Error())
	}
	newLogger.fields[key] = strings.Join(errStrs, ",")
	return newLogger
}

func (l *ZerologLogger) Timestamp() *ZerologLogger {
	newLogger := l.With()
	newLogger.fields["timestamp"] = time.Now()
	return newLogger
}

func (l *ZerologLogger) Stack() *ZerologLogger {
	newLogger := l.With()
	newLogger.fields["stack"] = "stack_trace_placeholder"
	return newLogger
}

func (l *ZerologLogger) Caller() *ZerologLogger {
	newLogger := l.With()
	newLogger.fields["caller"] = "caller_placeholder"
	return newLogger
}

// Log level methods that return events
func (l *ZerologLogger) Trace() *ZerologEvent {
	if l.level > ZerologTraceLevel {
		return &ZerologEvent{logger: l, level: ZerologDisabled}
	}
	return &ZerologEvent{
		logger: l,
		level:  ZerologTraceLevel,
		fields: make(map[string]interface{}),
	}
}

func (l *ZerologLogger) Debug() *ZerologEvent {
	if l.level > ZerologDebugLevel {
		return &ZerologEvent{logger: l, level: ZerologDisabled}
	}
	return &ZerologEvent{
		logger: l,
		level:  ZerologDebugLevel,
		fields: make(map[string]interface{}),
	}
}

func (l *ZerologLogger) Info() *ZerologEvent {
	if l.level > ZerologInfoLevel {
		return &ZerologEvent{logger: l, level: ZerologDisabled}
	}
	return &ZerologEvent{
		logger: l,
		level:  ZerologInfoLevel,
		fields: make(map[string]interface{}),
	}
}

func (l *ZerologLogger) Warn() *ZerologEvent {
	if l.level > ZerologWarnLevel {
		return &ZerologEvent{logger: l, level: ZerologDisabled}
	}
	return &ZerologEvent{
		logger: l,
		level:  ZerologWarnLevel,
		fields: make(map[string]interface{}),
	}
}

func (l *ZerologLogger) Error() *ZerologEvent {
	if l.level > ZerologErrorLevel {
		return &ZerologEvent{logger: l, level: ZerologDisabled}
	}
	return &ZerologEvent{
		logger: l,
		level:  ZerologErrorLevel,
		fields: make(map[string]interface{}),
	}
}

func (l *ZerologLogger) Fatal() *ZerologEvent {
	if l.level > ZerologFatalLevel {
		return &ZerologEvent{logger: l, level: ZerologDisabled}
	}
	return &ZerologEvent{
		logger: l,
		level:  ZerologFatalLevel,
		fields: make(map[string]interface{}),
	}
}

func (l *ZerologLogger) Panic() *ZerologEvent {
	if l.level > ZerologPanicLevel {
		return &ZerologEvent{logger: l, level: ZerologDisabled}
	}
	return &ZerologEvent{
		logger: l,
		level:  ZerologPanicLevel,
		fields: make(map[string]interface{}),
	}
}

func (l *ZerologLogger) Log() *ZerologEvent {
	return &ZerologEvent{
		logger: l,
		level:  ZerologNoLevel,
		fields: make(map[string]interface{}),
	}
}

// Print methods for printf-style logging
func (l *ZerologLogger) Print(v ...interface{}) {
	msg := fmt.Sprint(v...)
	_ = l.client.SendLogWithTimestampAndLevel(l.formatZerologMessage(msg, l.fields), time.Now(), models.LogLevelInfo)
}

func (l *ZerologLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	_ = l.client.SendLogWithTimestampAndLevel(l.formatZerologMessage(msg, l.fields), time.Now(), models.LogLevelInfo)
}

func (l *ZerologLogger) Close() error {
	return l.client.Close()
}

// ZerologEvent methods for fluent interface
func (e *ZerologEvent) Str(key, val string) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Strs(key string, vals []string) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = strings.Join(vals, ",")
	return e
}

func (e *ZerologEvent) Int(key string, val int) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Int64(key string, val int64) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Uint(key string, val uint) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Uint64(key string, val uint64) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Float32(key string, val float32) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Float64(key string, val float64) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Bool(key string, val bool) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Time(key string, val time.Time) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Dur(key string, val time.Duration) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Bytes(key string, val []byte) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Hex(key string, val []byte) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = fmt.Sprintf("%x", val)
	return e
}

func (e *ZerologEvent) Interface(key string, val interface{}) *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields[key] = val
	return e
}

func (e *ZerologEvent) Err(err error) *ZerologEvent {
	if e.level == ZerologDisabled || err == nil {
		return e
	}
	e.fields["error"] = err.Error()
	return e
}

func (e *ZerologEvent) AnErr(key string, err error) *ZerologEvent {
	if e.level == ZerologDisabled || err == nil {
		return e
	}
	e.fields[key] = err.Error()
	return e
}

func (e *ZerologEvent) Timestamp() *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields["timestamp"] = time.Now()
	return e
}

func (e *ZerologEvent) Stack() *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields["stack"] = "stack_trace_placeholder"
	return e
}

func (e *ZerologEvent) Caller() *ZerologEvent {
	if e.level == ZerologDisabled {
		return e
	}
	e.fields["caller"] = "caller_placeholder"
	return e
}

// Event termination methods
func (e *ZerologEvent) Msg(msg string) {
	if e.level == ZerologDisabled {
		return
	}

	// Combine logger fields with event fields
	allFields := make(map[string]interface{})
	for k, v := range e.logger.fields {
		allFields[k] = v
	}
	for k, v := range e.fields {
		allFields[k] = v
	}

	finalMsg := e.logger.formatZerologMessage(msg, allFields)
	_ = e.logger.client.SendLogWithTimestampAndLevel(finalMsg, time.Now(), mapZerologLevel(e.level))
}

func (e *ZerologEvent) Msgf(format string, v ...interface{}) {
	if e.level == ZerologDisabled {
		return
	}
	msg := fmt.Sprintf(format, v...)
	e.Msg(msg)
}

func (e *ZerologEvent) Send() {
	if e.level == ZerologDisabled {
		return
	}
	e.Msg("")
}

// Discard method for when events are disabled
func (e *ZerologEvent) Discard() *ZerologEvent {
	e.level = ZerologDisabled
	return e
}

// Enabled checks if the event is enabled
func (e *ZerologEvent) Enabled() bool {
	return e.level != ZerologDisabled
}
