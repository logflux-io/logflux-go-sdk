package payload

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestNewErrorPayload(t *testing.T) {
	err := errors.New("database connection failed")
	p := NewErrorPayload("api-server", err)

	if p.Message != "database connection failed" {
		t.Errorf("expected message, got %q", p.Message)
	}
	if p.ErrorType != "errors.errorString" {
		t.Errorf("expected errors.errorString, got %q", p.ErrorType)
	}
	if p.Level != 4 {
		t.Errorf("expected level 4 (error), got %d", p.Level)
	}
	if len(p.StackTrace) == 0 {
		t.Error("expected stack trace frames")
	}
	// First frame should be this test function
	if p.StackTrace[0].Function == "" {
		t.Error("expected function name in first frame")
	}
}

func TestNewErrorPayload_CustomType(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", errors.New("inner"))
	p := NewErrorPayload("svc", err)

	if p.ErrorType != "fmt.wrapError" {
		t.Errorf("expected fmt.wrapError, got %q", p.ErrorType)
	}
}

func TestErrorPayload_WithBreadcrumbs(t *testing.T) {
	ring := NewBreadcrumbRing(10)
	ring.Add(Breadcrumb{Message: "user clicked login"})
	ring.Add(Breadcrumb{Message: "fetching token"})

	err := errors.New("auth failed")
	p := NewErrorPayload("auth-svc", err)
	p.WithBreadcrumbs(ring)

	if len(p.Breadcrumbs) != 2 {
		t.Fatalf("expected 2 breadcrumbs, got %d", len(p.Breadcrumbs))
	}
	if p.Breadcrumbs[0].Message != "user clicked login" {
		t.Error("breadcrumbs not in order")
	}
}

func TestErrorPayload_Serialization(t *testing.T) {
	err := errors.New("test error")
	p := NewErrorPayload("svc", err)
	p.SetAttributes(map[string]string{"request_id": "abc"})

	ring := NewBreadcrumbRing(10)
	ring.Add(Breadcrumb{Category: "http", Message: "GET /api"})
	p.WithBreadcrumbs(ring)

	data, marshalErr := Marshal(p)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	if m["v"] != "2.0" {
		t.Errorf("expected v=2.0, got %v", m["v"])
	}
	if m["type"] != "log" {
		t.Errorf("expected type=log, got %v", m["type"])
	}
	if m["error_type"] != "errors.errorString" {
		t.Errorf("expected error_type, got %v", m["error_type"])
	}
	if m["stack_trace"] == nil {
		t.Error("expected stack_trace in JSON")
	}
	if m["breadcrumbs"] == nil {
		t.Error("expected breadcrumbs in JSON")
	}
	frames := m["stack_trace"].([]interface{})
	if len(frames) == 0 {
		t.Error("expected non-empty stack_trace")
	}
	crumbs := m["breadcrumbs"].([]interface{})
	if len(crumbs) != 1 {
		t.Errorf("expected 1 breadcrumb, got %d", len(crumbs))
	}
}

func TestErrorPayloadWithMessage(t *testing.T) {
	err := errors.New("connection refused")
	p := NewErrorPayloadWithMessage("svc", err, "Failed to connect to database")

	if p.Message != "Failed to connect to database" {
		t.Errorf("expected custom message, got %q", p.Message)
	}
	if p.Attributes["error"] != "connection refused" {
		t.Errorf("expected original error in attributes, got %q", p.Attributes["error"])
	}
}

func TestErrorChain_SingleError(t *testing.T) {
	err := errors.New("simple error")
	p := NewErrorPayload("svc", err)

	if p.ErrorChain != nil {
		t.Error("single error should not have a chain (redundant with top-level fields)")
	}
}

func TestErrorChain_WrappedError(t *testing.T) {
	inner := errors.New("connection refused")
	middle := fmt.Errorf("db query failed: %w", inner)
	outer := fmt.Errorf("handler error: %w", middle)

	p := NewErrorPayload("svc", outer)

	if len(p.ErrorChain) != 3 {
		t.Fatalf("expected 3-item chain, got %d", len(p.ErrorChain))
	}

	// Chain should be outermost → innermost
	if p.ErrorChain[0].Message != "handler error: db query failed: connection refused" {
		t.Errorf("chain[0] wrong: %s", p.ErrorChain[0].Message)
	}
	if p.ErrorChain[1].Message != "db query failed: connection refused" {
		t.Errorf("chain[1] wrong: %s", p.ErrorChain[1].Message)
	}
	if p.ErrorChain[2].Message != "connection refused" {
		t.Errorf("chain[2] wrong: %s", p.ErrorChain[2].Message)
	}

	// Types should be extracted
	if p.ErrorChain[0].Type != "fmt.wrapError" {
		t.Errorf("chain[0] type wrong: %s", p.ErrorChain[0].Type)
	}
	if p.ErrorChain[2].Type != "errors.errorString" {
		t.Errorf("chain[2] type wrong: %s", p.ErrorChain[2].Type)
	}
}

func TestErrorChain_Serialization(t *testing.T) {
	inner := errors.New("root cause")
	outer := fmt.Errorf("wrapper: %w", inner)
	p := NewErrorPayload("svc", outer)

	data, err := Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	chain, ok := m["error_chain"].([]interface{})
	if !ok {
		t.Fatal("expected error_chain array in JSON")
	}
	if len(chain) != 2 {
		t.Errorf("expected 2-item chain in JSON, got %d", len(chain))
	}
}
