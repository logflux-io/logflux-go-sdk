package types

import (
	"encoding/json"
	"time"
)

// LogEntry represents a log entry to be sent to the agent
// Matches the API specification for logflux-agent-api-v1.yaml
type LogEntry struct {
	Metadata    map[string]string `json:"metadata,omitempty"`
	Version     string            `json:"version,omitempty"`
	Payload     string            `json:"payload"`
	Source      string            `json:"source"`
	Timestamp   string            `json:"timestamp,omitempty"`
	PayloadType string            `json:"payloadType,omitempty"`
	EntryType   int               `json:"entryType"`
	LogLevel    int               `json:"logLevel"`
}

// LogBatch represents a batch of log entries
// Matches the API specification for logflux-agent-api-v1.yaml
type LogBatch struct {
	Version string     `json:"version,omitempty"` // Optional: Protocol version for compatibility
	Entries []LogEntry `json:"entries"`           // Required: Array of log entries (1-100 items)
}

// LogLevel constants for convenience (syslog severity levels as per API spec)
const (
	LevelEmergency = 1 // System is unusable
	LevelAlert     = 2 // Action must be taken immediately
	LevelCritical  = 3 // Critical conditions
	LevelError     = 4 // Error conditions
	LevelWarning   = 5 // Warning conditions
	LevelNotice    = 6 // Normal but significant condition
	LevelInfo      = 7 // Informational messages
	LevelDebug     = 8 // Debug-level messages
)

// EntryType constants for convenience
const (
	TypeLog = 1 // Standard log entry (default for all entries)
)

// DefaultProtocolVersion is the default protocol version used by the SDK
const DefaultProtocolVersion = "1.0"

// PayloadType identifies the structure and format of the log payload
type PayloadType string

const (
	PayloadTypeGeneric     PayloadType = "generic"      // Generic text logs
	PayloadTypeGenericJSON PayloadType = "generic_json" // Generic JSON data
)

// NewLogEntry creates a new log entry with default values and auto-detection
// Automatically detects JSON payload type. All entries default to TypeLog.
func NewLogEntry(payload, source string) LogEntry {
	if source == "" {
		source = "unknown"
	}
	// Auto-detect payload type (JSON vs generic text)
	payloadType := AutoDetectPayloadType(payload)

	return LogEntry{
		Version:     DefaultProtocolVersion,
		Payload:     payload,
		EntryType:   TypeLog,
		Source:      source,
		LogLevel:    LevelInfo, // Default to 7 (Info) as per API spec
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		PayloadType: string(payloadType),
		Metadata:    make(map[string]string),
	}
}

// WithLogLevel sets the log level (1-8 as per API spec, syslog severity levels)
func (e LogEntry) WithLogLevel(logLevel int) LogEntry {
	if logLevel < LevelEmergency || logLevel > LevelDebug {
		logLevel = LevelInfo // Default to Info if invalid
	}
	e.LogLevel = logLevel
	return e
}

// WithEntryType sets the entry type (only TypeLog supported in minimal SDK)
func (e LogEntry) WithEntryType(entryType int) LogEntry {
	// In minimal SDK, only TypeLog is supported
	e.EntryType = TypeLog
	return e
}

// WithSource sets the source identifier
func (e LogEntry) WithSource(source string) LogEntry {
	if source == "" {
		source = "unknown"
	}
	e.Source = source
	return e
}

// WithMetadata adds metadata to the entry
func (e LogEntry) WithMetadata(key, value string) LogEntry {
	if key == "" {
		return e // Skip empty keys
	}
	// Create a new metadata map to avoid race conditions
	newMetadata := make(map[string]string)
	for k, v := range e.Metadata {
		newMetadata[k] = v
	}
	newMetadata[key] = value
	e.Metadata = newMetadata
	return e
}

// WithAllMetadata sets multiple metadata fields
func (e LogEntry) WithAllMetadata(metadata map[string]string) LogEntry {
	// Create a new metadata map to avoid race conditions
	newMetadata := make(map[string]string)
	for k, v := range e.Metadata {
		newMetadata[k] = v
	}
	for k, v := range metadata {
		newMetadata[k] = v
	}
	e.Metadata = newMetadata
	return e
}

// WithTimestamp sets a custom timestamp in RFC3339 format (UTC)
func (e LogEntry) WithTimestamp(timestamp time.Time) LogEntry {
	e.Timestamp = timestamp.UTC().Format(time.RFC3339)
	return e
}

// WithTimestampString sets a custom timestamp from RFC3339 string
func (e LogEntry) WithTimestampString(timestamp string) LogEntry {
	e.Timestamp = timestamp
	return e
}

// WithPayloadType sets the payload type field as per API spec
func (e LogEntry) WithPayloadType(payloadType PayloadType) LogEntry {
	e.PayloadType = string(payloadType)
	return e
}

// WithVersion sets the protocol version
func (e LogEntry) WithVersion(version string) LogEntry {
	e.Version = version
	return e
}

// IsValidJSON checks if a string contains valid JSON.
// Returns true if the string can be unmarshaled as JSON, false otherwise.
func IsValidJSON(str string) bool {
	var js interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}

// AutoDetectPayloadType attempts to automatically detect the payload type based on content.
// If the message is valid JSON, returns PayloadTypeGenericJSON, otherwise PayloadTypeGeneric.
func AutoDetectPayloadType(message string) PayloadType {
	if IsValidJSON(message) {
		return PayloadTypeGenericJSON
	}
	return PayloadTypeGeneric
}

// PingRequest represents a ping health check request
type PingRequest struct {
	Version string `json:"version,omitempty"` // Optional: Protocol version for compatibility
	Action  string `json:"action"`            // Must be "ping"
}

// PongResponse represents a pong health check response
type PongResponse struct {
	Status string `json:"status"` // Must be "pong"
}

// AuthRequest represents an authentication request for TCP connections
type AuthRequest struct {
	Version      string `json:"version,omitempty"` // Optional: Protocol version for compatibility
	Action       string `json:"action"`            // Must be "authenticate"
	SharedSecret string `json:"shared_secret"`     // Shared secret for authentication
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Status  string `json:"status"`  // "success" or "error"
	Message string `json:"message"` // Success or error message
}

// NewPingRequest creates a new ping request
func NewPingRequest() PingRequest {
	return PingRequest{
		Version: DefaultProtocolVersion,
		Action:  "ping",
	}
}

// NewAuthRequest creates a new authentication request
func NewAuthRequest(sharedSecret string) AuthRequest {
	if sharedSecret == "" {
		panic("sharedSecret cannot be empty for authentication")
	}
	return AuthRequest{
		Version:      DefaultProtocolVersion,
		Action:       "authenticate",
		SharedSecret: sharedSecret,
	}
}
