package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestRetry_SucceedsAfterFailures(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxRetries = 3
	cfg.InitialDelay = 10 * time.Millisecond
	cfg.MaxDelay = 50 * time.Millisecond

	r := NewRetryer(cfg)

	attempts := 0
	err := r.Retry(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return &HTTPError{StatusCode: http.StatusServiceUnavailable, Message: "unavailable"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_StopsOnNonRetryable(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxRetries = 3
	cfg.InitialDelay = 10 * time.Millisecond

	r := NewRetryer(cfg)

	attempts := 0
	err := r.Retry(context.Background(), func() error {
		attempts++
		return errors.New("permanent failure")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt for non-retryable, got %d", attempts)
	}
}

func TestRetry_UsesRetryAfter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxRetries = 3
	cfg.InitialDelay = 1 * time.Millisecond
	cfg.MaxDelay = 2 * time.Second

	r := NewRetryer(cfg)

	attempts := 0
	started := time.Now()
	err := r.Retry(context.Background(), func() error {
		attempts++
		if attempts == 1 {
			// First call: rate limited with 1s Retry-After
			rr := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": []string{"1"}}}
			return NewHTTPErrorFromResponse(rr, "rate limited")
		}
		return nil // succeed on second call
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(started)
	if elapsed < 900*time.Millisecond {
		t.Fatalf("expected retry-after delay (~1s) to be respected, elapsed=%v", elapsed)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
