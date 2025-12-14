// Package google provides a Google Gemini API client for gollm-x.
package google

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
	ProviderID     = "google"
	ProviderName   = "Google Gemini"
	DefaultBaseURL = "https://generativelanguage.googleapis.com"
	ClientVersion  = "1.0.0"
)

// Client implements the gollmx.LLM interface for Google Gemini
type Client struct {
	config     *gollmx.Config
	httpClient *http.Client
	baseURL    string
	options    map[string]interface{}
}

func init() {
	gollmx.Register(ProviderID, NewClient)
}

// NewClient creates a new Google Gemini client
func NewClient(opts ...gollmx.Option) (gollmx.LLM, error) {
	config := gollmx.DefaultConfig()
	config.Apply(opts...)

	// Try to get API key from environment if not provided
	if config.APIKey == "" {
		config.APIKey = os.Getenv("GOOGLE_API_KEY")
		if config.APIKey == "" {
			config.APIKey = os.Getenv("GEMINI_API_KEY")
		}
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
		options:    make(map[string]interface{}),
	}, nil
}

// Provider information methods
func (c *Client) ID() string      { return ProviderID }
func (c *Client) Name() string    { return ProviderName }
func (c *Client) Version() string { return ClientVersion }
func (c *Client) BaseURL() string { return c.baseURL }

// Models returns all available Gemini models
func (c *Client) Models() []gollmx.Model {
	return GeminiModels
}

// GetModel returns information about a specific model
func (c *Client) GetModel(id string) (*gollmx.Model, error) {
	for _, model := range GeminiModels {
		if model.ID == id {
			return &model, nil
		}
	}
	return nil, gollmx.NewAPIError(gollmx.ErrorTypeModelNotFound, ProviderID, fmt.Sprintf("model not found: %s", id))
}

// Chat sends a chat request to Gemini's generateContent API
func (c *Client) Chat(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.ChatResponse, error) {
	geminiReq := c.convertChatRequest(req)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", c.baseURL, req.Model, c.config.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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

	var geminiResp geminiGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertChatResponse(req.Model, &geminiResp), nil
}

// ChatStream sends a streaming chat request
func (c *Client) ChatStream(ctx context.Context, req *gollmx.ChatRequest) (*gollmx.StreamReader, error) {
	geminiReq := c.convertChatRequest(req)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", c.baseURL, req.Model, c.config.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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
	go c.processStream(resp, req.Model, ch)

	return gollmx.NewStreamReader(ch), nil
}

// Complete converts to chat request for Gemini
func (c *Client) Complete(ctx context.Context, req *gollmx.CompletionRequest) (*gollmx.CompletionResponse, error) {
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

// Embed creates embeddings using Gemini's embedding model
func (c *Client) Embed(ctx context.Context, req *gollmx.EmbedRequest) (*gollmx.EmbedResponse, error) {
	model := req.Model
	if model == "" {
		model = "text-embedding-004"
	}

	// Use batch embedding for multiple inputs
	var requests []geminiEmbedRequest
	for _, text := range req.Input {
		requests = append(requests, geminiEmbedRequest{
			Model: fmt.Sprintf("models/%s", model),
			Content: geminiEmbedContent{
				Parts: []geminiPart{{Text: text}},
			},
		})
	}

	batchReq := geminiBatchEmbedRequest{Requests: requests}
	body, err := json.Marshal(batchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:batchEmbedContents?key=%s", c.baseURL, model, c.config.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var batchResp geminiBatchEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var embeddings []gollmx.Embedding
	for i, emb := range batchResp.Embeddings {
		embeddings = append(embeddings, gollmx.Embedding{
			Index:  i,
			Vector: emb.Values,
		})
	}

	return &gollmx.EmbedResponse{
		Provider:   ProviderID,
		Model:      model,
		Embeddings: embeddings,
	}, nil
}

// HasFeature checks if the provider supports a feature
func (c *Client) HasFeature(feature gollmx.Feature) bool {
	switch feature {
	case gollmx.FeatureChat, gollmx.FeatureStreaming, gollmx.FeatureVision,
		gollmx.FeatureTools, gollmx.FeatureJSON, gollmx.FeatureSystemPrompt,
		gollmx.FeatureEmbedding:
		return true
	case gollmx.FeatureCompletion:
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
// Private methods
// =============================================================================

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

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

	var errResp geminiErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return gollmx.NewAPIError(gollmx.ErrorTypeUnknown, ProviderID, string(body))
	}

	apiErr := &gollmx.APIError{
		Provider:   ProviderID,
		StatusCode: resp.StatusCode,
		Message:    errResp.Error.Message,
		Code:       errResp.Error.Status,
		Raw:        errResp,
	}

	switch resp.StatusCode {
	case 401, 403:
		apiErr.Type = gollmx.ErrorTypeAuth
	case 429:
		apiErr.Type = gollmx.ErrorTypeRateLimit
		apiErr.Retryable = true
	case 400:
		apiErr.Type = gollmx.ErrorTypeInvalidRequest
	case 500, 502, 503:
		apiErr.Type = gollmx.ErrorTypeServer
		apiErr.Retryable = true
	default:
		apiErr.Type = gollmx.ErrorTypeUnknown
	}

	return apiErr
}

func (c *Client) convertChatRequest(req *gollmx.ChatRequest) *geminiGenerateRequest {
	var contents []geminiContent
	var systemInstruction *geminiContent

	for _, msg := range req.Messages {
		switch msg.Role {
		case gollmx.RoleSystem:
			if content, ok := msg.Content.(string); ok {
				systemInstruction = &geminiContent{
					Parts: []geminiPart{{Text: content}},
				}
			}
		case gollmx.RoleUser:
			contents = append(contents, c.convertMessage("user", msg))
		case gollmx.RoleAssistant:
			contents = append(contents, c.convertMessage("model", msg))
		case gollmx.RoleTool:
			// Tool results
			if content, ok := msg.Content.(string); ok {
				contents = append(contents, geminiContent{
					Role: "function",
					Parts: []geminiPart{{
						FunctionResp: &geminiFunctionResp{
							Name:     msg.Name,
							Response: json.RawMessage(content),
						},
					}},
				})
			}
		}
	}

	geminiReq := &geminiGenerateRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
	}

	// Generation config
	genConfig := &geminiGenerationConfig{}
	hasConfig := false

	if req.MaxTokens > 0 {
		genConfig.MaxOutputTokens = req.MaxTokens
		hasConfig = true
	}
	if req.Temperature != nil {
		genConfig.Temperature = req.Temperature
		hasConfig = true
	}
	if req.TopP != nil {
		genConfig.TopP = req.TopP
		hasConfig = true
	}
	if len(req.Stop) > 0 {
		genConfig.StopSequences = req.Stop
		hasConfig = true
	}
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" {
		genConfig.ResponseMimeType = "application/json"
		hasConfig = true
	}

	if hasConfig {
		geminiReq.GenerationConfig = genConfig
	}

	// Convert tools
	if len(req.Tools) > 0 {
		geminiReq.Tools = c.convertTools(req.Tools)
	}

	return geminiReq
}

