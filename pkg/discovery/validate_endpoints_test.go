package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateEndpoints_Health2xx4xxOK(t *testing.T) {
	// returns 404 -> should still be considered reachable
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	d := NewDiscoveryClient(DiscoveryConfig{APIKey: "k", Timeout: time.Second})
	ep := &EndpointInfo{BaseURL: srv.URL}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := d.ValidateEndpoints(ctx, ep); err != nil {
		t.Fatalf("ValidateEndpoints should succeed on 4xx: %v", err)
	}
}

func TestValidateEndpoints_Health5xxFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	d := NewDiscoveryClient(DiscoveryConfig{APIKey: "k", Timeout: time.Second})
	ep := &EndpointInfo{BaseURL: srv.URL}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := d.ValidateEndpoints(ctx, ep); err == nil {
		t.Fatalf("expected validation failure on 5xx")
	}
}
