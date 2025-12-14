// Package anthropic provides Anthropic Claude API implementation for gollm-x
package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	gollmx "github.com/onlyhyde/gollm-x"
)

const (
	ProviderID      = "anthropic"
	ProviderName    = "Anthropic"
	DefaultBaseURL  = "https://api.anthropic.com/v1"
	DefaultModel    = "claude-3-5-sonnet-20241022"
	APIVersion      = "2023-06-01"
)

func init() {
	gollmx.Register(ProviderID, New)
}

// Client implements the gollmx.LLM interface for Anthropic
type Client struct {
	config  *gollmx.Config
	baseURL string
	options map[string]interface{}
}

// New creates a new Anthropic client
func New(opts ...gollmx.Option) (gollmx.LLM, error) {
	config := gollmx.DefaultConfig()
	config.Apply(opts...)

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	client := &Client{
		config:  config,
		baseURL: baseURL,
		options: make(map[string]interface{}),
	}

	return client, nil
}

// ID returns the provider identifier
func (c *Client) ID() string {
	return ProviderID
}

// Name returns the provider name
func (c *Client) Name() string {
	return ProviderName
}

// Version returns the client version
func (c *Client) Version() string {
	return "1.0.0"
}

// BaseURL returns the API base URL
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Models returns the list of available models
func (c *Client) Models() []gollmx.Model {
	return AnthropicModels
}

// GetModel returns information about a specific model
func (c *Client) GetModel(id string) (*gollmx.Model, error) {
	for _, m := range AnthropicModels {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, gollmx.NewAPIError(gollmx.ErrorTypeModelNotFound, ProviderID, fmt.Sprintf("model not found: %s", id))
}

// HasFeature checks if a feature is supported
func (c *Client) HasFeature(feature gollmx.Feature) bool {
	switch feature {
	case gollmx.FeatureChat, gollmx.FeatureStreaming, gollmx.FeatureVision,
		gollmx.FeatureTools, gollmx.FeatureJSON, gollmx.FeatureSystemPrompt:
		return true
	case gollmx.FeatureCompletion, gollmx.FeatureEmbedding:
		return false
	}
	return false
}

// Features returns all supported features
func (c *Client) Features() []gollmx.Feature {
	return []gollmx.Feature{
		gollmx.FeatureChat,
		gollmx.FeatureStreaming,
		gollmx.FeatureVision,
		gollmx.FeatureTools,
		gollmx.FeatureJSON,
		gollmx.FeatureSystemPrompt,
	}
}

// SetOption sets a provider-specific option
func (c *Client) SetOption(key string, value interface{}) error {
	c.options[key] = value
	return nil
}

// GetOption gets a provider-specific option
func (c *Client) GetOption(key string) (interface{}, bool) {
	v, ok := c.options[key]
	return v, ok
}

// =============================================================================
// Chat
// =============================================================================

// Chat performs a chat completion request
func (c *Client) Chat(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.config.DefaultModel
		if req.Model == "" {
			req.Model = DefaultModel
		}
	}

	anthropicReq := c.convertRequest(req)

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.config.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, c.handleError(err, 0, nil)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(nil, resp.StatusCode, respBody)
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return c.convertResponse(&anthropicResp), nil
}

// ChatStream performs a streaming chat completion request
func (c *Client) ChatStream(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.StreamReader, error) {
	if req.Model == "" {
		req.Model = c.config.DefaultModel
		if req.Model == "" {
			req.Model = DefaultModel
		}
	}

	anthropicReq := c.convertRequest(req)
	anthropicReq.Stream = true

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.config.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, c.handleError(err, 0, nil)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, c.handleError(nil, resp.StatusCode, respBody)
	}

	ch := make(chan gollmx.StreamChunk)
	go c.readStream(resp.Body, ch, req.Model)

	return gollmx.NewStreamReader(ch), nil
}

func (c *Client) readStream(body io.ReadCloser, ch chan gollmx.StreamChunk, model string) {
	defer close(ch)
	defer body.Close()

	var messageID string
	var currentToolCall *gollmx.ToolCall

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "" {
			continue
		}

		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			ch <- gollmx.StreamChunk{Error: err}
			return
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil {
				messageID = event.Message.ID
			}
		case "content_block_start":
			if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
				currentToolCall = &gollmx.ToolCall{
					ID:   event.ContentBlock.ID,
					Type: "function",
					Function: gollmx.FunctionCall{
						Name: event.ContentBlock.Name,
					},
				}
			}
		case "content_block_delta":
			if event.Delta != nil {
				chunk := gollmx.StreamChunk{
					ID:       messageID,
					Provider: ProviderID,
					Model:    model,
				}

				if event.Delta.Text != "" {
					chunk.Content = event.Delta.Text
				}

				if event.Delta.PartialJSON != "" && currentToolCall != nil {
					currentToolCall.Function.Arguments += event.Delta.PartialJSON
				}

				ch <- chunk
			}
		case "content_block_stop":
			if currentToolCall != nil {
				ch <- gollmx.StreamChunk{
					ID:        messageID,
					Provider:  ProviderID,
					Model:     model,
					ToolCalls: []gollmx.ToolCall{*currentToolCall},
				}
				currentToolCall = nil
			}
		case "message_delta":
			if event.Delta != nil {
				chunk := gollmx.StreamChunk{
					ID:           messageID,
					Provider:     ProviderID,
					Model:        model,
					FinishReason: event.Delta.StopReason,
				}
				if event.Usage != nil {
					chunk.Usage = gollmx.Usage{
						PromptTokens:     event.Usage.InputTokens,
						CompletionTokens: event.Usage.OutputTokens,
						TotalTokens:      event.Usage.InputTokens + event.Usage.OutputTokens,
					}
				}
				ch <- chunk
			}
		case "message_stop":
			// Stream complete
		case "error":
			ch <- gollmx.StreamChunk{Error: fmt.Errorf("stream error")}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- gollmx.StreamChunk{Error: err}
	}
}

