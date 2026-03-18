package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiscovery_SpecResponse(t *testing.T) {
	// mock discovery endpoint returning spec response
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data_residency": "eu",
			"backend_url":    "https://backend",
			"ingestor_url":   "https://ingestor",
			"environment":    "test",
			"features":       []string{"ingest"},
		})
	}))
	defer srv.Close()

	dc := NewDiscoveryClient(DiscoveryConfig{APIKey: "k", Timeout: time.Second})
	// override base to our server
	dc.baseURL = srv.URL

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ep, err := dc.DiscoverEndpoints(ctx, "")
	if err != nil {
		t.Fatalf("DiscoverEndpoints error: %v", err)
	}
	if ep.BaseURL == "" || ep.GetIngestURL() != "https://ingestor/v1/ingest" {
		t.Fatalf("unexpected endpoints: %+v", ep)
	}
}
