package gollmx

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig should not return nil")
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", cfg.Timeout)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("expected default max retries 3, got %d", cfg.MaxRetries)
	}

	if cfg.RetryDelay != time.Second {
		t.Errorf("expected default retry delay 1s, got %v", cfg.RetryDelay)
	}
}

func TestWithAPIKey(t *testing.T) {
	cfg := DefaultConfig()
	WithAPIKey("test-api-key")(cfg)

	if cfg.APIKey != "test-api-key" {
		t.Errorf("expected API key 'test-api-key', got '%s'", cfg.APIKey)
	}
}

func TestWithBaseURL(t *testing.T) {
	cfg := DefaultConfig()
	WithBaseURL("https://custom.api.com")(cfg)

	if cfg.BaseURL != "https://custom.api.com" {
		t.Errorf("expected base URL 'https://custom.api.com', got '%s'", cfg.BaseURL)
	}
}

func TestWithTimeout(t *testing.T) {
	cfg := DefaultConfig()
	WithTimeout(60 * time.Second)(cfg)

	if cfg.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", cfg.Timeout)
	}
}

func TestWithMaxRetries(t *testing.T) {
	cfg := DefaultConfig()
	WithMaxRetries(5)(cfg)

	if cfg.MaxRetries != 5 {
		t.Errorf("expected max retries 5, got %d", cfg.MaxRetries)
	}
}

func TestWithRetryDelay(t *testing.T) {
	cfg := DefaultConfig()
	WithRetryDelay(2 * time.Second)(cfg)

	if cfg.RetryDelay != 2*time.Second {
		t.Errorf("expected retry delay 2s, got %v", cfg.RetryDelay)
	}
}

func TestWithOrgID(t *testing.T) {
	cfg := DefaultConfig()
	WithOrgID("org-123")(cfg)

	if cfg.OrgID != "org-123" {
		t.Errorf("expected org ID 'org-123', got '%s'", cfg.OrgID)
	}
}

func TestWithProjectID(t *testing.T) {
	cfg := DefaultConfig()
	WithProjectID("proj-456")(cfg)

	if cfg.ProjectID != "proj-456" {
		t.Errorf("expected project ID 'proj-456', got '%s'", cfg.ProjectID)
	}
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 120 * time.Second}
	cfg := DefaultConfig()
	WithHTTPClient(customClient)(cfg)

	if cfg.HTTPClient != customClient {
		t.Error("HTTP client was not set correctly")
	}
}

func TestWithHeaders(t *testing.T) {
	headers := map[string]string{
		"X-Custom-Header": "custom-value",
	}
	cfg := DefaultConfig()
	WithHeaders(headers)(cfg)

	if cfg.Headers["X-Custom-Header"] != "custom-value" {
		t.Error("headers were not set correctly")
	}
}

func TestWithDebug(t *testing.T) {
	cfg := DefaultConfig()
	WithDebug(true)(cfg)

	if !cfg.Debug {
		t.Error("debug should be true")
	}
}

func TestWithDefaultModel(t *testing.T) {
	cfg := DefaultConfig()
	WithDefaultModel("gpt-4o")(cfg)

	if cfg.DefaultModel != "gpt-4o" {
		t.Errorf("expected default model 'gpt-4o', got '%s'", cfg.DefaultModel)
	}
}

func TestConfigApply(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Apply(
		WithAPIKey("key-123"),
		WithBaseURL("https://api.example.com"),
		WithTimeout(45*time.Second),
	)

	if cfg.APIKey != "key-123" {
		t.Errorf("expected API key 'key-123', got '%s'", cfg.APIKey)
	}

	if cfg.BaseURL != "https://api.example.com" {
		t.Errorf("expected base URL 'https://api.example.com', got '%s'", cfg.BaseURL)
	}

	if cfg.Timeout != 45*time.Second {
		t.Errorf("expected timeout 45s, got %v", cfg.Timeout)
	}
}

func TestConfigGetHTTPClient(t *testing.T) {
	// Test with nil HTTPClient (should create default)
	cfg := DefaultConfig()
	client := cfg.GetHTTPClient()

	if client == nil {
		t.Error("GetHTTPClient should not return nil")
	}

	// Test with custom HTTPClient
	customClient := &http.Client{Timeout: 120 * time.Second}
	cfg.HTTPClient = customClient
	client = cfg.GetHTTPClient()

	if client != customClient {
		t.Error("GetHTTPClient should return the custom client")
	}
}
