// Package anthropic provides an Anthropic Claude API client for gollm-x.
package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	gollmx "github.com/onlyhyde/gollm-x"
)

const (
	ProviderID     = "anthropic"
	ProviderName   = "Anthropic"
	DefaultBaseURL = "https://api.anthropic.com"
	DefaultVersion = "2023-06-01"
	ClientVersion  = "1.0.0"
)

// Client implements the gollmx.LLM interface for Anthropic
type Client struct {
	config     *gollmx.Config
	httpClient *http.Client
	baseURL    string
	apiVersion string
	options    map[string]interface{}
}

func init() {
	gollmx.Register(ProviderID, NewClient)
}

// NewClient creates a new Anthropic client
func NewClient(opts ...gollmx.Option) (gollmx.LLM, error) {
	config := gollmx.DefaultConfig()
	config.Apply(opts...)

	// Try to get API key from environment if not provided
	if config.APIKey == "" {
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		config:     config,
		httpClient: config.GetHTTPClient(),
		baseURL:    baseURL,
		apiVersion: DefaultVersion,
		options:    make(map[string]interface{}),
	}, nil
}

// Provider information methods
func (c *Client) ID() string      { return ProviderID }
func (c *Client) Name() string    { return ProviderName }
func (c *Client) Version() string { return ClientVersion }
func (c *Client) BaseURL() string { return c.baseURL }

// Models returns all available Anthropic models
func (c *Client) Models() []gollmx.Model {
	return AnthropicModels
}

// GetModel returns information about a specific model
func (c *Client) GetModel(id string) (*gollmx.Model, error) {
	for _, model := range AnthropicModels {
		if model.ID == id {
			return &model, nil
		}
	}
	return nil, gollmx.NewAPIError(gollmx.ErrorTypeModelNotFound, ProviderID, fmt.Sprintf("model not found: %s", id))
}

// Chat sends a chat request to Anthropic's Messages API
func (c *Client) Chat(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.ChatResponse, error) {
	anthropicReq, systemPrompt := c.convertChatRequest(req)
	if systemPrompt != "" {
		anthropicReq.System = systemPrompt
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.doRequestWithRetry(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var anthropicResp anthropicMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertChatResponse(&anthropicResp), nil
}

// ChatStream sends a streaming chat request
func (c *Client) ChatStream(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.StreamReader, error) {
	anthropicReq, systemPrompt := c.convertChatRequest(req)
	if systemPrompt != "" {
		anthropicReq.System = systemPrompt
	}
	anthropicReq.Stream = true

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, c.handleErrorResponse(resp)
	}

	ch := make(chan gollmx.StreamChunk, 100)
	go c.processStream(resp, ch)

	return gollmx.NewStreamReader(ch), nil
}

// Complete is not natively supported by Anthropic's Messages API
func (c *Client) Complete(ctx context.Context, req *gollmx.CompletionRequest) (*gollmx.CompletionResponse, error) {
	// Convert completion request to chat request
	chatReq := &gollmx.ChatRequest{
		Model:       req.Model,
		Messages:    []gollmx.Message{{Role: gollmx.RoleUser, Content: req.Prompt}},
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
		Choices: []gollmx.CompletionChoice{{
			Index:        0,
			Text:         chatResp.GetContent(),
			FinishReason: chatResp.Choices[0].FinishReason,
		}},
		Usage: chatResp.Usage,
	}, nil
}

// Embed is not supported by Anthropic
func (c *Client) Embed(ctx context.Context, req *gollmx.EmbedRequest) (*gollmx.EmbedResponse, error) {
	return nil, gollmx.NewAPIError(gollmx.ErrorTypeInvalidRequest, ProviderID, "Anthropic does not support embeddings")
}

// HasFeature checks if the provider supports a feature
func (c *Client) HasFeature(feature gollmx.Feature) bool {
	switch feature {
	case gollmx.FeatureChat, gollmx.FeatureStreaming, gollmx.FeatureVision,
		gollmx.FeatureTools, gollmx.FeatureJSON, gollmx.FeatureSystemPrompt:
		return true
	case gollmx.FeatureEmbedding, gollmx.FeatureCompletion:
		return false
	default:
		return false
	}
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
// Private methods
// =============================================================================

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", c.apiVersion)

	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}
}

func (c *Client) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.config.RetryDelay * time.Duration(attempt))

			// Clone request for retry
			newReq := req.Clone(req.Context())
			if req.Body != nil {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				newReq.Body = io.NopCloser(bytes.NewReader(body))
				req.Body = io.NopCloser(bytes.NewReader(body))
			}
			req = newReq
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Don't retry on success or client errors
		if resp.StatusCode < 500 {
			return resp, nil
		}

		// Server error, might be retryable
		resp.Body.Close()
		lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return nil, lastErr
}

func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp anthropicErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return gollmx.NewAPIError(gollmx.ErrorTypeUnknown, ProviderID, string(body))
	}

	apiErr := &gollmx.APIError{
		Provider:   ProviderID,
		StatusCode: resp.StatusCode,
		Message:    errResp.Error.Message,
		Code:       errResp.Error.Type,
		Raw:        errResp,
	}

	switch errResp.Error.Type {
	case "authentication_error":
		apiErr.Type = gollmx.ErrorTypeAuth
	case "rate_limit_error":
		apiErr.Type = gollmx.ErrorTypeRateLimit
		apiErr.Retryable = true
	case "invalid_request_error":
		apiErr.Type = gollmx.ErrorTypeInvalidRequest
	case "overloaded_error":
		apiErr.Type = gollmx.ErrorTypeServer
		apiErr.Retryable = true
	default:
		apiErr.Type = gollmx.ErrorTypeUnknown
	}

	return apiErr
}

