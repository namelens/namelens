package core

import (
	"strings"

	"github.com/namelens/namelens/internal/ailink"
)

// ExpertDigest is the normalized expert verdict for a single batch. It mirrors
// the row the markdown/table renderers synthesize at render time so that JSON
// consumers see the same digest without having to walk AILink/AILinkError
// themselves.
//
// Populated alongside AILink/AILinkError when expert is invoked for a batch
// (see internal/cmd/check.go). When neither AILink nor AILinkError is set —
// i.e., expert was not invoked — the digest stays nil and is omitted from JSON.
type ExpertDigest struct {
	// Name is the canonical name used in the expert row (chosen by
	// ExpertDisplayName so it matches what users see in markdown).
	Name string `json:"name"`
	// Status is a human-friendly verdict ("risk: high" | "risk: medium" |
	// "risk: low" | "risk: unknown" | "error").
	Status string `json:"status"`
	// Notes is the prose summary on success or the error message on failure.
	Notes string `json:"notes"`
	// RiskLevel is the raw provider-supplied level ("high"/"medium"/"low"/…)
	// without the "risk: " prefix, useful for programmatic consumers. Empty
	// when Status is "error" or when the provider returned no level.
	RiskLevel string `json:"risk_level,omitempty"`
}

// NewExpertDigest builds an ExpertDigest from the same source data the markdown
// renderer uses. Returns nil when there is neither a response nor an error
// (i.e., expert was not invoked for this batch).
func NewExpertDigest(name string, results []*CheckResult, resp *ailink.SearchResponse, errResp *ailink.SearchError) *ExpertDigest {
	if resp == nil && errResp == nil {
		return nil
	}

	digest := &ExpertDigest{Name: ExpertDisplayName(name, results)}

	if errResp != nil {
		digest.Status = "error"
		notes := errResp.Message
		if strings.TrimSpace(notes) == "" {
			notes = errResp.Details
		}
		digest.Notes = notes
		return digest
	}

	level := strings.TrimSpace(resp.RiskLevel)
	if level == "" {
		digest.Status = "risk: unknown"
	} else {
		digest.Status = "risk: " + level
		digest.RiskLevel = level
	}

	notes := strings.TrimSpace(resp.Summary)
	if notes == "" {
		if len(resp.Raw) > 0 {
			notes = "expert analysis complete (see raw JSON in --output=json)"
		} else {
			notes = "expert analysis complete"
		}
	}
	digest.Notes = notes
	return digest
}

// ExpertDisplayName chooses the best name to surface in the expert row: the
// batch's canonical Name if it appears in the per-checker results, otherwise
// the most common candidate inferred from the results themselves, otherwise
// the batch name as-is. Mirrors the logic the markdown renderer used prior
// to v0.2.5 so that pre/post-refactor renderings are byte-identical.
func ExpertDisplayName(name string, results []*CheckResult) string {
	trimmed := strings.TrimSpace(name)
	if trimmed != "" && expertNameAppearsInResults(trimmed, results) {
		return trimmed
	}
	if inferred := expertInferredName(results); inferred != "" {
		return inferred
	}
	return trimmed
}

func expertNameAppearsInResults(name string, results []*CheckResult) bool {
	needle := strings.ToLower(strings.TrimSpace(name))
	if needle == "" {
		return false
	}
	for _, result := range results {
		if expertResultCandidate(result) == needle {
			return true
		}
	}
	return false
}

func expertInferredName(results []*CheckResult) string {
	counts := make(map[string]int)
	order := make([]string, 0, len(results))
	for _, result := range results {
		candidate := expertResultCandidate(result)
		if candidate == "" {
			continue
		}
		if _, exists := counts[candidate]; !exists {
			order = append(order, candidate)
		}
		counts[candidate]++
	}

	best := ""
	bestCount := 0
	for _, candidate := range order {
		if count := counts[candidate]; count > bestCount {
			best = candidate
			bestCount = count
		}
	}
	return best
}

func expertResultCandidate(result *CheckResult) string {
	if result == nil {
		return ""
	}

	name := strings.ToLower(strings.TrimSpace(result.Name))
	if name == "" {
		return ""
	}

	if result.CheckType == CheckTypeGitHub {
		name = strings.TrimPrefix(name, "@")
	}

	if result.CheckType == CheckTypeDomain {
		tld := strings.ToLower(strings.TrimSpace(result.TLD))
		if tld != "" {
			suffix := "." + tld
			if strings.HasSuffix(name, suffix) && len(name) > len(suffix) {
				return strings.TrimSuffix(name, suffix)
			}
		}
		if label, _, ok := strings.Cut(name, "."); ok && label != "" {
			return label
		}
	}

	return name
}
