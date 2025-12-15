// Package cohere provides Cohere API implementation for gollm-x
package cohere

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	gollmx "github.com/onlyhyde/gollm-x"
)

const (
	ProviderID     = "cohere"
	ProviderName   = "Cohere"
	DefaultBaseURL = "https://api.cohere.ai/v1"
	DefaultModel   = "command-r-plus"
)

func init() {
	gollmx.Register(ProviderID, New)
}

// Client implements the gollmx.LLM interface for Cohere
type Client struct {
	config  *gollmx.Config
	baseURL string
	options map[string]interface{}
}

// New creates a new Cohere client
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
	return CohereModels
}

// GetModel returns information about a specific model
func (c *Client) GetModel(id string) (*gollmx.Model, error) {
	for _, m := range CohereModels {
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
		gollmx.FeatureSystemPrompt, gollmx.FeatureEmbedding:
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

	cohereReq := c.convertChatRequest(req)

	body, err := json.Marshal(cohereReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat", bytes.NewReader(body))
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

	var cohereResp chatResponse
	if err := json.Unmarshal(respBody, &cohereResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return c.convertChatResponse(&cohereResp, req.Model), nil
}

// ChatStream performs a streaming chat completion request
func (c *Client) ChatStream(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.StreamReader, error) {
	if req.Model == "" {
		req.Model = c.config.DefaultModel
		if req.Model == "" {
			req.Model = DefaultModel
		}
	}

	cohereReq := c.convertChatRequest(req)
	cohereReq.Stream = true

	body, err := json.Marshal(cohereReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat", bytes.NewReader(body))
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
		if line == "" {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		gollmxChunk := gollmx.StreamChunk{
			Provider: ProviderID,
			Model:    model,
		}

		switch event.EventType {
		case "text-generation":
			gollmxChunk.Content = event.Text
		case "stream-end":
			if event.Response != nil {
				gollmxChunk.FinishReason = string(event.Response.FinishReason)
				if event.Response.Meta != nil && event.Response.Meta.Tokens != nil {
					gollmxChunk.Usage = gollmx.Usage{
						PromptTokens:     event.Response.Meta.Tokens.InputTokens,
						CompletionTokens: event.Response.Meta.Tokens.OutputTokens,
						TotalTokens:      event.Response.Meta.Tokens.InputTokens + event.Response.Meta.Tokens.OutputTokens,
					}
				}
			}
		default:
			continue
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
		req.Model = "embed-english-v3.0"
	}

	cohereReq := embedRequest{
		Model:     req.Model,
		Texts:     req.Input,
		InputType: "search_document",
	}

	body, err := json.Marshal(cohereReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embed", bytes.NewReader(body))
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

	var cohereResp embedResponse
	if err := json.Unmarshal(respBody, &cohereResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	embeddings := make([]gollmx.Embedding, len(cohereResp.Embeddings))
	for i, emb := range cohereResp.Embeddings {
		embeddings[i] = gollmx.Embedding{
			Index:  i,
			Vector: emb,
		}
	}

	return &gollmx.EmbedResponse{
		Provider:   ProviderID,
		Model:      req.Model,
		Embeddings: embeddings,
		Usage: gollmx.Usage{
			TotalTokens: cohereResp.Meta.BilledUnits.InputTokens,
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
	cohereReq := &chatRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		P:           req.TopP,
		StopSequences: req.Stop,
	}

	// Extract system message and chat history
	var chatHistory []chatMessage
	for _, m := range req.Messages {
		switch m.Role {
		case gollmx.RoleSystem:
			cohereReq.Preamble = m.Content.(string)
		case gollmx.RoleUser:
			if content, ok := m.Content.(string); ok {
				// Last user message is the current message
				cohereReq.Message = content
			}
			chatHistory = append(chatHistory, chatMessage{
				Role:    "USER",
				Message: m.Content.(string),
			})
		case gollmx.RoleAssistant:
			chatHistory = append(chatHistory, chatMessage{
				Role:    "CHATBOT",
				Message: m.Content.(string),
			})
		}
	}

	// Remove last user message from history (it's the current message)
	if len(chatHistory) > 0 && chatHistory[len(chatHistory)-1].Role == "USER" {
		chatHistory = chatHistory[:len(chatHistory)-1]
	}

	if len(chatHistory) > 0 {
		cohereReq.ChatHistory = chatHistory
	}

	if len(req.Tools) > 0 {
		cohereReq.Tools = make([]tool, len(req.Tools))
		for i, t := range req.Tools {
			cohereReq.Tools[i] = tool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
			}
			// Parse parameters for tool
			if len(t.Function.Parameters) > 0 {
				var params map[string]interface{}
				if err := json.Unmarshal(t.Function.Parameters, &params); err == nil {
					if props, ok := params["properties"].(map[string]interface{}); ok {
						cohereReq.Tools[i].ParameterDefinitions = make(map[string]parameterDef)
						for name, prop := range props {
							if propMap, ok := prop.(map[string]interface{}); ok {
								def := parameterDef{}
								if t, ok := propMap["type"].(string); ok {
									def.Type = t
								}
								if d, ok := propMap["description"].(string); ok {
									def.Description = d
								}
								cohereReq.Tools[i].ParameterDefinitions[name] = def
							}
						}
					}
				}
			}
		}
	}

	return cohereReq
}

func (c *Client) convertChatResponse(resp *chatResponse, model string) *gollmx.ChatResponse {
	content := resp.Text

	chatResp := &gollmx.ChatResponse{
		ID:       resp.GenerationID,
		Provider: ProviderID,
		Model:    model,
		Choices: []gollmx.Choice{
			{
				Index: 0,
				Message: gollmx.Message{
					Role:    gollmx.RoleAssistant,
					Content: content,
				},
				FinishReason: string(resp.FinishReason),
			},
		},
		Raw: resp,
	}

	if resp.Meta != nil && resp.Meta.Tokens != nil {
		chatResp.Usage = gollmx.Usage{
			PromptTokens:     resp.Meta.Tokens.InputTokens,
			CompletionTokens: resp.Meta.Tokens.OutputTokens,
			TotalTokens:      resp.Meta.Tokens.InputTokens + resp.Meta.Tokens.OutputTokens,
		}
	}

	return chatResp
}
