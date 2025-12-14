package gollmx

import (
	"encoding/json"
	"time"
)

// =============================================================================
// Model Information
// =============================================================================

// Model represents an LLM model's metadata
type Model struct {
	ID           string   `json:"id"`           // Model identifier (e.g., "gpt-4o")
	Name         string   `json:"name"`         // Human-readable name
	Provider     string   `json:"provider"`     // Provider ID
	Description  string   `json:"description"`  // Model description
	ContextWindow int     `json:"contextWindow"` // Maximum context length in tokens
	MaxOutput    int      `json:"maxOutput"`    // Maximum output tokens
	InputPrice   float64  `json:"inputPrice"`   // Price per 1M input tokens (USD)
	OutputPrice  float64  `json:"outputPrice"`  // Price per 1M output tokens (USD)
	Features     []Feature `json:"features"`    // Supported features
	Deprecated   bool     `json:"deprecated"`   // Whether the model is deprecated
	ReleaseDate  string   `json:"releaseDate"`  // Release date (YYYY-MM-DD)
}

// SupportsFeature checks if the model supports a specific feature
func (m *Model) SupportsFeature(f Feature) bool {
	for _, feature := range m.Features {
		if feature == f {
			return true
		}
	}
	return false
}

// =============================================================================
// Chat Types
// =============================================================================

// Role represents the role of a message sender
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a single message in a conversation
type Message struct {
	Role       Role        `json:"role"`
	Content    interface{} `json:"content"` // string or []ContentPart for multimodal
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ContentPart represents a part of multimodal content
type ContentPart struct {
	Type     string    `json:"type"` // "text", "image_url", "image_base64"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image reference
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// TextContent creates a text content part
func TextContent(text string) ContentPart {
	return ContentPart{Type: "text", Text: text}
}

// ImageURLContent creates an image URL content part
func ImageURLContent(url string, detail string) ContentPart {
	return ContentPart{
		Type:     "image_url",
		ImageURL: &ImageURL{URL: url, Detail: detail},
	}
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
	Stream      bool      `json:"stream,omitempty"`

	// Tool/Function calling
	Tools      []Tool  `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"` // "auto", "none", or specific tool

	// Response format
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// Provider-specific options (passed through)
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// ResponseFormat specifies the format of the response
type ResponseFormat struct {
	Type       string          `json:"type"` // "text", "json_object", "json_schema"
	JSONSchema *JSONSchema     `json:"json_schema,omitempty"`
}

// JSONSchema represents a JSON schema for structured output
type JSONSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema"`
	Strict      bool            `json:"strict,omitempty"`
}

// Tool represents a tool/function that the model can call
type Tool struct {
	Type     string   `json:"type"` // "function"
	Function Function `json:"function"`
}

// Function represents a callable function
type Function struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"` // JSON Schema
	Strict      bool            `json:"strict,omitempty"`
}

// ToolCall represents a tool call made by the model
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function being called
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID        string   `json:"id"`
	Provider  string   `json:"provider"`
	Model     string   `json:"model"`
	Created   int64    `json:"created"`
	Choices   []Choice `json:"choices"`
	Usage     Usage    `json:"usage"`

	// Provider-specific data
	Raw interface{} `json:"raw,omitempty"`
}

// Choice represents a single completion choice
type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	FinishReason string   `json:"finish_reason"` // "stop", "length", "tool_calls", "content_filter"
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// GetContent returns the text content of the first choice
func (r *ChatResponse) GetContent() string {
	if len(r.Choices) == 0 {
		return ""
	}
	if content, ok := r.Choices[0].Message.Content.(string); ok {
		return content
	}
	return ""
}

// GetToolCalls returns tool calls from the first choice
func (r *ChatResponse) GetToolCalls() []ToolCall {
	if len(r.Choices) == 0 {
		return nil
	}
	return r.Choices[0].Message.ToolCalls
}

// =============================================================================
// Streaming Types
// =============================================================================

// StreamReader provides an iterator interface for streaming responses
type StreamReader struct {
	ch     <-chan StreamChunk
	err    error
	closed bool
}

// NewStreamReader creates a new StreamReader
func NewStreamReader(ch <-chan StreamChunk) *StreamReader {
	return &StreamReader{ch: ch}
}

// Next returns the next chunk, or false if the stream is exhausted
func (r *StreamReader) Next() (*StreamChunk, bool) {
	if r.closed {
		return nil, false
	}
	chunk, ok := <-r.ch
	if !ok {
		r.closed = true
		return nil, false
	}
	if chunk.Error != nil {
		r.err = chunk.Error
		return nil, false
	}
	return &chunk, true
}

