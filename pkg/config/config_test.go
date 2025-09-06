package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Network != "unix" {
		t.Errorf("Expected default network 'unix', got %s", cfg.Network)
	}

	if cfg.Address != "/tmp/logflux-agent.sock" {
		t.Errorf("Expected default address '/tmp/logflux-agent.sock', got %s", cfg.Address)
	}

	if cfg.Timeout <= 0 {
		t.Errorf("Expected positive timeout, got %v", cfg.Timeout)
	}

	if cfg.BatchSize <= 0 {
		t.Errorf("Expected positive batch size, got %d", cfg.BatchSize)
	}

	if cfg.MaxRetries < 0 {
		t.Errorf("Expected non-negative max retries, got %d", cfg.MaxRetries)
	}
}

func TestDefaultBatchConfig(t *testing.T) {
	cfg := DefaultBatchConfig()

	if cfg.MaxBatchSize <= 0 {
		t.Errorf("Expected positive max batch size, got %d", cfg.MaxBatchSize)
	}

	if cfg.FlushInterval <= 0 {
		t.Errorf("Expected positive flush interval, got %v", cfg.FlushInterval)
	}

	if !cfg.AutoFlush {
		t.Error("Expected auto flush to be enabled by default")
	}
}

func TestCalculateBackoffDelay(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "attempt 0 returns initial delay",
			attempt:  0,
			expected: cfg.RetryDelay,
		},
		{
			name:     "negative attempt returns initial delay",
			attempt:  -1,
			expected: cfg.RetryDelay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := cfg.CalculateBackoffDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("Expected delay %v, got %v", tt.expected, delay)
			}
		})
	}

	// Test exponential backoff progression
	t.Run("exponential backoff progression", func(t *testing.T) {
		delay1 := cfg.CalculateBackoffDelay(1)
		delay2 := cfg.CalculateBackoffDelay(2)

		// Should have exponential growth (allowing for jitter)
		expectedMin1 := time.Duration(float64(cfg.RetryDelay) * cfg.RetryMultiplier * 0.9) // 90% of expected (accounting for jitter)
		expectedMin2 := time.Duration(float64(cfg.RetryDelay) * cfg.RetryMultiplier * cfg.RetryMultiplier * 0.9)

		if delay1 < expectedMin1 {
			t.Errorf("Delay1 %v should be at least %v", delay1, expectedMin1)
		}

		if delay2 < expectedMin2 {
			t.Errorf("Delay2 %v should be at least %v", delay2, expectedMin2)
		}
	})

	// Test maximum delay cap
	t.Run("respects maximum delay", func(t *testing.T) {
		delay := cfg.CalculateBackoffDelay(10) // Large attempt number

		// Should not exceed max delay by much (jitter can add up to 10%)
		maxAllowed := time.Duration(float64(cfg.MaxRetryDelay) * 1.1)
		if delay > maxAllowed {
			t.Errorf("Delay %v should not exceed max %v (with jitter)", delay, maxAllowed)
		}
	})

	// Test minimum delay enforcement
	t.Run("never goes below initial delay", func(t *testing.T) {
		// Test with config that has jitter
		for attempt := 1; attempt <= 5; attempt++ {
			delay := cfg.CalculateBackoffDelay(attempt)
			if delay < cfg.RetryDelay {
				t.Errorf("Delay %v should never be less than initial delay %v", delay, cfg.RetryDelay)
			}
		}
	})
}
