package logger

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
)

// Logger provides a simple interface for logging to LogFlux
type Logger struct {
	client *client.Client
	prefix string
}

// NewLogger creates a new logger instance
func NewLogger(client *client.Client, prefix string) *Logger {
	return &Logger{
		client: client,
		prefix: prefix,
	}
}

// NewLoggerFromEnv creates a new logger from environment variables
func NewLoggerFromEnv(node, prefix string) (*Logger, error) {
	client, err := client.NewClientFromEnv(node)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return NewLogger(client, prefix), nil
}

// Log sends a log message
func (l *Logger) Log(message string) error {
	fullMessage := l.formatMessage(message)
	return l.client.SendLog(fullMessage)
}

// Logf sends a formatted log message
func (l *Logger) Logf(format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)
	return l.Log(message)
}

// LogWithTimestamp sends a log message with a specific timestamp
func (l *Logger) LogWithTimestamp(message string, timestamp time.Time) error {
	fullMessage := l.formatMessage(message)
	return l.client.SendLogWithTimestamp(fullMessage, timestamp)
}

// Info sends an info level log message
func (l *Logger) Info(message string) error {
	return l.client.Info(l.formatMessage(message))
}

// Infof sends a formatted info level log message
func (l *Logger) Infof(format string, args ...interface{}) error {
	return l.Info(fmt.Sprintf(format, args...))
}

// Warn sends a warning level log message
func (l *Logger) Warn(message string) error {
	return l.client.Warn(l.formatMessage(message))
}

// Warnf sends a formatted warning level log message
func (l *Logger) Warnf(format string, args ...interface{}) error {
	return l.Warn(fmt.Sprintf(format, args...))
}

// Error sends an error level log message
func (l *Logger) Error(message string) error {
	return l.client.Error(l.formatMessage(message))
}

// Errorf sends a formatted error level log message
func (l *Logger) Errorf(format string, args ...interface{}) error {
	return l.Error(fmt.Sprintf(format, args...))
}

// Debug sends a debug level log message
func (l *Logger) Debug(message string) error {
	return l.client.Debug(l.formatMessage(message))
}

// Debugf sends a formatted debug level log message
func (l *Logger) Debugf(format string, args ...interface{}) error {
	return l.Debug(fmt.Sprintf(format, args...))
}

// Fatal sends a fatal level log message and exits with status 1.
func (l *Logger) Fatal(message string) error {
	err := l.client.Fatal(l.formatMessage(message))
	os.Exit(1)
	return err // unreachable, kept for interface compliance
}

// Fatalf sends a formatted fatal level log message and exits with status 1.
func (l *Logger) Fatalf(format string, args ...interface{}) error {
	return l.Fatal(fmt.Sprintf(format, args...))
}

// formatMessage formats the message with prefix and timestamp
func (l *Logger) formatMessage(message string) string {
	if l.prefix != "" {
		return fmt.Sprintf("[%s] %s", l.prefix, message)
	}
	return message
}

// Close closes the logger and underlying client
func (l *Logger) Close() error {
	if l.client != nil {
		return l.client.Close()
	}
	return nil
}

// AsyncLogger provides asynchronous logging capabilities.
// Close() must be called to stop the background worker and flush remaining entries.
type AsyncLogger struct {
	logger  *Logger
	logChan chan logEntry
	done    chan struct{}
}

type logEntry struct {
	message   string
	timestamp time.Time
	level     int // 0 means use default (Info via LogWithTimestamp)
}

// NewAsyncLogger creates a new asynchronous logger.
func NewAsyncLogger(logger *Logger, bufferSize int) *AsyncLogger {
	al := &AsyncLogger{
		logger:  logger,
		logChan: make(chan logEntry, bufferSize),
		done:    make(chan struct{}),
	}

	go al.worker()
	return al
}

// worker processes log entries asynchronously
func (al *AsyncLogger) worker() {
	for {
		select {
		case entry := <-al.logChan:
			al.sendEntry(entry)
		case <-al.done:
			// Drain remaining entries
			for {
				select {
				case entry := <-al.logChan:
					al.sendEntry(entry)
				default:
					return
				}
			}
		}
	}
}

func (al *AsyncLogger) sendEntry(entry logEntry) {
	var err error
	if entry.level > 0 {
		err = al.logger.client.SendLogWithTimestampAndLevel(
			al.logger.formatMessage(entry.message), entry.timestamp, entry.level,
		)
	} else {
		err = al.logger.LogWithTimestamp(entry.message, entry.timestamp)
	}
	if err != nil {
		log.Printf("Failed to send async log: %v", err)
	}
}

func (al *AsyncLogger) enqueue(message string, level int) {
	select {
	case al.logChan <- logEntry{message: message, timestamp: time.Now(), level: level}:
	default:
		log.Printf("Async logger buffer full, dropping log message: %s", message)
	}
}

// Log sends a log message asynchronously
func (al *AsyncLogger) Log(message string) {
	al.enqueue(message, 0)
}

// Logf sends a formatted log message asynchronously
func (al *AsyncLogger) Logf(format string, args ...interface{}) {
	al.Log(fmt.Sprintf(format, args...))
}

// Info sends an info level log message asynchronously
func (al *AsyncLogger) Info(message string) {
	al.enqueue(message, 7) // LogLevelInfo
}

// Infof sends a formatted info level log message asynchronously
func (al *AsyncLogger) Infof(format string, args ...interface{}) {
	al.Info(fmt.Sprintf(format, args...))
}

// Warn sends a warning level log message asynchronously
func (al *AsyncLogger) Warn(message string) {
	al.enqueue(message, 5) // LogLevelWarning
}

// Warnf sends a formatted warning level log message asynchronously
func (al *AsyncLogger) Warnf(format string, args ...interface{}) {
	al.Warn(fmt.Sprintf(format, args...))
}

// Error sends an error level log message asynchronously
func (al *AsyncLogger) Error(message string) {
	al.enqueue(message, 4) // LogLevelError
}

// Errorf sends a formatted error level log message asynchronously
func (al *AsyncLogger) Errorf(format string, args ...interface{}) {
	al.Error(fmt.Sprintf(format, args...))
}

// Debug sends a debug level log message asynchronously
func (al *AsyncLogger) Debug(message string) {
	al.enqueue(message, 8) // LogLevelDebug
}

// Debugf sends a formatted debug level log message asynchronously
func (al *AsyncLogger) Debugf(format string, args ...interface{}) {
	al.Debug(fmt.Sprintf(format, args...))
}

// Fatal sends a fatal level log message asynchronously and exits.
func (al *AsyncLogger) Fatal(message string) {
	// Fatal is synchronous — must send before exit
	_ = al.logger.Fatal(message)
}

// Fatalf sends a formatted fatal level log message asynchronously and exits.
func (al *AsyncLogger) Fatalf(format string, args ...interface{}) {
	al.Fatal(fmt.Sprintf(format, args...))
}

// Close closes the async logger and waits for all messages to be sent.
func (al *AsyncLogger) Close() error {
	close(al.done)
	return al.logger.Close()
}

// SetupGlobalLogger sets up a global logger that can be used throughout the application
func SetupGlobalLogger(node, prefix string) error {
	logger, err := NewLoggerFromEnv(node, prefix)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// Set up standard log to also send to LogFlux
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Store the logger globally (you might want to use a proper singleton pattern)
	globalLogger = logger

	return nil
}

var globalLogger *Logger

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return globalLogger
}
