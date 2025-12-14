package gollmx

import (
	"net/http"
	"time"
)

// Config holds the configuration for an LLM client
type Config struct {
	APIKey      string
	BaseURL     string
	OrgID       string            // Organization ID (for OpenAI)
	ProjectID   string            // Project ID
	HTTPClient  *http.Client
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	Headers     map[string]string // Custom headers
	Debug       bool

	// Rate limiting
	RateLimit   int           // Requests per minute (0 = no limit)

	// Default model
	DefaultModel string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
		Headers:    make(map[string]string),
	}
}

// Option is a function that modifies Config
type Option func(*Config)

// WithAPIKey sets the API key
func WithAPIKey(key string) Option {
	return func(c *Config) {
		c.APIKey = key
	}
}

// WithBaseURL sets a custom base URL
func WithBaseURL(url string) Option {
	return func(c *Config) {
		c.BaseURL = url
	}
}

// WithOrgID sets the organization ID
func WithOrgID(orgID string) Option {
	return func(c *Config) {
		c.OrgID = orgID
	}
}

// WithProjectID sets the project ID
func WithProjectID(projectID string) Option {
	return func(c *Config) {
		c.ProjectID = projectID
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = client
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(n int) Option {
	return func(c *Config) {
		c.MaxRetries = n
	}
}

// WithRetryDelay sets the delay between retries
func WithRetryDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.RetryDelay = delay
	}
}

// WithHeaders sets custom HTTP headers
func WithHeaders(headers map[string]string) Option {
	return func(c *Config) {
		for k, v := range headers {
			c.Headers[k] = v
		}
	}
}

// WithHeader adds a single custom header
func WithHeader(key, value string) Option {
	return func(c *Config) {
		c.Headers[key] = value
	}
}

// WithDebug enables debug mode
func WithDebug(debug bool) Option {
	return func(c *Config) {
		c.Debug = debug
	}
}

// WithRateLimit sets the rate limit (requests per minute)
func WithRateLimit(rpm int) Option {
	return func(c *Config) {
		c.RateLimit = rpm
	}
}

// WithDefaultModel sets the default model to use
func WithDefaultModel(model string) Option {
	return func(c *Config) {
		c.DefaultModel = model
	}
}

// Apply applies all options to the config
func (c *Config) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return NewAPIError(ErrorTypeAuth, "", "API key is required")
	}
	return nil
}

// GetHTTPClient returns the HTTP client, creating a default one if needed
func (c *Config) GetHTTPClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{
		Timeout: c.Timeout,
	}
}
