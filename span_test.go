package logflux

import (
	"testing"
	"time"
)

func TestStartSpan_GeneratesIDs(t *testing.T) {
	span := StartSpan("http.server", "GET /users")

	if len(span.TraceID()) != 32 {
		t.Errorf("expected 32-char trace ID, got %d: %s", len(span.TraceID()), span.TraceID())
	}
	if len(span.SpanID()) != 16 {
		t.Errorf("expected 16-char span ID, got %d: %s", len(span.SpanID()), span.SpanID())
	}
	if span.ParentSpanID() != "" {
		t.Error("root span should have no parent")
	}
}

func TestStartChild_InheritsTraceID(t *testing.T) {
	parent := StartSpan("http.server", "GET /users")
	child := parent.StartChild("db.query", "SELECT * FROM users")

	if child.TraceID() != parent.TraceID() {
		t.Error("child should inherit parent's trace ID")
	}
	if child.ParentSpanID() != parent.SpanID() {
		t.Error("child's parent should be parent's span ID")
	}
	if child.SpanID() == parent.SpanID() {
		t.Error("child should have its own span ID")
	}
}

func TestSpan_SetError(t *testing.T) {
	span := StartSpan("http.server", "GET /users")
	span.SetError(errForTest("connection refused"))

	span.mu.Lock()
	if span.status != "error" {
		t.Error("expected error status")
	}
	if span.attributes["error.message"] != "connection refused" {
		t.Error("expected error.message attribute")
	}
	span.mu.Unlock()
}

func TestSpan_EndIdempotent(t *testing.T) {
	// End() without a global client should be a no-op and not panic
	span := StartSpan("test.op", "test")
	err1 := span.End()
	err2 := span.End()
	if err1 != nil || err2 != nil {
		t.Error("End() should be no-op without global client")
	}
}

func TestSpan_SetAttributes(t *testing.T) {
	span := StartSpan("http.server", "GET /users")
	span.SetAttribute("http.method", "GET")
	span.SetAttributes(Fields{"http.url": "/users", "http.status_code": "200"})

	span.mu.Lock()
	if span.attributes["http.method"] != "GET" {
		t.Error("expected http.method")
	}
	if span.attributes["http.url"] != "/users" {
		t.Error("expected http.url")
	}
	span.mu.Unlock()
}

func TestStartSpanWithTraceID(t *testing.T) {
	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	span := StartSpanWithTraceID(traceID, "http.client", "GET /api")

	if span.TraceID() != traceID {
		t.Errorf("expected trace ID %s, got %s", traceID, span.TraceID())
	}
}

func TestSpan_Timing(t *testing.T) {
	span := StartSpan("test.op", "timed work")
	time.Sleep(10 * time.Millisecond)
	// Can't test End() send without a server, but we can verify it doesn't panic
	_ = span.End()
}

type errForTest string

func (e errForTest) Error() string { return string(e) }
