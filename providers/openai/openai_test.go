package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
		gollmx.WithTimeout(60*time.Second),
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
		if m.ID == "gpt-4o" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected to find gpt-4o model")
	}
}

func TestGetModel(t *testing.T) {
	client, _ := New()

	model, err := client.GetModel("gpt-4o")
	if err != nil {
		t.Fatalf("failed to get model: %v", err)
	}

	if model.ID != "gpt-4o" {
		t.Errorf("expected model ID 'gpt-4o', got '%s'", model.ID)
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

	// Should support all these features
	features := []gollmx.Feature{
		gollmx.FeatureChat,
		gollmx.FeatureCompletion,
		gollmx.FeatureEmbedding,
		gollmx.FeatureStreaming,
		gollmx.FeatureVision,
		gollmx.FeatureTools,
		gollmx.FeatureJSON,
		gollmx.FeatureSystemPrompt,
	}

	for _, f := range features {
		if !client.HasFeature(f) {
			t.Errorf("should support %s feature", f)
		}
	}
}

func TestFeatures(t *testing.T) {
	client, _ := New()
	features := client.Features()

	if len(features) == 0 {
		t.Error("expected at least one feature")
	}

	if len(features) != 8 {
		t.Errorf("expected 8 features, got %d", len(features))
	}
}

func TestSetGetOption(t *testing.T) {
	client, _ := New()
	openaiClient := client.(*Client)

	err := openaiClient.SetOption("custom_key", "custom_value")
	if err != nil {
		t.Fatalf("failed to set option: %v", err)
	}

	value, ok := openaiClient.GetOption("custom_key")
	if !ok {
		t.Error("option should exist")
	}

	if value != "custom_value" {
		t.Errorf("expected 'custom_value', got '%v'", value)
	}

	// Non-existent option
	_, ok = openaiClient.GetOption("non_existent")
	if ok {
		t.Error("non-existent option should not exist")
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path '/chat/completions', got '%s'", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("expected POST method, got '%s'", r.Method)
		}

		// Check headers
		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type application/json")
		}

		response := openAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
			Choices: []openAIChoice{
				{
					Index: 0,
					Message: openAIMessageResp{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: openAIUsage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
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
		Model: "gpt-4o-mini",
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

	if resp.Usage.CompletionTokens != 8 {
		t.Errorf("expected 8 completion tokens, got %d", resp.Usage.CompletionTokens)
	}
}

func TestChatWithSystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify system message is included
		if len(req.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("expected first message to be system, got %s", req.Messages[0].Role)
		}

		response := openAIChatResponse{
			ID:      "chatcmpl-123",
			Model:   req.Model,
			Choices: []openAIChoice{{Message: openAIMessageResp{Role: "assistant", Content: "Response"}, FinishReason: "stop"}},
			Usage:   openAIUsage{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL), gollmx.WithAPIKey("test"))

	_, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleSystem, Content: "You are helpful"},
			{Role: gollmx.RoleUser, Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
}

func TestChatWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify tools are included
		if len(req.Tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Function.Name != "get_weather" {
			t.Errorf("expected tool name 'get_weather', got '%s'", req.Tools[0].Function.Name)
		}

		response := openAIChatResponse{
			ID:    "chatcmpl-123",
			Model: req.Model,
			Choices: []openAIChoice{
				{
					Message: openAIMessageResp{
						Role: "assistant",
						ToolCalls: []openAIToolCall{
							{
								ID:   "call_123",
								Type: "function",
								Function: openAIFunctionCall{
									Name:      "get_weather",
									Arguments: `{"location": "Seoul"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
			Usage: openAIUsage{PromptTokens: 20, CompletionTokens: 10, TotalTokens: 30},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL), gollmx.WithAPIKey("test"))

	resp, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "What's the weather in Seoul?"},
		},
		Tools: []gollmx.Tool{
			{
				Type: "function",
				Function: gollmx.Function{
					Name:        "get_weather",
					Description: "Get weather for a location",
					Parameters:  json.RawMessage(`{"type": "object", "properties": {"location": {"type": "string"}}}`),
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	toolCalls := resp.GetToolCalls()
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
	}

	if toolCalls[0].Function.Name != "get_weather" {
		t.Errorf("expected function name 'get_weather', got '%s'", toolCalls[0].Function.Name)
	}
}

func TestChatError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorType  gollmx.ErrorType
	}{
		{"unauthorized", 401, gollmx.ErrorTypeAuth},
		{"rate_limit", 429, gollmx.ErrorTypeRateLimit},
		{"bad_request", 400, gollmx.ErrorTypeInvalidRequest},
		{"not_found", 404, gollmx.ErrorTypeModelNotFound},
		{"server_error", 500, gollmx.ErrorTypeServer},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				json.NewEncoder(w).Encode(openAIErrorResponse{
					Error: &openAIError{
						Message: "Test error",
						Type:    tc.name,
					},
				})
			}))
			defer server.Close()

			client, _ := New(gollmx.WithBaseURL(server.URL), gollmx.WithAPIKey("test"))

			_, err := client.Chat(context.Background(), &gollmx.ChatRequest{
				Model:    "gpt-4o-mini",
				Messages: []gollmx.Message{{Role: gollmx.RoleUser, Content: "Hello"}},
			})

			if err == nil {
				t.Error("expected error")
				return
			}

			apiErr, ok := err.(*gollmx.APIError)
			if !ok {
				t.Fatalf("expected APIError, got %T", err)
			}

			if apiErr.Type != tc.errorType {
				t.Errorf("expected error type %s, got %s", tc.errorType, apiErr.Type)
			}
		})
	}
}

func TestComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := openAIChatResponse{
			ID:    "chatcmpl-123",
			Model: "gpt-4o-mini",
			Choices: []openAIChoice{
				{Message: openAIMessageResp{Role: "assistant", Content: "Completed text"}, FinishReason: "stop"},
			},
			Usage: openAIUsage{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL), gollmx.WithAPIKey("test"))

	resp, err := client.Complete(context.Background(), &gollmx.CompletionRequest{
		Model:  "gpt-4o-mini",
		Prompt: "Complete this",
	})

	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	if resp.GetText() != "Completed text" {
		t.Errorf("unexpected text: %s", resp.GetText())
	}

	if resp.Provider != ProviderID {
		t.Errorf("expected provider '%s', got '%s'", ProviderID, resp.Provider)
	}
}

func TestEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("expected path '/embeddings', got '%s'", r.URL.Path)
		}

		response := openAIEmbedResponse{
			Object: "list",
			Data: []openAIEmbedData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5}},
			},
			Model: "text-embedding-3-small",
			Usage: openAIEmbedUsage{PromptTokens: 5, TotalTokens: 5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL), gollmx.WithAPIKey("test"))

	resp, err := client.Embed(context.Background(), &gollmx.EmbedRequest{
		Model: "text-embedding-3-small",
		Input: []string{"Hello world"},
	})

	if err != nil {
		t.Fatalf("embed failed: %v", err)
	}

	if len(resp.Embeddings) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(resp.Embeddings))
	}

	if len(resp.Embeddings[0].Vector) != 5 {
		t.Errorf("expected 5 dimensions, got %d", len(resp.Embeddings[0].Vector))
	}

	if resp.Provider != ProviderID {
		t.Errorf("expected provider '%s', got '%s'", ProviderID, resp.Provider)
	}
}

func TestEmbedMultipleInputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := openAIEmbedResponse{
			Object: "list",
			Data: []openAIEmbedData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2, 0.3}},
				{Object: "embedding", Index: 1, Embedding: []float64{0.4, 0.5, 0.6}},
			},
			Model: "text-embedding-3-small",
			Usage: openAIEmbedUsage{PromptTokens: 10, TotalTokens: 10},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL), gollmx.WithAPIKey("test"))

	resp, err := client.Embed(context.Background(), &gollmx.EmbedRequest{
		Model: "text-embedding-3-small",
		Input: []string{"Hello", "World"},
	})

	if err != nil {
		t.Fatalf("embed failed: %v", err)
	}

	if len(resp.Embeddings) != 2 {
		t.Fatalf("expected 2 embeddings, got %d", len(resp.Embeddings))
	}
}

func TestChatWithOrgAndProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check org and project headers
		if r.Header.Get("OpenAI-Organization") != "org-123" {
			t.Errorf("expected org header 'org-123', got '%s'", r.Header.Get("OpenAI-Organization"))
		}
		if r.Header.Get("OpenAI-Project") != "proj-456" {
			t.Errorf("expected project header 'proj-456', got '%s'", r.Header.Get("OpenAI-Project"))
		}

		response := openAIChatResponse{
			ID:      "chatcmpl-123",
			Model:   "gpt-4o-mini",
			Choices: []openAIChoice{{Message: openAIMessageResp{Role: "assistant", Content: "OK"}, FinishReason: "stop"}},
			Usage:   openAIUsage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(
		gollmx.WithBaseURL(server.URL),
		gollmx.WithAPIKey("test"),
		gollmx.WithOrgID("org-123"),
		gollmx.WithProjectID("proj-456"),
	)

	_, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model:    "gpt-4o-mini",
		Messages: []gollmx.Message{{Role: gollmx.RoleUser, Content: "Hi"}},
	})

	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
}

func TestDefaultModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Model != DefaultModel {
			t.Errorf("expected default model '%s', got '%s'", DefaultModel, req.Model)
		}

		response := openAIChatResponse{
			ID:      "chatcmpl-123",
			Model:   req.Model,
			Choices: []openAIChoice{{Message: openAIMessageResp{Role: "assistant", Content: "OK"}, FinishReason: "stop"}},
			Usage:   openAIUsage{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL), gollmx.WithAPIKey("test"))

	// Request without model should use default
	_, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Messages: []gollmx.Message{{Role: gollmx.RoleUser, Content: "Hi"}},
	})

	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
}

func TestVersion(t *testing.T) {
	client, _ := New()

	if client.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", client.Version())
	}
}
