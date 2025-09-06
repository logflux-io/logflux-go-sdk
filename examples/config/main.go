package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// AppConfig represents application-level configuration
// Users can use any format they prefer (JSON, YAML, TOML, etc.)
type AppConfig struct {
	LogFlux LogFluxConfig `json:"logflux"`
	// ... other application settings
}

// LogFluxConfig maps to SDK configuration
type LogFluxConfig struct {
	SocketPath   string `json:"socket_path"`
	Network      string `json:"network"`
	SharedSecret string `json:"shared_secret,omitempty"`
	MaxRetries   int    `json:"max_retries"`
	TimeoutMs    int    `json:"timeout_ms"`
	BatchSize    int    `json:"batch_size"`
	FlushMs      int    `json:"flush_interval_ms"`
}

// ToSDKConfig converts application config to SDK config
func (c *LogFluxConfig) ToSDKConfig() *config.Config {
	cfg := config.DefaultConfig()

	if c.SocketPath != "" {
		cfg.Address = c.SocketPath
	}
	if c.Network != "" {
		cfg.Network = c.Network
	}
	if c.SharedSecret != "" {
		cfg.SharedSecret = c.SharedSecret
	}
	if c.MaxRetries > 0 {
		cfg.MaxRetries = c.MaxRetries
	}
	if c.TimeoutMs > 0 {
		cfg.Timeout = time.Duration(c.TimeoutMs) * time.Millisecond
	}
	if c.BatchSize > 0 {
		cfg.BatchSize = c.BatchSize
	}
	if c.FlushMs > 0 {
		cfg.FlushInterval = time.Duration(c.FlushMs) * time.Millisecond
	}

	return cfg
}

func main() {
	// Example: Load configuration from JSON file (user's choice)
	configFile := "app-config.json"

	// Create example config file if it doesn't exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		exampleConfig := AppConfig{
			LogFlux: LogFluxConfig{
				SocketPath: "/tmp/logflux-agent.sock",
				Network:    "unix",
				MaxRetries: 3,
				TimeoutMs:  10000,
				BatchSize:  10,
				FlushMs:    5000,
			},
		}

		data, _ := json.MarshalIndent(exampleConfig, "", "  ")
		os.WriteFile(configFile, data, 0644)
		log.Printf("Created example config file: %s", configFile)
	}

	// Load user's configuration (JSON in this example)
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var appConfig AppConfig
	if err := json.Unmarshal(data, &appConfig); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Convert to SDK configuration
	sdkConfig := appConfig.LogFlux.ToSDKConfig()

	// Use the SDK with user-managed configuration
	c := client.NewClient(sdkConfig)

	ctx := context.Background()
	if err := c.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer c.Close()

	// Send a log entry
	entry := types.NewLogEntry("User-managed configuration example!", "config-example").
		WithLogLevel(types.LevelInfo).
		WithMetadata("config_source", "json")

	if err := c.SendLogEntry(entry); err != nil {
		log.Fatalf("Failed to send log: %v", err)
	}

	log.Println("Log sent successfully using user-managed configuration!")
}
