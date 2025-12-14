// Package gollmx provides a unified interface for interacting with various LLM providers.
// Inspired by ccxt for cryptocurrency exchanges, gollm-x abstracts away the differences
// between LLM APIs, allowing you to use a consistent interface across providers.
//
// Example usage:
//
//	client := gollmx.New("openai", gollmx.WithAPIKey("sk-..."))
//	resp, err := client.Chat(ctx, &gollmx.ChatRequest{
//	    Model: "gpt-4o",
//	    Messages: []gollmx.Message{{Role: "user", Content: "Hello!"}},
//	})
package gollmx

import (
	"context"
	"fmt"
	"sync"
)

// LLM is the main interface that all providers must implement.
// It provides a unified API for interacting with different LLM services.
type LLM interface {
	// Provider information
	ID() string          // Unique identifier (e.g., "openai", "anthropic")
	Name() string        // Human-readable name
	Version() string     // Provider client version
	BaseURL() string     // API base URL

	// Model information
	Models() []Model                    // List available models
	GetModel(id string) (*Model, error) // Get specific model info

	// Core chat functionality
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// Streaming chat
	ChatStream(ctx context.Context, req *ChatRequest) (*StreamReader, error)

	// Text completion (legacy, not all providers support)
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Embeddings
	Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error)

	// Feature detection
	HasFeature(feature Feature) bool
	Features() []Feature

	// Configuration
	SetOption(key string, value interface{}) error
	GetOption(key string) (interface{}, bool)
}

// Feature represents optional capabilities that providers may support
type Feature string

const (
	FeatureChat         Feature = "chat"
	FeatureCompletion   Feature = "completion"
	FeatureEmbedding    Feature = "embedding"
	FeatureStreaming    Feature = "streaming"
	FeatureVision       Feature = "vision"
	FeatureTools        Feature = "tools"        // Function calling
	FeatureJSON         Feature = "json_mode"    // Structured JSON output
	FeatureSystemPrompt Feature = "system_prompt"
)

// ProviderFactory is a function that creates a new LLM instance
type ProviderFactory func(opts ...Option) (LLM, error)

// Registry holds all registered providers
var (
	registry   = make(map[string]ProviderFactory)
	registryMu sync.RWMutex
)

// Register adds a new provider to the registry
func Register(id string, factory ProviderFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[id] = factory
}

// New creates a new LLM client for the specified provider
func New(providerID string, opts ...Option) (LLM, error) {
	registryMu.RLock()
	factory, ok := registry[providerID]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: %v)", providerID, Providers())
	}

	return factory(opts...)
}

// MustNew creates a new LLM client, panicking on error
func MustNew(providerID string, opts ...Option) LLM {
	llm, err := New(providerID, opts...)
	if err != nil {
		panic(err)
	}
	return llm
}

// Providers returns a list of all registered provider IDs
func Providers() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	return ids
}

// HasProvider checks if a provider is registered
func HasProvider(id string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := registry[id]
	return ok
}
