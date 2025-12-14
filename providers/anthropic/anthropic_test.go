package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gollmx "github.com/onlyhyde/gollm-x"
)

func TestNew(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if client.ID() != ProviderID {
		t.Errorf("expected ID '%s', got '%s'", ProviderID, client.ID())
	}

	if client.Name() != ProviderName {
		t.Errorf("expected name '%s', got '%s'", ProviderName, client.Name())
	}

	if client.BaseURL() != DefaultBaseURL {
		t.Errorf("expected base URL '%s', got '%s'", DefaultBaseURL, client.BaseURL())
	}
}

func TestNewWithOptions(t *testing.T) {
	client, err := New(
		gollmx.WithBaseURL("https://custom.api.com"),
		gollmx.WithAPIKey("test-key"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if client.BaseURL() != "https://custom.api.com" {
		t.Errorf("expected custom base URL, got '%s'", client.BaseURL())
	}
}

func TestModels(t *testing.T) {
	client, _ := New()
	models := client.Models()

	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Check for specific model
	found := false
	for _, m := range models {
		if m.ID == "claude-3-5-sonnet-20241022" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected to find claude-3-5-sonnet-20241022 model")
	}
}

func TestGetModel(t *testing.T) {
	client, _ := New()

	model, err := client.GetModel("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("failed to get model: %v", err)
	}

	if model.ID != "claude-3-5-sonnet-20241022" {
		t.Errorf("expected model ID 'claude-3-5-sonnet-20241022', got '%s'", model.ID)
	}
}

func TestGetModelNotFound(t *testing.T) {
	client, _ := New()

	_, err := client.GetModel("non-existent-model")
	if err == nil {
		t.Error("expected error for non-existent model")
	}
}

func TestHasFeature(t *testing.T) {
	client, _ := New()

	// Should support these features
	if !client.HasFeature(gollmx.FeatureChat) {
		t.Error("should support chat feature")
	}
	if !client.HasFeature(gollmx.FeatureStreaming) {
		t.Error("should support streaming feature")
	}
	if !client.HasFeature(gollmx.FeatureTools) {
		t.Error("should support tools feature")
	}
	if !client.HasFeature(gollmx.FeatureVision) {
		t.Error("should support vision feature")
	}

	// Should not support embedding
	if client.HasFeature(gollmx.FeatureEmbedding) {
		t.Error("should not support embedding feature")
	}
}

func TestFeatures(t *testing.T) {
	client, _ := New()
	features := client.Features()

	if len(features) == 0 {
		t.Error("expected at least one feature")
	}
}

func TestSetGetOption(t *testing.T) {
	client, _ := New()
	anthropicClient := client.(*Client)

	err := anthropicClient.SetOption("custom_key", "custom_value")
	if err != nil {
		t.Fatalf("failed to set option: %v", err)
	}

	value, ok := anthropicClient.GetOption("custom_key")
	if !ok {
		t.Error("option should exist")
	}

	if value != "custom_value" {
		t.Errorf("expected 'custom_value', got '%v'", value)
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Errorf("expected path '/messages', got '%s'", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("expected POST method, got '%s'", r.Method)
		}

		// Check headers
		if r.Header.Get("x-api-key") == "" {
			t.Error("expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") != APIVersion {
			t.Errorf("expected anthropic-version '%s', got '%s'", APIVersion, r.Header.Get("anthropic-version"))
		}

		response := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContentBlock{
				{Type: "text", Text: "Hello! How can I help you today?"},
			},
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  10,
				OutputTokens: 8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(
		gollmx.WithBaseURL(server.URL),
		gollmx.WithAPIKey("test-key"),
	)

	resp, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "Hello!"},
		},
	})

	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.GetContent() != "Hello! How can I help you today?" {
		t.Errorf("unexpected content: %s", resp.GetContent())
	}

	if resp.Provider != ProviderID {
		t.Errorf("expected provider '%s', got '%s'", ProviderID, resp.Provider)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}
}

func TestChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(anthropicErrorResponse{
			Type: "error",
			Error: &anthropicError{
				Type:    "authentication_error",
				Message: "Invalid API key",
			},
		})
	}))
	defer server.Close()

	client, _ := New(
		gollmx.WithBaseURL(server.URL),
		gollmx.WithAPIKey("invalid-key"),
	)

	_, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "Hello!"},
		},
	})

	if err == nil {
		t.Error("expected error for unauthorized request")
	}

	apiErr, ok := err.(*gollmx.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Type != gollmx.ErrorTypeAuth {
		t.Errorf("expected auth error type, got %s", apiErr.Type)
	}
}

func TestEmbedNotSupported(t *testing.T) {
	client, _ := New()

	_, err := client.Embed(context.Background(), &gollmx.EmbedRequest{
		Model: "test",
		Input: []string{"test"},
	})

	if err == nil {
		t.Error("expected error for embed request")
	}
}

func TestChatWithSystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.System == "" {
			t.Error("expected system prompt to be set")
		}

		response := anthropicResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       "assistant",
			Content:    []anthropicContentBlock{{Type: "text", Text: "Response"}},
			Model:      req.Model,
			StopReason: "end_turn",
			Usage:      anthropicUsage{InputTokens: 5, OutputTokens: 3},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL), gollmx.WithAPIKey("test"))

	_, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleSystem, Content: "You are helpful"},
			{Role: gollmx.RoleUser, Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
}
