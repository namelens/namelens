package ailink

import "encoding/json"

// SearchRequest is the high-level request for an expert search.
type SearchRequest struct {
	Role       string
	Name       string
	PromptSlug string
	Depth      string
	Model      string
	TimeoutSec int
	UseTools   bool
}

// SearchResponse captures the parsed response plus raw JSON.
type SearchResponse struct {
	Summary         string          `json:"summary"`
	LikelyAvailable *bool           `json:"likely_available,omitempty"`
	RiskLevel       string          `json:"risk_level,omitempty"`
	Confidence      *float64        `json:"confidence,omitempty"`
	Insights        []string        `json:"insights,omitempty"`
	Mentions        []SearchMention `json:"mentions,omitempty"`
	Recommendations []string        `json:"recommendations,omitempty"`
	Raw             json.RawMessage `json:"raw,omitempty"`
}

// SearchMention represents a single mention returned by the model.
type SearchMention struct {
	Source      string `json:"source"`
	Description string `json:"description"`
	URL         string `json:"url,omitempty"`
	Relevance   string `json:"relevance,omitempty"`
	Sentiment   string `json:"sentiment,omitempty"`
	Date        string `json:"date,omitempty"`
}

// SearchError captures an ailink failure without breaking the command.
type SearchError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// GenerateRequest is the high-level request for name generation.
type GenerateRequest struct {
	Role       string
	PromptSlug string
	Variables  map[string]string
	Depth      string
	Model      string
	TimeoutSec int
	UseTools   bool
}

// GenerateResponse captures the raw JSON response from generation prompts.
type GenerateResponse struct {
	Raw json.RawMessage `json:"raw"`
}
