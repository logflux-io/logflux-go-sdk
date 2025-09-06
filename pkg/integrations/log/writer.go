package log

import (
	"io"
	"strings"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// Writer implements io.Writer to capture standard log package output.
// It integrates with Go's standard log package by acting as a custom output destination.
type Writer struct {
	client *client.BatchClient
	source string
}

// NewWriter creates a new LogFlux log writer.
// Use with log.SetOutput() to redirect standard log output to LogFlux.
func NewWriter(client *client.BatchClient, source string) *Writer {
	if source == "" {
		source = "log"
	}
	return &Writer{
		client: client,
		source: source,
	}
}

// Write implements io.Writer interface.
// Processes log messages from the standard log package.
func (w *Writer) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	if message == "" {
		return len(p), nil
	}

	// Standard log package doesn't provide level info, so we default to Info
	entry := types.NewLogEntry(message, w.source).
		WithLogLevel(types.LevelInfo)

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
