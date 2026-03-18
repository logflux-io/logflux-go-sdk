package config

import (
	"os"
	"testing"
)

func withEnv(vars map[string]string, fn func()) {
	orig := map[string]string{}
	for k := range vars {
		orig[k] = os.Getenv(k)
	}
	for k, v := range vars {
		os.Setenv(k, v)
	}
	defer func() {
		for k, v := range orig {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()
	fn()
}

func TestLoadConfigFromEnv_Success(t *testing.T) {
	withEnv(map[string]string{
		"LOGFLUX_API_KEY":            "eu-lf_testkey123",
		"LOGFLUX_QUEUE_SIZE":         "123",
		"LOGFLUX_FLUSH_INTERVAL":     "7",
		"LOGFLUX_MAX_RETRIES":        "5",
		"LOGFLUX_INITIAL_DELAY":      "250",
		"LOGFLUX_MAX_DELAY":          "9",
		"LOGFLUX_BACKOFF_FACTOR":     "1.7",
		"LOGFLUX_HTTP_TIMEOUT":       "11",
		"LOGFLUX_FAILSAFE_MODE":      "true",
		"LOGFLUX_WORKER_COUNT":       "3",
		"LOGFLUX_ENABLE_COMPRESSION": "false",
	}, func() {
		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("LoadConfigFromEnv error: %v", err)
		}
		if cfg.APIKey != "eu-lf_testkey123" || cfg.QueueSize != 123 || cfg.FlushInterval != 7 || cfg.MaxRetries != 5 || cfg.InitialDelay != 250 || cfg.MaxDelay != 9 {
			t.Fatalf("unexpected cfg values: %+v", cfg)
		}
		if cfg.BackoffFactor != 1.7 {
			t.Fatalf("backoff factor mismatch: %v", cfg.BackoffFactor)
		}
		if cfg.HTTPTimeout != 11 || !cfg.FailsafeMode || cfg.WorkerCount != 3 || cfg.EnableCompression {
			t.Fatalf("unexpected toggles: %+v", cfg)
		}
	})
}

func TestLoadConfigFromEnv_MissingAPIKey(t *testing.T) {
	withEnv(map[string]string{}, func() {
		if _, err := LoadConfigFromEnv(); err == nil {
			t.Fatalf("expected error when api key missing")
		}
	})
}
