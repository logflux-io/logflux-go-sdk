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
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

func newServerWithHandshakeAndIngestForBatch(t *testing.T) *httptest.Server {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/handshake/init", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"public_key": string(pemBytes)})
	})
	mux.HandleFunc("/v1/handshake/complete", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"key_uuid": "test-key"})
	})
	mux.HandleFunc("/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"api_version": "v1"})
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return httptest.NewServer(mux)
}

func TestClient_BatchLimitsAndHappyPath(t *testing.T) {
	srv := newServerWithHandshakeAndIngestForBatch(t)
	defer srv.Close()

	c, err := NewClientWithCustomEndpoint("eu-lf_testkey123", srv.URL, "node")
	if err != nil {
		t.Fatalf("NewClientWithCustomEndpoint: %v", err)
	}

	// too large batch
	msgs := make([]LogMessage, 1001)
	for i := range msgs {
		msgs[i] = LogMessage{Message: "m", Timestamp: time.Now(), Level: models.LogLevelInfo}
	}
	if err := c.SendLogBatch(msgs); err == nil {
		t.Fatalf("expected batch size error")
	}

	// small happy path
	msgs = []LogMessage{{Message: "a", Timestamp: time.Now(), Level: models.LogLevelInfo}, {Message: "b", Timestamp: time.Now(), Level: models.LogLevelWarning}}
	if err := c.SendLogBatch(msgs); err != nil {
		t.Fatalf("batch send error: %v", err)
	}
}

func TestClient_Version_And_HealthCheck(t *testing.T) {
	srv := newServerWithHandshakeAndIngestForBatch(t)
	defer srv.Close()

	c, err := NewClientWithCustomEndpoint("eu-lf_testkey123", srv.URL, "node")
	if err != nil {
		t.Fatalf("NewClientWithCustomEndpoint: %v", err)
	}

	v, err := c.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if v["api_version"] != "v1" {
		t.Fatalf("unexpected version map: %#v", v)
	}

	if err := c.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
}
