// Example: Using tools/function calling with gollm-x
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	gollmx "github.com/onlyhyde/gollm-x"
	_ "github.com/onlyhyde/gollm-x/providers"
)

// Define tool parameters as JSON schema
var getWeatherSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"location": {
			"type": "string",
			"description": "The city and country, e.g., 'Tokyo, Japan'"
		},
		"unit": {
			"type": "string",
			"enum": ["celsius", "fahrenheit"],
			"description": "Temperature unit"
		}
	},
	"required": ["location"]
}`)

func main() {
	ctx := context.Background()

	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable required")
	}

	client, err := gollmx.New("openai")
	if err != nil {
		log.Fatal(err)
	}

	// Define tools
	tools := []gollmx.Tool{
		{
			Type: "function",
			Function: gollmx.Function{
				Name:        "get_weather",
				Description: "Get the current weather for a location",
				Parameters:  getWeatherSchema,
			},
		},
	}

	// First call: Model decides to use the tool
	resp, err := client.Chat(ctx, &gollmx.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "What's the weather like in Seoul?"},
		},
		Tools:     tools,
		MaxTokens: 200,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Check if model wants to call a tool
	toolCalls := resp.GetToolCalls()
	if len(toolCalls) > 0 {
		fmt.Println("Model wants to call tools:")
		for _, tc := range toolCalls {
			fmt.Printf("  - %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
		}

		// Simulate tool execution
		weatherResult := `{"temperature": 22, "unit": "celsius", "condition": "sunny"}`

		// Second call: Provide tool result
		messages := []gollmx.Message{
			{Role: gollmx.RoleUser, Content: "What's the weather like in Seoul?"},
			{
				Role:      gollmx.RoleAssistant,
				Content:   resp.GetContent(),
				ToolCalls: toolCalls,
			},
			{
				Role:       gollmx.RoleTool,
				Content:    weatherResult,
				ToolCallID: toolCalls[0].ID,
			},
		}

		finalResp, err := client.Chat(ctx, &gollmx.ChatRequest{
			Model:     "gpt-4o-mini",
			Messages:  messages,
			Tools:     tools,
			MaxTokens: 200,
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("\nFinal response:")
		fmt.Println(finalResp.GetContent())
	} else {
		fmt.Println("Response (no tool call):")
		fmt.Println(resp.GetContent())
	}
}
