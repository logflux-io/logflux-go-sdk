package logflux

import (
	"sync"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/payload"
)

// Scope provides per-request context isolation. Attributes and breadcrumbs
// set on a scope are merged into every entry sent through it, without
// affecting other scopes or the global state.
//
// Usage:
//
//	logflux.WithScope(func(scope *logflux.Scope) {
//	    scope.SetAttribute("request_id", "abc-123")
//	    scope.SetAttribute("user_id", "usr_456")
//	    scope.AddBreadcrumb("http", "GET /api/users", nil)
//	    scope.Info("processing request")
//	    scope.CaptureError(err) // includes scope attrs + breadcrumbs
//	})
type Scope struct {
	mu          sync.RWMutex
	attributes  Fields
	breadcrumbs *payload.BreadcrumbRing
	traceID     string
	spanID      string
}

// newScope creates a scope with its own breadcrumb buffer.
func newScope() *Scope {
	return &Scope{
		attributes:  make(Fields),
		breadcrumbs: payload.NewBreadcrumbRing(100),
	}
}

// WithScope runs fn with an isolated scope. Scope attributes and breadcrumbs
// are merged into every entry sent through the scope.
func WithScope(fn func(scope *Scope)) {
	scope := newScope()
	fn(scope)
}

// --- Attribute setters ---

// SetAttribute sets a key-value pair that will be merged into every entry.
func (s *Scope) SetAttribute(key, value string) {
	s.mu.Lock()
	s.attributes[key] = value
	s.mu.Unlock()
}

// SetAttributes sets multiple attributes at once.
func (s *Scope) SetAttributes(attrs Fields) {
	s.mu.Lock()
	for k, v := range attrs {
		s.attributes[k] = v
	}
	s.mu.Unlock()
}

// SetUser is a convenience for setting user context.
func (s *Scope) SetUser(userID string) {
	s.mu.Lock()
	s.attributes["user.id"] = userID
	s.mu.Unlock()
}

// SetRequest is a convenience for setting request context.
func (s *Scope) SetRequest(method, path, requestID string) {
	s.mu.Lock()
	s.attributes["http.method"] = method
	s.attributes["http.path"] = path
	if requestID != "" {
		s.attributes["request_id"] = requestID
	}
	s.mu.Unlock()
}

// --- Trace context ---

// SetTraceContext sets the trace/span IDs for this scope.
func (s *Scope) SetTraceContext(traceID, spanID string) {
	s.traceID = traceID
	s.spanID = spanID
}

// --- Breadcrumbs ---

// AddBreadcrumb adds a breadcrumb to this scope's trail.
func (s *Scope) AddBreadcrumb(category, message string, data Fields) {
	s.breadcrumbs.Add(payload.Breadcrumb{
		Category: category,
		Message:  message,
		Data:     data,
	})
}

// --- Log methods ---

func (s *Scope) Debug(message string) error     { return s.Log(models.LogLevelDebug, message) }
func (s *Scope) Info(message string) error      { return s.Log(models.LogLevelInfo, message) }
func (s *Scope) Notice(message string) error    { return s.Log(models.LogLevelNotice, message) }
func (s *Scope) Warn(message string) error      { return s.Log(models.LogLevelWarning, message) }
func (s *Scope) Error(message string) error     { return s.Log(models.LogLevelError, message) }
func (s *Scope) Critical(message string) error  { return s.Log(models.LogLevelCritical, message) }

// Log sends a log entry with scope attributes merged in.
func (s *Scope) Log(level int, message string) error {
	c := getClient()
	if c == nil {
		return nil
	}
	p := payload.NewLog("", message, level)
	payload.ApplyContext(p)
	s.applyScope(p)

	if level <= models.LogLevelInfo {
		s.breadcrumbs.Add(payload.Breadcrumb{
			Category: "log",
			Message:  message,
			Level:    levelString(level),
		})
	}

	data, err := payload.Marshal(p)
	if err != nil {
		return err
	}
	return c.SendLogWithEntryType(string(data), level, models.EntryTypeLog)
}

// CaptureError captures an error with scope context + breadcrumbs.
func (s *Scope) CaptureError(err error) error {
	c := getClient()
	if c == nil || err == nil {
		return nil
	}
	p := payload.NewErrorPayload("", err)
	payload.ApplyContext(p)
	s.applyScope(p)
	p.WithBreadcrumbs(s.breadcrumbs)

	data, marshalErr := payload.Marshal(p)
	if marshalErr != nil {
		return marshalErr
	}
	return c.SendLogWithEntryType(string(data), models.LogLevelError, models.EntryTypeLog)
}

// Event sends an event with scope attributes.
func (s *Scope) Event(event string, attrs Fields) error {
	c := getClient()
	if c == nil {
		return nil
	}
	p := payload.NewEvent("", event)
	payload.ApplyContext(p)
	s.applyScope(p)
	if attrs != nil {
		for k, v := range attrs {
			if p.Attributes == nil {
				p.Attributes = make(Fields)
			}
			p.Attributes[k] = v
		}
	}

	s.breadcrumbs.Add(payload.Breadcrumb{
		Category: "event",
		Message:  event,
		Data:     attrs,
	})

	data, err := payload.Marshal(p)
	if err != nil {
		return err
	}
	return c.SendLogWithEntryType(string(data), models.LogLevelInfo, models.EntryTypeEvent)
}

// applyScope merges scope attributes into a payload's attributes.
func (s *Scope) applyScope(p interface {
	SetAttributes(Fields)
	GetAttributes() Fields
}) {
	s.mu.RLock()
	attrsCopy := make(Fields, len(s.attributes))
	for k, v := range s.attributes {
		attrsCopy[k] = v
	}
	s.mu.RUnlock()

	existing := p.GetAttributes()
	if existing == nil {
		existing = make(Fields, len(attrsCopy))
	}
	// Scope attributes are defaults — don't overwrite explicit ones
	for k, v := range attrsCopy {
		if _, exists := existing[k]; !exists {
			existing[k] = v
		}
	}
	p.SetAttributes(existing)
}

