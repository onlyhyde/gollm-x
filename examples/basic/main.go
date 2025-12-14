// Example: Basic usage of gollm-x
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	gollmx "github.com/onlyhyde/gollm-x"
	// Import all providers
	_ "github.com/onlyhyde/gollm-x/providers"
)

func main() {
	ctx := context.Background()

	// List available providers
	fmt.Println("Available providers:", gollmx.Providers())

	// Example 1: Using OpenAI
	if os.Getenv("OPENAI_API_KEY") != "" {
		openaiExample(ctx)
	}

	// Example 2: Using Anthropic
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		anthropicExample(ctx)
	}

	// Example 3: Using Google Gemini
	if os.Getenv("GOOGLE_API_KEY") != "" {
		geminiExample(ctx)
	}
}

func openaiExample(ctx context.Context) {
	fmt.Println("\n=== OpenAI Example ===")

	client, err := gollmx.New("openai")
	if err != nil {
		log.Printf("Failed to create OpenAI client: %v", err)
		return
	}

	// List models
	fmt.Printf("Provider: %s (%s)\n", client.Name(), client.ID())
	fmt.Printf("Available models: %d\n", len(client.Models()))

	// Simple chat
	resp, err := client.Chat(ctx, &gollmx.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleSystem, Content: "You are a helpful assistant. Be concise."},
			{Role: gollmx.RoleUser, Content: "What is 2+2?"},
		},
		MaxTokens: 100,
	})
	if err != nil {
		log.Printf("Chat error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.GetContent())
	fmt.Printf("Tokens: %d prompt, %d completion\n", resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
}

func anthropicExample(ctx context.Context) {
	fmt.Println("\n=== Anthropic Example ===")

	client, err := gollmx.New("anthropic")
	if err != nil {
		log.Printf("Failed to create Anthropic client: %v", err)
		return
	}

	fmt.Printf("Provider: %s (%s)\n", client.Name(), client.ID())

	resp, err := client.Chat(ctx, &gollmx.ChatRequest{
		Model: "claude-3-5-haiku-20241022",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "What is 2+2? Reply with just the number."},
		},
		MaxTokens: 100,
	})
	if err != nil {
		log.Printf("Chat error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.GetContent())
}

func geminiExample(ctx context.Context) {
	fmt.Println("\n=== Google Gemini Example ===")

	client, err := gollmx.New("google")
	if err != nil {
		log.Printf("Failed to create Gemini client: %v", err)
		return
	}

	fmt.Printf("Provider: %s (%s)\n", client.Name(), client.ID())

	resp, err := client.Chat(ctx, &gollmx.ChatRequest{
		Model: "gemini-1.5-flash",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "What is 2+2? Reply with just the number."},
		},
		MaxTokens: 100,
	})
	if err != nil {
		log.Printf("Chat error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.GetContent())
}
