package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink"
)

func TestApplyGenerateProviderOverrideSetsRoleRouting(t *testing.T) {
	cfg := ailink.Config{
		Providers: map[string]ailink.ProviderInstanceConfig{
			"namelens-openai": {Enabled: true},
		},
		Routing: map[string]string{
			"name-alternatives": "namelens-xai",
		},
	}

	out, err := applyGenerateProviderOverride(cfg, "name-alternatives", "namelens-openai")
	require.NoError(t, err)
	require.Equal(t, "namelens-openai", out.Routing["name-alternatives"])
	require.Equal(t, "namelens-xai", cfg.Routing["name-alternatives"], "source config must not be mutated")
}

func TestApplyGenerateProviderOverrideUnknownProvider(t *testing.T) {
	cfg := ailink.Config{
		Providers: map[string]ailink.ProviderInstanceConfig{
			"namelens-anthropic": {Enabled: true},
			"namelens-openai":    {Enabled: true},
		},
	}

	_, err := applyGenerateProviderOverride(cfg, "name-alternatives", "missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown provider "missing"`)
	require.Contains(t, err.Error(), "namelens-anthropic, namelens-openai")
}

func TestApplyGenerateProviderOverrideDisabledProvider(t *testing.T) {
	cfg := ailink.Config{
		Providers: map[string]ailink.ProviderInstanceConfig{
			"namelens-openai": {Enabled: false},
		},
	}

	_, err := applyGenerateProviderOverride(cfg, "name-alternatives", "namelens-openai")
	require.Error(t, err)
	require.Equal(t, `provider "namelens-openai" is disabled`, err.Error())
}

func TestApplyGenerateProviderOverrideEmptyProviderNoop(t *testing.T) {
	cfg := ailink.Config{
		Providers: map[string]ailink.ProviderInstanceConfig{
			"namelens-openai": {Enabled: true},
		},
	}

	out, err := applyGenerateProviderOverride(cfg, "name-alternatives", "")
	require.NoError(t, err)
	require.Equal(t, cfg, out)
}

func TestRenderGenerateResultsBrandProposalSchema(t *testing.T) {
	raw := []byte(`{
		"summary": "Strong developer-tool fit with low conflict risk.",
		"likely_available": true,
		"risk_level": "low",
		"confidence": 0.82,
		"brand_assessment": {
			"memorability": "9/10 - short and sticky.",
			"developer_appeal": "8/10 - sounds like an infra tool.",
			"pronunciation": "Easy - reads clearly on first pass.",
			"visual_potential": "Strong - angular letterforms support a crisp mark."
		},
		"conflict_analysis": {
			"existing_software": ["Dimlock"],
			"trademark_concerns": "No direct trademark match found in software, but close variants should be checked.",
			"social_presence": "Sparse current usage across major developer channels."
		},
		"domain_strategy": {
			"recommended_tld": ".dev",
			"rationale": "Best fit for a developer-facing command-line tool.",
			"alternatives": [".io", ".sh"]
		},
		"insights": ["No active developer tool with the exact name surfaced.", "The sound is short and memorable."],
		"mentions": [{"source": "web", "relevance": "high", "description": "A dormant personal site mentions Dimlox.", "url": "https://example.com/dimlox"}],
		"recommendations": ["Run domain and package checks before locking the brand."]
	}`)

	var buf bytes.Buffer
	rendered, err := renderGenerateResults(&buf, raw, "dimlox", "brand-proposal", "ailink/v0/search-response")
	require.NoError(t, err)
	require.True(t, rendered)

	out := buf.String()
	require.Contains(t, out, "Brand assessment for: dimlox")
	require.Contains(t, out, "Summary: Strong developer-tool fit with low conflict risk.")
	require.Contains(t, out, "Likely available: yes")
	require.Contains(t, out, "Risk level: low")
	require.Contains(t, out, "Confidence: 82%")
	require.Contains(t, out, "Brand Assessment:")
	require.Contains(t, out, "Memorability: 9/10 - short and sticky.")
	require.Contains(t, out, "Conflict Analysis:")
	require.Contains(t, out, "Existing software: Dimlock")
	require.Contains(t, out, "Domain Strategy:")
	require.Contains(t, out, "Recommended TLD: .dev")
	require.Contains(t, out, "Insights:")
	require.Contains(t, out, "Mentions:")
	require.Contains(t, out, "Recommendations:")
	require.Contains(t, out, "Run 'namelens check <name>' to verify availability.")
}

func TestRenderGenerateResultsBrandPlanSchema(t *testing.T) {
	raw := []byte(`{
		"summary": "Launch-ready with a distinctive technical voice.",
		"likely_available": true,
		"risk_level": "medium",
		"brand_identity": {
			"positioning_statement": "A practical naming copilot for software teams.",
			"tagline_options": ["Name with confidence", "Brand before you build"],
			"brand_voice": "Clear, technical, reassuring",
			"visual_direction": "Precision optics with warm industrial accents"
		},
		"immediate_actions": {
			"domains_to_register": ["dimlox.dev", "dimlox.io"],
			"handles_to_claim": ["github.com/dimlox", "x.com/dimlox"],
			"trademark_considerations": "Run a US trademark knockout search before launch."
		},
		"launch_checklist": [{
			"phase": "Week 1",
			"priority": "high",
			"actions": ["Claim core handles", "Publish landing page"]
		}],
		"competitive_landscape": {
			"similar_tools": ["Namechk", "BrandBucket"],
			"differentiation_opportunities": ["Developer-native CLI workflow", "Expert AI review"]
		},
		"recommendations": ["Prepare a concise launch story for early adopters."]
	}`)

	var buf bytes.Buffer
	rendered, err := renderGenerateResults(&buf, raw, "dimlox", "brand-plan", "ailink/v0/brand-plan-response")
	require.NoError(t, err)
	require.True(t, rendered)

	out := buf.String()
	require.Contains(t, out, "Brand plan for: dimlox")
	require.Contains(t, out, "Brand Identity:")
	require.Contains(t, out, "Immediate Actions:")
	require.Contains(t, out, "Launch Checklist:")
	require.Contains(t, out, "Competitive Landscape:")
	require.Contains(t, out, "Prepare a concise launch story for early adopters.")
	require.Contains(t, out, "Run 'namelens check <name>' to verify availability.")
}

func TestRenderGenerateResultsSearchResponseWithoutBrandProposalSections(t *testing.T) {
	raw := []byte(`{
		"summary": "Low risk overall.",
		"likely_available": true,
		"risk_level": "low",
		"insights": ["No strong collisions found."]
	}`)

	var buf bytes.Buffer
	rendered, err := renderGenerateResults(&buf, raw, "dimlox", "name-availability", "ailink/v0/search-response")
	require.NoError(t, err)
	require.True(t, rendered)

	out := buf.String()
	require.NotContains(t, out, "Brand Assessment:")
	require.NotContains(t, out, "Conflict Analysis:")
	require.NotContains(t, out, "Domain Strategy:")
}

func TestPrintGenerateRawFallbackPrettyPrintsUnknownSchema(t *testing.T) {
	raw := []byte(`{"summary":"hello","custom":{"answer":42}}`)

	var buf bytes.Buffer
	err := printGenerateRawFallback(&buf, raw, "dimlox")
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "Generated output for: dimlox")
	require.Contains(t, out, "\"summary\": \"hello\"")
	require.Contains(t, out, "\"answer\": 42")
	require.Contains(t, out, "Run 'namelens check <name>' to verify availability.")
}
