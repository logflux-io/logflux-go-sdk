package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExtractRegionFromKey(t *testing.T) {
	tests := []struct {
		key            string
		expectedRegion string
		expectedKey    string
	}{
		{"eu-lf_abc123", "eu", "lf_abc123"},
		{"us-lf_abc123", "us", "lf_abc123"},
		{"ca-lf_abc123", "ca", "lf_abc123"},
		{"au-lf_abc123", "au", "lf_abc123"},
		{"ap-lf_abc123", "ap", "lf_abc123"},
		{"lf_abc123", "", "lf_abc123"},
		{"", "", ""},
		{"xx-lf_abc123", "", "xx-lf_abc123"},
		{"eu-lf_usr_abc123", "eu", "lf_usr_abc123"},
	}

	for _, tt := range tests {
		region, stripped := ExtractRegionFromKey(tt.key)
		if region != tt.expectedRegion {
			t.Errorf("ExtractRegionFromKey(%q) region = %q, want %q", tt.key, region, tt.expectedRegion)
		}
		if stripped != tt.expectedKey {
			t.Errorf("ExtractRegionFromKey(%q) key = %q, want %q", tt.key, stripped, tt.expectedKey)
		}
	}
}

func TestStaticDiscoveryURL(t *testing.T) {
	if got := StaticDiscoveryURL("eu"); got != "https://discover.eu.logflux.io" {
		t.Errorf("StaticDiscoveryURL(eu) = %q", got)
	}
	if got := StaticDiscoveryURL("us"); got != "https://discover.us.logflux.io" {
		t.Errorf("StaticDiscoveryURL(us) = %q", got)
	}
}

func TestDiscovery_StaticEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("static discovery should not send Authorization header")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": "1.0",
			"region":  "eu",
			"endpoints": map[string]any{
				"backend_url":   "https://api.inspect.eu.logflux.io",
				"ingestor_url":  "https://api.ingest.eu.logflux.io",
				"dashboard_url": "https://dashboard.logflux.io",
			},
			"updated_at": "2026-03-14T00:00:00Z",
		})
	}))
	defer srv.Close()

	dc := NewDiscoveryClient(DiscoveryConfig{APIKey: "eu-lf_testkey", Timeout: time.Second})
	// Override the static discovery URL by making tryStaticDiscovery hit our test server.
	// We test via the exported method by overriding baseURL to something that will fail,
	// but since the key has eu- prefix, static discovery will be tried first.
	// We need to intercept the static URL. Use a direct call instead.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Test tryStaticDiscovery directly using the test server
	// Override the function by calling the method on a client pointed at the test server
	dc.httpClient = srv.Client()
	ep, err := dc.tryStaticDiscovery(ctx, "eu")
	if err == nil {
		// The test server doesn't match the real URL, so tryStaticDiscovery will try to reach
		// discover.eu.logflux.io which won't resolve to our test server.
		// Let's test a different way - use a mock transport.
		_ = ep
	}
	cancel()

	// Use a transport that redirects all requests to our test server
	transport := &redirectTransport{target: srv.URL}
	dc2 := &DiscoveryClient{
		httpClient: &http.Client{Timeout: time.Second, Transport: transport},
		apiKey:     "eu-lf_testkey",
		baseURL:    srv.URL,
		timeout:    time.Second,
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()

	ep, err = dc2.tryStaticDiscovery(ctx2, "eu")
	if err != nil {
		t.Fatalf("tryStaticDiscovery error: %v", err)
	}
	if ep.BaseURL != "https://api.ingest.eu.logflux.io" {
		t.Errorf("unexpected BaseURL: %s", ep.BaseURL)
	}
	if ep.Region != "eu" {
		t.Errorf("unexpected Region: %s", ep.Region)
	}
	if ep.Metadata["discovery"] != "static" {
		t.Errorf("expected discovery=static in metadata")
	}
	if ep.Metadata["backend_url"] != "https://api.inspect.eu.logflux.io" {
		t.Errorf("unexpected backend_url: %s", ep.Metadata["backend_url"])
	}
}

func TestDiscovery_StaticFallsBackToAuthenticated(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Return authenticated discovery response
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data_residency": "eu",
			"backend_url":    "https://backend",
			"ingestor_url":   "https://ingestor-fallback",
			"environment":    "test",
			"features":       []string{"ingest"},
		})
	}))
	defer srv.Close()

	// Use a transport that fails for static discovery but works for authenticated
	transport := &selectiveTransport{
		staticFail: true,
		fallback:   srv.Client().Transport,
		serverURL:  srv.URL,
	}

	dc := &DiscoveryClient{
		httpClient: &http.Client{Timeout: time.Second, Transport: transport},
		apiKey:     "eu-lf_testkey",
		baseURL:    srv.URL,
		timeout:    time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ep, err := dc.DiscoverEndpoints(ctx, "")
	if err != nil {
		t.Fatalf("DiscoverEndpoints error: %v", err)
	}
	if ep.BaseURL != "https://ingestor-fallback" {
		t.Errorf("expected fallback ingestor URL, got: %s", ep.BaseURL)
	}
}

func TestDiscovery_LegacyKeySkipsStatic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should be authenticated discovery (has Authorization header)
		if r.Header.Get("Authorization") == "" {
			t.Error("legacy key should use authenticated discovery with Authorization header")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data_residency": "eu",
			"backend_url":    "https://backend",
			"ingestor_url":   "https://ingestor-authenticated",
			"environment":    "test",
			"features":       []string{"ingest"},
		})
	}))
	defer srv.Close()

	dc := NewDiscoveryClient(DiscoveryConfig{APIKey: "lf_legacykey", Timeout: time.Second})
	dc.baseURL = srv.URL

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ep, err := dc.DiscoverEndpoints(ctx, "")
	if err != nil {
		t.Fatalf("DiscoverEndpoints error: %v", err)
	}
	if ep.BaseURL != "https://ingestor-authenticated" {
		t.Errorf("expected authenticated discovery URL, got: %s", ep.BaseURL)
	}
}

// redirectTransport redirects all requests to the target URL.
type redirectTransport struct {
	target string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())
	newReq.URL.Scheme = "http"
	newReq.URL.Host = t.target[len("http://"):]
	return http.DefaultTransport.RoundTrip(newReq)
}

// selectiveTransport fails static discovery requests but passes through others.
type selectiveTransport struct {
	staticFail bool
	fallback   http.RoundTripper
	serverURL  string
}

func (t *selectiveTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.staticFail && req.Header.Get("Authorization") == "" {
		return nil, &http.MaxBytesError{}
	}
	// Redirect to test server
	newReq := req.Clone(req.Context())
	newReq.URL.Scheme = "http"
	newReq.URL.Host = t.serverURL[len("http://"):]
	if t.fallback != nil {
		return t.fallback.RoundTrip(newReq)
	}
	return http.DefaultTransport.RoundTrip(newReq)
}
