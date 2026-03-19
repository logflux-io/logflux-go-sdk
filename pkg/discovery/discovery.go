package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/api"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/sdkversion"
)

// Response body size limits for discovery responses.
const maxDiscoveryResponseSize = 64 << 10 // 64 KiB

var discoveryUserAgent = "logflux-go-sdk/" + sdkversion.Version

// validRegionPrefixes lists the recognized region prefixes for API keys.
var validRegionPrefixes = []string{"eu-", "us-", "ca-", "au-", "ap-"}

// ExtractRegionFromKey extracts the region prefix from an API key.
// Returns the region code (e.g. "eu") and the key with the prefix stripped.
// If no region prefix is found, returns empty region and the original key.
func ExtractRegionFromKey(key string) (region string, strippedKey string) {
	for _, prefix := range validRegionPrefixes {
		if strings.HasPrefix(key, prefix) {
			return strings.TrimSuffix(prefix, "-"), key[len(prefix):]
		}
	}
	return "", key
}

// StaticDiscoveryURL returns the static discovery URL for a region.
func StaticDiscoveryURL(region string) string {
	return fmt.Sprintf("https://discover.%s.logflux.io", region)
}

// EndpointInfo represents information about discovered endpoints
type EndpointInfo struct {
	BaseURL      string            `json:"base_url"` // Base ingestor URL
	Region       string            `json:"region,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	RateLimit    *RateLimitInfo    `json:"rate_limit,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// GetIngestURL returns the ingest endpoint URL
func (e *EndpointInfo) GetIngestURL() string {
	return e.BaseURL + api.DefaultPaths.IngestPath
}

// GetBatchURL returns the batch endpoint URL
func (e *EndpointInfo) GetBatchURL() string {
	return e.BaseURL + api.DefaultPaths.BatchPath
}

// GetVersionURL returns the version endpoint URL
func (e *EndpointInfo) GetVersionURL() string {
	return e.BaseURL + api.DefaultPaths.VersionPath
}

// GetHealthURL returns the health endpoint URL
func (e *EndpointInfo) GetHealthURL() string {
	return e.BaseURL + api.DefaultPaths.HealthPath
}

// GetHandshakeURL returns the handshake endpoint URL
func (e *EndpointInfo) GetHandshakeURL() string {
	// SDK connects only to the ingestor; build handshake URL from ingestor BaseURL
	return e.BaseURL + api.DefaultPaths.HandshakeBasePath
}

// RateLimitInfo contains rate limiting information for the discovered endpoints
type RateLimitInfo struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	BurstLimit        int `json:"burst_limit"`
	WindowSize        int `json:"window_size"` // in seconds
}

// DiscoveryClient handles endpoint discovery for LogFlux services
type DiscoveryClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	timeout    time.Duration
}

// DiscoveryConfig holds configuration for the discovery client
type DiscoveryConfig struct {
	APIKey     string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// NewDiscoveryClient creates a new endpoint discovery client
func NewDiscoveryClient(config DiscoveryConfig) *DiscoveryClient {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}
	if httpClient.Timeout == 0 {
		httpClient.Timeout = 10 * time.Second
	}

	// Use API gateway per spec as primary discovery base. We'll try regional gateways as fallbacks.
	baseURL := "https://api.logflux.io"

	return &DiscoveryClient{
		httpClient: httpClient,
		apiKey:     config.APIKey,
		baseURL:    baseURL,
		timeout:    httpClient.Timeout,
	}
}

// DiscoverEndpoints discovers LogFlux regional endpoints.
// If the API key has a region prefix (e.g. "eu-lf_..."), it first tries the
// static, unauthenticated discovery endpoint at discover.{region}.logflux.io.
// Falls back to the authenticated API gateway discovery for legacy keys or on failure.
func (d *DiscoveryClient) DiscoverEndpoints(ctx context.Context, identifier string) (*EndpointInfo, error) {
	// If key has a region prefix, try static discovery first
	region, _ := ExtractRegionFromKey(d.apiKey)
	if region != "" {
		endpoints, err := d.tryStaticDiscovery(ctx, region)
		if err == nil {
			return endpoints, nil
		}
		// Static discovery failed, fall through to authenticated discovery
	}

	// Authenticated discovery: try configured baseURL first (for tests/overrides),
	// then public API gateways as fallback.
	discoveryURLs := []string{
		d.baseURL + "/api/discovery",
		"https://api.logflux.io/api/discovery",
		"https://eu.api.logflux.io/api/discovery",
		"https://us.api.logflux.io/api/discovery",
	}

	var lastErr error
	for _, url := range discoveryURLs {
		endpoints, err := d.tryDiscoveryURL(ctx, url)
		if err == nil {
			return endpoints, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all discovery URLs failed, last error: %w", lastErr)
}

// tryStaticDiscovery fetches endpoints from the static, unauthenticated
// per-region discovery endpoint (e.g. https://discover.eu.logflux.io/).
func (d *DiscoveryClient) tryStaticDiscovery(ctx context.Context, region string) (*EndpointInfo, error) {
	url := StaticDiscoveryURL(region)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create static discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", discoveryUserAgent)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("static discovery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxDiscoveryResponseSize))
		return nil, fmt.Errorf("static discovery returned status %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxDiscoveryResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read static discovery response: %w", err)
	}

	var staticResp struct {
		Version   string `json:"version"`
		Region    string `json:"region"`
		Endpoints struct {
			BackendURL   string `json:"backend_url"`
			IngestorURL  string `json:"ingestor_url"`
			DashboardURL string `json:"dashboard_url"`
		} `json:"endpoints"`
		UpdatedAt string `json:"updated_at"`
	}

	if err := json.Unmarshal(bodyBytes, &staticResp); err != nil {
		return nil, fmt.Errorf("failed to parse static discovery response: %w", err)
	}

	if staticResp.Endpoints.IngestorURL == "" {
		return nil, fmt.Errorf("static discovery response missing ingestor_url")
	}

	return &EndpointInfo{
		BaseURL: staticResp.Endpoints.IngestorURL,
		Region:  staticResp.Region,
		Metadata: map[string]string{
			"backend_url":   staticResp.Endpoints.BackendURL,
			"dashboard_url": staticResp.Endpoints.DashboardURL,
			"discovery_url": url,
			"discovery":     "static",
		},
	}, nil
}

