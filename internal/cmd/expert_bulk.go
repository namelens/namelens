package cmd

import (
	"context"
	"strings"
	"time"

	"github.com/namelens/namelens/internal/ailink"
)

// expertBulkKey returns the canonical map key used by the --expert-bulk
// merge path: lower-cased and whitespace-trimmed. The same key MUST be used
// at every site that interacts with the bulk results map — live response
// merge, cache hydration, and per-name lookup in the worker — or the merge
// will silently miss on any name whose raw input differs from the bulk
// response's `item.Name` in case or surrounding whitespace.
//
// Pre-v0.2.5 PR-3 the three sites used three different keys:
//   - live merge: lower+trim (canonical-shaped, but isolated)
//   - cache merge: raw item.Name (no normalization)
//   - per-name lookup: raw cobra-args input (no normalization)
//
// The mismatch was invisible when names happened to line up exactly but
// produced ~40% silent merge failures across mixed-case shortlists in
// dogfood evidence. Centralizing here means future call sites can't drift.
func expertBulkKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// bulkItemsToMap converts a slice of BulkSearchItem (fresh from the service
// or hydrated from cache) into the map shape consumed by the per-name
// worker loop. Always uses expertBulkKey on the entry, so the live and
// cache paths produce identically-keyed maps.
func bulkItemsToMap(items []ailink.BulkSearchItem) map[string]*ailink.SearchResponse {
	out := make(map[string]*ailink.SearchResponse, len(items))
	for _, item := range items {
		resp := &ailink.SearchResponse{
			Summary:         item.Summary,
			LikelyAvailable: item.LikelyAvailable,
			RiskLevel:       item.RiskLevel,
			Confidence:      item.Confidence,
			Insights:        item.Insights,
			Mentions:        item.Mentions,
			Recommendations: item.Recommendations,
		}
		out[expertBulkKey(item.Name)] = resp
	}
	return out
}

// waitForFallbackSlot blocks until targetStart or ctx.Done, returning nil
// on a successful wait or a canceled-tagged SearchError if context expired
// during the wait. Extracted in v0.2.5 PR-3 so the cancel branch can be
// exercised in tests and so the worker can fall through to the standard
// batches[job.index] = summarizeResults(...) write rather than bare
// `continue` (which previously left the slot nil and made the missing-
// item case indistinguishable from a never-started job).
func waitForFallbackSlot(ctx context.Context, targetStart time.Time) *ailink.SearchError {
	delay := time.Until(targetStart)
	if delay <= 0 {
		return nil
	}
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return &ailink.SearchError{
			Code:    "AILINK_FALLBACK_CANCELED",
			Message: "expert fallback canceled before retry",
		}
	}
}