// =============================================================================
// Completion (not supported by Anthropic)
// =============================================================================

// Complete performs a text completion request
func (c *Client) Complete(ctx context.Context, req *gollmx.CompletionRequest) (*gollmx.CompletionResponse, error) {
	// Convert to chat completion
	chatReq := &gollmx.ChatRequest{
		Model: req.Model,
		Messages: []gollmx.Message{
			{Role: gollmx.RoleUser, Content: req.Prompt},
		},
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
	}

	chatResp, err := c.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	return &gollmx.CompletionResponse{
		ID:       chatResp.ID,
		Provider: ProviderID,
		Model:    chatResp.Model,
		Created:  chatResp.Created,
		Choices: []gollmx.CompletionChoice{
			{
				Index:        0,
				Text:         chatResp.GetContent(),
				FinishReason: chatResp.Choices[0].FinishReason,
			},
		},
		Usage: chatResp.Usage,
	}, nil
}

// =============================================================================
// Embedding (not supported by Anthropic)
// =============================================================================

// Embed generates embeddings (not supported)
func (c *Client) Embed(ctx context.Context, req *gollmx.EmbedRequest) (*gollmx.EmbedResponse, error) {
	return nil, gollmx.NewAPIError(gollmx.ErrorTypeInvalidRequest, ProviderID, "embedding not supported by Anthropic")
}

// =============================================================================
// Helpers
// =============================================================================

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", APIVersion)

	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}
}

func (c *Client) handleError(err error, statusCode int, body []byte) error {
	if err != nil {
		return &gollmx.APIError{
			Type:     gollmx.ErrorTypeNetwork,
			Provider: ProviderID,
			Message:  err.Error(),
		}
	}

	apiErr := &gollmx.APIError{
		Provider:   ProviderID,
		StatusCode: statusCode,
	}

	var errResp anthropicErrorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != nil {
		apiErr.Message = errResp.Error.Message
		apiErr.Code = errResp.Error.Type
	} else {
		apiErr.Message = string(body)
	}

	switch statusCode {
	case 401:
		apiErr.Type = gollmx.ErrorTypeAuth
	case 429:
		apiErr.Type = gollmx.ErrorTypeRateLimit
		apiErr.Retryable = true
		apiErr.RetryAfter = 60 * time.Second
	case 400:
		apiErr.Type = gollmx.ErrorTypeInvalidRequest
	case 404:
		apiErr.Type = gollmx.ErrorTypeModelNotFound
	case 500, 502, 503, 529:
		apiErr.Type = gollmx.ErrorTypeServer
		apiErr.Retryable = true
	default:
		apiErr.Type = gollmx.ErrorTypeUnknown
	}

	return apiErr
}

func (c *Client) convertRequest(req *gollmx.ChatRequest) *anthropicRequest {
	var systemPrompt string
	messages := make([]anthropicMessage, 0, len(req.Messages))

	for _, m := range req.Messages {
		if m.Role == gollmx.RoleSystem {
			systemPrompt = m.Content.(string)
			continue
		}

		msg := anthropicMessage{
			Role: string(m.Role),
		}

		// Handle tool results
		if m.Role == gollmx.RoleTool {
			msg.Role = "user"
			msg.Content = []anthropicContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content.(string),
				},
			}
		} else if len(m.ToolCalls) > 0 {
			// Assistant message with tool calls
			blocks := make([]anthropicContentBlock, 0)
			if content, ok := m.Content.(string); ok && content != "" {
				blocks = append(blocks, anthropicContentBlock{
					Type: "text",
					Text: content,
				})
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, anthropicContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: json.RawMessage(tc.Function.Arguments),
				})
			}
			msg.Content = blocks
		} else {
			msg.Content = m.Content
		}

		messages = append(messages, msg)
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	anthropicReq := &anthropicRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		System:      systemPrompt,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		StopSeqs:    req.Stop,
	}

	if len(req.Tools) > 0 {
		anthropicReq.Tools = make([]anthropicTool, len(req.Tools))
		for i, t := range req.Tools {
			anthropicReq.Tools[i] = anthropicTool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				InputSchema: t.Function.Parameters,
			}
		}
	}

	return anthropicReq
}

func (c *Client) convertResponse(resp *anthropicResponse) *gollmx.ChatResponse {
	var content string
	var toolCalls []gollmx.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			tc := gollmx.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: gollmx.FunctionCall{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			}
			toolCalls = append(toolCalls, tc)
		}
	}

	message := gollmx.Message{
		Role:      gollmx.RoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}

	finishReason := resp.StopReason
	if finishReason == "end_turn" {
		finishReason = "stop"
	} else if finishReason == "tool_use" {
		finishReason = "tool_calls"
	}

	return &gollmx.ChatResponse{
		ID:       resp.ID,
		Provider: ProviderID,
		Model:    resp.Model,
		Created:  time.Now().Unix(),
		Choices: []gollmx.Choice{
			{
				Index:        0,
				Message:      message,
				FinishReason: finishReason,
			},
		},
		Usage: gollmx.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		Raw: resp,
	}
}
