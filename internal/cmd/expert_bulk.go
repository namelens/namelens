package cmd

import (
	"context"
	"strings"
	"time"

	"github.com/namelens/namelens/internal/ailink"
)

// expertBulkKey returns the canonical map key for the --expert-bulk merge
// path: lower-cased and whitespace-trimmed. Every site that interacts with
// the bulk results map — live response merge, cache hydration, and the
// per-name lookup in the worker — must use this function. Otherwise the
// merge silently misses when a name's raw input differs from the bulk
// response's item.Name in case or surrounding whitespace.
func expertBulkKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// bulkItemsToMap converts a slice of BulkSearchItem (live response or cache
// hydration) into the map shape consumed by the per-name worker loop,
// keyed by expertBulkKey so live and cache paths produce identically-keyed
// maps.
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
// during the wait. The error-tagged return lets the worker produce a
// populated batch slot for canceled fallbacks rather than leaving them
// indistinguishable from never-started jobs.
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
