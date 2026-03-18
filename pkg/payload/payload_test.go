package payload

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLog_Serialization(t *testing.T) {
	p := NewLog("api-server", "connection timeout", 4)
	p.SetAttributes(map[string]string{"request_id": "abc"})
	p.Logger = "db.pool"

	data, err := Marshal(p)
	if err != nil {
		t.Fatal(err)
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
	if m["message"] != "connection timeout" {
		t.Errorf("expected message, got %v", m["message"])
	}
	if m["logger"] != "db.pool" {
		t.Errorf("expected logger=db.pool, got %v", m["logger"])
	}
	attrs := m["attributes"].(map[string]interface{})
	if attrs["request_id"] != "abc" {
		t.Errorf("expected request_id=abc, got %v", attrs["request_id"])
	}
}

func TestMetric_Counter(t *testing.T) {
	p := NewCounter("api", "requests.total", 42)
	data, _ := Marshal(p)

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["type"] != "metric" {
		t.Errorf("expected type=metric, got %v", m["type"])
	}
	if m["name"] != "requests.total" {
		t.Errorf("expected name, got %v", m["name"])
	}
	if m["value"] != 42.0 {
		t.Errorf("expected value=42, got %v", m["value"])
	}
	if m["kind"] != "counter" {
		t.Errorf("expected kind=counter, got %v", m["kind"])
	}
}

func TestTrace_Serialization(t *testing.T) {
	start := time.Date(2026, 3, 15, 14, 30, 45, 0, time.UTC)
	end := start.Add(143 * time.Millisecond)
	p := NewTrace("api", "aaa", "bbb", "http.server", "GET /users", start, end)
	p.ParentSpanID = "ccc"

	data, _ := Marshal(p)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["type"] != "trace" {
		t.Errorf("expected type=trace, got %v", m["type"])
	}
	if m["trace_id"] != "aaa" {
		t.Errorf("expected trace_id=aaa, got %v", m["trace_id"])
	}
	if m["duration_ms"] != 143.0 {
		t.Errorf("expected duration_ms=143, got %v", m["duration_ms"])
	}
	if m["parent_span_id"] != "ccc" {
		t.Errorf("expected parent_span_id=ccc, got %v", m["parent_span_id"])
	}
}

func TestEvent_Serialization(t *testing.T) {
	p := NewEvent("auth", "user.signup")
	p.SetAttributes(map[string]string{"plan": "starter"})

	data, _ := Marshal(p)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["type"] != "event" {
		t.Errorf("expected type=event, got %v", m["type"])
	}
	if m["event"] != "user.signup" {
		t.Errorf("expected event=user.signup, got %v", m["event"])
	}
}

func TestAudit_Serialization(t *testing.T) {
	p := NewAudit("billing", "record.deleted", "usr_456", "invoice", "inv_789")
	p.SetAttributes(map[string]string{"reason": "customer_request"})

	data, _ := Marshal(p)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["type"] != "audit" {
		t.Errorf("expected type=audit, got %v", m["type"])
	}
	if m["action"] != "record.deleted" {
		t.Errorf("expected action, got %v", m["action"])
	}
	if m["actor"] != "usr_456" {
		t.Errorf("expected actor, got %v", m["actor"])
	}
	if m["outcome"] != "success" {
		t.Errorf("expected outcome=success, got %v", m["outcome"])
	}
}

func TestTelemetry_Serialization(t *testing.T) {
	p := NewTelemetry("gateway", "dev_001", []Reading{
		{Name: "cpu_temp", Value: 72.5, Unit: "celsius"},
		{Name: "memory", Value: 85.2, Unit: "percent"},
	})

	data, _ := Marshal(p)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["type"] != "telemetry" {
		t.Errorf("expected type=telemetry, got %v", m["type"])
	}
	if m["device_id"] != "dev_001" {
		t.Errorf("expected device_id, got %v", m["device_id"])
	}
	readings := m["readings"].([]interface{})
	if len(readings) != 2 {
		t.Errorf("expected 2 readings, got %d", len(readings))
	}
}

func TestNoPayloadField(t *testing.T) {
	// v2 payloads must NOT have a "payload" field
	p := NewLog("svc", "hello", 7)
	data, _ := Marshal(p)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if _, exists := m["payload"]; exists {
		t.Error("v2 payload must not contain a 'payload' field")
	}
}
