package logflux

import (
	"fmt"
	"sync"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/payload"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/retry"
)

var (
	globalMu     sync.RWMutex
	globalClient *client.ResilientClient
	breadcrumbs  *payload.BreadcrumbRing
	hooks        sendHooks
	sampler      *payload.Sampler
)

// getClient returns the global client under a read lock.
func getClient() *client.ResilientClient {
	globalMu.RLock()
	c := globalClient
	globalMu.RUnlock()
	return c
}

// getSampler returns the global sampler under a read lock.
func getSampler() *payload.Sampler {
	globalMu.RLock()
	s := sampler
	globalMu.RUnlock()
	return s
}

// getBreadcrumbs returns the global breadcrumb ring under a read lock.
func getBreadcrumbs() *payload.BreadcrumbRing {
	globalMu.RLock()
	b := breadcrumbs
	globalMu.RUnlock()
	return b
}

// getHooks returns the global hooks under a read lock.
func getHooks() sendHooks {
	globalMu.RLock()
	h := hooks
	globalMu.RUnlock()
	return h
}

// Typed BeforeSend callbacks. Return nil to drop the entry.
type sendHooks struct {
	Log       func(*payload.Log) *payload.Log
	Error     func(*payload.ErrorPayload) *payload.ErrorPayload
	Metric    func(*payload.Metric) *payload.Metric
	Event     func(*payload.Event) *payload.Event
	Audit     func(*payload.Audit) *payload.Audit
	Trace     func(*payload.Trace) *payload.Trace
	Telemetry func(*payload.Telemetry) *payload.Telemetry
}

// Fields is a convenience alias for attributes.
type Fields = map[string]string

// Options configures the LogFlux SDK.
type Options struct {
	APIKey            string
	Node              string
	Source            string // Auto-attached to all payloads
	Environment       string // Auto-attached to all payload meta
	Release           string // Auto-attached to all payload meta
	LogGroup          string
	CustomEndpointURL string

	QueueSize     int
	FlushInterval time.Duration
	BatchSize     int
	WorkerCount   int

	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64

	HTTPTimeout       time.Duration
	Failsafe          bool
	EnableCompression bool
	Debug             bool

	MaxBreadcrumbs int     // Ring buffer size (default: 100)
	SampleRate     float64 // 0.0-1.0, probability of sending an entry (default: 1.0 = send all)

	// Global BeforeSend — runs on all entry types at the transport level.
	BeforeSend client.BeforeSendFunc

	// Per-type BeforeSend callbacks. Return nil to drop the entry.
	BeforeSendLog       func(*payload.Log) *payload.Log
	BeforeSendError     func(*payload.ErrorPayload) *payload.ErrorPayload
	BeforeSendMetric    func(*payload.Metric) *payload.Metric
	BeforeSendEvent     func(*payload.Event) *payload.Event
	BeforeSendAudit     func(*payload.Audit) *payload.Audit
	BeforeSendTrace     func(*payload.Trace) *payload.Trace
	BeforeSendTelemetry func(*payload.Telemetry) *payload.Telemetry
}

