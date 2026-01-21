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

// ImageGenerator is an optional interface implemented by drivers that support image generation.
//
// This keeps the core Driver interface stable for text-only providers.
type ImageGenerator interface {
	GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error)
}

type ImageRequest struct {
	Model        string
	Prompt       string
	Count        int
	Size         string
	OutputFormat string
	Background   string
	Quality      string
	PromptSlug   string
}

type ImageResponse struct {
	Created      int64
	OutputFormat string
	Size         string
	Quality      string
	Images       []content.ContentBlock
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
//
// Note: Some providers (e.g. OpenAI) support additional structured modes such as
// "json_schema".
type ResponseFormat struct {
	Type string `json:"type"` // "text", "json_object", "json_schema"

	// JSONSchema is provider-specific configuration for strict JSON schema output.
	// When Type is "json_schema", drivers may use this to request schema-constrained output.
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

type JSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
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
