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

func TestClient_ErrorParsing_FlatAndLegacy(t *testing.T) {
	mux := http.NewServeMux()

	// Generate a proper 2048-bit RSA key for handshake
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	mux.HandleFunc("/v1/handshake/init", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"public_key": string(pemBytes)})
	})
	mux.HandleFunc("/v1/handshake/complete", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"key_uuid": "test-key"})
	})

	// flat error
	mux.HandleFunc("/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("Retry-After", "2")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":       "rate_limited",
			"message":    "too many requests",
			"details":    "slow down",
			"request_id": "abc",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClientWithCustomEndpoint("eu-lf_testkey123", srv.URL, "node1")
	if err != nil {
		t.Fatalf("NewClientWithCustomEndpoint: %v", err)
	}

	err = c.Info("hello")
	if err == nil {
		t.Fatalf("expected error for 429")
	}
	if got := err.Error(); got == "" || got == "unexpected response status" {
		t.Fatalf("unexpected error text: %v", got)
	}

	// legacy error on a separate server to avoid mux re-registration
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/v1/handshake/init", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"public_key": string(pemBytes)})
	})
	mux2.HandleFunc("/v1/handshake/complete", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"key_uuid": "test-key"})
	})
	mux2.HandleFunc("/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "error",
			"error": map[string]string{
				"code":    "bad_gateway",
				"message": "upstream failure",
				"details": "x",
			},
			"request_id": "def",
		})
	})
	srv2 := httptest.NewServer(mux2)
	defer srv2.Close()
	c2, err := NewClientWithCustomEndpoint("eu-lf_testkey123", srv2.URL, "node1")
	if err != nil {
		t.Fatalf("NewClientWithCustomEndpoint(legacy): %v", err)
	}
	err = c2.Info("hello2")
	if err == nil {
		t.Fatalf("expected error for legacy failure")
	}
}

func TestClient_RateLimitHeaders(t *testing.T) {
	mux := http.NewServeMux()
	// Proper RSA for handshake
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	mux.HandleFunc("/v1/handshake/init", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"public_key": string(pemBytes)})
	})
	mux.HandleFunc("/v1/handshake/complete", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"key_uuid": "test-key"})
	})
	mux.HandleFunc("/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "60")
		w.Header().Set("X-RateLimit-Reset", "1234567890")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClientWithCustomEndpoint("eu-lf_testkey123", srv.URL, "node1")
	if err != nil {
		t.Fatalf("NewClientWithCustomEndpoint: %v", err)
	}

	if err := c.Info("hello"); err != nil {
		t.Fatalf("info send: %v", err)
	}

	limit, remaining, reset := c.GetRateLimitInfo()
	if limit != 100 || remaining != 60 || reset != 1234567890 {
		t.Fatalf("unexpected rate limit info: %d %d %d", limit, remaining, reset)
	}
}
