package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all SDK configuration loaded from environment variables.
type Config struct {
	APIKey      string
	Environment string
	Node        string
	LogGroup    string

	QueueSize     int
	FlushInterval int // seconds
	BatchSize     int

	MaxRetries    int
	InitialDelay  int // milliseconds
	MaxDelay      int // seconds
	BackoffFactor float64

	HTTPTimeout int // seconds

	FailsafeMode      bool
	WorkerCount       int
	EnableCompression bool
	Debug             bool
}

// ValidateAPIKey checks if the key matches <region>-lf_<key> format.
func ValidateAPIKey(key string) error {
	parts := strings.SplitN(key, "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid API key format: must be <region>-lf_<key>")
	}
	region := parts[0]
	validRegions := map[string]bool{"eu": true, "us": true, "ca": true, "au": true, "ap": true}
	if !validRegions[region] {
		return fmt.Errorf("invalid API key region: %q (expected eu, us, ca, au, or ap)", region)
	}
	if !strings.HasPrefix(parts[1], "lf_") {
		return fmt.Errorf("invalid API key format: key must start with lf_")
	}
	if len(parts[1]) <= 3 { // "lf_" + at least one char
		return fmt.Errorf("invalid API key format: key body is empty")
	}
	return nil
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() (*Config, error) {
	return LoadConfigFromEnv()
}

// LoadConfigFromEnv loads configuration from environment variables.
func LoadConfigFromEnv() (*Config, error) {
	config := &Config{
		APIKey:      os.Getenv("LOGFLUX_API_KEY"),
		Environment: os.Getenv("LOGFLUX_ENVIRONMENT"),
		Node:        os.Getenv("LOGFLUX_NODE"),
		LogGroup:    os.Getenv("LOGFLUX_LOG_GROUP"),
	}

	if v := os.Getenv("LOGFLUX_QUEUE_SIZE"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			config.QueueSize = val
		}
	}
	if v := os.Getenv("LOGFLUX_FLUSH_INTERVAL"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			config.FlushInterval = val
		}
	}
	if v := os.Getenv("LOGFLUX_BATCH_SIZE"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			config.BatchSize = val
		}
	}
	if v := os.Getenv("LOGFLUX_MAX_RETRIES"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			config.MaxRetries = val
		}
	}
	if v := os.Getenv("LOGFLUX_INITIAL_DELAY"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			config.InitialDelay = val
		}
	}
	if v := os.Getenv("LOGFLUX_MAX_DELAY"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			config.MaxDelay = val
		}
	}
	if v := os.Getenv("LOGFLUX_BACKOFF_FACTOR"); v != "" {
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			config.BackoffFactor = val
		}
	}
	if v := os.Getenv("LOGFLUX_HTTP_TIMEOUT"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			config.HTTPTimeout = val
		}
	}
	if v := os.Getenv("LOGFLUX_FAILSAFE_MODE"); v != "" {
		if val, err := strconv.ParseBool(v); err == nil {
			config.FailsafeMode = val
		}
	}
	if v := os.Getenv("LOGFLUX_WORKER_COUNT"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			config.WorkerCount = val
		}
	}

	// Compression defaults to true
	config.EnableCompression = true
	if v := os.Getenv("LOGFLUX_ENABLE_COMPRESSION"); v != "" {
		if val, err := strconv.ParseBool(v); err == nil {
			config.EnableCompression = val
		}
	}

	if v := os.Getenv("LOGFLUX_DEBUG"); v != "" {
		if val, err := strconv.ParseBool(v); err == nil {
			config.Debug = val
		}
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("LOGFLUX_API_KEY environment variable is required")
	}
	if err := ValidateAPIKey(config.APIKey); err != nil {
		return nil, err
	}

	return config, nil
}
