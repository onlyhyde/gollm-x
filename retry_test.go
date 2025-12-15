package gollmx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", config.MaxRetries)
	}
	if config.InitialDelay != 1*time.Second {
		t.Errorf("expected InitialDelay 1s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay 30s, got %v", config.MaxDelay)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("expected Multiplier 2.0, got %f", config.Multiplier)
	}
	if len(config.RetryableTypes) == 0 {
		t.Error("expected at least one retryable type")
	}
}

func TestRetryerOptions(t *testing.T) {
	retryer := NewRetryer(
		WithRetryMaxRetries(5),
		WithRetryInitialDelay(2*time.Second),
		WithRetryMaxDelay(60*time.Second),
		WithRetryMultiplier(3.0),
		WithRetryJitter(0.2),
		WithRetryableTypes(ErrorTypeRateLimit, ErrorTypeServer),
	)

	if retryer.config.MaxRetries != 5 {
		t.Errorf("expected MaxRetries 5, got %d", retryer.config.MaxRetries)
	}
	if retryer.config.InitialDelay != 2*time.Second {
		t.Errorf("expected InitialDelay 2s, got %v", retryer.config.InitialDelay)
	}
	if retryer.config.MaxDelay != 60*time.Second {
		t.Errorf("expected MaxDelay 60s, got %v", retryer.config.MaxDelay)
	}
	if retryer.config.Multiplier != 3.0 {
		t.Errorf("expected Multiplier 3.0, got %f", retryer.config.Multiplier)
	}
	if retryer.config.Jitter != 0.2 {
		t.Errorf("expected Jitter 0.2, got %f", retryer.config.Jitter)
	}
	if len(retryer.config.RetryableTypes) != 2 {
		t.Errorf("expected 2 retryable types, got %d", len(retryer.config.RetryableTypes))
	}
}

func TestRetryerDoSuccess(t *testing.T) {
	retryer := NewRetryer(WithRetryMaxRetries(3))

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryerDoSuccessAfterRetries(t *testing.T) {
	retryer := NewRetryer(
		WithRetryMaxRetries(3),
		WithRetryInitialDelay(1*time.Millisecond),
	)

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return &APIError{Type: ErrorTypeRateLimit, Message: "rate limited", Retryable: true}
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryerDoMaxRetriesExceeded(t *testing.T) {
	retryer := NewRetryer(
		WithRetryMaxRetries(2),
		WithRetryInitialDelay(1*time.Millisecond),
	)

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		return &APIError{Type: ErrorTypeRateLimit, Message: "rate limited", Retryable: true}
	})

	if err == nil {
		t.Error("expected error after max retries")
	}
	if attempts != 3 { // Initial + 2 retries
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryerDoNonRetryableError(t *testing.T) {
	retryer := NewRetryer(
		WithRetryMaxRetries(3),
		WithRetryInitialDelay(1*time.Millisecond),
	)

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		return &APIError{Type: ErrorTypeAuth, Message: "unauthorized", Retryable: false}
	})

	if err == nil {
		t.Error("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetryerDoContextCancelled(t *testing.T) {
	retryer := NewRetryer(
		WithRetryMaxRetries(3),
		WithRetryInitialDelay(100*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retryer.Do(ctx, func() error {
		attempts++
		return &APIError{Type: ErrorTypeRateLimit, Message: "rate limited", Retryable: true}
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRetryerDoWithRetryAfter(t *testing.T) {
	retryer := NewRetryer(
		WithRetryMaxRetries(1),
		WithRetryInitialDelay(1*time.Second), // Should be overridden by RetryAfter
	)

	start := time.Now()
	attempts := 0
	_ = retryer.Do(context.Background(), func() error {
		attempts++
		if attempts == 1 {
			return &APIError{
				Type:       ErrorTypeRateLimit,
				Message:    "rate limited",
				Retryable:  true,
				RetryAfter: 10 * time.Millisecond,
			}
		}
		return nil
	})

	elapsed := time.Since(start)

	// Should wait closer to RetryAfter (10ms) rather than InitialDelay (1s)
	if elapsed > 500*time.Millisecond {
		t.Errorf("expected delay closer to 10ms, but took %v", elapsed)
	}
}

func TestDoWithResult(t *testing.T) {
	retryer := NewRetryer(
		WithRetryMaxRetries(2),
		WithRetryInitialDelay(1*time.Millisecond),
	)

	attempts := 0
	result, err := DoWithResult(context.Background(), retryer, func() (int, error) {
		attempts++
		if attempts < 2 {
			return 0, &APIError{Type: ErrorTypeServer, Message: "server error", Retryable: true}
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("expected result 42, got %d", result)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestCalculateDelay(t *testing.T) {
	retryer := NewRetryer(
		WithRetryInitialDelay(100*time.Millisecond),
		WithRetryMultiplier(2.0),
		WithRetryMaxDelay(1*time.Second),
		WithRetryJitter(0), // No jitter for predictable testing
	)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1 * time.Second}, // Capped at MaxDelay
		{5, 1 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		delay := retryer.calculateDelay(tt.attempt, nil)
		if delay != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, delay)
		}
	}
}

func TestShouldRetryByType(t *testing.T) {
	retryer := NewRetryer(
		WithRetryableTypes(ErrorTypeRateLimit, ErrorTypeServer),
	)

	tests := []struct {
		err      error
		expected bool
	}{
		{&APIError{Type: ErrorTypeRateLimit}, true},
		{&APIError{Type: ErrorTypeServer}, true},
		{&APIError{Type: ErrorTypeAuth}, false},
		{&APIError{Type: ErrorTypeInvalidRequest}, false},
		{&APIError{Type: ErrorTypeRateLimit, Retryable: true}, true},
		{errors.New("connection refused"), true}, // Network error
		{errors.New("random error"), false},
	}

	for i, tt := range tests {
		result := retryer.shouldRetry(tt.err)
		if result != tt.expected {
			t.Errorf("test %d: expected %v, got %v for error %v", i, tt.expected, result, tt.err)
		}
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{nil, false},
		{errors.New("connection refused"), true},
		{errors.New("Connection Reset by peer"), true},
		{errors.New("no such host"), true},
		{errors.New("network is unreachable"), true},
		{errors.New("i/o timeout"), true},
		{errors.New("EOF"), true},
		{errors.New("broken pipe"), true},
		{errors.New("random application error"), false},
		{errors.New("invalid input"), false},
	}

	for i, tt := range tests {
		result := isNetworkError(tt.err)
		if result != tt.expected {
			t.Errorf("test %d: expected %v, got %v for error '%v'", i, tt.expected, result, tt.err)
		}
	}
}

func TestRetryableClientInterface(t *testing.T) {
	// This test verifies that RetryableClient properly implements the LLM interface
	var _ LLM = (*RetryableClient)(nil)
}

func TestRetryerNoRetries(t *testing.T) {
	retryer := NewRetryer(WithRetryMaxRetries(0))

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		return &APIError{Type: ErrorTypeRateLimit, Message: "rate limited", Retryable: true}
	})

	if err == nil {
		t.Error("expected error with no retries")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt with no retries, got %d", attempts)
	}
}
