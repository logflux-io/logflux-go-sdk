package adapters

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

// ZapLevel represents zap log levels
type ZapLevel int8

const (
	// ZapDebugLevel logs are typically voluminous, and are usually disabled in production
	ZapDebugLevel ZapLevel = iota - 1
	// ZapInfoLevel is the default logging priority
	ZapInfoLevel
	// ZapWarnLevel logs are more important than Info, but don't need individual human review
	ZapWarnLevel
	// ZapErrorLevel logs are high-priority. If an application is running smoothly, it shouldn't generate any error-level logs
	ZapErrorLevel
	// ZapDPanicLevel logs are particularly important errors. In development the logger panics after writing the message
	ZapDPanicLevel
	// ZapPanicLevel logs a message, then panics
	ZapPanicLevel
	// ZapFatalLevel logs a message, then calls os.Exit(1)
	ZapFatalLevel
)

// ZapField represents a structured log field
type ZapField struct {
	Key   string
	Value interface{}
}

// ZapLogger provides a drop-in replacement for zap.Logger
type ZapLogger struct {
	client LoggerInterface
	level  ZapLevel
	fields []ZapField
}

// NewZapLogger creates a new zap logger adapter
func NewZapLogger(client LoggerInterface) *ZapLogger {
	return &ZapLogger{
		client: client,
		level:  ZapInfoLevel,
		fields: make([]ZapField, 0),
	}
}

// NewZapLoggerFromEnv creates a zap logger adapter from environment variables
func NewZapLoggerFromEnv(node string) (*ZapLogger, error) {
	resilientClient, err := client.NewResilientClientFromEnvWithHandshake(node)
	if err != nil {
		return nil, err
	}

	return NewZapLogger(resilientClient), nil
}

// mapZapLevel converts zap levels to LogFlux levels
func mapZapLevel(level ZapLevel) int {
	switch level {
	case ZapDebugLevel:
		return models.LogLevelDebug
	case ZapInfoLevel:
		return models.LogLevelInfo
	case ZapWarnLevel:
		return models.LogLevelWarning
	case ZapErrorLevel:
		return models.LogLevelError
	case ZapDPanicLevel, ZapPanicLevel, ZapFatalLevel:
		return models.LogLevelCritical
	default:
		return models.LogLevelInfo
	}
}

// formatMessage formats a message with structured fields
func (l *ZapLogger) formatMessage(message string, fields []ZapField) string {
	if len(fields) == 0 {
		return message
	}

	var parts []string
	for _, field := range fields {
		parts = append(parts, field.Key+"="+formatZapValue(field.Value))
	}

	if message != "" {
		return message + " " + strings.Join(parts, " ")
	}
	return strings.Join(parts, " ")
}

