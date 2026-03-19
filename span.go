package logflux

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/payload"
)

// Span represents an in-flight trace span. Call End() to finish and send it.
//
// Usage:
//
//	span := logflux.StartSpan("http.server", "GET /api/users")
//	defer span.End()
//	span.SetAttribute("http.method", "GET")
//	// ... do work ...
//	if err != nil {
//	    span.SetStatus("error")
//	}
type Span struct {
	traceID      string
	spanID       string
	parentSpanID string
	operation    string
	name         string
	startTime    time.Time
	status       string
	attributes   Fields
	mu           sync.Mutex
	ended        bool
}

// StartSpan creates and starts a new root span (generates new trace ID).
func StartSpan(operation, name string) *Span {
	return &Span{
		traceID:   generateTraceID(),
		spanID:    generateSpanID(),
		operation: operation,
		name:      name,
		startTime: time.Now(),
		status:    "ok",
	}
}

// StartSpanWithTraceID creates a root span with a specific trace ID.
func StartSpanWithTraceID(traceID, operation, name string) *Span {
	return &Span{
		traceID:   traceID,
		spanID:    generateSpanID(),
		operation: operation,
		name:      name,
		startTime: time.Now(),
		status:    "ok",
	}
}

// StartChild creates a child span under this span (same trace ID).
func (s *Span) StartChild(operation, name string) *Span {
	return &Span{
		traceID:      s.traceID,
		spanID:       generateSpanID(),
		parentSpanID: s.spanID,
		operation:    operation,
		name:         name,
		startTime:    time.Now(),
		status:       "ok",
	}
}

// End finishes the span, computes duration, and sends it as a trace entry.
func (s *Span) End() error {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return nil
	}
	s.ended = true
	endTime := time.Now()
	attrs := s.attributes
	s.mu.Unlock()

	c := getClient()
	if c == nil {
		return nil
	}

	p := payload.NewTrace("", s.traceID, s.spanID, s.operation, s.name, s.startTime, endTime)
	p.ParentSpanID = s.parentSpanID
	p.Status = s.status
	payload.ApplyContext(p)
	if attrs != nil {
		p.SetAttributes(attrs)
	}
	h := getHooks()
	if h.Trace != nil {
		p = h.Trace(p)
		if p == nil {
			return nil
		}
	}

	data, err := payload.Marshal(p)
	if err != nil {
		return err
	}
	return c.SendLogWithEntryType(string(data), models.LogLevelInfo, models.EntryTypeTrace)
}

// --- Setters ---

// SetAttribute sets a span attribute.
func (s *Span) SetAttribute(key, value string) {
	s.mu.Lock()
	if s.attributes == nil {
		s.attributes = make(Fields)
	}
	s.attributes[key] = value
	s.mu.Unlock()
}

// SetAttributes sets multiple span attributes.
func (s *Span) SetAttributes(attrs Fields) {
	s.mu.Lock()
	if s.attributes == nil {
		s.attributes = make(Fields, len(attrs))
	}
	for k, v := range attrs {
		s.attributes[k] = v
	}
	s.mu.Unlock()
}

// SetStatus sets the span status ("ok" or "error").
func (s *Span) SetStatus(status string) {
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()
}

// SetError marks the span as errored and records the error message.
func (s *Span) SetError(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	s.status = "error"
	if s.attributes == nil {
		s.attributes = make(Fields)
	}
	s.attributes["error.message"] = err.Error()
	s.mu.Unlock()
}

// --- Getters ---

// TraceID returns the span's trace ID.
func (s *Span) TraceID() string { return s.traceID }

// SpanID returns the span's span ID.
func (s *Span) SpanID() string { return s.spanID }

// ParentSpanID returns the span's parent span ID.
func (s *Span) ParentSpanID() string { return s.parentSpanID }

// --- ID generation ---

func generateTraceID() string {
	b := make([]byte, 16) // 32 hex chars
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func generateSpanID() string {
	b := make([]byte, 8) // 16 hex chars
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
