package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HTTPError represents an HTTP error with status code and retry information.
type HTTPError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("HTTP %d: %s (retry after %v)", e.StatusCode, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func NewHTTPErrorFromResponse(resp *http.Response, message string) *HTTPError {
	httpErr := &HTTPError{
		StatusCode: resp.StatusCode,
		Message:    message,
	}
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			httpErr.RetryAfter = time.Duration(seconds) * time.Second
		}
	}
	return httpErr
}

// IsQuotaExceeded returns true if this is a 507 quota error.
func (e *HTTPError) IsQuotaExceeded() bool {
	return e.StatusCode == http.StatusInsufficientStorage
}

// IsRateLimited returns true if this is a 429 rate limit error.
func (e *HTTPError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// Config holds retry configuration.
type Config struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	JitterEnabled bool

	ResilientMode      bool
	HealthCheckURL     string
	HealthCheckTimeout time.Duration
	HealthCheckRetries int
}

func DefaultConfig() Config {
	return Config{
		MaxRetries:         3,
		InitialDelay:       1 * time.Second,
		MaxDelay:           30 * time.Second,
		BackoffFactor:      2.0,
		JitterEnabled:      true,
		ResilientMode:      false,
		HealthCheckTimeout: 10 * time.Second,
		HealthCheckRetries: 3,
	}
}

func BasicConfig() Config {
	return DefaultConfig()
}

func ResilientConfig() Config {
	c := DefaultConfig()
	c.MaxRetries = 10
	c.InitialDelay = 100 * time.Millisecond
	c.MaxDelay = 60 * time.Second
	c.ResilientMode = true
	c.HealthCheckRetries = 5
	return c
}

type RetryFunc func() error

// Retryer handles retry logic with exponential backoff and jitter.
type Retryer struct {
	config Config
	rng    *rand.Rand
	mu     sync.Mutex
}

func NewRetryer(config Config) *Retryer {
	if config.InitialDelay <= 0 {
		config.InitialDelay = 1 * time.Second
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.BackoffFactor <= 1.0 {
		config.BackoffFactor = 2.0
	}
	return &Retryer{
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func NewRetryerWithDefaults() *Retryer {
	return NewRetryer(DefaultConfig())
}

// Retry executes fn with retry. Rate-limit waits do not count against max_retries.
func (r *Retryer) Retry(ctx context.Context, fn RetryFunc) error {
	var lastErr error
	attempt := 0

	for {
		if r.config.MaxRetries >= 0 && attempt > r.config.MaxRetries {
			break
		}

		if r.config.ResilientMode && attempt > 0 {
			_ = r.waitForServerReadiness(ctx)
		}

		if err := fn(); err != nil {
			lastErr = err

			if !r.isRetryable(err) {
				return err
			}

			// Rate-limit: wait server-specified duration, don't count as retry attempt
			if httpErr, ok := err.(*HTTPError); ok && httpErr.IsRateLimited() {
				delay := httpErr.RetryAfter
				if delay <= 0 {
					delay = 60 * time.Second // minimum for bare 429
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}
				continue // don't increment attempt
			}

			if r.config.MaxRetries >= 0 && attempt >= r.config.MaxRetries {
				break
			}

			delay := r.calculateDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		} else {
			return nil
		}

		attempt++
	}

	return fmt.Errorf("retry failed after %d attempts: %w", attempt, lastErr)
}

func (r *Retryer) calculateDelay(attempt int) time.Duration {
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(attempt))
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}
	// 25% jitter per protocol spec
	if r.config.JitterEnabled {
		r.mu.Lock()
		jitter := r.rng.Float64() * 0.25
		r.mu.Unlock()
		delay = delay * (1 + jitter)
	}
	return time.Duration(delay)
}

func (r *Retryer) isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if httpErr, ok := err.(*HTTPError); ok {
		switch httpErr.StatusCode {
		case http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			// 400, 401, 413, 507 are NOT retryable
			return false
		}
	}
	// Network errors are retryable
	errStr := strings.ToLower(err.Error())
	for _, keyword := range []string{
		"connection refused", "timeout", "network error",
		"eof", "broken pipe", "connection reset", "no route to host",
	} {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}
	return false
}

func (r *Retryer) waitForServerReadiness(ctx context.Context) error {
	if !r.config.ResilientMode || r.config.HealthCheckURL == "" {
		return nil
	}
	client := &http.Client{Timeout: r.config.HealthCheckTimeout}
	backoff := r.config.InitialDelay

	for attempt := 0; attempt < r.config.HealthCheckRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, err := http.NewRequestWithContext(ctx, "GET", r.config.HealthCheckURL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		if attempt < r.config.HealthCheckRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff = time.Duration(float64(backoff) * r.config.BackoffFactor)
			if backoff > r.config.MaxDelay {
				backoff = r.config.MaxDelay
			}
		}
	}
	return fmt.Errorf("server not ready after %d health checks", r.config.HealthCheckRetries)
}

func (r *Retryer) SetHealthCheckURL(url string) {
	r.config.HealthCheckURL = url
}

func (r *Retryer) EnableResilientMode(enabled bool) {
	r.config.ResilientMode = enabled
}

func RetryWithDefaults(ctx context.Context, fn RetryFunc) error {
	return NewRetryerWithDefaults().Retry(ctx, fn)
}

func RetryWithConfig(ctx context.Context, config Config, fn RetryFunc) error {
	return NewRetryer(config).Retry(ctx, fn)
}
