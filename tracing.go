package logflux

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Trace context propagation header.
// Format: <trace_id>-<span_id>-<sampled>
// Example: 4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-1
const TraceHeader = "X-LogFlux-Trace"

// TraceContext holds propagated trace information.
type TraceContext struct {
	TraceID string
	SpanID  string
	Sampled bool
}

// InjectTraceContext sets the trace header on an outgoing HTTP request.
func InjectTraceContext(req *http.Request, span *Span) {
	if span == nil || req == nil {
		return
	}
	sampled := "1"
	req.Header.Set(TraceHeader, fmt.Sprintf("%s-%s-%s", span.TraceID(), span.SpanID(), sampled))
}

// ExtractTraceContext reads trace context from an incoming HTTP request.
// Returns nil if the header is missing or malformed.
func ExtractTraceContext(req *http.Request) *TraceContext {
	if req == nil {
		return nil
	}
	header := req.Header.Get(TraceHeader)
	if header == "" {
		return nil
	}
	return ParseTraceHeader(header)
}

// ParseTraceHeader parses a trace header value.
func ParseTraceHeader(header string) *TraceContext {
	parts := strings.SplitN(header, "-", 3)
	if len(parts) < 2 {
		return nil
	}

	traceID := parts[0]
	spanID := parts[1]

	// Validate lengths
	if len(traceID) != 32 || len(spanID) != 16 {
		return nil
	}

	// Validate hex characters
	if _, err := hex.DecodeString(traceID); err != nil {
		return nil
	}
	if _, err := hex.DecodeString(spanID); err != nil {
		return nil
	}

	sampled := true
	if len(parts) == 3 && parts[2] == "0" {
		sampled = false
	}

	return &TraceContext{
		TraceID: traceID,
		SpanID:  spanID,
		Sampled: sampled,
	}
}

// FormatTraceHeader formats a trace context as a header value.
func FormatTraceHeader(tc *TraceContext) string {
	if tc == nil {
		return ""
	}
	sampled := "1"
	if !tc.Sampled {
		sampled = "0"
	}
	return fmt.Sprintf("%s-%s-%s", tc.TraceID, tc.SpanID, sampled)
}

// ContinueFromRequest creates a child span that continues a trace from an incoming request.
// If no trace header is present, starts a new root span.
func ContinueFromRequest(req *http.Request, operation, name string) *Span {
	tc := ExtractTraceContext(req)
	if tc == nil {
		return StartSpan(operation, name)
	}
	return &Span{
		traceID:      tc.TraceID,
		spanID:       generateSpanID(),
		parentSpanID: tc.SpanID,
		operation:    operation,
		name:         name,
		startTime:    timeNow(),
		status:       "ok",
	}
}

// TracingMiddleware wraps an HTTP handler with automatic span creation.
// Creates a span for each request with operation "http.server" and
// propagates the trace context.
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := ContinueFromRequest(r, "http.server", r.Method+" "+r.URL.Path)
		span.SetAttribute("http.method", r.Method)
		span.SetAttribute("http.url", r.URL.String())

		// Wrap response writer to capture status code
		sw := &statusWriter{ResponseWriter: w, status: 200}

		next.ServeHTTP(sw, r)

		span.SetAttribute("http.status_code", fmt.Sprintf("%d", sw.status))
		if sw.status >= 500 {
			span.SetStatus("error")
		}
		_ = span.End()
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush delegates to the underlying ResponseWriter if it implements http.Flusher.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack delegates to the underlying ResponseWriter if it implements http.Hijacker.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// timeNow is a var for testing.
var timeNow = func() time.Time { return time.Now() }
