package core

import (
	"encoding/json"
	"time"

	"github.com/namelens/namelens/internal/ailink"
)

// BatchResult captures the results for a single name check.
type BatchResult struct {
	Name        string                 `json:"name"`
	Results     []*CheckResult         `json:"results"`
	Score       int                    `json:"score"`
	Total       int                    `json:"total"`
	Unknown     int                    `json:"unknown"`
	CompletedAt time.Time              `json:"completed_at"`
	AILink      *ailink.SearchResponse `json:"ailink,omitempty"`
	AILinkError *ailink.SearchError    `json:"ailink_error,omitempty"`
	// Expert is a normalized digest of the AILink/AILinkError pair that
	// mirrors the row the markdown/table renderers synthesize. Added in
	// v0.2.5 so JSON consumers see the same digest the markdown renderer
	// shows without walking the raw AILink payload themselves. Nil when
	// expert was not invoked for this batch.
	Expert           *ExpertDigest       `json:"expert,omitempty"`
	Phonetics        json.RawMessage     `json:"phonetics,omitempty"`
	PhoneticsError   *ailink.SearchError `json:"phonetics_error,omitempty"`
	Suitability      json.RawMessage     `json:"suitability,omitempty"`
	SuitabilityError *ailink.SearchError `json:"suitability_error,omitempty"`
}
