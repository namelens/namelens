package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink"
	"github.com/namelens/namelens/internal/core"
)

func TestParseFormat(t *testing.T) {
	format, err := ParseFormat("table")
	require.NoError(t, err)
	require.Equal(t, FormatTable, format)

	format, err = ParseFormat("JSON")
	require.NoError(t, err)
	require.Equal(t, FormatJSON, format)

	format, err = ParseFormat("")
	require.NoError(t, err)
	require.Equal(t, FormatTable, format)

	_, err = ParseFormat("csv")
	require.Error(t, err)
}

func TestFormatBatchListJSON(t *testing.T) {
	results := []*core.BatchResult{
		{
			Name:  "alpha",
			Score: 1,
			Total: 1,
			Results: []*core.CheckResult{
				{
					Name:      "alpha",
					CheckType: core.CheckTypeNPM,
					Available: core.AvailabilityAvailable,
				},
			},
		},
	}

	rendered, err := FormatBatchList(FormatJSON, results)
	require.NoError(t, err)
	require.Contains(t, rendered, "\"name\": \"alpha\"")
	require.Contains(t, rendered, "\"check_type\": \"npm\"")
}

func TestFormatters(t *testing.T) {
	result := &core.BatchResult{
		Name:  "delta",
		Score: 1,
		Total: 2,
		Results: []*core.CheckResult{
			{
				Name:      "delta.com",
				CheckType: core.CheckTypeDomain,
				TLD:       "com",
				Available: core.AvailabilityAvailable,
				ExtraData: map[string]any{"expiration": time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)},
			},
			{
				Name:      "delta",
				CheckType: core.CheckTypeNPM,
				Available: core.AvailabilityTaken,
				ExtraData: map[string]any{"latest_version": "1.2.3"},
			},
		},
	}

	tableRendered, err := NewFormatter(FormatTable).FormatBatch(result)
	require.NoError(t, err)
	require.Contains(t, tableRendered, "TYPE")
	require.Contains(t, tableRendered, "domain")
	require.Contains(t, tableRendered, "available")

	jsonRendered, err := NewFormatter(FormatJSON).FormatBatch(result)
	require.NoError(t, err)
	require.Contains(t, jsonRendered, "\"name\": \"delta\"")

	markdownRendered, err := NewFormatter(FormatMarkdown).FormatBatch(result)
	require.NoError(t, err)
	require.Contains(t, markdownRendered, "| Type | Name | Status | Notes |")
	require.Contains(t, markdownRendered, "delta.com")
}

func TestAnalysisRendering(t *testing.T) {
	result := &core.BatchResult{
		Name:  "delta",
		Score: 1,
		Total: 1,
		Results: []*core.CheckResult{
			{
				Name:      "delta.com",
				CheckType: core.CheckTypeDomain,
				Available: core.AvailabilityAvailable,
			},
		},
		Phonetics:   json.RawMessage(`{"syllables":{"count":2,"breakdown":"del-ta"},"pronunciation":{"ipa_primary":"/del-ta/"},"typeability":{"overall_score":82},"cli_suitability":{"score":88},"overall_assessment":{"recommendation":"Solid and easy to say"}}`),
		Suitability: json.RawMessage(`{"overall_suitability":{"score":92,"rating":"suitable","summary":"No major issues"},"risk_assessment":{"legal":{"level":"high"},"profanity":{"level":"clear"}}}`),
	}

	tableRendered, err := NewFormatter(FormatTable).FormatBatch(result)
	require.NoError(t, err)
	require.Contains(t, tableRendered, "Phonetics Analysis")
	require.Contains(t, tableRendered, "Syllables: 2")
	require.Contains(t, tableRendered, "Suitability Analysis")
	require.Contains(t, tableRendered, "Risks: legal=high")

	markdownRendered, err := NewFormatter(FormatMarkdown).FormatBatch(result)
	require.NoError(t, err)
	require.Contains(t, markdownRendered, "### Phonetics Analysis")
	require.Contains(t, markdownRendered, "- Syllables: 2")
	require.Contains(t, markdownRendered, "### Suitability Analysis")
}

func TestDisplayName(t *testing.T) {
	require.Equal(t, "@octocat", displayName(&core.CheckResult{
		Name:      "octocat",
		CheckType: core.CheckTypeGitHub,
	}))
	require.Equal(t, "example.com", displayName(&core.CheckResult{
		Name:      "example.com",
		CheckType: core.CheckTypeDomain,
		TLD:       "com",
	}))
	require.Equal(t, ".com", displayName(&core.CheckResult{
		CheckType: core.CheckTypeDomain,
		TLD:       "com",
	}))
	require.Equal(t, "pkg", displayName(&core.CheckResult{
		Name:      "pkg",
		CheckType: core.CheckTypeNPM,
	}))
}

func TestFormatNotesRateLimited(t *testing.T) {
	result := &core.CheckResult{
		Name:      "alpha",
		CheckType: core.CheckTypeNPM,
		Available: core.AvailabilityRateLimited,
		ExtraData: map[string]any{"retry_after": "10s"},
	}

	notes := formatNotes(result)
	require.Contains(t, notes, "retry: 10s")
}

