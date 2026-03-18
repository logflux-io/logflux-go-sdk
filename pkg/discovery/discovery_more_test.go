package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiscovery_LegacyWrappedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"base_url":     "https://ingestor.example",
				"region":       "eu",
				"capabilities": []string{"ingest"},
			},
		})
	}))
	defer srv.Close()

	dc := NewDiscoveryClient(DiscoveryConfig{APIKey: "k", Timeout: time.Second})
	dc.baseURL = srv.URL
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ep, err := dc.DiscoverEndpoints(ctx, "")
	if err != nil {
		t.Fatalf("DiscoverEndpoints: %v", err)
	}
	if ep.BaseURL != "https://ingestor.example" {
		t.Fatalf("unexpected base: %s", ep.BaseURL)
	}
}

func TestDiscovery_SpecMissingIngestorURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data_residency": "eu",
			"backend_url":    "https://backend",
			// ingestor_url intentionally missing
			"environment": "test",
			"features":    []string{"ingest"},
		})
	}))
	defer srv.Close()

	dc := NewDiscoveryClient(DiscoveryConfig{APIKey: "k", Timeout: time.Second})
	dc.baseURL = srv.URL
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := dc.DiscoverEndpoints(ctx, ""); err == nil {
		t.Fatalf("expected error due to missing ingestor_url")
	}
}