func (c *Client) convertChatRequest(req *gollmx.ChatRequest) (*anthropicMessagesRequest, string) {
	var messages []anthropicMessage
	var systemPrompt string

	for _, msg := range req.Messages {
		switch msg.Role {
		case gollmx.RoleSystem:
			// Extract system message
			if content, ok := msg.Content.(string); ok {
				systemPrompt = content
			}
		case gollmx.RoleUser, gollmx.RoleAssistant:
			messages = append(messages, c.convertMessage(msg))
		case gollmx.RoleTool:
			// Tool results are added to the previous user message
			messages = append(messages, c.convertToolResultMessage(msg))
		}
	}

	anthropicReq := &anthropicMessagesRequest{
		Model:    req.Model,
		Messages: messages,
	}

	if req.MaxTokens > 0 {
		anthropicReq.MaxTokens = req.MaxTokens
	} else {
		anthropicReq.MaxTokens = 4096 // Default
	}

	if req.Temperature != nil {
		anthropicReq.Temperature = req.Temperature
	}
	if req.TopP != nil {
		anthropicReq.TopP = req.TopP
	}
	if len(req.Stop) > 0 {
		anthropicReq.StopSeqs = req.Stop
	}

	// Convert tools
	if len(req.Tools) > 0 {
		anthropicReq.Tools = c.convertTools(req.Tools)
	}

	return anthropicReq, systemPrompt
}

func (c *Client) convertMessage(msg gollmx.Message) anthropicMessage {
	role := string(msg.Role)

	// Handle multimodal content
	switch content := msg.Content.(type) {
	case string:
		if len(msg.ToolCalls) > 0 {
			// Assistant message with tool calls
			blocks := []anthropicContentBlock{{Type: "text", Text: content}}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, anthropicContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: json.RawMessage(tc.Function.Arguments),
				})
			}
			return anthropicMessage{Role: role, Content: blocks}
		}
		return anthropicMessage{Role: role, Content: content}

	case []gollmx.ContentPart:
		var blocks []anthropicContentBlock
		for _, part := range content {
			switch part.Type {
			case "text":
				blocks = append(blocks, anthropicContentBlock{Type: "text", Text: part.Text})
			case "image_url":
				if part.ImageURL != nil {
					blocks = append(blocks, anthropicContentBlock{
						Type: "image",
						Source: &anthropicImageSource{
							Type: "url",
							URL:  part.ImageURL.URL,
						},
					})
				}
			}
		}
		return anthropicMessage{Role: role, Content: blocks}

	default:
		return anthropicMessage{Role: role, Content: ""}
	}
}

func (c *Client) convertToolResultMessage(msg gollmx.Message) anthropicMessage {
	content := ""
	if c, ok := msg.Content.(string); ok {
		content = c
	}

	return anthropicMessage{
		Role: "user",
		Content: []anthropicContentBlock{{
			Type:      "tool_result",
			ToolUseID: msg.ToolCallID,
			Content:   content,
		}},
	}
}

func (c *Client) convertTools(tools []gollmx.Tool) []anthropicTool {
	var result []anthropicTool
	for _, tool := range tools {
		if tool.Type == "function" {
			result = append(result, anthropicTool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				InputSchema: tool.Function.Parameters,
			})
		}
	}
	return result
}

func (c *Client) convertChatResponse(resp *anthropicMessagesResponse) *gollmx.ChatResponse {
	var content string
	var toolCalls []gollmx.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, gollmx.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: gollmx.FunctionCall{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})
		}
	}

	finishReason := c.convertStopReason(resp.StopReason)

	return &gollmx.ChatResponse{
		ID:       resp.ID,
		Provider: ProviderID,
		Model:    resp.Model,
		Created:  time.Now().Unix(),
		Choices: []gollmx.Choice{{
			Index: 0,
			Message: gollmx.Message{
				Role:      gollmx.RoleAssistant,
				Content:   content,
				ToolCalls: toolCalls,
			},
			FinishReason: finishReason,
		}},
		Usage: gollmx.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		Raw: resp,
	}
}

func (c *Client) convertStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	default:
		return reason
	}
}

func (c *Client) processStream(resp *http.Response, ch chan<- gollmx.StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	var messageID, model string
	var inputTokens int

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				ch <- gollmx.StreamChunk{Error: err}
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}

		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil {
				messageID = event.Message.ID
				model = event.Message.Model
				inputTokens = event.Message.Usage.InputTokens
			}

		case "content_block_delta":
			var delta anthropicStreamDelta
			if err := json.Unmarshal(event.Delta, &delta); err != nil {
				continue
			}

			if delta.Text != "" {
				ch <- gollmx.StreamChunk{
					ID:       messageID,
					Provider: ProviderID,
					Model:    model,
					Content:  delta.Text,
				}
			}

		case "message_delta":
			var delta anthropicStreamDelta
			if err := json.Unmarshal(event.Delta, &delta); err != nil {
				continue
			}

			chunk := gollmx.StreamChunk{
				ID:           messageID,
				Provider:     ProviderID,
				Model:        model,
				FinishReason: c.convertStopReason(delta.StopReason),
			}

			if event.Usage != nil {
				chunk.Usage = gollmx.Usage{
					PromptTokens:     inputTokens,
					CompletionTokens: event.Usage.OutputTokens,
					TotalTokens:      inputTokens + event.Usage.OutputTokens,
				}
			}

			ch <- chunk
		}
	}
}
