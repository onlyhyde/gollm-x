package gollmx

import (
	"context"
	"testing"
)

func TestRegisterAndNew(t *testing.T) {
	// Register a mock provider
	mockFactory := func(opts ...Option) (LLM, error) {
		return &mockLLM{id: "mock"}, nil
	}

	Register("mock", mockFactory)

	// Check if provider is registered
	if !HasProvider("mock") {
		t.Error("mock provider should be registered")
	}

	// Create new client
	client, err := New("mock")
	if err != nil {
		t.Fatalf("failed to create mock client: %v", err)
	}

	if client.ID() != "mock" {
		t.Errorf("expected ID 'mock', got '%s'", client.ID())
	}
}

func TestNewWithUnknownProvider(t *testing.T) {
	_, err := New("unknown-provider-xyz")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestProviders(t *testing.T) {
	// Register test provider
	Register("test-provider", func(opts ...Option) (LLM, error) {
		return &mockLLM{id: "test-provider"}, nil
	})

	providers := Providers()
	found := false
	for _, p := range providers {
		if p == "test-provider" {
			found = true
			break
		}
	}

	if !found {
		t.Error("test-provider should be in providers list")
	}
}

func TestMustNew(t *testing.T) {
	Register("must-test", func(opts ...Option) (LLM, error) {
		return &mockLLM{id: "must-test"}, nil
	})

	// Should not panic
	client := MustNew("must-test")
	if client.ID() != "must-test" {
		t.Errorf("expected ID 'must-test', got '%s'", client.ID())
	}
}

func TestMustNewPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNew should panic for unknown provider")
		}
	}()

	MustNew("non-existent-provider-xyz")
}

// mockLLM is a minimal mock implementation for testing
type mockLLM struct {
	id string
}

func (m *mockLLM) ID() string                   { return m.id }
func (m *mockLLM) Name() string                 { return "Mock LLM" }
func (m *mockLLM) Version() string              { return "1.0.0" }
func (m *mockLLM) BaseURL() string              { return "http://mock" }
func (m *mockLLM) Models() []Model              { return nil }
func (m *mockLLM) GetModel(id string) (*Model, error) { return nil, nil }
func (m *mockLLM) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) { return nil, nil }
func (m *mockLLM) ChatStream(ctx context.Context, req *ChatRequest) (*StreamReader, error) { return nil, nil }
func (m *mockLLM) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) { return nil, nil }
func (m *mockLLM) Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error) { return nil, nil }
func (m *mockLLM) HasFeature(feature Feature) bool { return false }
func (m *mockLLM) Features() []Feature          { return nil }
func (m *mockLLM) SetOption(key string, value interface{}) error { return nil }
func (m *mockLLM) GetOption(key string) (interface{}, bool) { return nil, false }
