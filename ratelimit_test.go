package gollmx

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(&RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         5,
	})

	if limiter == nil {
		t.Fatal("expected non-nil limiter")
	}

	if limiter.maxTokens != 5 {
		t.Errorf("expected maxTokens 5, got %f", limiter.maxTokens)
	}

	if limiter.refillRate != 1.0 { // 60 RPM = 1 per second
		t.Errorf("expected refillRate 1.0, got %f", limiter.refillRate)
	}
}

func TestNewRateLimiterNil(t *testing.T) {
	limiter := NewRateLimiter(&RateLimitConfig{
		RequestsPerMinute: 0,
	})

	if limiter != nil {
		t.Error("expected nil limiter for 0 RPM")
	}
}

func TestNewRateLimiterDefaultBurst(t *testing.T) {
	limiter := NewRateLimiter(&RateLimitConfig{
		RequestsPerMinute: 100,
		BurstSize:         0, // Should default to RPM/10 = 10
	})

	if limiter.maxTokens != 10 {
		t.Errorf("expected default burst size 10, got %f", limiter.maxTokens)
	}
}

func TestRateLimiterTryAcquire(t *testing.T) {
	limiter := NewRateLimiter(&RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         3,
	})

	// Should be able to acquire burst size tokens immediately
	for i := 0; i < 3; i++ {
		if !limiter.TryAcquire() {
			t.Errorf("expected to acquire token %d", i+1)
		}
	}

	// Next acquire should fail (no tokens left)
	if limiter.TryAcquire() {
		t.Error("expected TryAcquire to fail when no tokens available")
	}
}

func TestRateLimiterAcquire(t *testing.T) {
	limiter := NewRateLimiter(&RateLimitConfig{
		RequestsPerMinute: 600, // 10 per second
		BurstSize:         2,
		WaitTimeout:       5 * time.Second,
	})

	ctx := context.Background()

	// Acquire burst tokens
	for i := 0; i < 2; i++ {
		if err := limiter.Acquire(ctx); err != nil {
			t.Errorf("expected no error on acquire %d, got %v", i+1, err)
		}
	}

	// Next acquire should wait and succeed
	start := time.Now()
	if err := limiter.Acquire(ctx); err != nil {
		t.Errorf("expected no error after wait, got %v", err)
	}
	elapsed := time.Since(start)

	// Should have waited approximately 100ms (1 token / 10 per second)
	if elapsed < 50*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("expected wait around 100ms, got %v", elapsed)
	}
}

func TestRateLimiterAcquireTimeout(t *testing.T) {
	limiter := NewRateLimiter(&RateLimitConfig{
		RequestsPerMinute: 60, // 1 per second
		BurstSize:         1,
		WaitTimeout:       50 * time.Millisecond,
	})

	ctx := context.Background()

	// Use the one token
	limiter.TryAcquire()

	// Next acquire should timeout
	start := time.Now()
	err := limiter.Acquire(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected timeout error")
	}

	if elapsed < 40*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Errorf("expected wait around 50ms timeout, got %v", elapsed)
	}
}

func TestRateLimiterAcquireContextCancel(t *testing.T) {
	limiter := NewRateLimiter(&RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         1,
		WaitTimeout:       5 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())

	// Use the one token
	limiter.TryAcquire()

	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := limiter.Acquire(ctx)
	if err == nil {
		t.Error("expected context cancel error")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	limiter := NewRateLimiter(&RateLimitConfig{
		RequestsPerMinute: 600, // 10 per second
		BurstSize:         5,
	})

	// Use all tokens
	for i := 0; i < 5; i++ {
		limiter.TryAcquire()
	}

	// Wait for refill
	time.Sleep(200 * time.Millisecond) // Should refill ~2 tokens

	available := limiter.Available()
	if available < 1.5 || available > 2.5 {
		t.Errorf("expected ~2 tokens after 200ms, got %f", available)
	}
}

func TestRateLimiterConcurrent(t *testing.T) {
	limiter := NewRateLimiter(&RateLimitConfig{
		RequestsPerMinute: 6000, // 100 per second for faster test
		BurstSize:         10,
		WaitTimeout:       5 * time.Second,
	})

	ctx := context.Background()
	var acquired int64
	var wg sync.WaitGroup

	// Start 20 goroutines trying to acquire
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := limiter.Acquire(ctx); err == nil {
				atomic.AddInt64(&acquired, 1)
			}
		}()
	}

	wg.Wait()

	if acquired != 20 {
		t.Errorf("expected all 20 acquires to succeed, got %d", acquired)
	}
}

func TestRateLimiterNil(t *testing.T) {
	var limiter *RateLimiter

	// All operations should succeed on nil limiter
	if err := limiter.Acquire(context.Background()); err != nil {
		t.Errorf("expected nil limiter Acquire to succeed, got %v", err)
	}

	if !limiter.TryAcquire() {
		t.Error("expected nil limiter TryAcquire to return true")
	}

	if limiter.Available() != -1 {
		t.Errorf("expected nil limiter Available to return -1, got %f", limiter.Available())
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.RequestsPerMinute != 60 {
		t.Errorf("expected RPM 60, got %d", config.RequestsPerMinute)
	}

	if config.BurstSize != 6 {
		t.Errorf("expected BurstSize 6, got %d", config.BurstSize)
	}

	if config.WaitTimeout != 30*time.Second {
		t.Errorf("expected WaitTimeout 30s, got %v", config.WaitTimeout)
	}
}

func TestNewRateLimitedClient(t *testing.T) {
	// This test verifies that RateLimitedClient implements LLM interface
	var _ LLM = (*RateLimitedClient)(nil)
}

func TestRateLimitedClientUnwrap(t *testing.T) {
	// Create a mock client to wrap (mockLLM defined in llm_test.go)
	mockClient := &mockLLM{}
	rateLimited := NewRateLimitedClient(mockClient, 60)

	if rateLimited.Unwrap() != mockClient {
		t.Error("Unwrap should return the original client")
	}

	if rateLimited.Limiter() == nil {
		t.Error("Limiter should not be nil")
	}
}
