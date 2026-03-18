package config

import (
	"testing"
)

func TestConfigFieldsForResilientMapping(t *testing.T) {
	// Verify that Config fields used for resilient client mapping load correctly
	// and have expected types (int seconds/milliseconds, float64 backoff).
	withEnv(map[string]string{
		"LOGFLUX_API_KEY":        "eu-lf_testkey123",
		"LOGFLUX_QUEUE_SIZE":     "500",
		"LOGFLUX_FLUSH_INTERVAL": "7",
		"LOGFLUX_BATCH_SIZE":     "50",
		"LOGFLUX_MAX_RETRIES":    "4",
		"LOGFLUX_INITIAL_DELAY":  "250",
		"LOGFLUX_MAX_DELAY":      "9",
		"LOGFLUX_BACKOFF_FACTOR": "1.5",
		"LOGFLUX_HTTP_TIMEOUT":   "11",
		"LOGFLUX_FAILSAFE_MODE":  "true",
		"LOGFLUX_WORKER_COUNT":   "3",
		"LOGFLUX_ENABLE_COMPRESSION": "false",
	}, func() {
		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("LoadConfigFromEnv error: %v", err)
		}

		if cfg.APIKey != "eu-lf_testkey123" {
			t.Fatalf("APIKey mismatch: %s", cfg.APIKey)
		}
		if cfg.QueueSize != 500 {
			t.Fatalf("QueueSize mismatch: %d", cfg.QueueSize)
		}
		if cfg.FlushInterval != 7 {
			t.Fatalf("FlushInterval mismatch (should be int seconds): %d", cfg.FlushInterval)
		}
		if cfg.BatchSize != 50 {
			t.Fatalf("BatchSize mismatch: %d", cfg.BatchSize)
		}
		if cfg.MaxRetries != 4 {
			t.Fatalf("MaxRetries mismatch: %d", cfg.MaxRetries)
		}
		if cfg.InitialDelay != 250 {
			t.Fatalf("InitialDelay mismatch (should be int milliseconds): %d", cfg.InitialDelay)
		}
		if cfg.MaxDelay != 9 {
			t.Fatalf("MaxDelay mismatch (should be int seconds): %d", cfg.MaxDelay)
		}
		if cfg.BackoffFactor != 1.5 {
			t.Fatalf("BackoffFactor mismatch: %v", cfg.BackoffFactor)
		}
		if cfg.HTTPTimeout != 11 {
			t.Fatalf("HTTPTimeout mismatch (should be int seconds): %d", cfg.HTTPTimeout)
		}
		if !cfg.FailsafeMode {
			t.Fatalf("FailsafeMode should be true")
		}
		if cfg.WorkerCount != 3 {
			t.Fatalf("WorkerCount mismatch: %d", cfg.WorkerCount)
		}
		if cfg.EnableCompression {
			t.Fatalf("EnableCompression should be false")
		}
	})
}