// Init initializes LogFlux with the given options.
func Init(opts Options) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	// Configure global payload context
	source := opts.Source
	if source == "" {
		source = opts.Node
	}
	payload.Configure(source, opts.Environment, opts.Release)

	// Initialize breadcrumb ring buffer
	maxCrumbs := opts.MaxBreadcrumbs
	if maxCrumbs <= 0 {
		maxCrumbs = 100
	}
	breadcrumbs = payload.NewBreadcrumbRing(maxCrumbs)

	// Initialize sampler. SampleRate <= 0 means "send all" (default).
	// Only values in (0.0, 1.0] are treated as an explicit sample rate.
	rate := opts.SampleRate
	if rate <= 0 {
		rate = 1.0
	}
	sampler = payload.NewSampler(rate)

	cfg := client.DefaultResilientClientConfig()
	cfg.APIKey = opts.APIKey
	cfg.Node = opts.Node
	cfg.Environment = opts.Environment
	cfg.LogGroup = opts.LogGroup
	cfg.CustomEndpointURL = opts.CustomEndpointURL
	cfg.BeforeSend = opts.BeforeSend

	if opts.QueueSize > 0 {
		cfg.QueueSize = opts.QueueSize
	}
	if opts.FlushInterval > 0 {
		cfg.FlushInterval = opts.FlushInterval
	}
	if opts.BatchSize > 0 {
		cfg.BatchSize = opts.BatchSize
	}
	if opts.WorkerCount > 0 {
		cfg.WorkerCount = opts.WorkerCount
	}
	if opts.HTTPTimeout > 0 {
		cfg.HTTPTimeout = opts.HTTPTimeout
	}
	if opts.MaxRetries > 0 {
		cfg.RetryConfig.MaxRetries = opts.MaxRetries
	}
	if opts.InitialDelay > 0 {
		cfg.RetryConfig.InitialDelay = opts.InitialDelay
	}
	if opts.MaxDelay > 0 {
		cfg.RetryConfig.MaxDelay = opts.MaxDelay
	}
	if opts.BackoffFactor > 0 {
		cfg.RetryConfig.BackoffFactor = opts.BackoffFactor
	}

	cfg.FailsafeMode = opts.Failsafe
	cfg.EnableCompression = opts.EnableCompression

	// Store typed hooks
	hooks = sendHooks{
		Log:       opts.BeforeSendLog,
		Error:     opts.BeforeSendError,
		Metric:    opts.BeforeSendMetric,
		Event:     opts.BeforeSendEvent,
		Audit:     opts.BeforeSendAudit,
		Trace:     opts.BeforeSendTrace,
		Telemetry: opts.BeforeSendTelemetry,
	}

	var err error
	globalClient, err = client.NewResilientClientWithHandshake(cfg)
	return err
}

// InitSimple initializes LogFlux with just an API key and node name.
func InitSimple(apiKey, node string) error {
	return Init(Options{
		APIKey:            apiKey,
		Node:              node,
		Source:            node,
		Failsafe:          true,
		EnableCompression: true,
	})
}

// InitFromEnv initializes LogFlux from environment variables.
func InitFromEnv(node string) error {
	globalMu.Lock()
	defer globalMu.Unlock()
	breadcrumbs = payload.NewBreadcrumbRing(100)
	sampler = payload.NewSampler(1.0)
	var err error
	globalClient, err = client.NewResilientClientFromEnvWithHandshake(node)
	return err
}

// InitWithConfig initializes LogFlux with a ResilientClientConfig directly.
func InitWithConfig(config client.ResilientClientConfig) error {
	globalMu.Lock()
	defer globalMu.Unlock()
	payload.Configure(config.Node, config.Environment, "")
	breadcrumbs = payload.NewBreadcrumbRing(100)
	sampler = payload.NewSampler(1.0)
	var err error
	globalClient, err = client.NewResilientClientWithHandshake(config)
	return err
}

// --- Breadcrumbs ---

// AddBreadcrumb adds a breadcrumb to the trail.
func AddBreadcrumb(category, message string, data Fields) {
	b := getBreadcrumbs()
	if b == nil {
		return
	}
	b.Add(payload.Breadcrumb{
		Category: category,
		Message:  message,
		Data:     data,
	})
}

// AddBreadcrumbWithLevel adds a breadcrumb with a severity level.
func AddBreadcrumbWithLevel(category, message, level string, data Fields) {
	b := getBreadcrumbs()
	if b == nil {
		return
	}
	b.Add(payload.Breadcrumb{
		Category: category,
		Message:  message,
		Level:    level,
		Data:     data,
	})
}

// ClearBreadcrumbs removes all breadcrumbs.
func ClearBreadcrumbs() {
	b := getBreadcrumbs()
	if b != nil {
		b.Clear()
	}
}

// --- Log convenience (type 1) ---

func Debug(message string) error     { return Log(models.LogLevelDebug, message, nil) }
func Info(message string) error      { return Log(models.LogLevelInfo, message, nil) }
func Notice(message string) error    { return Log(models.LogLevelNotice, message, nil) }
func Warn(message string) error      { return Log(models.LogLevelWarning, message, nil) }
func Warning(message string) error   { return Log(models.LogLevelWarning, message, nil) }
func Critical(message string) error  { return Log(models.LogLevelCritical, message, nil) }
func Alert(message string) error     { return Log(models.LogLevelAlert, message, nil) }
func Emergency(message string) error { return Log(models.LogLevelEmergency, message, nil) }
func Fatal(message string) error     { return Log(models.LogLevelCritical, message, nil) }

// Debugf sends a formatted debug log.
func Debugf(format string, args ...interface{}) error {
	return Log(models.LogLevelDebug, fmt.Sprintf(format, args...), nil)
}

