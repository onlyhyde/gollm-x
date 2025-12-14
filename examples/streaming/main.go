// Example: Streaming responses with gollm-x
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	gollmx "github.com/onlyhyde/gollm-x"
	_ "github.com/onlyhyde/gollm-x/providers"
)

func main() {
	ctx := context.Background()

	// Use OpenAI for this example
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable required")
	}

	client, err := gollmx.New("openai")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Streaming response from GPT-4o-mini:")
	fmt.Println("---")

	// Create streaming request
	stream, err := client.ChatStream(ctx, &gollmx.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "Write a short poem about coding."},
		},
		MaxTokens: 200,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Read chunks as they arrive
	for {
		chunk, ok := stream.Next()
		if !ok {
			break
		}
		fmt.Print(chunk.Content)
	}

	// Check for errors
	if err := stream.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n---")
	fmt.Println("Stream complete!")
}
