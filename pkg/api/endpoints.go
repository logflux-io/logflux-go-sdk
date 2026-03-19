package api

// Paths holds all API endpoint path segments used by the SDK.
// These default values reflect the currently implemented public API.
// They are centralized here to make future spec-driven changes atomic.
type Paths struct {
	IngestPath              string // Single log ingest
	BatchPath               string // Batch ingest
	VersionPath             string // Service info
	HealthPath              string // Health check
	HandshakeBasePath       string // Base handshake path (without suffixes)
	HandshakeInitSuffix     string // Suffix appended for handshake init
	HandshakeCompleteSuffix string // Suffix appended for handshake complete
}

// DefaultPaths contains the effective endpoint paths used by the SDK.
var DefaultPaths = Paths{
	IngestPath:              "/v1/ingest",
	BatchPath:               "/v1/batch",
	VersionPath:             "/info",
	HealthPath:              "/health",
	HandshakeBasePath:       "/v1/handshake",
	HandshakeInitSuffix:     "/init",
	HandshakeCompleteSuffix: "/complete",
}

// GetHandshakeInitPath returns full init path relative to base URL.
func (p Paths) GetHandshakeInitPath() string { return p.HandshakeBasePath + p.HandshakeInitSuffix }

// GetHandshakeCompletePath returns full complete path relative to base URL.
func (p Paths) GetHandshakeCompletePath() string {
	return p.HandshakeBasePath + p.HandshakeCompleteSuffix
}