// tryDiscoveryURL attempts discovery from a specific URL
func (d *DiscoveryClient) tryDiscoveryURL(ctx context.Context, discoveryURL string) (*EndpointInfo, error) {
	// Create HTTP request (GET /api/discovery per spec)
	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}

	// Set authentication header
	req.Header.Set("Authorization", "Bearer "+d.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", discoveryUserAgent)

	// Send request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send discovery request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxDiscoveryResponseSize))
		return nil, fmt.Errorf("discovery request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response body once
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxDiscoveryResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read discovery response: %w", err)
	}

	// Spec format
	var specResp struct {
		DataResidency string   `json:"data_residency"`
		BackendURL    string   `json:"backend_url"`
		IngestorURL   string   `json:"ingestor_url"`
		Environment   string   `json:"environment"`
		Features      []string `json:"features"`
	}
	if err := json.Unmarshal(bodyBytes, &specResp); err == nil && (specResp.IngestorURL != "" || specResp.BackendURL != "") {
		if specResp.IngestorURL == "" {
			return nil, fmt.Errorf("discovery response missing ingestor_url")
		}
		return &EndpointInfo{
			BaseURL:      specResp.IngestorURL,
			Region:       specResp.DataResidency,
			Capabilities: specResp.Features,
			Metadata: map[string]string{
				"backend_url":   specResp.BackendURL,
				"environment":   specResp.Environment,
				"discovery_url": discoveryURL,
			},
		}, nil
	}

	// Legacy wrapped format { status, data }
	var wrapped struct {
		Status string `json:"status"`
		Data   *struct {
			BaseURL      string   `json:"base_url"`
			Region       string   `json:"region"`
			Capabilities []string `json:"capabilities"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(bodyBytes, &wrapped); err == nil && wrapped.Data != nil {
		return &EndpointInfo{
			BaseURL:      wrapped.Data.BaseURL,
			Region:       wrapped.Data.Region,
			Capabilities: wrapped.Data.Capabilities,
		}, nil
	}

	return nil, fmt.Errorf("failed to parse discovery response: %s", string(bodyBytes))
}

// ValidateEndpoints performs basic validation on discovered endpoints
func (d *DiscoveryClient) ValidateEndpoints(ctx context.Context, endpoints *EndpointInfo) error {
	if endpoints == nil {
		return fmt.Errorf("endpoints cannot be nil")
	}

	// Validate required base URL
	if endpoints.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	// Test health endpoint
	if err := d.testEndpoint(ctx, endpoints.GetHealthURL()); err != nil {
		return fmt.Errorf("health endpoint validation failed: %w", err)
	}

	return nil
}

// testEndpoint performs a simple GET request to test if an endpoint is reachable
func (d *DiscoveryClient) testEndpoint(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Accept any 2xx or 4xx status (4xx means endpoint exists but may require auth)
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return nil
	}

	return fmt.Errorf("endpoint returned status %d", resp.StatusCode)
}

// RefreshEndpoints re-discovers endpoints and updates the cache
func (d *DiscoveryClient) RefreshEndpoints(ctx context.Context, identifier string) (*EndpointInfo, error) {
	return d.DiscoverEndpoints(ctx, identifier)
}

// GetDiscoveryURL returns the base discovery URL being used
func (d *DiscoveryClient) GetDiscoveryURL() string {
	return d.baseURL
}

// SetCustomEndpoint allows overriding the discovered endpoint with a custom URL
// This bypasses discovery and uses the provided URL directly
func (d *DiscoveryClient) SetCustomEndpoint(baseURL string) *EndpointInfo {
	return &EndpointInfo{
		BaseURL: baseURL,
	}
}
