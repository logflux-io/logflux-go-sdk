package api

import "testing"

func TestDefaultPathsAndBuilders(t *testing.T) {
	if DefaultPaths.IngestPath != "/v1/ingest" {
		t.Fatalf("IngestPath mismatch: %q", DefaultPaths.IngestPath)
	}
	if DefaultPaths.BatchPath != "/v1/batch" {
		t.Fatalf("BatchPath mismatch: %q", DefaultPaths.BatchPath)
	}
	if DefaultPaths.VersionPath != "/info" {
		t.Fatalf("VersionPath mismatch: %q", DefaultPaths.VersionPath)
	}
	if DefaultPaths.HealthPath != "/health" {
		t.Fatalf("HealthPath mismatch: %q", DefaultPaths.HealthPath)
	}
	if DefaultPaths.HandshakeBasePath != "/v1/handshake" {
		t.Fatalf("HandshakeBasePath mismatch: %q", DefaultPaths.HandshakeBasePath)
	}
	if DefaultPaths.HandshakeInitSuffix != "/init" {
		t.Fatalf("HandshakeInitSuffix mismatch: %q", DefaultPaths.HandshakeInitSuffix)
	}
	if DefaultPaths.HandshakeCompleteSuffix != "/complete" {
		t.Fatalf("HandshakeCompleteSuffix mismatch: %q", DefaultPaths.HandshakeCompleteSuffix)
	}

	if got := DefaultPaths.GetHandshakeInitPath(); got != "/v1/handshake/init" {
		t.Fatalf("GetHandshakeInitPath = %q", got)
	}
	if got := DefaultPaths.GetHandshakeCompletePath(); got != "/v1/handshake/complete" {
		t.Fatalf("GetHandshakeCompletePath = %q", got)
	}
}
