package adapters

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

// LoggerInterface defines the interface that clients must implement
type LoggerInterface interface {
	SendLogWithTimestampAndLevel(message string, timestamp time.Time, level int) error
	Close() error
}

// StdlibLogger provides a drop-in replacement for the standard library log.Logger
type StdlibLogger struct {
	client LoggerInterface
	prefix string
}

// NewStdlibLogger creates a new standard library logger adapter
func NewStdlibLogger(client LoggerInterface, prefix string) *StdlibLogger {
	return &StdlibLogger{
		client: client,
		prefix: prefix,
	}
}

// NewStdlibLoggerFromEnv creates a standard library logger adapter from environment variables
func NewStdlibLoggerFromEnv(node, prefix string) (*StdlibLogger, error) {
	resilientClient, err := client.NewResilientClientFromEnvWithHandshake(node)
	if err != nil {
		return nil, err
	}

	return NewStdlibLogger(resilientClient, prefix), nil
}

// Write implements io.Writer interface for compatibility with standard library
func (l *StdlibLogger) Write(p []byte) (n int, err error) {
	message := string(p)
	message = strings.TrimSpace(message)

	// Remove common log prefixes if present
	if strings.Contains(message, ": ") {
		parts := strings.SplitN(message, ": ", 2)
		if len(parts) == 2 {
			message = parts[1]
		}
	}

	// Add prefix if configured
	if l.prefix != "" {
		message = l.prefix + message
	}

	// Send as Info level by default
	err = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelInfo)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// Print calls Output to print to the standard logger
func (l *StdlibLogger) Print(v ...interface{}) {
	message := l.prefix + fmt.Sprint(v...)
	_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelInfo)
}

// Printf calls Output to print to the standard logger
func (l *StdlibLogger) Printf(format string, v ...interface{}) {
	message := l.prefix + fmt.Sprintf(format, v...)
	_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelInfo)
}

// Println calls Output to print to the standard logger
func (l *StdlibLogger) Println(v ...interface{}) {
	message := l.prefix + fmt.Sprintln(v...)
	_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelInfo)
}

// Fatal is equivalent to Print() followed by a call to os.Exit(1)
func (l *StdlibLogger) Fatal(v ...interface{}) {
	message := l.prefix + fmt.Sprint(v...)
	_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelCritical)
	os.Exit(1)
}

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1)
func (l *StdlibLogger) Fatalf(format string, v ...interface{}) {
	message := l.prefix + fmt.Sprintf(format, v...)
	_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelCritical)
	os.Exit(1)
}

// Fatalln is equivalent to Println() followed by a call to os.Exit(1)
func (l *StdlibLogger) Fatalln(v ...interface{}) {
	message := l.prefix + fmt.Sprintln(v...)
	_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelCritical)
	os.Exit(1)
}

// Panic is equivalent to Print() followed by a call to panic()
func (l *StdlibLogger) Panic(v ...interface{}) {
	message := l.prefix + fmt.Sprint(v...)
	_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelCritical)
	panic(message)
}

// Panicf is equivalent to Printf() followed by a call to panic()
func (l *StdlibLogger) Panicf(format string, v ...interface{}) {
	message := l.prefix + fmt.Sprintf(format, v...)
	_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelCritical)
	panic(message)
}

// Panicln is equivalent to Println() followed by a call to panic()
func (l *StdlibLogger) Panicln(v ...interface{}) {
	message := l.prefix + fmt.Sprintln(v...)
	_ = l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelCritical)
	panic(message)
}

// SetOutput sets the output destination for the logger
func (l *StdlibLogger) SetOutput(w io.Writer) {
	// LogFlux sends to backend, so we ignore this
}

// SetPrefix sets the output prefix for the logger
func (l *StdlibLogger) SetPrefix(prefix string) {
	l.prefix = prefix
}

// SetFlags sets the output flags for the logger
func (l *StdlibLogger) SetFlags(flag int) {
	// LogFlux handles formatting, so we ignore this
}

// Prefix returns the output prefix for the logger
func (l *StdlibLogger) Prefix() string {
	return l.prefix
}

// Flags returns the output flags for the logger
func (l *StdlibLogger) Flags() int {
	// Return default flags since we don't use them
	return log.LstdFlags
}

// Output writes the output for a logging event
func (l *StdlibLogger) Output(calldepth int, s string) error {
	message := l.prefix + s
	return l.client.SendLogWithTimestampAndLevel(message, time.Now(), models.LogLevelInfo)
}

// Close closes the underlying client
func (l *StdlibLogger) Close() error {
	return l.client.Close()
}

// ReplaceStandardLogger replaces the standard library's default logger with LogFlux
func ReplaceStandardLogger(client LoggerInterface, prefix string) *StdlibLogger {
	adapter := NewStdlibLogger(client, prefix)

	// Replace the standard logger's output
	log.SetOutput(adapter)

	return adapter
}

// ReplaceStandardLoggerFromEnv replaces the standard library's default logger using environment config
func ReplaceStandardLoggerFromEnv(node, prefix string) (*StdlibLogger, error) {
	adapter, err := NewStdlibLoggerFromEnv(node, prefix)
	if err != nil {
		return nil, err
	}

	// Replace the standard logger's output
	log.SetOutput(adapter)

	return adapter, nil
}
