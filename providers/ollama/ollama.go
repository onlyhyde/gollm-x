// Package ollama provides an Ollama LLM client implementation
package ollama

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
	ProviderID     = "ollama"
	ProviderName   = "Ollama"
	DefaultBaseURL = "http://localhost:11434"
	DefaultModel   = "llama3.2"
)

func init() {
	gollmx.Register(ProviderID, New)
}

// Client implements the gollmx.LLM interface for Ollama
type Client struct {
	config  *gollmx.Config
	baseURL string
	options map[string]interface{}
}

// New creates a new Ollama client
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

// Models returns available models
func (c *Client) Models() []gollmx.Model {
	return defaultModels
}

// GetModel returns a specific model by ID
func (c *Client) GetModel(id string) (*gollmx.Model, error) {
	for _, m := range defaultModels {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, gollmx.NewAPIError(gollmx.ErrorTypeModelNotFound, ProviderID, fmt.Sprintf("model not found: %s", id))
}

// HasFeature checks if a feature is supported
func (c *Client) HasFeature(feature gollmx.Feature) bool {
	switch feature {
	case gollmx.FeatureChat, gollmx.FeatureStreaming, gollmx.FeatureCompletion, gollmx.FeatureEmbedding:
		return true
	case gollmx.FeatureVision:
		return true // Some Ollama models support vision
	case gollmx.FeatureTools:
		return false // Ollama has limited tool support
	default:
		return false
	}
}

// Features returns all supported features
func (c *Client) Features() []gollmx.Feature {
	return []gollmx.Feature{
		gollmx.FeatureChat,
		gollmx.FeatureStreaming,
		gollmx.FeatureCompletion,
		gollmx.FeatureEmbedding,
		gollmx.FeatureVision,
	}
}

// SetOption sets a configuration option
func (c *Client) SetOption(key string, value interface{}) error {
	c.options[key] = value
	return nil
}

// GetOption gets a configuration option
func (c *Client) GetOption(key string) (interface{}, bool) {
	v, ok := c.options[key]
	return v, ok
}

// Chat performs a chat completion request
func (c *Client) Chat(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.config.DefaultModel
		if req.Model == "" {
			req.Model = DefaultModel
		}
	}

	ollamaReq := c.buildChatRequest(req)

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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

	var ollamaResp ChatResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertResponse(&ollamaResp), nil
}

// ChatStream performs a streaming chat completion request
func (c *Client) ChatStream(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.StreamReader, error) {
	if req.Model == "" {
		req.Model = c.config.DefaultModel
		if req.Model == "" {
			req.Model = DefaultModel
		}
	}

	ollamaReq := c.buildChatRequest(req)
	ollamaReq.Stream = true

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp ChatResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			ch <- gollmx.StreamChunk{Error: err}
			return
		}

		chunk := gollmx.StreamChunk{
			ID:       fmt.Sprintf("ollama-%d", resp.CreatedAt.Unix()),
			Provider: ProviderID,
			Model:    resp.Model,
			Content:  resp.Message.Content,
		}

		if resp.Done {
			chunk.FinishReason = "stop"
			chunk.Usage = gollmx.Usage{
				PromptTokens:     resp.PromptEvalCount,
				CompletionTokens: resp.EvalCount,
				TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
			}
		}

		ch <- chunk
	}

	if err := scanner.Err(); err != nil {
		ch <- gollmx.StreamChunk{Error: err}
	}
}

// Complete performs a text completion request (uses chat internally)
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

// Embed performs an embedding request
func (c *Client) Embed(ctx context.Context, req *gollmx.EmbedRequest) (*gollmx.EmbedResponse, error) {
	if req.Model == "" {
		req.Model = "nomic-embed-text"
	}

	ollamaReq := EmbedRequest{
		Model:  req.Model,
		Prompt: req.Input[0], // Ollama takes single prompt
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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

	var ollamaResp EmbedResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &gollmx.EmbedResponse{
		Provider: ProviderID,
		Model:    req.Model,
		Embeddings: []gollmx.Embedding{
			{
				Index:  0,
				Vector: ollamaResp.Embedding,
			},
		},
	}, nil
}

// buildChatRequest converts gollmx.ChatRequest to Ollama format
func (c *Client) buildChatRequest(req *gollmx.ChatRequest) *ChatRequest {
	messages := make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		content, _ := m.Content.(string)
		messages[i] = Message{
			Role:    string(m.Role),
			Content: content,
		}
	}

	ollamaReq := &ChatRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   false,
	}

	// Set options
	options := make(map[string]interface{})
	if req.Temperature != nil {
		options["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		options["top_p"] = *req.TopP
	}
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	}
	if len(req.Stop) > 0 {
		options["stop"] = req.Stop
	}
	if len(options) > 0 {
		ollamaReq.Options = options
	}

	return ollamaReq
}

// convertResponse converts Ollama response to gollmx format
func (c *Client) convertResponse(resp *ChatResponse) *gollmx.ChatResponse {
	return &gollmx.ChatResponse{
		ID:       fmt.Sprintf("ollama-%d", resp.CreatedAt.Unix()),
		Provider: ProviderID,
		Model:    resp.Model,
		Created:  resp.CreatedAt.Unix(),
		Choices: []gollmx.Choice{
			{
				Index: 0,
				Message: gollmx.Message{
					Role:    gollmx.Role(resp.Message.Role),
					Content: resp.Message.Content,
				},
				FinishReason: "stop",
			},
		},
		Usage: gollmx.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
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
		Message:    string(body),
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