func TestMarkdownEscaping(t *testing.T) {
	result := &core.BatchResult{
		Name:  "pipe|test",
		Score: 1,
		Total: 1,
		Results: []*core.CheckResult{
			{
				Name:      "pkg",
				CheckType: core.CheckTypePyPI,
				Available: core.AvailabilityTaken,
				ExtraData: map[string]any{"summary": "foo|bar"},
			},
		},
	}

	rendered, err := NewFormatter(FormatMarkdown).FormatBatch(result)
	require.NoError(t, err)
	require.Contains(t, rendered, "pipe\\|test")
	require.Contains(t, rendered, "foo\\|bar")
}

func TestStatusLabel(t *testing.T) {
	require.Equal(t, "rate limited", statusLabel(&core.CheckResult{
		Available: core.AvailabilityRateLimited,
	}))
	require.Equal(t, "unsupported", statusLabel(&core.CheckResult{
		Available: core.AvailabilityUnsupported,
	}))
	require.Equal(t, "error", statusLabel(&core.CheckResult{
		Available: core.AvailabilityError,
	}))
}

func TestFormatBatchListNonJSON(t *testing.T) {
	result := &core.BatchResult{
		Name:  "alpha",
		Score: 1,
		Total: 1,
		Results: []*core.CheckResult{
			{
				Name:      "alpha.com",
				CheckType: core.CheckTypeDomain,
				Available: core.AvailabilityAvailable,
			},
		},
	}

	rendered, err := FormatBatchList(FormatMarkdown, []*core.BatchResult{result})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(rendered, "## "))
}

func TestExpertRowInfersNameFromChecksWhenBatchNameMismatches(t *testing.T) {
	result := &core.BatchResult{
		Name: "ailink",
		Results: []*core.CheckResult{
			{
				CheckType: core.CheckTypeDomain,
				Name:      "idpbolt.com",
				TLD:       "com",
				Available: core.AvailabilityTaken,
			},
			{
				CheckType: core.CheckTypeNPM,
				Name:      "idpbolt",
				Available: core.AvailabilityAvailable,
			},
		},
		AILink: &ailink.SearchResponse{RiskLevel: "low", Summary: "looks clear"},
	}

	rowType, name, status, notes, ok := expertRow(result)
	require.True(t, ok)
	require.Equal(t, "expert", rowType)
	require.Equal(t, "idpbolt", name)
	require.Equal(t, "risk: low", status)
	require.Equal(t, "looks clear", notes)
}

func TestExpertRowKeepsBatchNameWhenItMatchesChecks(t *testing.T) {
	result := &core.BatchResult{
		Name: "ailink",
		Results: []*core.CheckResult{
			{
				CheckType: core.CheckTypeDomain,
				Name:      "ailink.com",
				TLD:       "com",
				Available: core.AvailabilityTaken,
			},
		},
		AILink: &ailink.SearchResponse{RiskLevel: "low", Summary: "ok"},
	}

	_, name, _, _, ok := expertRow(result)
	require.True(t, ok)
	require.Equal(t, "ailink", name)
}

// TestJSONExpertParityWithMarkdownRow asserts the v0.2.5 fix: when expert was
// invoked for a batch (AILink or AILinkError set, Expert digest populated),
// JSON output exposes an `expert` block whose name/status/notes match what the
// markdown renderer's expert row would surface.
func TestJSONExpertParityWithMarkdownRow(t *testing.T) {
	results := []*core.CheckResult{
		{
			CheckType: core.CheckTypeDomain,
			Name:      "acme.com",
			TLD:       "com",
			Available: core.AvailabilityTaken,
		},
	}
	resp := &ailink.SearchResponse{
		RiskLevel: "high",
		Summary:   "synthetic high-risk fixture for the parity test",
	}
	batch := &core.BatchResult{
		Name:    "acme",
		Results: results,
		AILink:  resp,
		Expert:  core.NewExpertDigest("acme", results, resp, nil),
	}

	// Expert digest is populated and matches markdown synthesis.
	require.NotNil(t, batch.Expert)
	_, mdName, mdStatus, mdNotes, ok := expertRow(batch)
	require.True(t, ok)
	require.Equal(t, "acme", mdName)
	require.Equal(t, "risk: high", mdStatus)
	require.Equal(t, resp.Summary, mdNotes)

	// JSON output carries the same digest under the top-level `expert` key.
	rendered, err := FormatBatchList(FormatJSON, []*core.BatchResult{batch})
	require.NoError(t, err)
	var parsed []map[string]any
	require.NoError(t, json.Unmarshal([]byte(rendered), &parsed))
	require.Len(t, parsed, 1)
	expert, ok := parsed[0]["expert"].(map[string]any)
	require.True(t, ok, "JSON output is missing top-level 'expert' digest field; rendered=%s", rendered)
	require.Equal(t, mdName, expert["name"])
	require.Equal(t, mdStatus, expert["status"])
	require.Equal(t, mdNotes, expert["notes"])
	require.Equal(t, "high", expert["risk_level"])
}

// TestJSONExpertOmittedWhenExpertNotInvoked guards against accidentally adding
// an empty expert object to non-expert runs.
func TestJSONExpertOmittedWhenExpertNotInvoked(t *testing.T) {
	batch := &core.BatchResult{
		Name: "alpha",
		Results: []*core.CheckResult{
			{CheckType: core.CheckTypeNPM, Name: "alpha", Available: core.AvailabilityAvailable},
		},
	}
	rendered, err := FormatBatchList(FormatJSON, []*core.BatchResult{batch})
	require.NoError(t, err)
	require.NotContains(t, rendered, "\"expert\":", "non-expert run should omit 'expert' field; rendered=%s", rendered)
}
