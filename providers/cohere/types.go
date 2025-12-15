package cohere

// =============================================================================
// Request Types
// =============================================================================

type chatRequest struct {
	Model         string        `json:"model"`
	Message       string        `json:"message"`
	Preamble      string        `json:"preamble,omitempty"`
	ChatHistory   []chatMessage `json:"chat_history,omitempty"`
	MaxTokens     int           `json:"max_tokens,omitempty"`
	Temperature   *float64      `json:"temperature,omitempty"`
	P             *float64      `json:"p,omitempty"`
	StopSequences []string      `json:"stop_sequences,omitempty"`
	Stream        bool          `json:"stream,omitempty"`
	Tools         []tool        `json:"tools,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Message string `json:"message"`
}

type tool struct {
	Name                 string                  `json:"name"`
	Description          string                  `json:"description"`
	ParameterDefinitions map[string]parameterDef `json:"parameter_definitions,omitempty"`
}

type parameterDef struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type embedRequest struct {
	Model     string   `json:"model"`
	Texts     []string `json:"texts"`
	InputType string   `json:"input_type"`
}

// =============================================================================
// Response Types
// =============================================================================

type chatResponse struct {
	GenerationID string        `json:"generation_id"`
	Text         string        `json:"text"`
	FinishReason finishReason  `json:"finish_reason"`
	ChatHistory  []chatMessage `json:"chat_history,omitempty"`
	Meta         *meta         `json:"meta,omitempty"`
	ToolCalls    []toolCall    `json:"tool_calls,omitempty"`
}

type finishReason string

const (
	FinishReasonComplete   finishReason = "COMPLETE"
	FinishReasonStopSequence finishReason = "STOP_SEQUENCE"
	FinishReasonMaxTokens  finishReason = "MAX_TOKENS"
	FinishReasonToolCall   finishReason = "TOOL_CALL"
)

type meta struct {
	APIVersion  apiVersion `json:"api_version,omitempty"`
	Tokens      *tokens    `json:"tokens,omitempty"`
	BilledUnits *billedUnits `json:"billed_units,omitempty"`
}

type apiVersion struct {
	Version string `json:"version"`
}

type tokens struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type billedUnits struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type toolCall struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

type streamEvent struct {
	EventType string        `json:"event_type"`
	Text      string        `json:"text,omitempty"`
	Response  *chatResponse `json:"response,omitempty"`
}

type embedResponse struct {
	ID         string      `json:"id"`
	Embeddings [][]float64 `json:"embeddings"`
	Meta       embedMeta   `json:"meta"`
}

type embedMeta struct {
	APIVersion  apiVersion  `json:"api_version"`
	BilledUnits billedUnits `json:"billed_units"`
}

// =============================================================================
// Error Types
// =============================================================================

type errorResponse struct {
	Message string `json:"message"`
}