func (c *Client) convertMessage(role string, msg gollmx.Message) geminiContent {
	var parts []geminiPart

	switch content := msg.Content.(type) {
	case string:
		parts = append(parts, geminiPart{Text: content})

	case []gollmx.ContentPart:
		for _, part := range content {
			switch part.Type {
			case "text":
				parts = append(parts, geminiPart{Text: part.Text})
			case "image_url":
				if part.ImageURL != nil {
					// Gemini requires base64 inline data or file URI
					// For URL, we'd need to fetch and convert
					parts = append(parts, geminiPart{
						InlineData: &geminiInlineData{
							MimeType: "image/jpeg", // Would need to detect
							Data:     part.ImageURL.URL,
						},
					})
				}
			}
		}
	}

	// Add tool calls from assistant messages
	for _, tc := range msg.ToolCalls {
		parts = append(parts, geminiPart{
			FunctionCall: &geminiFunctionCall{
				Name: tc.Function.Name,
				Args: json.RawMessage(tc.Function.Arguments),
			},
		})
	}

	return geminiContent{Role: role, Parts: parts}
}

func (c *Client) convertTools(tools []gollmx.Tool) []geminiTool {
	var functions []geminiFunctionDecl
	for _, tool := range tools {
		if tool.Type == "function" {
			functions = append(functions, geminiFunctionDecl{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			})
		}
	}

	if len(functions) > 0 {
		return []geminiTool{{FunctionDeclarations: functions}}
	}
	return nil
}

func (c *Client) convertChatResponse(model string, resp *geminiGenerateResponse) *gollmx.ChatResponse {
	var choices []gollmx.Choice

	for _, candidate := range resp.Candidates {
		var content string
		var toolCalls []gollmx.ToolCall

		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					content += part.Text
				}
				if part.FunctionCall != nil {
					toolCalls = append(toolCalls, gollmx.ToolCall{
						ID:   fmt.Sprintf("call_%d", len(toolCalls)),
						Type: "function",
						Function: gollmx.FunctionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(part.FunctionCall.Args),
						},
					})
				}
			}
		}

		finishReason := c.convertFinishReason(candidate.FinishReason)

		choices = append(choices, gollmx.Choice{
			Index: candidate.Index,
			Message: gollmx.Message{
				Role:      gollmx.RoleAssistant,
				Content:   content,
				ToolCalls: toolCalls,
			},
			FinishReason: finishReason,
		})
	}

	usage := gollmx.Usage{}
	if resp.UsageMetadata != nil {
		usage.PromptTokens = resp.UsageMetadata.PromptTokenCount
		usage.CompletionTokens = resp.UsageMetadata.CandidatesTokenCount
		usage.TotalTokens = resp.UsageMetadata.TotalTokenCount
	}

	return &gollmx.ChatResponse{
		ID:       fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
		Provider: ProviderID,
		Model:    model,
		Created:  time.Now().Unix(),
		Choices:  choices,
		Usage:    usage,
		Raw:      resp,
	}
}

func (c *Client) convertFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	case "OTHER":
		return "stop"
	default:
		return reason
	}
}

func (c *Client) processStream(resp *http.Response, model string, ch chan<- gollmx.StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

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

		var geminiResp geminiGenerateResponse
		if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
			continue
		}

		for _, candidate := range geminiResp.Candidates {
			if candidate.Content == nil {
				continue
			}

			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					chunk := gollmx.StreamChunk{
						ID:       fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
						Provider: ProviderID,
						Model:    model,
						Content:  part.Text,
					}

					if candidate.FinishReason != "" {
						chunk.FinishReason = c.convertFinishReason(candidate.FinishReason)
					}

					if geminiResp.UsageMetadata != nil {
						chunk.Usage = gollmx.Usage{
							PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
							CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
							TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
						}
					}

					ch <- chunk
				}
			}
		}
	}
}
