package anthropic

import "encoding/json"

// =============================================================================
// Request Types
// =============================================================================

type anthropicMessagesRequest struct {
	Model       string              `json:"model"`
	Messages    []anthropicMessage  `json:"messages"`
	System      string              `json:"system,omitempty"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature *float64            `json:"temperature,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
	TopK        *int                `json:"top_k,omitempty"`
	StopSeqs    []string            `json:"stop_sequences,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	Tools       []anthropicTool     `json:"tools,omitempty"`
	ToolChoice  *anthropicToolChoice `json:"tool_choice,omitempty"`
	Metadata    *anthropicMetadata  `json:"metadata,omitempty"`
}

type anthropicMessage struct {
	Role    string      `json:"role"` // "user" or "assistant"
	Content interface{} `json:"content"` // string or []anthropicContentBlock
}

type anthropicContentBlock struct {
	Type      string `json:"type"` // "text", "image", "tool_use", "tool_result"
	Text      string `json:"text,omitempty"`

	// For images
	Source    *anthropicImageSource `json:"source,omitempty"`

	// For tool use (assistant response)
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`

	// For tool result (user message)
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"` // string or []anthropicContentBlock
	IsError   bool        `json:"is_error,omitempty"`
}

type anthropicImageSource struct {
	Type      string `json:"type"` // "base64" or "url"
	MediaType string `json:"media_type,omitempty"` // "image/jpeg", "image/png", etc.
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicToolChoice struct {
	Type string `json:"type"` // "auto", "any", "tool"
	Name string `json:"name,omitempty"` // Required when type is "tool"
}

type anthropicMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// =============================================================================
// Response Types
// =============================================================================

type anthropicMessagesResponse struct {
	ID           string                   `json:"id"`
	Type         string                   `json:"type"` // "message"
	Role         string                   `json:"role"` // "assistant"
	Content      []anthropicContentBlock  `json:"content"`
	Model        string                   `json:"model"`
	StopReason   string                   `json:"stop_reason"` // "end_turn", "max_tokens", "stop_sequence", "tool_use"
	StopSequence string                   `json:"stop_sequence,omitempty"`
	Usage        anthropicUsage           `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// =============================================================================
// Streaming Types
// =============================================================================

type anthropicStreamEvent struct {
	Type  string          `json:"type"`
	Index int             `json:"index,omitempty"`
	Delta json.RawMessage `json:"delta,omitempty"`

	// For message_start
	Message *anthropicMessagesResponse `json:"message,omitempty"`

	// For content_block_start
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`

	// For message_delta
	Usage *anthropicStreamUsage `json:"usage,omitempty"`
}

type anthropicStreamDelta struct {
	Type         string `json:"type,omitempty"`
	Text         string `json:"text,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`

	// For tool use delta
	PartialJSON string `json:"partial_json,omitempty"`
}

type anthropicStreamUsage struct {
	OutputTokens int `json:"output_tokens"`
}

// =============================================================================
// Error Types
// =============================================================================

type anthropicErrorResponse struct {
	Type  string          `json:"type"` // "error"
	Error *anthropicError `json:"error"`
}

type anthropicError struct {
	Type    string `json:"type"`    // "invalid_request_error", "authentication_error", etc.
	Message string `json:"message"`
}
