package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLogEntry_Fields(t *testing.T) {
	ts := time.Unix(1723766400, 0).UTC()
	labels := map[string]string{"env": "test"}
	entry := LogEntry{
		Message:      "hello world",
		Timestamp:    ts,
		Level:        LogLevelInfo,
		EntryType:    EntryTypeLog,
		PayloadType:  PayloadTypeAES256GCMGzipJSON,
		Node:         "node-a",
		Labels:       labels,
		SearchTokens: []string{"hello", "world"},
	}

	if entry.Message != "hello world" {
		t.Fatalf("message mismatch: %s", entry.Message)
	}
	if entry.Level != LogLevelInfo {
		t.Fatalf("level mismatch: %d", entry.Level)
	}
	if entry.EntryType != EntryTypeLog {
		t.Fatalf("entry type mismatch: %d", entry.EntryType)
	}
	if entry.Node != "node-a" {
		t.Fatalf("node mismatch: %s", entry.Node)
	}
	if len(entry.Labels) != 1 || entry.Labels["env"] != "test" {
		t.Fatalf("labels mismatch: %v", entry.Labels)
	}
	if len(entry.SearchTokens) != 2 {
		t.Fatalf("search tokens mismatch: %v", entry.SearchTokens)
	}
}

func TestIngestResponse_JSONShape(t *testing.T) {
	raw := `{"status":"ok","message":"accepted","request_id":"abc","data":{"success":true,"timestamp":"2024-01-01T00:00:00Z","entry_type":1,"payload_type":1}}`
	var resp IngestResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status mismatch: %s", resp.Status)
	}
	if !resp.Data.Success {
		t.Fatalf("data.success should be true")
	}
	if resp.Data.EntryType != 1 {
		t.Fatalf("data.entry_type mismatch: %d", resp.Data.EntryType)
	}
}

func TestBatchIngestResponse_JSONShape(t *testing.T) {
	raw := `{"status":"ok","message":"batch accepted","request_id":"def","data":{"success":true,"total":3,"succeeded":2,"failed":1,"failures":[{"index":1,"error":"bad entry"}]}}`
	var resp BatchIngestResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.Total != 3 || resp.Data.Succeeded != 2 || resp.Data.Failed != 1 {
		t.Fatalf("batch counts mismatch: %+v", resp.Data)
	}
	if len(resp.Data.Failures) != 1 || resp.Data.Failures[0].Index != 1 {
		t.Fatalf("failures mismatch: %+v", resp.Data.Failures)
	}
}

func TestErrorResponse_JSONShape(t *testing.T) {
	raw := `{"status":"error","error":{"code":"rate_limited","message":"too many requests","details":"slow down","retry_after":60},"request_id":"xyz"}`
	var resp ErrorResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Code != "rate_limited" {
		t.Fatalf("error code mismatch: %s", resp.Error.Code)
	}
	if resp.Error.RetryAfter != 60 {
		t.Fatalf("retry_after mismatch: %d", resp.Error.RetryAfter)
	}
}

func TestEntryTypeCategory(t *testing.T) {
	cases := []struct {
		entryType int
		want      string
	}{
		{EntryTypeLog, CategoryEvents},
		{EntryTypeMetric, CategoryEvents},
		{EntryTypeEvent, CategoryEvents},
		{EntryTypeTrace, CategoryTraces},
		{EntryTypeTelemetry, CategoryTraces},
		{EntryTypeTelemetryManaged, CategoryTraces},
		{EntryTypeAudit, CategoryAudit},
		{99, CategoryEvents}, // unknown defaults to events
	}
	for _, c := range cases {
		if got := EntryTypeCategory(c.entryType); got != c.want {
			t.Fatalf("EntryTypeCategory(%d) = %s, want %s", c.entryType, got, c.want)
		}
	}
}

func TestEntryTypeRequiresEncryption(t *testing.T) {
	for _, et := range []int{1, 2, 3, 4, 5, 6} {
		if !EntryTypeRequiresEncryption(et) {
			t.Fatalf("expected type %d to require encryption", et)
		}
	}
	if EntryTypeRequiresEncryption(EntryTypeTelemetryManaged) {
		t.Fatalf("type 7 should not require encryption")
	}
}

func TestDefaultPayloadType(t *testing.T) {
	if DefaultPayloadType(EntryTypeLog) != PayloadTypeAES256GCMGzipJSON {
		t.Fatalf("default for log should be AES256GCMGzipJSON")
	}
	if DefaultPayloadType(EntryTypeTelemetryManaged) != PayloadTypeGzipJSON {
		t.Fatalf("default for TelemetryManaged should be GzipJSON")
	}
}
