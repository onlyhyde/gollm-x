package gollmx

import (
	"testing"
)

func TestModelSupportsFeature(t *testing.T) {
	model := Model{
		ID:       "test-model",
		Features: []Feature{FeatureChat, FeatureStreaming, FeatureVision},
	}

	if !model.SupportsFeature(FeatureChat) {
		t.Error("model should support chat feature")
	}

	if !model.SupportsFeature(FeatureStreaming) {
		t.Error("model should support streaming feature")
	}

	if model.SupportsFeature(FeatureTools) {
		t.Error("model should not support tools feature")
	}
}

func TestChatResponseGetContent(t *testing.T) {
	resp := &ChatResponse{
		Choices: []Choice{
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "Hello, world!",
				},
			},
		},
	}

	if resp.GetContent() != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got '%s'", resp.GetContent())
	}

	// Test empty choices
	emptyResp := &ChatResponse{Choices: []Choice{}}
	if emptyResp.GetContent() != "" {
		t.Error("expected empty string for empty choices")
	}
}

func TestChatResponseGetToolCalls(t *testing.T) {
	toolCalls := []ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: FunctionCall{
				Name:      "get_weather",
				Arguments: `{"location": "Seoul"}`,
			},
		},
	}

	resp := &ChatResponse{
		Choices: []Choice{
			{
				Message: Message{
					Role:      RoleAssistant,
					ToolCalls: toolCalls,
				},
			},
		},
	}

	gotToolCalls := resp.GetToolCalls()
	if len(gotToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(gotToolCalls))
	}

	if gotToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("expected function name 'get_weather', got '%s'", gotToolCalls[0].Function.Name)
	}

	// Test empty choices
	emptyResp := &ChatResponse{Choices: []Choice{}}
	if emptyResp.GetToolCalls() != nil {
		t.Error("expected nil for empty choices")
	}
}

func TestCompletionResponseGetText(t *testing.T) {
	resp := &CompletionResponse{
		Choices: []CompletionChoice{
			{
				Text: "Generated text",
			},
		},
	}

	if resp.GetText() != "Generated text" {
		t.Errorf("expected 'Generated text', got '%s'", resp.GetText())
	}

	// Test empty choices
	emptyResp := &CompletionResponse{Choices: []CompletionChoice{}}
	if emptyResp.GetText() != "" {
		t.Error("expected empty string for empty choices")
	}
}

func TestAPIError(t *testing.T) {
	err := NewAPIError(ErrorTypeRateLimit, "test-provider", "rate limit exceeded")

	if err.Type != ErrorTypeRateLimit {
		t.Errorf("expected type RateLimit, got %s", err.Type)
	}

	if err.Provider != "test-provider" {
		t.Errorf("expected provider 'test-provider', got '%s'", err.Provider)
	}

	if err.Error() != "rate limit exceeded" {
		t.Errorf("expected message 'rate limit exceeded', got '%s'", err.Error())
	}
}

func TestTextContent(t *testing.T) {
	content := TextContent("Hello")

	if content.Type != "text" {
		t.Errorf("expected type 'text', got '%s'", content.Type)
	}

	if content.Text != "Hello" {
		t.Errorf("expected text 'Hello', got '%s'", content.Text)
	}
}

func TestImageURLContent(t *testing.T) {
	content := ImageURLContent("https://example.com/image.png", "high")

	if content.Type != "image_url" {
		t.Errorf("expected type 'image_url', got '%s'", content.Type)
	}

	if content.ImageURL == nil {
		t.Fatal("ImageURL should not be nil")
	}

	if content.ImageURL.URL != "https://example.com/image.png" {
		t.Errorf("expected URL 'https://example.com/image.png', got '%s'", content.ImageURL.URL)
	}

	if content.ImageURL.Detail != "high" {
		t.Errorf("expected detail 'high', got '%s'", content.ImageURL.Detail)
	}
}

func TestRoleConstants(t *testing.T) {
	if RoleSystem != "system" {
		t.Errorf("expected RoleSystem 'system', got '%s'", RoleSystem)
	}
	if RoleUser != "user" {
		t.Errorf("expected RoleUser 'user', got '%s'", RoleUser)
	}
	if RoleAssistant != "assistant" {
		t.Errorf("expected RoleAssistant 'assistant', got '%s'", RoleAssistant)
	}
	if RoleTool != "tool" {
		t.Errorf("expected RoleTool 'tool', got '%s'", RoleTool)
	}
}

func TestFeatureConstants(t *testing.T) {
	features := []Feature{
		FeatureChat,
		FeatureCompletion,
		FeatureEmbedding,
		FeatureStreaming,
		FeatureVision,
		FeatureTools,
		FeatureJSON,
		FeatureSystemPrompt,
	}

	for _, f := range features {
		if f == "" {
			t.Error("feature should not be empty")
		}
	}
}

func TestErrorTypes(t *testing.T) {
	errorTypes := []ErrorType{
		ErrorTypeAuth,
		ErrorTypeRateLimit,
		ErrorTypeInvalidRequest,
		ErrorTypeServer,
		ErrorTypeNetwork,
		ErrorTypeTimeout,
		ErrorTypeContentFilter,
		ErrorTypeModelNotFound,
		ErrorTypeQuota,
		ErrorTypeUnknown,
	}

	for _, et := range errorTypes {
		if et == "" {
			t.Error("error type should not be empty")
		}
	}
}
