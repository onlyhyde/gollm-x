// Package mistral provides Mistral AI API implementation for gollm-x
// Mistral uses OpenAI-compatible API
package mistral

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
	ProviderID     = "mistral"
	ProviderName   = "Mistral AI"
	DefaultBaseURL = "https://api.mistral.ai/v1"
	DefaultModel   = "mistral-small-latest"
)

func init() {
	gollmx.Register(ProviderID, New)
}

// Client implements the gollmx.LLM interface for Mistral
type Client struct {
	config  *gollmx.Config
	baseURL string
	options map[string]interface{}
}

// New creates a new Mistral client
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
	return MistralModels
}

// GetModel returns information about a specific model
func (c *Client) GetModel(id string) (*gollmx.Model, error) {
	for _, m := range MistralModels {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, gollmx.NewAPIError(gollmx.ErrorTypeModelNotFound, ProviderID, fmt.Sprintf("model not found: %s", id))
}

// HasFeature checks if a feature is supported
func (c *Client) HasFeature(feature gollmx.Feature) bool {
	switch feature {
	case gollmx.FeatureChat, gollmx.FeatureStreaming, gollmx.FeatureTools,
		gollmx.FeatureJSON, gollmx.FeatureSystemPrompt, gollmx.FeatureEmbedding:
		return true
	}
	return false
}

// Features returns all supported features
func (c *Client) Features() []gollmx.Feature {
	return []gollmx.Feature{
		gollmx.FeatureChat,
		gollmx.FeatureStreaming,
		gollmx.FeatureTools,
		gollmx.FeatureJSON,
		gollmx.FeatureSystemPrompt,
		gollmx.FeatureEmbedding,
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

	mistralReq := c.convertChatRequest(req)

	body, err := json.Marshal(mistralReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
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

	var mistralResp chatResponse
	if err := json.Unmarshal(respBody, &mistralResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return c.convertChatResponse(&mistralResp), nil
}

// ChatStream performs a streaming chat completion request
func (c *Client) ChatStream(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.StreamReader, error) {
	if req.Model == "" {
		req.Model = c.config.DefaultModel
		if req.Model == "" {
			req.Model = DefaultModel
		}
	}

	mistralReq := c.convertChatRequest(req)
	mistralReq.Stream = true

	body, err := json.Marshal(mistralReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
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

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			ch <- gollmx.StreamChunk{Error: err}
			return
		}

		gollmxChunk := gollmx.StreamChunk{
			ID:       chunk.ID,
			Provider: ProviderID,
			Model:    chunk.Model,
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			gollmxChunk.Content = delta.Content
			gollmxChunk.FinishReason = chunk.Choices[0].FinishReason

			if len(delta.ToolCalls) > 0 {
				for _, tc := range delta.ToolCalls {
					gollmxChunk.ToolCalls = append(gollmxChunk.ToolCalls, gollmx.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: gollmx.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					})
				}
			}
		}

		if chunk.Usage != nil {
			gollmxChunk.Usage = gollmx.Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

		ch <- gollmxChunk
	}

	if err := scanner.Err(); err != nil {
		ch <- gollmx.StreamChunk{Error: err}
	}
}

// =============================================================================
// Completion
// =============================================================================

// Complete performs a text completion request
func (c *Client) Complete(ctx context.Context, req *gollmx.CompletionRequest) (*gollmx.CompletionResponse, error) {
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
// Embedding
// =============================================================================

// Embed generates embeddings
func (c *Client) Embed(ctx context.Context, req *gollmx.EmbedRequest) (*gollmx.EmbedResponse, error) {
	if req.Model == "" {
		req.Model = "mistral-embed"
	}

	mistralReq := embedRequest{
		Model: req.Model,
		Input: req.Input,
	}

	body, err := json.Marshal(mistralReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(body))
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

	var mistralResp embedResponse
	if err := json.Unmarshal(respBody, &mistralResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	embeddings := make([]gollmx.Embedding, len(mistralResp.Data))
	for i, d := range mistralResp.Data {
		embeddings[i] = gollmx.Embedding{
			Index:  d.Index,
			Vector: d.Embedding,
		}
	}

	return &gollmx.EmbedResponse{
		Provider:   ProviderID,
		Model:      req.Model,
		Embeddings: embeddings,
		Usage: gollmx.Usage{
			PromptTokens: mistralResp.Usage.PromptTokens,
			TotalTokens:  mistralResp.Usage.TotalTokens,
		},
	}, nil
}

// =============================================================================
// Helpers
// =============================================================================

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

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

	var errResp errorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
		apiErr.Message = errResp.Message
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
	case 500, 502, 503:
		apiErr.Type = gollmx.ErrorTypeServer
		apiErr.Retryable = true
	default:
		apiErr.Type = gollmx.ErrorTypeUnknown
	}

	return apiErr
}

func (c *Client) convertChatRequest(req *gollmx.ChatRequest) *chatRequest {
	messages := make([]message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = message{
			Role:       string(m.Role),
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			messages[i].ToolCalls = make([]toolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				messages[i].ToolCalls[j] = toolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: functionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	mistralReq := &chatRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
		Stream:      req.Stream,
	}

	if len(req.Tools) > 0 {
		mistralReq.Tools = make([]tool, len(req.Tools))
		for i, t := range req.Tools {
			mistralReq.Tools[i] = tool{
				Type: t.Type,
				Function: function{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
		mistralReq.ToolChoice = req.ToolChoice
	}

	if req.ResponseFormat != nil {
		mistralReq.ResponseFormat = &responseFormat{
			Type: req.ResponseFormat.Type,
		}
	}

	return mistralReq
}

func (c *Client) convertChatResponse(resp *chatResponse) *gollmx.ChatResponse {
	choices := make([]gollmx.Choice, len(resp.Choices))
	for i, ch := range resp.Choices {
		msg := gollmx.Message{
			Role:    gollmx.Role(ch.Message.Role),
			Content: ch.Message.Content,
		}

		if len(ch.Message.ToolCalls) > 0 {
			msg.ToolCalls = make([]gollmx.ToolCall, len(ch.Message.ToolCalls))
			for j, tc := range ch.Message.ToolCalls {
				msg.ToolCalls[j] = gollmx.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: gollmx.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		choices[i] = gollmx.Choice{
			Index:        ch.Index,
			Message:      msg,
			FinishReason: ch.FinishReason,
		}
	}

	return &gollmx.ChatResponse{
		ID:       resp.ID,
		Provider: ProviderID,
		Model:    resp.Model,
		Created:  resp.Created,
		Choices:  choices,
		Usage: gollmx.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Raw: resp,
	}
}
