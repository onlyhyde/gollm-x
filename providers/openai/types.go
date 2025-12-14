package openai

import "encoding/json"

// =============================================================================
// Request Types
// =============================================================================

type openAIChatRequest struct {
	Model          string               `json:"model"`
	Messages       []openAIMessage      `json:"messages"`
	MaxTokens      int                  `json:"max_tokens,omitempty"`
	Temperature    *float64             `json:"temperature,omitempty"`
	TopP           *float64             `json:"top_p,omitempty"`
	Stop           []string             `json:"stop,omitempty"`
	Stream         bool                 `json:"stream,omitempty"`
	StreamOptions  *streamOptions       `json:"stream_options,omitempty"`
	Tools          []openAITool         `json:"tools,omitempty"`
	ToolChoice     interface{}          `json:"tool_choice,omitempty"`
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"` // string or []content_part
	Name       string           `json:"name,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      bool            `json:"strict,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIResponseFormat struct {
	Type       string           `json:"type"`
	JSONSchema *openAIJSONSchema `json:"json_schema,omitempty"`
}

type openAIJSONSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema"`
	Strict      bool            `json:"strict,omitempty"`
}

type openAIEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// =============================================================================
// Response Types
// =============================================================================

type openAIChatResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []openAIChoice  `json:"choices"`
	Usage   openAIUsage     `json:"usage"`
}

type openAIChoice struct {
	Index        int                `json:"index"`
	Message      openAIMessageResp  `json:"message"`
	FinishReason string             `json:"finish_reason"`
}

type openAIMessageResp struct {
	Role      string           `json:"role"`
	Content   interface{}      `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIStreamChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage        `json:"usage,omitempty"`
}

type openAIStreamChoice struct {
	Index        int                 `json:"index"`
	Delta        openAIStreamDelta   `json:"delta"`
	FinishReason string              `json:"finish_reason"`
}

type openAIStreamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIEmbedResponse struct {
	Object string              `json:"object"`
	Data   []openAIEmbedData   `json:"data"`
	Model  string              `json:"model"`
	Usage  openAIEmbedUsage    `json:"usage"`
}

type openAIEmbedData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type openAIEmbedUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// =============================================================================
// Error Types
// =============================================================================

type openAIErrorResponse struct {
	Error *openAIError `json:"error"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}
