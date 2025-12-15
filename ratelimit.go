package gollmx

import (
	"context"
	"sync"
	"time"
)

// RateLimiter controls the rate of API requests using token bucket algorithm
type RateLimiter struct {
	mu           sync.Mutex
	tokens       float64
	maxTokens    float64
	refillRate   float64 // tokens per second
	lastRefill   time.Time
	waitTimeout  time.Duration
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	RequestsPerMinute int           // Maximum requests per minute (0 = unlimited)
	BurstSize         int           // Maximum burst size (defaults to RPM/10 or 1)
	WaitTimeout       time.Duration // Maximum time to wait for a token (0 = no wait, return error)
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         6,
		WaitTimeout:       30 * time.Second,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	if config.RequestsPerMinute <= 0 {
		return nil // No rate limiting
	}

	burstSize := config.BurstSize
	if burstSize <= 0 {
		burstSize = config.RequestsPerMinute / 10
		if burstSize < 1 {
			burstSize = 1
		}
	}

	return &RateLimiter{
		tokens:      float64(burstSize),
		maxTokens:   float64(burstSize),
		refillRate:  float64(config.RequestsPerMinute) / 60.0, // per second
		lastRefill:  time.Now(),
		waitTimeout: config.WaitTimeout,
	}
}

// Acquire blocks until a token is available or context is cancelled
func (r *RateLimiter) Acquire(ctx context.Context) error {
	if r == nil {
		return nil // No rate limiting
	}

	// Create timeout context if configured
	if r.waitTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.waitTimeout)
		defer cancel()
	}

	for {
		r.mu.Lock()
		r.refill()

		if r.tokens >= 1 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}

		// Calculate wait time for next token
		waitTime := time.Duration((1 - r.tokens) / r.refillRate * float64(time.Second))
		r.mu.Unlock()

		// Wait for token or context cancellation
		select {
		case <-ctx.Done():
			return &APIError{
				Type:    ErrorTypeRateLimit,
				Message: "rate limit wait timeout",
			}
		case <-time.After(waitTime):
			// Try again
		}
	}
}

// TryAcquire attempts to acquire a token without blocking
func (r *RateLimiter) TryAcquire() bool {
	if r == nil {
		return true // No rate limiting
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}

// refill adds tokens based on elapsed time (must be called with lock held)
func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.refillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now
}

// Available returns the current number of available tokens
func (r *RateLimiter) Available() float64 {
	if r == nil {
		return -1 // Unlimited
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.refill()
	return r.tokens
}

// =============================================================================
// Rate Limited Client Wrapper
// =============================================================================

// RateLimitedClient wraps an LLM client with rate limiting
type RateLimitedClient struct {
	client  LLM
	limiter *RateLimiter
}

// NewRateLimitedClient wraps an LLM client with rate limiting
func NewRateLimitedClient(client LLM, rpm int) *RateLimitedClient {
	return NewRateLimitedClientWithConfig(client, &RateLimitConfig{
		RequestsPerMinute: rpm,
		BurstSize:         rpm / 10,
		WaitTimeout:       30 * time.Second,
	})
}

// NewRateLimitedClientWithConfig wraps an LLM client with custom rate limit configuration
func NewRateLimitedClientWithConfig(client LLM, config *RateLimitConfig) *RateLimitedClient {
	return &RateLimitedClient{
		client:  client,
		limiter: NewRateLimiter(config),
	}
}

// ID returns the provider identifier
func (c *RateLimitedClient) ID() string {
	return c.client.ID()
}

// Name returns the provider name
func (c *RateLimitedClient) Name() string {
	return c.client.Name()
}

// Version returns the client version
func (c *RateLimitedClient) Version() string {
	return c.client.Version()
}

// BaseURL returns the API base URL
func (c *RateLimitedClient) BaseURL() string {
	return c.client.BaseURL()
}

// Models returns available models
func (c *RateLimitedClient) Models() []Model {
	return c.client.Models()
}

// GetModel returns a specific model
func (c *RateLimitedClient) GetModel(id string) (*Model, error) {
	return c.client.GetModel(id)
}

// Chat performs a chat completion with rate limiting
func (c *RateLimitedClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if err := c.limiter.Acquire(ctx); err != nil {
		return nil, err
	}
	return c.client.Chat(ctx, req)
}

// ChatStream performs a streaming chat completion with rate limiting
func (c *RateLimitedClient) ChatStream(ctx context.Context, req *ChatRequest) (*StreamReader, error) {
	if err := c.limiter.Acquire(ctx); err != nil {
		return nil, err
	}
	return c.client.ChatStream(ctx, req)
}

// Complete performs a text completion with rate limiting
func (c *RateLimitedClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if err := c.limiter.Acquire(ctx); err != nil {
		return nil, err
	}
	return c.client.Complete(ctx, req)
}

// Embed generates embeddings with rate limiting
func (c *RateLimitedClient) Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error) {
	if err := c.limiter.Acquire(ctx); err != nil {
		return nil, err
	}
	return c.client.Embed(ctx, req)
}

// HasFeature checks if a feature is supported
func (c *RateLimitedClient) HasFeature(feature Feature) bool {
	return c.client.HasFeature(feature)
}

// Features returns all supported features
func (c *RateLimitedClient) Features() []Feature {
	return c.client.Features()
}

// SetOption sets a provider-specific option
func (c *RateLimitedClient) SetOption(key string, value interface{}) error {
	return c.client.SetOption(key, value)
}

// GetOption gets a provider-specific option
func (c *RateLimitedClient) GetOption(key string) (interface{}, bool) {
	return c.client.GetOption(key)
}

// Unwrap returns the underlying LLM client
func (c *RateLimitedClient) Unwrap() LLM {
	return c.client
}

// Limiter returns the rate limiter instance
func (c *RateLimitedClient) Limiter() *RateLimiter {
	return c.limiter
}

// Ensure RateLimitedClient implements LLM interface
var _ LLM = (*RateLimitedClient)(nil)
