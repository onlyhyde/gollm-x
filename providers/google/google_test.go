package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gollmx "github.com/onlyhyde/gollm-x"
)

func TestNew(t *testing.T) {
	client, err := NewClient(gollmx.WithAPIKey("test-key"))
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
	client, err := NewClient(
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
	client, _ := NewClient(gollmx.WithAPIKey("test-key"))
	models := client.Models()

	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Check for specific model
	found := false
	for _, m := range models {
		if m.ID == "gemini-1.5-pro" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected to find gemini-1.5-pro model")
	}
}

func TestGetModel(t *testing.T) {
	client, _ := NewClient(gollmx.WithAPIKey("test-key"))

	model, err := client.GetModel("gemini-1.5-pro")
	if err != nil {
		t.Fatalf("failed to get model: %v", err)
	}

	if model.ID != "gemini-1.5-pro" {
		t.Errorf("expected model ID 'gemini-1.5-pro', got '%s'", model.ID)
	}
}

func TestGetModelNotFound(t *testing.T) {
	client, _ := NewClient(gollmx.WithAPIKey("test-key"))

	_, err := client.GetModel("non-existent-model")
	if err == nil {
		t.Error("expected error for non-existent model")
	}
}

func TestHasFeature(t *testing.T) {
	client, _ := NewClient(gollmx.WithAPIKey("test-key"))

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
	if !client.HasFeature(gollmx.FeatureEmbedding) {
		t.Error("should support embedding feature")
	}
}

func TestFeatures(t *testing.T) {
	client, _ := NewClient(gollmx.WithAPIKey("test-key"))
	features := client.Features()

	if len(features) == 0 {
		t.Error("expected at least one feature")
	}
}

func TestSetGetOption(t *testing.T) {
	client, _ := NewClient(gollmx.WithAPIKey("test-key"))
	googleClient := client.(*Client)

	err := googleClient.SetOption("custom_key", "custom_value")
	if err != nil {
		t.Fatalf("failed to set option: %v", err)
	}

	value, ok := googleClient.GetOption("custom_key")
	if !ok {
		t.Error("option should exist")
	}

	if value != "custom_value" {
		t.Errorf("expected 'custom_value', got '%v'", value)
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got '%s'", r.Method)
		}

		response := geminiGenerateResponse{
			Candidates: []geminiCandidate{
				{
					Content: &geminiContent{
						Role: "model",
						Parts: []geminiPart{
							{Text: "Hello! How can I help you today?"},
						},
					},
					FinishReason: "STOP",
					Index:        0,
				},
			},
			UsageMetadata: &geminiUsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 8,
				TotalTokenCount:      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := NewClient(
		gollmx.WithBaseURL(server.URL),
		gollmx.WithAPIKey("test-key"),
	)

	resp, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model: "gemini-1.5-pro",
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
		json.NewEncoder(w).Encode(geminiErrorResponse{
			Error: &geminiError{
				Code:    401,
				Message: "API key not valid",
				Status:  "UNAUTHENTICATED",
			},
		})
	}))
	defer server.Close()

	client, _ := NewClient(
		gollmx.WithBaseURL(server.URL),
		gollmx.WithAPIKey("invalid-key"),
	)

	_, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "Hello!"},
		},
	})

	if err == nil {
		t.Error("expected error for unauthorized request")
	}
}

func TestEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := geminiBatchEmbedResponse{
			Embeddings: []geminiEmbedding{
				{Values: []float64{0.1, 0.2, 0.3, 0.4, 0.5}},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := NewClient(
		gollmx.WithBaseURL(server.URL),
		gollmx.WithAPIKey("test-key"),
	)

	resp, err := client.Embed(context.Background(), &gollmx.EmbedRequest{
		Model: "text-embedding-004",
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
}

func TestComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := geminiGenerateResponse{
			Candidates: []geminiCandidate{
				{
					Content: &geminiContent{
						Role:  "model",
						Parts: []geminiPart{{Text: "Completed text"}},
					},
					FinishReason: "STOP",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := NewClient(
		gollmx.WithBaseURL(server.URL),
		gollmx.WithAPIKey("test-key"),
	)

	resp, err := client.Complete(context.Background(), &gollmx.CompletionRequest{
		Model:  "gemini-1.5-pro",
		Prompt: "Complete this",
	})

	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	if resp.GetText() != "Completed text" {
		t.Errorf("unexpected text: %s", resp.GetText())
	}
}
