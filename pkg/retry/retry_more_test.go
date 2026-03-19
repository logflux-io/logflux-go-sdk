package retry

import (
	"testing"
	"time"
)

func TestIsRetryable_StatusCodes(t *testing.T) {
	r := NewRetryer(DefaultConfig())
	cases := []struct {
		code int
		want bool
	}{
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{404, false},
		{400, false},
	}
	for _, c := range cases {
		if got := r.isRetryable(&HTTPError{StatusCode: c.code}); got != c.want {
			t.Fatalf("code %d retryable=%v", c.code, got)
		}
	}
}

func TestCalculateDelay_CapsToMax(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxDelay = 200 * time.Millisecond
	cfg.JitterEnabled = false // disable jitter for deterministic test
	r := NewRetryer(cfg)

	// At high attempt count, delay should be capped to MaxDelay
	got := r.calculateDelay(100)
	if got != cfg.MaxDelay {
		t.Fatalf("expected delay capped to %v, got %v", cfg.MaxDelay, got)
	}
}
