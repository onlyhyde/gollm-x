# gollm-x

A unified Go library for interacting with multiple LLM providers. Inspired by [ccxt](https://github.com/ccxt/ccxt) for cryptocurrency exchanges, gollm-x provides a consistent interface across different LLM APIs.

## Features

- **Unified API**: Same interface for all providers (OpenAI, Anthropic, Google Gemini, etc.)
- **Provider Registry**: Dynamic provider registration with auto-discovery
- **Streaming Support**: Built-in streaming with iterator pattern
- **Tool/Function Calling**: Consistent tool calling across providers
- **Multimodal**: Vision support for models that support it
- **Embeddings**: Vector embeddings for semantic search
- **Feature Detection**: Query provider capabilities at runtime
- **Retry Logic**: Built-in retry with exponential backoff
- **Rate Limiting**: Token bucket rate limiter for API throttling
- **Type Safety**: Strongly typed requests and responses

## Installation

```bash
go get github.com/onlyhyde/gollm-x
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    gollmx "github.com/onlyhyde/gollm-x"
    _ "github.com/onlyhyde/gollm-x/providers" // Import all providers
)

func main() {
    // Create a client for any provider
    client, err := gollmx.New("openai")  // or "anthropic", "google"
    if err != nil {
        log.Fatal(err)
    }

    // Same API for all providers
    resp, err := client.Chat(context.Background(), &gollmx.ChatRequest{
        Model: "gpt-4o-mini",
        Messages: []gollmx.Message{
            {Role: gollmx.RoleUser, Content: "Hello!"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.GetContent())
}
```

## Supported Providers

| Provider | ID | Chat | Streaming | Vision | Tools | Embeddings |
|----------|-----|------|-----------|--------|-------|------------|
| OpenAI | `openai` | Yes | Yes | Yes | Yes | Yes |
| Anthropic | `anthropic` | Yes | Yes | Yes | Yes | No |
| Google Gemini | `google` | Yes | Yes | Yes | Yes | Yes |
| Ollama | `ollama` | Yes | Yes | Yes | No | Yes |

## Configuration

### Using Environment Variables

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_API_KEY="..."
```

### Using Options

```go
client, err := gollmx.New("openai",
    gollmx.WithAPIKey("sk-..."),
    gollmx.WithBaseURL("https://custom-endpoint.com"),
    gollmx.WithTimeout(60 * time.Second),
    gollmx.WithMaxRetries(5),
)
```

## Streaming

```go
stream, err := client.ChatStream(ctx, &gollmx.ChatRequest{
    Model: "gpt-4o-mini",
    Messages: []gollmx.Message{
        {Role: gollmx.RoleUser, Content: "Tell me a story"},
    },
})
if err != nil {
    log.Fatal(err)
}

for {
    chunk, ok := stream.Next()
    if !ok {
        break
    }
    fmt.Print(chunk.Content)
}
```

## Tool Calling

```go
tools := []gollmx.Tool{
    {
        Type: "function",
        Function: gollmx.Function{
            Name:        "get_weather",
            Description: "Get weather for a location",
            Parameters:  json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
        },
    },
}

resp, err := client.Chat(ctx, &gollmx.ChatRequest{
    Model:    "gpt-4o-mini",
    Messages: messages,
    Tools:    tools,
})

// Check if model wants to call a tool
if toolCalls := resp.GetToolCalls(); len(toolCalls) > 0 {
    // Handle tool calls
}
```

## Feature Detection

```go
if client.HasFeature(gollmx.FeatureVision) {
    // Send image
}

// List all supported features
features := client.Features()
```

## Embeddings

```go
resp, err := client.Embed(ctx, &gollmx.EmbedRequest{
    Model: "text-embedding-3-small",
    Input: []string{
        "Hello, world!",
        "How are you today?",
    },
})
if err != nil {
    log.Fatal(err)
}

for i, emb := range resp.Embeddings {
    fmt.Printf("Embedding %d: %d dimensions\n", i, len(emb.Vector))
}
```

## Retry Logic

Wrap any client with automatic retry and exponential backoff:

```go
client = gollmx.WithRetry(client,
    gollmx.WithRetryMaxRetries(3),
    gollmx.WithRetryInitialDelay(1*time.Second),
    gollmx.WithRetryMaxDelay(30*time.Second),
    gollmx.WithRetryMultiplier(2.0),
)

// Retries automatically on rate limits, server errors, and network issues
resp, err := client.Chat(ctx, req)
```

## Rate Limiting

Control API request rates with token bucket rate limiter:

```go
// Simple: 60 requests per minute
client = gollmx.NewRateLimitedClient(client, 60)

// Advanced: custom configuration
client = gollmx.NewRateLimitedClientWithConfig(client, &gollmx.RateLimitConfig{
    RequestsPerMinute: 100,
    BurstSize:         10,
    WaitTimeout:       30 * time.Second,
})

// Blocks until token available or timeout
resp, err := client.Chat(ctx, req)
```

## Available Models

```go
// List all models for a provider
models := client.Models()

// Get specific model info
model, err := client.GetModel("gpt-4o")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Context window: %d tokens\n", model.ContextWindow)
```

## Selective Provider Import

Import only the providers you need:

```go
import (
    gollmx "github.com/onlyhyde/gollm-x"
    _ "github.com/onlyhyde/gollm-x/providers/openai"
    // Only OpenAI is available
)
```

Or import all at once:

```go
import (
    gollmx "github.com/onlyhyde/gollm-x"
    _ "github.com/onlyhyde/gollm-x/providers"
    // All providers available
)
```

## Error Handling

```go
resp, err := client.Chat(ctx, req)
if err != nil {
    if apiErr, ok := err.(*gollmx.APIError); ok {
        switch apiErr.Type {
        case gollmx.ErrorTypeRateLimit:
            // Wait and retry
        case gollmx.ErrorTypeAuth:
            // Check API key
        case gollmx.ErrorTypeInvalidRequest:
            // Fix request
        }
    }
}
```

## Contributing

Contributions are welcome! To add a new provider:

1. Create a new package under `providers/`
2. Implement the `gollmx.LLM` interface
3. Register the provider in `init()` using `gollmx.Register()`
4. Add the import to `providers/providers.go`

## License

MIT License
