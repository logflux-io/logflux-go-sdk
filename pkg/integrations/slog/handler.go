package slog

import (
	"context"
	"log/slog"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// Handler implements slog.Handler to send logs to LogFlux.
// It integrates with Go's standard structured logging library (Go 1.21+).
type Handler struct {
	client *client.BatchClient
	source string
	attrs  []slog.Attr
}

// NewHandler creates a new LogFlux slog handler.
// Uses batch client for better performance with structured logging.
func NewHandler(client *client.BatchClient, source string) *Handler {
	if source == "" {
		source = "slog"
	}
	return &Handler{
		client: client,
		source: source,
	}
}

// Enabled reports whether the handler handles records at the given level.
// Currently accepts all levels.
func (h *Handler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle processes a log record and sends it to LogFlux.
func (h *Handler) Handle(_ context.Context, record slog.Record) error {
	// Convert slog level to LogFlux level
	logLevel := convertLevel(record.Level)

	// Create LogFlux entry
	entry := types.NewLogEntry(record.Message, h.source).
		WithLogLevel(logLevel)

	// Add attributes as metadata
	record.Attrs(func(attr slog.Attr) bool {
		entry = entry.WithMetadata(attr.Key, attr.Value.String())
		return true
	})

	// Add handler-level attributes
	for _, attr := range h.attrs {
		entry = entry.WithMetadata(attr.Key, attr.Value.String())
	}

	return h.client.SendLogEntry(entry)
}

// WithAttrs returns a new Handler with additional attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &Handler{
		client: h.client,
		source: h.source,
		attrs:  newAttrs,
	}
}

// WithGroup returns a new Handler with a group name.
// Groups are flattened into metadata keys with dot notation.
func (h *Handler) WithGroup(name string) slog.Handler {
	// For simplicity, we'll prefix future attributes with the group name
	return &Handler{
		client: h.client,
		source: h.source + "." + name,
		attrs:  h.attrs,
	}
}

// convertLevel converts slog.Level to LogFlux log level
func convertLevel(level slog.Level) int {
	switch {
	case level >= slog.LevelError:
		return types.LevelError
	case level >= slog.LevelWarn:
		return types.LevelWarning
	case level >= slog.LevelInfo:
		return types.LevelInfo
	default: // DEBUG and below
		return types.LevelDebug
	}
}
