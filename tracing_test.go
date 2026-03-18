package logflux

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInjectExtractTraceContext(t *testing.T) {
	span := StartSpan("http.client", "GET /api")

	req := httptest.NewRequest("GET", "/api", nil)
	InjectTraceContext(req, span)

	header := req.Header.Get(TraceHeader)
	if header == "" {
		t.Fatal("expected trace header to be set")
	}

	tc := ExtractTraceContext(req)
	if tc == nil {
		t.Fatal("expected trace context to be extracted")
	}
	if tc.TraceID != span.TraceID() {
		t.Errorf("trace ID mismatch: %s != %s", tc.TraceID, span.TraceID())
	}
	if tc.SpanID != span.SpanID() {
		t.Errorf("span ID mismatch: %s != %s", tc.SpanID, span.SpanID())
	}
	if !tc.Sampled {
		t.Error("expected sampled=true")
	}
}

func TestParseTraceHeader_Valid(t *testing.T) {
	tc := ParseTraceHeader("4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-1")
	if tc == nil {
		t.Fatal("expected valid parse")
	}
	if tc.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("wrong trace ID: %s", tc.TraceID)
	}
	if tc.SpanID != "00f067aa0ba902b7" {
		t.Errorf("wrong span ID: %s", tc.SpanID)
	}
	if !tc.Sampled {
		t.Error("expected sampled=true")
	}
}

func TestParseTraceHeader_Unsampled(t *testing.T) {
	tc := ParseTraceHeader("4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-0")
	if tc == nil {
		t.Fatal("expected valid parse")
	}
	if tc.Sampled {
		t.Error("expected sampled=false")
	}
}

func TestParseTraceHeader_Invalid(t *testing.T) {
	cases := []string{
		"",
		"abc",
		"abc-def",
		"short-00f067aa0ba902b7-1",
		"4bf92f3577b34da6a3ce929d0e0e4736-short-1",
	}
	for _, c := range cases {
		if tc := ParseTraceHeader(c); tc != nil {
			t.Errorf("expected nil for %q, got %+v", c, tc)
		}
	}
}

func TestExtractTraceContext_MissingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	tc := ExtractTraceContext(req)
	if tc != nil {
		t.Error("expected nil for missing header")
	}
}

func TestContinueFromRequest_WithHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set(TraceHeader, "4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-1")

	span := ContinueFromRequest(req, "http.server", "GET /api/users")

	if span.TraceID() != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Error("expected inherited trace ID")
	}
	if span.ParentSpanID() != "00f067aa0ba902b7" {
		t.Error("expected parent span ID from header")
	}
}

func TestContinueFromRequest_NoHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)

	span := ContinueFromRequest(req, "http.server", "GET /api/users")

	if len(span.TraceID()) != 32 {
		t.Error("expected new trace ID")
	}
	if span.ParentSpanID() != "" {
		t.Error("expected no parent for new trace")
	}
}

func TestTracingMiddleware(t *testing.T) {
	called := false
	handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler not called")
	}
	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestFormatTraceHeader(t *testing.T) {
	tc := &TraceContext{
		TraceID: "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:  "00f067aa0ba902b7",
		Sampled: true,
	}
	header := FormatTraceHeader(tc)
	expected := "4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-1"
	if header != expected {
		t.Errorf("expected %s, got %s", expected, header)
	}
}
