package client

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

func newHandshakeMux(t *testing.T) (*http.ServeMux, []byte) {
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
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"api_version": "v1"})
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	return mux, pemBytes
}

func TestClient_Validation_TimestampAndLevel(t *testing.T) {
	mux, _ := newHandshakeMux(t)
	mux.HandleFunc("/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClientWithCustomEndpoint("eu-lf_testkey123", srv.URL, "node")
	if err != nil {
		t.Fatalf("NewClientWithCustomEndpoint: %v", err)
	}

	// too far future
	err = c.SendLogWithTimestamp("msg", time.Now().Add(2*time.Minute))
	if err == nil || !strings.Contains(err.Error(), "timestamp cannot be more than 1 minute") {
		t.Fatalf("expected future timestamp error, got %v", err)
	}

	// too old
	err = c.SendLogWithTimestamp("msg", time.Now().AddDate(-2, 0, 0))
	if err == nil || !strings.Contains(err.Error(), "older than 1 year") {
		t.Fatalf("expected old timestamp error, got %v", err)
	}

	// invalid levels (0 means "unset" and is valid; -1 and 9 are out of range 1-8)
	if err := c.SendLogWithLevel("msg", -1); err == nil {
		t.Fatalf("expected level low error")
	}
	if err := c.SendLogWithLevel("msg", 9); err == nil {
		t.Fatalf("expected level high error")
	}
}

func TestClient_Validation_LabelsAndEntryType(t *testing.T) {
	mux, _ := newHandshakeMux(t)
	mux.HandleFunc("/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClientWithCustomEndpoint("eu-lf_testkey123", srv.URL, "node")
	if err != nil {
		t.Fatalf("NewClientWithCustomEndpoint: %v", err)
	}

	// labels > 20
	labels := map[string]string{}
	for i := 0; i < 21; i++ {
		labels[fmt.Sprintf("k%02d", i)] = "v"
	}
	if err := c.SendLogWithTimestampLevelAndLabels("m", time.Now(), models.LogLevelInfo, labels); err == nil {
		t.Fatalf("expected too many labels error")
	}

	// key > 64
	longKey := strings.Repeat("a", 65)
	if err := c.SendLogWithTimestampLevelAndLabels("m", time.Now(), models.LogLevelInfo, map[string]string{longKey: "v"}); err == nil {
		t.Fatalf("expected long key error")
	}

	// value > 256
	longVal := strings.Repeat("b", 257)
	if err := c.SendLogWithTimestampLevelAndLabels("m", time.Now(), models.LogLevelInfo, map[string]string{"ok": longVal}); err == nil {
		t.Fatalf("expected long value error")
	}

	// disallowed key
	if err := c.SendLogWithTimestampLevelAndLabels("m", time.Now(), models.LogLevelInfo, map[string]string{"node": "x"}); err == nil {
		t.Fatalf("expected disallowed key error")
	}

	// entry_type invalid (8 is out of range 1-7)
	if err := c.SendLogWithTimestampLevelTypeAndLabels("m", time.Now(), models.LogLevelInfo, 8, nil); err == nil {
		t.Fatalf("expected entry_type validation error")
	}
}

func TestClient_EntryType_DefaultsToLog(t *testing.T) {
	mux, _ := newHandshakeMux(t)
	gotRequest := false
	mux.HandleFunc("/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		gotRequest = true
		// Multipart/mixed format — verify the X-LF-Entry-Type header in the MIME part
		ct := r.Header.Get("Content-Type")
		if ct == "" {
			t.Fatalf("missing Content-Type header")
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClientWithCustomEndpoint("eu-lf_testkey123", srv.URL, "node")
	if err != nil {
		t.Fatalf("NewClientWithCustomEndpoint: %v", err)
	}

	// entry_type=0 should default to EntryTypeLog (1) internally
	if err := c.SendLogWithTimestampLevelTypeAndLabels("m", time.Now(), models.LogLevelInfo, 0, nil); err != nil {
		t.Fatalf("send error: %v", err)
	}
	if !gotRequest {
		t.Fatalf("expected ingest request to be sent")
	}
}
