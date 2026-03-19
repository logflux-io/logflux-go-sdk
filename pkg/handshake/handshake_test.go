package handshake_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/api"
	h "github.com/logflux-io/logflux-go-sdk/v3/pkg/handshake"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/testutils"
)

func TestPerformHandshakeWithMockServer_Succeeds(t *testing.T) {
	// Use existing testutils.MockServer which implements /v1/handshake/init and /complete
	ms := testutils.NewMockServer(t)
	defer ms.Close()

	base := ms.Server.URL + api.DefaultPaths.HandshakeBasePath
	httpClient := &http.Client{Timeout: 2 * time.Second}

	res, err := h.PerformHandshakeWithURL(base, "test-api-key", httpClient)
	if err != nil {
		t.Fatalf("PerformHandshakeWithURL error: %v", err)
	}
	if res == nil || len(res.AESKey) == 0 || res.KeyUUID == "" {
		t.Fatalf("unexpected result: %#v", res)
	}
	if res.ServerPublicKeyPEM == "" || res.ServerKeyFingerprint == "" {
		t.Fatalf("expected server public key info to be populated")
	}
}

func TestPerformHandshake_NetworkError(t *testing.T) {
	// point to a closed port to simulate connectivity failure
	base := "http://127.0.0.1:1" + api.DefaultPaths.HandshakeBasePath
	httpClient := &http.Client{Timeout: 300 * time.Millisecond}

	_, err := h.PerformHandshakeWithURL(base, "key", httpClient)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, h.ErrIngestorUnavailable) {
		t.Fatalf("expected ErrIngestorUnavailable wrapping, got: %v", err)
	}
	// message should include 'cannot connect to'
	if !contains(err.Error(), "cannot connect to") {
		t.Fatalf("expected friendly message, got: %v", err)
	}
}

func TestPerformHandshake_InvalidInitResponse(t *testing.T) {
	// server returns OK but without public_key
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case api.DefaultPaths.HandshakeInitSuffix: // this will not match directly because we need base path
			// unreachable branch; we'll switch on full path below
		}
		if r.URL.Path == api.DefaultPaths.HandshakeBasePath+api.DefaultPaths.HandshakeInitSuffix {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"foo":"bar"}`))
			return
		}
		// complete should not be hit, but return OK if it is
		if r.URL.Path == api.DefaultPaths.HandshakeBasePath+api.DefaultPaths.HandshakeCompleteSuffix {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"key_uuid":"x"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	base := srv.URL + api.DefaultPaths.HandshakeBasePath
	httpClient := &http.Client{Timeout: 2 * time.Second}
	_, err := h.PerformHandshakeWithURL(base, "k", httpClient)
	if err == nil {
		t.Fatalf("expected error for invalid init response")
	}
}

func TestPerformHandshake_CompleteFailure(t *testing.T) {
	// init OK, complete returns 500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == api.DefaultPaths.HandshakeBasePath+api.DefaultPaths.HandshakeInitSuffix {
			// Return a valid minimal PEM public key (invalid content doesn't matter for client parsing)
			pem := "-----BEGIN PUBLIC KEY-----\nMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBALn6vZc5k2K8c5E7bO7A5a1Cz7Z0cJgq\nH0Yg6u1o7+d3Zy1gP0QnqY5jH8K0wMfrL1Jm7dY2tZ8mQ3U2rWjYqYsCAwEAAQ==\n-----END PUBLIC KEY-----"
			_, _ = w.Write([]byte(`{"public_key":"` + pem + `"}`))
			return
		}
		if r.URL.Path == api.DefaultPaths.HandshakeBasePath+api.DefaultPaths.HandshakeCompleteSuffix {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	base := srv.URL + api.DefaultPaths.HandshakeBasePath
	httpClient := &http.Client{Timeout: 2 * time.Second}
	_, err := h.PerformHandshakeWithURL(base, "k", httpClient)
	if err == nil {
		t.Fatalf("expected error for complete failure")
	}
}

// local contains to avoid importing retry internals
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if toLower(s[i+j]) != toLower(sub[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
