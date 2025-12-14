package ollama

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
		gollmx.WithBaseURL("http://custom:11434"),
		gollmx.WithTimeout(60*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if client.BaseURL() != "http://custom:11434" {
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
		if m.ID == "llama3.2" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected to find llama3.2 model")
	}
}

func TestGetModel(t *testing.T) {
	client, _ := New()

	model, err := client.GetModel("llama3.2")
	if err != nil {
		t.Fatalf("failed to get model: %v", err)
	}

	if model.ID != "llama3.2" {
		t.Errorf("expected model ID 'llama3.2', got '%s'", model.ID)
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
	if !client.HasFeature(gollmx.FeatureEmbedding) {
		t.Error("should support embedding feature")
	}

	// Should not support tools
	if client.HasFeature(gollmx.FeatureTools) {
		t.Error("should not support tools feature")
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
	ollamaClient := client.(*Client)

	err := ollamaClient.SetOption("custom_key", "custom_value")
	if err != nil {
		t.Fatalf("failed to set option: %v", err)
	}

	value, ok := ollamaClient.GetOption("custom_key")
	if !ok {
		t.Error("option should exist")
	}

	if value != "custom_value" {
		t.Errorf("expected 'custom_value', got '%v'", value)
	}
}

func TestChat(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected path '/api/chat', got '%s'", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("expected POST method, got '%s'", r.Method)
		}

		response := ChatResponse{
			Model:     "llama3.2",
			CreatedAt: time.Now(),
			Message: Message{
				Role:    "assistant",
				Content: "Hello! How can I help you today?",
			},
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       8,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL))

	resp, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model: "llama3.2",
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
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL))

	_, err := client.Chat(context.Background(), &gollmx.ChatRequest{
		Model: "llama3.2",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "Hello!"},
		},
	})

	if err == nil {
		t.Error("expected error for server error")
	}
}

func TestComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ChatResponse{
			Model:     "llama3.2",
			CreatedAt: time.Now(),
			Message: Message{
				Role:    "assistant",
				Content: "Completed text",
			},
			Done: true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL))

	resp, err := client.Complete(context.Background(), &gollmx.CompletionRequest{
		Model:  "llama3.2",
		Prompt: "Complete this",
	})

	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	if resp.GetText() != "Completed text" {
		t.Errorf("unexpected text: %s", resp.GetText())
	}
}

func TestEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("expected path '/api/embeddings', got '%s'", r.URL.Path)
		}

		response := EmbedResponse{
			Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, _ := New(gollmx.WithBaseURL(server.URL))

	resp, err := client.Embed(context.Background(), &gollmx.EmbedRequest{
		Model: "nomic-embed-text",
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

func TestBuildChatRequest(t *testing.T) {
	client, _ := New()
	ollamaClient := client.(*Client)

	temp := 0.7
	topP := 0.9

	req := &gollmx.ChatRequest{
		Model: "llama3.2",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleSystem, Content: "You are helpful"},
			{Role: gollmx.RoleUser, Content: "Hello"},
		},
		Temperature: &temp,
		TopP:        &topP,
		MaxTokens:   100,
		Stop:        []string{"END"},
	}

	ollamaReq := ollamaClient.buildChatRequest(req)

	if ollamaReq.Model != "llama3.2" {
		t.Errorf("expected model 'llama3.2', got '%s'", ollamaReq.Model)
	}

	if len(ollamaReq.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(ollamaReq.Messages))
	}

	if ollamaReq.Options["temperature"] != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", ollamaReq.Options["temperature"])
	}

	if ollamaReq.Options["num_predict"] != 100 {
		t.Errorf("expected num_predict 100, got %v", ollamaReq.Options["num_predict"])
	}
}