// Infof sends a formatted info log.
func Infof(format string, args ...interface{}) error {
	return Log(models.LogLevelInfo, fmt.Sprintf(format, args...), nil)
}

// Warnf sends a formatted warning log.
func Warnf(format string, args ...interface{}) error {
	return Log(models.LogLevelWarning, fmt.Sprintf(format, args...), nil)
}

// Errorf sends a formatted error log.
func Errorf(format string, args ...interface{}) error {
	return Log(models.LogLevelError, fmt.Sprintf(format, args...), nil)
}

// Log sends a log entry (type 1) with the given level and attributes.
func Log(level int, message string, attrs Fields) error {
	c := getClient()
	if c == nil {
		return nil
	}
	if s := getSampler(); s != nil && !s.ShouldSample() {
		return nil
	}
	p := payload.NewLog("", message, level)
	payload.ApplyContext(p)
	if attrs != nil {
		p.SetAttributes(attrs)
	}
	h := getHooks()
	if h.Log != nil {
		p = h.Log(p)
		if p == nil {
			return nil
		}
	}

	if b := getBreadcrumbs(); b != nil && level <= models.LogLevelInfo {
		addLogBreadcrumb(b, level, message)
	}

	data, err := payload.Marshal(p)
	if err != nil {
		return err
	}
	return c.SendLogWithEntryType(string(data), level, models.EntryTypeLog)
}

// Error sends an error-level log message.
func Error(message string) error {
	return Log(models.LogLevelError, message, nil)
}

// --- CaptureError (type 1 with stack trace + breadcrumbs) ---

// CaptureError captures a Go error with automatic stack trace and breadcrumbs.
func CaptureError(err error) error {
	return CaptureErrorWithAttrs(err, nil)
}

// CaptureErrorWithAttrs captures a Go error with stack trace, breadcrumbs, and attributes.
func CaptureErrorWithAttrs(err error, attrs Fields) error {
	c := getClient()
	if c == nil || err == nil {
		return nil
	}
	if s := getSampler(); s != nil && !s.ShouldSample() {
		return nil
	}

	p := payload.NewErrorPayload("", err)
	payload.ApplyContext(p)
	if attrs != nil {
		p.SetAttributes(attrs)
	}
	if b := getBreadcrumbs(); b != nil {
		p.WithBreadcrumbs(b)
	}
	h := getHooks()
	if h.Error != nil {
		p = h.Error(p)
		if p == nil {
			return nil
		}
	}

	data, marshalErr := payload.Marshal(p)
	if marshalErr != nil {
		return marshalErr
	}
	return c.SendLogWithEntryType(string(data), models.LogLevelError, models.EntryTypeLog)
}

// CaptureErrorWithMessage captures with a custom message (error goes into attributes).
func CaptureErrorWithMessage(err error, message string, attrs Fields) error {
	c := getClient()
	if c == nil || err == nil {
		return nil
	}
	if s := getSampler(); s != nil && !s.ShouldSample() {
		return nil
	}

	p := payload.NewErrorPayloadWithMessage("", err, message)
	payload.ApplyContext(p)
	if attrs != nil {
		for k, v := range attrs {
			if p.Attributes == nil {
				p.Attributes = make(Fields)
			}
			p.Attributes[k] = v
		}
	}
	if b := getBreadcrumbs(); b != nil {
		p.WithBreadcrumbs(b)
	}
	h := getHooks()
	if h.Error != nil {
		p = h.Error(p)
		if p == nil {
			return nil
		}
	}

	data, marshalErr := payload.Marshal(p)
	if marshalErr != nil {
		return marshalErr
	}
	return c.SendLogWithEntryType(string(data), models.LogLevelError, models.EntryTypeLog)
}

// --- Metric convenience (type 2) ---

// Metric sends a metric entry (type 2).
func Metric(name string, value float64, kind string, attrs Fields) error {
	c := getClient()
	if c == nil {
		return nil
	}
	if s := getSampler(); s != nil && !s.ShouldSample() {
		return nil
	}
	p := payload.NewGauge("", name, value, "")
	p.Kind = kind
	payload.ApplyContext(p)
	if attrs != nil {
		p.SetAttributes(attrs)
	}
	h := getHooks()
	if h.Metric != nil {
		p = h.Metric(p)
		if p == nil {
			return nil
		}
	}
	data, err := payload.Marshal(p)
	if err != nil {
		return err
	}
	return c.SendLogWithEntryType(string(data), models.LogLevelInfo, models.EntryTypeMetric)
}

