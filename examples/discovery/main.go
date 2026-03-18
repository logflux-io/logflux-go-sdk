package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/discovery"
)

func main() {
	fmt.Println("LogFlux Go SDK - Endpoint Discovery Example")
	fmt.Println("==========================================")

	// Example 1: Manual endpoint discovery
	fmt.Println("\n1. Manual Endpoint Discovery")
	manualDiscoveryExample()

	// Example 2: Client with automatic endpoint discovery
	fmt.Println("\n2. Client with Automatic Discovery")
	clientWithDiscoveryExample()
}

func manualDiscoveryExample() {
	// Configure discovery client
	discoveryConfig := discovery.DiscoveryConfig{
		APIKey:  getEnvOrDefault("LOGFLUX_API_KEY", "lf_demo_key123"),
		Timeout: 10 * time.Second,
	}

	discoveryClient := discovery.NewDiscoveryClient(discoveryConfig)

	// Discover endpoints for the authenticated user
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Printf("Discovering endpoints (authentication handled via API key)\n")

	endpoints, err := discoveryClient.DiscoverEndpoints(ctx, "")
	if err != nil {
		fmt.Printf("Discovery failed: %v\n", err)
		fmt.Println("This is expected in demo mode - discovery.logflux.io service may not be available")
		fmt.Println("Discovery will try regional fallback URLs for resilience")
		return
	}

	// Display discovered endpoints
	fmt.Printf("✓ Discovered endpoints:\n")
	fmt.Printf("  Base URL:      %s\n", endpoints.BaseURL)
	fmt.Printf("  Ingest URL:    %s\n", endpoints.GetIngestURL())
	fmt.Printf("  Batch URL:     %s\n", endpoints.GetBatchURL())
	fmt.Printf("  Version URL:   %s\n", endpoints.GetVersionURL())
	fmt.Printf("  Health URL:    %s\n", endpoints.GetHealthURL())
	fmt.Printf("  Handshake URL: %s\n", endpoints.GetHandshakeURL())
	fmt.Printf("  Region:        %s\n", endpoints.Region)

	if endpoints.RateLimit != nil {
		fmt.Printf("  Rate Limit:    %d req/min (burst: %d)\n",
			endpoints.RateLimit.RequestsPerMinute,
			endpoints.RateLimit.BurstLimit)
	}

	if len(endpoints.Capabilities) > 0 {
		fmt.Printf("  Capabilities:  %v\n", endpoints.Capabilities)
	}

	// Validate endpoints
	err = discoveryClient.ValidateEndpoints(ctx, endpoints)
	if err != nil {
		fmt.Printf("Endpoint validation failed: %v\n", err)
	} else {
		fmt.Println("✓ Endpoints validated successfully")
	}
}

func clientWithDiscoveryExample() {
	apiKey := getEnvOrDefault("LOGFLUX_API_KEY", "lf_demo_key123")

	fmt.Printf("Creating client with mandatory discovery...\n")
	fmt.Printf("API Key: %s\n", maskAPIKey(apiKey))

	// Create client with discovery (always enabled)
	clientConfig := client.ClientConfig{
		APIKey:            apiKey,
		Node:              "demo-node",
		HTTPTimeout:       30 * time.Second,
		DiscoveryTimeout:  10 * time.Second,
		EnableCompression: true,
	}

	fmt.Println("Performing endpoint discovery...")
	logClient, err := client.NewClientWithConfig(clientConfig)
	if err != nil {
		fmt.Printf("Client creation failed: %v\n", err)
		fmt.Println("This is expected in demo mode - discovery service may not be available")
		fmt.Println("In production, ensure:")
		fmt.Println("  1. Valid API key (authentication handled automatically)")
		fmt.Println("  2. Network access to discovery.logflux.io")
		fmt.Println("  4. Network access to fallback URLs: api.ingest.eu.logflux.io, api.ingest.us.logflux.io")
		return
	}

	fmt.Println("✓ Client created successfully with discovered endpoints")

	// Test logging with discovered endpoints
	err = logClient.Info("Test message using discovered endpoints")
	if err != nil {
		fmt.Printf("Logging failed: %v\n", err)
		fmt.Println("This is expected in demo mode - ingestor may not be available")
	} else {
		fmt.Println("✓ Log message sent successfully")
	}

	// Test health check
	err = logClient.HealthCheck()
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		fmt.Println("This is expected in demo mode - ingestor may not be available")
	} else {
		fmt.Println("✓ Health check passed")
	}

	// Test version info
	version, err := logClient.GetVersion()
	if err != nil {
		fmt.Printf("Version check failed: %v\n", err)
		fmt.Println("This is expected in demo mode - ingestor may not be available")
	} else {
		fmt.Printf("✓ Version info: %v\n", version)
	}

	logClient.Close()
}

func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}

func init() {
	// Set up demo environment if no real configuration is provided
	if os.Getenv("LOGFLUX_API_KEY") == "" {
		fmt.Println("No LOGFLUX_API_KEY found, using demo configuration")
		fmt.Println("Set environment variables for real usage:")
		fmt.Println("  export LOGFLUX_API_KEY=<your_api_key>")
		fmt.Println()
		fmt.Println("Discovery will automatically try these URLs in order:")
		fmt.Println("  1. https://discovery.logflux.io (primary - provides regional endpoints)")
		fmt.Println("  2. https://api.ingest.eu.logflux.io (EU regional fallback)")
		fmt.Println("  3. https://api.ingest.us.logflux.io (US regional fallback)")
		fmt.Println()
	}
}