// Err returns any error that occurred during streaming
func (r *StreamReader) Err() error {
	return r.err
}

// Collect reads all chunks and returns the complete response
func (r *StreamReader) Collect() (*ChatResponse, error) {
	var response ChatResponse
	var content string
	var toolCalls []ToolCall

	for {
		chunk, ok := r.Next()
		if !ok {
			break
		}
		if chunk.Content != "" {
			content += chunk.Content
		}
		if len(chunk.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}
		response.ID = chunk.ID
		response.Model = chunk.Model
		response.Provider = chunk.Provider
		if chunk.FinishReason != "" {
			response.Choices = []Choice{{
				Index:        0,
				Message:      Message{Role: RoleAssistant, Content: content, ToolCalls: toolCalls},
				FinishReason: chunk.FinishReason,
			}}
		}
		response.Usage = chunk.Usage
	}

	if r.err != nil {
		return nil, r.err
	}

	if len(response.Choices) == 0 {
		response.Choices = []Choice{{
			Index:   0,
			Message: Message{Role: RoleAssistant, Content: content, ToolCalls: toolCalls},
		}}
	}

	return &response, nil
}

// StreamChunk represents a single chunk in a streaming response
type StreamChunk struct {
	ID           string     `json:"id"`
	Provider     string     `json:"provider"`
	Model        string     `json:"model"`
	Content      string     `json:"content"`       // Delta content
	ToolCalls    []ToolCall `json:"tool_calls"`    // Delta tool calls
	FinishReason string     `json:"finish_reason"`
	Usage        Usage      `json:"usage"`
	Error        error      `json:"error,omitempty"`
}

// =============================================================================
// Completion Types (Legacy)
// =============================================================================

// CompletionRequest represents a text completion request
type CompletionRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
	Echo        bool     `json:"echo,omitempty"`

	Extra map[string]interface{} `json:"extra,omitempty"`
}

// CompletionResponse represents a text completion response
type CompletionResponse struct {
	ID       string             `json:"id"`
	Provider string             `json:"provider"`
	Model    string             `json:"model"`
	Created  int64              `json:"created"`
	Choices  []CompletionChoice `json:"choices"`
	Usage    Usage              `json:"usage"`
	Raw      interface{}        `json:"raw,omitempty"`
}

// CompletionChoice represents a single completion choice
type CompletionChoice struct {
	Index        int    `json:"index"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

// GetText returns the text of the first choice
func (r *CompletionResponse) GetText() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Text
}

// =============================================================================
// Embedding Types
// =============================================================================

// EmbedRequest represents an embedding request
type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"` // Text(s) to embed

	Extra map[string]interface{} `json:"extra,omitempty"`
}

// EmbedResponse represents an embedding response
type EmbedResponse struct {
	Provider   string       `json:"provider"`
	Model      string       `json:"model"`
	Embeddings []Embedding  `json:"embeddings"`
	Usage      Usage        `json:"usage"`
	Raw        interface{}  `json:"raw,omitempty"`
}

// Embedding represents a single embedding vector
type Embedding struct {
	Index  int       `json:"index"`
	Vector []float64 `json:"vector"`
}

// =============================================================================
// Error Types
// =============================================================================

// ErrorType categorizes API errors
type ErrorType string

const (
	ErrorTypeAuth          ErrorType = "authentication"
	ErrorTypeRateLimit     ErrorType = "rate_limit"
	ErrorTypeInvalidRequest ErrorType = "invalid_request"
	ErrorTypeServer        ErrorType = "server"
	ErrorTypeNetwork       ErrorType = "network"
	ErrorTypeTimeout       ErrorType = "timeout"
	ErrorTypeContentFilter ErrorType = "content_filter"
	ErrorTypeModelNotFound ErrorType = "model_not_found"
	ErrorTypeQuota         ErrorType = "quota_exceeded"
	ErrorTypeUnknown       ErrorType = "unknown"
)

// APIError represents an error from an LLM API
type APIError struct {
	Type       ErrorType `json:"type"`
	Provider   string    `json:"provider"`
	StatusCode int       `json:"status_code,omitempty"`
	Message    string    `json:"message"`
	Code       string    `json:"code,omitempty"`     // Provider-specific error code
	Param      string    `json:"param,omitempty"`    // Parameter that caused the error
	Retryable  bool      `json:"retryable"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	Raw        interface{} `json:"raw,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError creates a new APIError
func NewAPIError(errType ErrorType, provider string, message string) *APIError {
	return &APIError{
		Type:     errType,
		Provider: provider,
		Message:  message,
	}
}

// IsRetryable returns true if the error can be retried
func (e *APIError) IsRetryable() bool {
	return e.Retryable
}
