package models

import "time"

// Log level constants (syslog severity 1-8)
const (
	LogLevelEmergency = 1
	LogLevelAlert     = 2
	LogLevelCritical  = 3
	LogLevelError     = 4
	LogLevelWarning   = 5
	LogLevelNotice    = 6
	LogLevelInfo      = 7
	LogLevelDebug     = 8
)

// Entry type constants
const (
	EntryTypeLog              = 1
	EntryTypeMetric           = 2
	EntryTypeTrace            = 3
	EntryTypeEvent            = 4
	EntryTypeAudit            = 5
	EntryTypeTelemetry        = 6
	EntryTypeTelemetryManaged = 7
)

// Payload type constants
const (
	PayloadTypeAES256GCMGzipJSON = 1 // AES-256-GCM + gzip (default for types 1-6)
	PayloadTypeGzipJSON          = 3 // gzip only (default for type 7)
)

// Pricing categories for quota tracking
const (
	CategoryEvents = "events" // Types 1, 2, 4
	CategoryTraces = "traces" // Types 3, 6, 7
	CategoryAudit  = "audit"  // Type 5
)

// EntryTypeCategory maps an entry type to its pricing category.
func EntryTypeCategory(entryType int) string {
	switch entryType {
	case EntryTypeLog, EntryTypeMetric, EntryTypeEvent:
		return CategoryEvents
	case EntryTypeTrace, EntryTypeTelemetry, EntryTypeTelemetryManaged:
		return CategoryTraces
	case EntryTypeAudit:
		return CategoryAudit
	default:
		return CategoryEvents
	}
}

// EntryTypeRequiresEncryption returns true if the entry type needs E2E encryption.
func EntryTypeRequiresEncryption(entryType int) bool {
	return entryType >= 1 && entryType <= 6
}

// DefaultPayloadType returns the default payload type for an entry type.
func DefaultPayloadType(entryType int) int {
	if entryType == EntryTypeTelemetryManaged {
		return PayloadTypeGzipJSON
	}
	return PayloadTypeAES256GCMGzipJSON
}

// LogEntry represents an entry ready for transmission.
type LogEntry struct {
	Message      string
	Timestamp    time.Time
	Level        int
	EntryType    int
	PayloadType  int
	Node         string
	Labels       map[string]string
	SearchTokens []string
}

// IngestResponse is the server response for a single entry.
type IngestResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Data      struct {
		Success     bool   `json:"success"`
		Timestamp   string `json:"timestamp"`
		EntryType   int    `json:"entry_type"`
		PayloadType int    `json:"payload_type"`
	} `json:"data"`
}

// BatchIngestResponse is the server response for a batch.
type BatchIngestResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Data      struct {
		Success   bool             `json:"success"`
		Total     int              `json:"total"`
		Succeeded int              `json:"succeeded"`
		Failed    int              `json:"failed"`
		Failures  []BatchFailure   `json:"failures"`
	} `json:"data"`
}

// BatchFailure describes a single failed entry in a batch.
type BatchFailure struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

// ErrorResponse is the server error envelope.
type ErrorResponse struct {
	Status string `json:"status"`
	Error  struct {
		Code       string `json:"code"`
		Message    string `json:"message"`
		Details    string `json:"details"`
		RetryAfter int    `json:"retry_after"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

// HandshakeLimits contains server-enforced limits from the handshake response.
type HandshakeLimits struct {
	MaxBatchSize   int `json:"max_batch_size"`
	MaxPayloadSize int `json:"max_payload_size"`
	MaxRequestSize int `json:"max_request_size"`
}
