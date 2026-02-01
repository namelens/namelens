package ailink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// BulkSearchRequest is a multi-name variant of Search.
//
// It is designed to evaluate a short shortlist (e.g. up to ~10 names) in a single provider call.
type BulkSearchRequest struct {
	Role       string
	Names      []string
	PromptSlug string
	Depth      string
	Model      string
	TimeoutSec int
	UseTools   bool
	IncludeRaw bool
}

// BulkSearchResponse is the validated response for a bulk search.
type BulkSearchResponse struct {
	Summary string           `json:"summary,omitempty"`
	Items   []BulkSearchItem `json:"items"`
	Raw     json.RawMessage  `json:"raw,omitempty"`
}

// BulkSearchItem is a per-name assessment.
//
// Fields intentionally mirror SearchResponse so callers can map easily.
type BulkSearchItem struct {
	Name            string          `json:"name"`
	Summary         string          `json:"summary"`
	LikelyAvailable *bool           `json:"likely_available,omitempty"`
	RiskLevel       string          `json:"risk_level,omitempty"`
	Confidence      *float64        `json:"confidence,omitempty"`
	Insights        []string        `json:"insights,omitempty"`
	Mentions        []SearchMention `json:"mentions,omitempty"`
	Recommendations []string        `json:"recommendations,omitempty"`
}

// SearchBulk runs a bulk expert search using a prompt that accepts a list of names.
func (s *Service) SearchBulk(ctx context.Context, req BulkSearchRequest) (*BulkSearchResponse, error) {
	if s == nil || s.Providers == nil {
		return nil, errors.New("ailink provider registry not configured")
	}
	if s.Registry == nil {
		return nil, errors.New("ailink prompt registry not configured")
	}

	names := normalizeBulkNames(req.Names)
	if len(names) == 0 {
		return nil, errors.New("at least one name is required")
	}

	slug := strings.TrimSpace(req.PromptSlug)
	if slug == "" {
		slug = "name-availability-bulk"
	}

	// Represent names as a stable, human-readable bullet list.
	// Names are validated elsewhere at the CLI boundary, but keep this stable and deterministic.
	var list strings.Builder
	for _, name := range names {
		list.WriteString("- ")
		list.WriteString(name)
		list.WriteByte('\n')
	}

	variables := map[string]string{
		"names": strings.TrimSpace(list.String()),
		"count": fmt.Sprintf("%d", len(names)),
	}
	if strings.TrimSpace(req.Depth) != "" {
		variables["depth"] = strings.TrimSpace(req.Depth)
	}

	gen, err := s.Generate(ctx, GenerateRequest{
		Role:       req.Role,
		PromptSlug: slug,
		Variables:  variables,
		Depth:      req.Depth,
		Model:      req.Model,
		TimeoutSec: req.TimeoutSec,
		UseTools:   req.UseTools,
		IncludeRaw: req.IncludeRaw,
	})
	if err != nil {
		// Allow best-effort recovery of partial results if schema validation failed.
		// This enables callers to optionally fallback for missing items.
		var rawErr *RawResponseError
		if errors.As(err, &rawErr) {
			parsed, decodeErr := decodeBulkSearchResponseLenient(rawErr.Raw)
			if decodeErr == nil && parsed != nil && len(parsed.Items) > 0 {
				if isRawCaptureEnabled(s.Providers.cfg, req.IncludeRaw) {
					parsed.Raw = append(parsed.Raw[:0], rawErr.Raw...)
					parsed.Raw = truncateJSONRaw(parsed.Raw, rawLimit(s.Providers.cfg))
				}
				return parsed, err
			}
		}
		return nil, err
	}

	parsed, err := decodeBulkSearchResponse(gen.Raw)
	if err != nil {
		return nil, &RawResponseError{Err: err, Raw: append(json.RawMessage(nil), gen.Raw...)}
	}

	if isRawCaptureEnabled(s.Providers.cfg, req.IncludeRaw) {
		parsed.Raw = append(parsed.Raw[:0], gen.Raw...)
		parsed.Raw = truncateJSONRaw(parsed.Raw, rawLimit(s.Providers.cfg))
	}

	return parsed, nil
}

func decodeBulkSearchResponseLenient(raw []byte) (*BulkSearchResponse, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}

	itemsRaw, ok := payload["items"].([]any)
	if !ok || len(itemsRaw) == 0 {
		return nil, errors.New("missing required field: items")
	}

	parsed := &BulkSearchResponse{}
	if summary, ok := payload["summary"].(string); ok {
		parsed.Summary = summary
	}

	parsed.Items = make([]BulkSearchItem, 0, len(itemsRaw))
	for _, itemRaw := range itemsRaw {
		obj, ok := itemRaw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := obj["name"].(string)
		summary, _ := obj["summary"].(string)
		name = strings.TrimSpace(name)
		summary = strings.TrimSpace(summary)
		if name == "" || summary == "" {
			continue
		}

		item := BulkSearchItem{Name: name, Summary: summary}
		if v, ok := obj["likely_available"].(bool); ok {
			item.LikelyAvailable = &v
		}
		if v, ok := obj["risk_level"].(string); ok {
			item.RiskLevel = strings.TrimSpace(v)
		}
		if v, ok := obj["confidence"].(float64); ok {
			item.Confidence = &v
		}

		parsed.Items = append(parsed.Items, item)
	}
	if len(parsed.Items) == 0 {
		return nil, errors.New("items contains no valid entries")
	}
	return parsed, nil
}

func decodeBulkSearchResponse(raw []byte) (*BulkSearchResponse, error) {
	var parsed BulkSearchResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Items) == 0 {
		return nil, errors.New("missing required field: items")
	}
	return &parsed, nil
}

func normalizeBulkNames(names []string) []string {
	seen := make(map[string]struct{})
	for _, raw := range names {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