// Counter sends a counter metric (type 2, kind=counter).
func Counter(name string, value float64, attrs Fields) error {
	return Metric(name, value, "counter", attrs)
}

// Gauge sends a gauge metric (type 2, kind=gauge).
func Gauge(name string, value float64, attrs Fields) error {
	return Metric(name, value, "gauge", attrs)
}

// --- Event convenience (type 4) ---

// Event sends an event entry (type 4).
func Event(event string, attrs Fields) error {
	c := getClient()
	if c == nil {
		return nil
	}
	if s := getSampler(); s != nil && !s.ShouldSample() {
		return nil
	}
	p := payload.NewEvent("", event)
	payload.ApplyContext(p)
	if attrs != nil {
		p.SetAttributes(attrs)
	}
	h := getHooks()
	if h.Event != nil {
		p = h.Event(p)
		if p == nil {
			return nil
		}
	}

	if b := getBreadcrumbs(); b != nil {
		b.Add(payload.Breadcrumb{
			Category: "event",
			Message:  event,
			Data:     attrs,
		})
	}

	data, err := payload.Marshal(p)
	if err != nil {
		return err
	}
	return c.SendLogWithEntryType(string(data), models.LogLevelInfo, models.EntryTypeEvent)
}

// --- Audit convenience (type 5) ---

// Audit sends an audit entry (type 5, Object Lock).
func Audit(action, actor, resource, resourceID string, attrs Fields) error {
	c := getClient()
	if c == nil {
		return nil
	}
	// Audit entries are never sampled — compliance requirement
	p := payload.NewAudit("", action, actor, resource, resourceID)
	payload.ApplyContext(p)
	if attrs != nil {
		p.SetAttributes(attrs)
	}
	h := getHooks()
	if h.Audit != nil {
		p = h.Audit(p)
		if p == nil {
			return nil
		}
	}
	data, err := payload.Marshal(p)
	if err != nil {
		return err
	}
	return c.SendLogWithEntryType(string(data), models.LogLevelNotice, models.EntryTypeAudit)
}

// --- Lifecycle ---

func Close() error {
	c := getClient()
	if c == nil {
		return nil
	}
	return c.Close()
}

func Flush(timeout time.Duration) error {
	c := getClient()
	if c == nil {
		return nil
	}
	return c.Flush(timeout)
}

func Stats() client.ClientStats {
	c := getClient()
	if c == nil {
		return client.ClientStats{}
	}
	return c.GetStats()
}

func GetStats() client.ClientStats { return Stats() }

// --- Helpers ---

func addLogBreadcrumb(b *payload.BreadcrumbRing, level int, message string) {
	b.Add(payload.Breadcrumb{
		Category: "log",
		Message:  message,
		Level:    levelString(level),
	})
}

// levelString converts a numeric log level to a human-readable string.
// Shared by logflux.go and scope.go within this package.
func levelString(level int) string {
	switch {
	case level <= models.LogLevelCritical:
		return "error"
	case level == models.LogLevelError:
		return "error"
	case level == models.LogLevelWarning:
		return "warning"
	default:
		return "info"
	}
}

// --- Constants ---

const (
	LogLevelEmergency = models.LogLevelEmergency
	LogLevelAlert     = models.LogLevelAlert
	LogLevelCritical  = models.LogLevelCritical
	LogLevelError     = models.LogLevelError
	LogLevelWarning   = models.LogLevelWarning
	LogLevelNotice    = models.LogLevelNotice
	LogLevelInfo      = models.LogLevelInfo
	LogLevelDebug     = models.LogLevelDebug
)

const (
	EntryTypeLog              = models.EntryTypeLog
	EntryTypeMetric           = models.EntryTypeMetric
	EntryTypeTrace            = models.EntryTypeTrace
	EntryTypeEvent            = models.EntryTypeEvent
	EntryTypeAudit            = models.EntryTypeAudit
	EntryTypeTelemetry        = models.EntryTypeTelemetry
	EntryTypeTelemetryManaged = models.EntryTypeTelemetryManaged
)

var (
	DefaultRetryConfig   = retry.DefaultConfig
	ResilientRetryConfig = retry.ResilientConfig
)
