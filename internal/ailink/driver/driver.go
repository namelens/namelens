package driver

import (
	"context"

	"github.com/namelens/namelens/internal/ailink/content"
)

// Driver defines the interface for AI completion providers.
type Driver interface {
	// Complete sends a completion request and returns the response.
	Complete(ctx context.Context, req *Request) (*Response, error)
	// Name returns the driver identifier (e.g., "xai").
	Name() string
	// Capabilities returns what this driver supports.
	Capabilities() Capabilities
}

// Capabilities describes driver features.
type Capabilities struct {
	SupportsTools     bool
	SupportsImages    bool
	SupportsStreaming bool
	SupportedModels   []string
}

// Tool represents a server-side tool.
type Tool struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

// ResponseFormat specifies the expected response format.
type ResponseFormat struct {
	Type string `json:"type"` // "text", "json_object"
}

// Usage contains token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SearchParameters for xAI Live Search (request-level).
type SearchParameters struct {
	Mode            string   `json:"mode,omitempty"`
	ReturnCitations bool     `json:"return_citations,omitempty"`
	Sources         []Source `json:"sources,omitempty"`
}

// Source for search_parameters.sources.
type Source struct {
	Type string `json:"type"`
}

// Request is a provider-agnostic completion request.
type Request struct {
	Model            string
	Messages         []content.Message
	Tools            []Tool
	SearchParameters *SearchParameters
	ResponseFormat   *ResponseFormat
	Temperature      *float64
	MaxTokens        *int
	PromptSlug       string
	Metadata         map[string]string
}

// Response is a provider-agnostic completion response.
type Response struct {
	Content      []content.ContentBlock
	FinishReason string
	Usage        *Usage
	ToolCalls    []ToolCall
}

// ToolCall represents a tool invocation captured by the provider.
type ToolCall struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Name   string         `json:"name,omitempty"`
	Input  map[string]any `json:"input,omitempty"`
	Result map[string]any `json:"result,omitempty"`
}
