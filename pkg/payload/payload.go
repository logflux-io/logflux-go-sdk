// Package payload provides v2 payload schema types for all 7 entry types.
// These types serialize to flat JSON with no string-in-string encoding.
package payload

import (
	"encoding/json"
	"time"
)

// Common fields present in every v2 payload.
type common struct {
	V          string            `json:"v"`
	Type       string            `json:"type"`
	Source     string            `json:"source"`
	Level      int               `json:"level,omitempty"`
	Ts         string            `json:"ts,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Meta       map[string]string `json:"meta,omitempty"`
}

func newCommon(typeName, source string, level int) common {
	return common{
		V:      "2.0",
		Type:   typeName,
		Source: source,
		Level:  level,
		Ts:     time.Now().UTC().Format(time.RFC3339Nano),
	}
}

// --- Type 1: Log ---

// Log represents a v2 log payload.
type Log struct {
	common
	Message string `json:"message"`
	Logger  string `json:"logger,omitempty"`
}

// NewLog creates a log payload.
func NewLog(source, message string, level int) *Log {
	return &Log{
		common:  newCommon("log", source, level),
		Message: message,
	}
}

// --- Type 2: Metric ---

// Metric represents a v2 metric payload.
type Metric struct {
	common
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
	Kind  string  `json:"kind,omitempty"`
}

// NewCounter creates a counter metric.
func NewCounter(source, name string, value float64) *Metric {
	return &Metric{
		common: newCommon("metric", source, 7),
		Name:   name,
		Value:  value,
		Kind:   "counter",
	}
}

// NewGauge creates a gauge metric.
func NewGauge(source, name string, value float64, unit string) *Metric {
	return &Metric{
		common: newCommon("metric", source, 7),
		Name:   name,
		Value:  value,
		Unit:   unit,
		Kind:   "gauge",
	}
}

// NewDistribution creates a distribution metric.
func NewDistribution(source, name string, value float64, unit string) *Metric {
	return &Metric{
		common: newCommon("metric", source, 7),
		Name:   name,
		Value:  value,
		Unit:   unit,
		Kind:   "distribution",
	}
}

// --- Type 3: Trace ---

// Trace represents a v2 trace span payload.
type Trace struct {
	common
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
	Operation    string `json:"operation"`
	Name         string `json:"name"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	Status       string `json:"status,omitempty"`
}

// NewTrace creates a trace span payload.
func NewTrace(source, traceID, spanID, operation, name string, start, end time.Time) *Trace {
	return &Trace{
		common:     newCommon("trace", source, 7),
		TraceID:    traceID,
		SpanID:     spanID,
		Operation:  operation,
		Name:       name,
		StartTime:  start.UTC().Format(time.RFC3339Nano),
		EndTime:    end.UTC().Format(time.RFC3339Nano),
		DurationMs: end.Sub(start).Milliseconds(),
		Status:     "ok",
	}
}

// --- Type 4: Event ---

// Event represents a v2 event payload.
type Event struct {
	common
	EventName string `json:"event"`
}

// NewEvent creates an event payload.
func NewEvent(source, event string) *Event {
	return &Event{
		common:    newCommon("event", source, 7),
		EventName: event,
	}
}

// --- Type 5: Audit ---

// Audit represents a v2 audit payload.
type Audit struct {
	common
	Action     string `json:"action"`
	Actor      string `json:"actor"`
	ActorType  string `json:"actor_type,omitempty"`
	Resource   string `json:"resource"`
	ResourceID string `json:"resource_id"`
	Outcome    string `json:"outcome,omitempty"`
}

// NewAudit creates an audit payload.
func NewAudit(source, action, actor, resource, resourceID string) *Audit {
	return &Audit{
		common:     newCommon("audit", source, 6), // notice level
		Action:     action,
		Actor:      actor,
		Resource:   resource,
		ResourceID: resourceID,
		Outcome:    "success",
	}
}

// --- Type 6/7: Telemetry ---

// Reading is a single sensor measurement.
type Reading struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

// Telemetry represents a v2 telemetry payload.
type Telemetry struct {
	common
	DeviceID string    `json:"device_id"`
	Readings []Reading `json:"readings"`
}

// NewTelemetry creates a telemetry payload.
func NewTelemetry(source, deviceID string, readings []Reading) *Telemetry {
	return &Telemetry{
		common:   newCommon("telemetry", source, 7),
		DeviceID: deviceID,
		Readings: readings,
	}
}

// --- Fluent setters (all types via embedded common) ---

func (c *common) SetAttributes(attrs map[string]string)    { c.Attributes = attrs }
func (c *common) SetMeta(meta map[string]string)           { c.Meta = meta }
func (c *common) SetLevel(level int)                       { c.Level = level }
func (c *common) SetSource(source string)                  { c.Source = source }
func (c *common) SetTimestamp(ts time.Time)                { c.Ts = ts.UTC().Format(time.RFC3339Nano) }

// Exported getter for scope integration.
func (c *common) GetAttributes() map[string]string { return c.Attributes }

// --- Serialization ---

// Marshal serializes any payload type to JSON bytes.
func Marshal(p interface{}) ([]byte, error) {
	return json.Marshal(p)
}