// formatZapValue formats a field value as string
func formatZapValue(value interface{}) string {
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
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Core methods that match zap's interface
func (l *ZapLogger) Core() interface{} {
	return l // Return self as core for compatibility
}

func (l *ZapLogger) Sugar() *ZapSugar {
	return NewZapSugar(l)
}

func (l *ZapLogger) Named(name string) *ZapLogger {
	newLogger := &ZapLogger{
		client: l.client,
		level:  l.level,
		fields: make([]ZapField, len(l.fields)),
	}
	copy(newLogger.fields, l.fields)
	newLogger.fields = append(newLogger.fields, ZapField{Key: "logger", Value: name})
	return newLogger
}

func (l *ZapLogger) With(fields ...ZapField) *ZapLogger {
	newLogger := &ZapLogger{
		client: l.client,
		level:  l.level,
		fields: make([]ZapField, len(l.fields)+len(fields)),
	}
	copy(newLogger.fields, l.fields)
	copy(newLogger.fields[len(l.fields):], fields)
	return newLogger
}

func (l *ZapLogger) WithOptions(opts ...interface{}) *ZapLogger {
	// For simplicity, we'll ignore options and return a copy
	return l.With()
}

func (l *ZapLogger) Level() ZapLevel {
	return l.level
}

func (l *ZapLogger) Check(level ZapLevel, message string) *ZapLogger {
	if level >= l.level {
		return l
	}
	return nil
}

// Structured logging methods
func (l *ZapLogger) Debug(message string, fields ...ZapField) {
	if l.level <= ZapDebugLevel {
		allFields := append(l.fields, fields...)
		msg := l.formatMessage(message, allFields)
		_ = l.client.SendLogWithTimestampAndLevel(msg, time.Now(), mapZapLevel(ZapDebugLevel))
	}
}

func (l *ZapLogger) Info(message string, fields ...ZapField) {
	if l.level <= ZapInfoLevel {
		allFields := append(l.fields, fields...)
		msg := l.formatMessage(message, allFields)
		_ = l.client.SendLogWithTimestampAndLevel(msg, time.Now(), mapZapLevel(ZapInfoLevel))
	}
}

func (l *ZapLogger) Warn(message string, fields ...ZapField) {
	if l.level <= ZapWarnLevel {
		allFields := append(l.fields, fields...)
		msg := l.formatMessage(message, allFields)
		_ = l.client.SendLogWithTimestampAndLevel(msg, time.Now(), mapZapLevel(ZapWarnLevel))
	}
}

func (l *ZapLogger) Error(message string, fields ...ZapField) {
	if l.level <= ZapErrorLevel {
		allFields := append(l.fields, fields...)
		msg := l.formatMessage(message, allFields)
		_ = l.client.SendLogWithTimestampAndLevel(msg, time.Now(), mapZapLevel(ZapErrorLevel))
	}
}

func (l *ZapLogger) DPanic(message string, fields ...ZapField) {
	if l.level <= ZapDPanicLevel {
		allFields := append(l.fields, fields...)
		msg := l.formatMessage(message, allFields)
		_ = l.client.SendLogWithTimestampAndLevel(msg, time.Now(), mapZapLevel(ZapDPanicLevel))
		// In development, this would panic, but we'll skip for compatibility
	}
}

func (l *ZapLogger) Panic(message string, fields ...ZapField) {
	if l.level <= ZapPanicLevel {
		allFields := append(l.fields, fields...)
		msg := l.formatMessage(message, allFields)
		_ = l.client.SendLogWithTimestampAndLevel(msg, time.Now(), mapZapLevel(ZapPanicLevel))
		panic(msg)
	}
}

func (l *ZapLogger) Fatal(message string, fields ...ZapField) {
	if l.level <= ZapFatalLevel {
		allFields := append(l.fields, fields...)
		msg := l.formatMessage(message, allFields)
		_ = l.client.SendLogWithTimestampAndLevel(msg, time.Now(), mapZapLevel(ZapFatalLevel))
	}
	os.Exit(1)
}

func (l *ZapLogger) Sync() error {
	// For compatibility with zap's Sync method
	return nil
}

func (l *ZapLogger) Close() error {
	return l.client.Close()
}

// Field constructors (matching zap's field constructors)
func String(key, val string) ZapField {
	return ZapField{Key: key, Value: val}
}

func Int(key string, val int) ZapField {
	return ZapField{Key: key, Value: val}
}

func Int64(key string, val int64) ZapField {
	return ZapField{Key: key, Value: val}
}

func Uint(key string, val uint) ZapField {
	return ZapField{Key: key, Value: val}
}

func Uint64(key string, val uint64) ZapField {
	return ZapField{Key: key, Value: val}
}

func Float64(key string, val float64) ZapField {
	return ZapField{Key: key, Value: val}
}

func Bool(key string, val bool) ZapField {
	return ZapField{Key: key, Value: val}
}

func Time(key string, val time.Time) ZapField {
	return ZapField{Key: key, Value: val}
}

func Duration(key string, val time.Duration) ZapField {
	return ZapField{Key: key, Value: val}
}

func Any(key string, val interface{}) ZapField {
	return ZapField{Key: key, Value: val}
}

func Error(err error) ZapField {
	if err == nil {
		return ZapField{Key: "error", Value: "<nil>"}
	}
	return ZapField{Key: "error", Value: err.Error()}
}

func NamedError(key string, err error) ZapField {
	if err == nil {
		return ZapField{Key: key, Value: "<nil>"}
	}
	return ZapField{Key: key, Value: err.Error()}
}

// ZapSugar provides a sugared logger interface similar to zap's SugaredLogger
type ZapSugar struct {
	logger *ZapLogger
}

func NewZapSugar(logger *ZapLogger) *ZapSugar {
	return &ZapSugar{logger: logger}
}

func (s *ZapSugar) Desugar() *ZapLogger {
	return s.logger
}

func (s *ZapSugar) Named(name string) *ZapSugar {
	return NewZapSugar(s.logger.Named(name))
}

func (s *ZapSugar) With(args ...interface{}) *ZapSugar {
	fields := make([]ZapField, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields = append(fields, ZapField{Key: key, Value: args[i+1]})
		}
	}
	return NewZapSugar(s.logger.With(fields...))
}

