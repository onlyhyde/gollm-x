package gollmx

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retry attempts (0 = no retries)
	InitialDelay   time.Duration // Initial delay before first retry
	MaxDelay       time.Duration // Maximum delay between retries
	Multiplier     float64       // Multiplier for exponential backoff
	Jitter         float64       // Random jitter factor (0-1)
	RetryableTypes []ErrorType   // Error types that should be retried
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
		RetryableTypes: []ErrorType{
			ErrorTypeRateLimit,
			ErrorTypeServer,
			ErrorTypeNetwork,
			ErrorTypeTimeout,
		},
	}
}

// RetryOption is a function that modifies RetryConfig
type RetryOption func(*RetryConfig)

// WithRetryMaxRetries sets the maximum number of retries
func WithRetryMaxRetries(n int) RetryOption {
	return func(c *RetryConfig) {
		c.MaxRetries = n
	}
}

// WithRetryInitialDelay sets the initial delay
func WithRetryInitialDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) {
		c.InitialDelay = d
	}
}

// WithRetryMaxDelay sets the maximum delay
func WithRetryMaxDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) {
		c.MaxDelay = d
	}
}

// WithRetryMultiplier sets the backoff multiplier
func WithRetryMultiplier(m float64) RetryOption {
	return func(c *RetryConfig) {
		c.Multiplier = m
	}
}

// WithRetryJitter sets the jitter factor
func WithRetryJitter(j float64) RetryOption {
	return func(c *RetryConfig) {
		c.Jitter = j
	}
}

// WithRetryableTypes sets the retryable error types
func WithRetryableTypes(types ...ErrorType) RetryOption {
	return func(c *RetryConfig) {
		c.RetryableTypes = types
	}
}

// Retryer handles retry logic with exponential backoff
type Retryer struct {
	config *RetryConfig
	rng    *rand.Rand
}

// NewRetryer creates a new Retryer with the given options
func NewRetryer(opts ...RetryOption) *Retryer {
	config := DefaultRetryConfig()
	for _, opt := range opts {
		opt(config)
	}
	return &Retryer{
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Do executes the given function with retry logic
func (r *Retryer) Do(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check context before attempting
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !r.shouldRetry(err) {
			return err
		}

		// Check if we have more retries
		if attempt >= r.config.MaxRetries {
			return err
		}

		// Calculate delay with exponential backoff and jitter
		delay := r.calculateDelay(attempt, err)

		// Wait or return if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// DoWithResult executes a function that returns a value and error with retry logic
func DoWithResult[T any](ctx context.Context, r *Retryer, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check context before attempting
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		// Execute the function
		res, err := fn()
		if err == nil {
			return res, nil
		}

		lastErr = err

		// Check if we should retry
		if !r.shouldRetry(err) {
			return result, err
		}

		// Check if we have more retries
		if attempt >= r.config.MaxRetries {
			return result, err
		}

		// Calculate delay with exponential backoff and jitter
		delay := r.calculateDelay(attempt, err)

		// Wait or return if context is cancelled
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return result, lastErr
}

// shouldRetry determines if an error should be retried
func (r *Retryer) shouldRetry(err error) bool {
	// Check if it's an APIError with Retryable flag
	if apiErr, ok := err.(*APIError); ok {
		if apiErr.Retryable {
			return true
		}
		// Check against configured retryable types
		for _, t := range r.config.RetryableTypes {
			if apiErr.Type == t {
				return true
			}
		}
		return false
	}

	// For non-API errors, check if it looks like a network error
	return isNetworkError(err)
}

// calculateDelay calculates the delay for the next retry attempt
func (r *Retryer) calculateDelay(attempt int, err error) time.Duration {
	// Check if the error specifies a retry-after duration
	if apiErr, ok := err.(*APIError); ok && apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter
	}

	// Calculate exponential backoff
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempt))

	// Apply maximum cap
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Apply jitter
	if r.config.Jitter > 0 {
		jitterRange := delay * r.config.Jitter
		delay += (r.rng.Float64()*2 - 1) * jitterRange
	}

	return time.Duration(delay)
}

// isNetworkError checks if an error appears to be a network-related error
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common network error patterns in error message
	errStr := err.Error()
	networkPatterns := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"EOF",
		"broken pipe",
		"connection timed out",
	}

	for _, pattern := range networkPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if s contains substr (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldASCII(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// =============================================================================
// Retry Wrapper for LLM Client
// =============================================================================

// RetryableClient wraps an LLM client with automatic retry logic
type RetryableClient struct {
	client  LLM
	retryer *Retryer
}

// WithRetry wraps an LLM client with retry logic
func WithRetry(client LLM, opts ...RetryOption) *RetryableClient {
	return &RetryableClient{
		client:  client,
		retryer: NewRetryer(opts...),
	}
}

// ID returns the provider identifier
func (c *RetryableClient) ID() string {
	return c.client.ID()
}

// Name returns the provider name
func (c *RetryableClient) Name() string {
	return c.client.Name()
}

// Version returns the client version
func (c *RetryableClient) Version() string {
	return c.client.Version()
}

// BaseURL returns the API base URL
func (c *RetryableClient) BaseURL() string {
	return c.client.BaseURL()
}

// Models returns available models
func (c *RetryableClient) Models() []Model {
	return c.client.Models()
}

// GetModel returns a specific model
func (c *RetryableClient) GetModel(id string) (*Model, error) {
	return c.client.GetModel(id)
}

// Chat performs a chat completion with retry
func (c *RetryableClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return DoWithResult(ctx, c.retryer, func() (*ChatResponse, error) {
		return c.client.Chat(ctx, req)
	})
}

// ChatStream performs a streaming chat completion (no retry for streams)
func (c *RetryableClient) ChatStream(ctx context.Context, req *ChatRequest) (*StreamReader, error) {
	// Streaming doesn't support retry as it's a continuous connection
	return c.client.ChatStream(ctx, req)
}

// Complete performs a text completion with retry
func (c *RetryableClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return DoWithResult(ctx, c.retryer, func() (*CompletionResponse, error) {
		return c.client.Complete(ctx, req)
	})
}

// Embed generates embeddings with retry
func (c *RetryableClient) Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error) {
	return DoWithResult(ctx, c.retryer, func() (*EmbedResponse, error) {
		return c.client.Embed(ctx, req)
	})
}

// HasFeature checks if a feature is supported
func (c *RetryableClient) HasFeature(feature Feature) bool {
	return c.client.HasFeature(feature)
}

// Features returns all supported features
func (c *RetryableClient) Features() []Feature {
	return c.client.Features()
}

// SetOption sets a provider-specific option
func (c *RetryableClient) SetOption(key string, value interface{}) error {
	return c.client.SetOption(key, value)
}

// GetOption gets a provider-specific option
func (c *RetryableClient) GetOption(key string) (interface{}, bool) {
	return c.client.GetOption(key)
}

// Unwrap returns the underlying LLM client
func (c *RetryableClient) Unwrap() LLM {
	return c.client
}

// Ensure RetryableClient implements LLM interface
var _ LLM = (*RetryableClient)(nil)
