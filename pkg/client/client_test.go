package client

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mock ingestor with handshake endpoints; bypass discovery by using custom endpoint URL
func newMockIngestorServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Generate RSA keypair for handshake
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	pubDer, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDer})

	// handshake init returns PEM
	mux.HandleFunc("/v1/handshake/init", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"public_key": string(pubPEM)})
	})

	// handshake complete returns key uuid
	mux.HandleFunc("/v1/handshake/complete", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"key_uuid": "test-key"})
	})

	// ingest
	mux.HandleFunc("/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	// version
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"api_version": "v1"})
	})

	// health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	srv := httptest.NewServer(mux)
	return srv
}

func TestClient_SendLog_Smoke(t *testing.T) {
	srv := newMockIngestorServer(t)
	defer srv.Close()

	// Create client using custom endpoint to bypass discovery
	c, err := NewClientWithCustomEndpoint("eu-lf_testkey123", srv.URL, "node-1")
	if err != nil {
		t.Fatalf("NewClientWithCustomEndpoint error: %v", err)
	}

	// ensure we can send a simple log
	if err := c.Info("hello"); err != nil {
		t.Fatalf("Send info failed: %v", err)
	}
}