// Sugar logging methods
func (s *ZapSugar) Debug(args ...interface{}) {
	msg := fmt.Sprint(args...)
	s.logger.Debug(msg)
}

func (s *ZapSugar) Debugf(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	s.logger.Debug(msg)
}

func (s *ZapSugar) Debugw(msg string, keysAndValues ...interface{}) {
	fields := make([]ZapField, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields = append(fields, ZapField{Key: key, Value: keysAndValues[i+1]})
		}
	}
	s.logger.Debug(msg, fields...)
}

func (s *ZapSugar) Info(args ...interface{}) {
	msg := fmt.Sprint(args...)
	s.logger.Info(msg)
}

func (s *ZapSugar) Infof(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	s.logger.Info(msg)
}

func (s *ZapSugar) Infow(msg string, keysAndValues ...interface{}) {
	fields := make([]ZapField, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields = append(fields, ZapField{Key: key, Value: keysAndValues[i+1]})
		}
	}
	s.logger.Info(msg, fields...)
}

func (s *ZapSugar) Warn(args ...interface{}) {
	msg := fmt.Sprint(args...)
	s.logger.Warn(msg)
}

func (s *ZapSugar) Warnf(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	s.logger.Warn(msg)
}

func (s *ZapSugar) Warnw(msg string, keysAndValues ...interface{}) {
	fields := make([]ZapField, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields = append(fields, ZapField{Key: key, Value: keysAndValues[i+1]})
		}
	}
	s.logger.Warn(msg, fields...)
}

func (s *ZapSugar) Error(args ...interface{}) {
	msg := fmt.Sprint(args...)
	s.logger.Error(msg)
}

func (s *ZapSugar) Errorf(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	s.logger.Error(msg)
}

func (s *ZapSugar) Errorw(msg string, keysAndValues ...interface{}) {
	fields := make([]ZapField, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields = append(fields, ZapField{Key: key, Value: keysAndValues[i+1]})
		}
	}
	s.logger.Error(msg, fields...)
}

func (s *ZapSugar) DPanic(args ...interface{}) {
	msg := fmt.Sprint(args...)
	s.logger.DPanic(msg)
}

func (s *ZapSugar) DPanicf(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	s.logger.DPanic(msg)
}

func (s *ZapSugar) DPanicw(msg string, keysAndValues ...interface{}) {
	fields := make([]ZapField, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields = append(fields, ZapField{Key: key, Value: keysAndValues[i+1]})
		}
	}
	s.logger.DPanic(msg, fields...)
}

func (s *ZapSugar) Panic(args ...interface{}) {
	msg := fmt.Sprint(args...)
	s.logger.Panic(msg)
}

func (s *ZapSugar) Panicf(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	s.logger.Panic(msg)
}

func (s *ZapSugar) Panicw(msg string, keysAndValues ...interface{}) {
	fields := make([]ZapField, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields = append(fields, ZapField{Key: key, Value: keysAndValues[i+1]})
		}
	}
	s.logger.Panic(msg, fields...)
}

func (s *ZapSugar) Fatal(args ...interface{}) {
	msg := fmt.Sprint(args...)
	s.logger.Fatal(msg)
}

func (s *ZapSugar) Fatalf(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	s.logger.Fatal(msg)
}

func (s *ZapSugar) Fatalw(msg string, keysAndValues ...interface{}) {
	fields := make([]ZapField, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields = append(fields, ZapField{Key: key, Value: keysAndValues[i+1]})
		}
	}
	s.logger.Fatal(msg, fields...)
}

func (s *ZapSugar) Sync() error {
	return s.logger.Sync()
}

func (s *ZapSugar) Close() error {
	return s.logger.Close()
}
