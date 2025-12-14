package google

import "encoding/json"

// =============================================================================
// Request Types
// =============================================================================

type geminiGenerateRequest struct {
	Contents         []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
	Tools            []geminiTool           `json:"tools,omitempty"`
	ToolConfig       *geminiToolConfig      `json:"toolConfig,omitempty"`
	SafetySettings   []geminiSafetySetting  `json:"safetySettings,omitempty"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"` // "user", "model", "function"
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	InlineData   *geminiInlineData   `json:"inlineData,omitempty"`
	FileData     *geminiFileData     `json:"fileData,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResp *geminiFunctionResp `json:"functionResponse,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // Base64 encoded
}

type geminiFileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

type geminiFunctionCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type geminiFunctionResp struct {
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations,omitempty"`
}

type geminiFunctionDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type geminiToolConfig struct {
	FunctionCallingConfig *geminiFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type geminiFunctionCallingConfig struct {
	Mode             string   `json:"mode,omitempty"` // "AUTO", "ANY", "NONE"
	AllowedFunctions []string `json:"allowedFunctionNames,omitempty"`
}

type geminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type geminiGenerationConfig struct {
	StopSequences   []string `json:"stopSequences,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	CandidateCount  int      `json:"candidateCount,omitempty"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"` // "text/plain" or "application/json"
}

// =============================================================================
// Response Types
// =============================================================================

type geminiGenerateResponse struct {
	Candidates     []geminiCandidate    `json:"candidates"`
	PromptFeedback *geminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *geminiUsageMetadata  `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content       *geminiContent       `json:"content"`
	FinishReason  string               `json:"finishReason"`
	SafetyRatings []geminiSafetyRating `json:"safetyRatings,omitempty"`
	Index         int                  `json:"index"`
}

type geminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type geminiPromptFeedback struct {
	SafetyRatings []geminiSafetyRating `json:"safetyRatings,omitempty"`
	BlockReason   string               `json:"blockReason,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// =============================================================================
// Streaming Types
// =============================================================================

// Streaming uses the same response format with partial candidates

// =============================================================================
// Embedding Types
// =============================================================================

type geminiEmbedRequest struct {
	Model   string             `json:"model"`
	Content geminiEmbedContent `json:"content"`
}

type geminiEmbedContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiBatchEmbedRequest struct {
	Requests []geminiEmbedRequest `json:"requests"`
}

type geminiEmbedResponse struct {
	Embedding *geminiEmbedding `json:"embedding"`
}

type geminiBatchEmbedResponse struct {
	Embeddings []geminiEmbedding `json:"embeddings"`
}

type geminiEmbedding struct {
	Values []float64 `json:"values"`
}

// =============================================================================
// Error Types
// =============================================================================

type geminiErrorResponse struct {
	Error *geminiError `json:"error"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}
