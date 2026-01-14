package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink"
	"github.com/namelens/namelens/internal/ailink/prompt"
)

type stubPromptRegistry struct {
	get  func(string) (*prompt.Prompt, error)
	list func() []*prompt.Prompt
}

func (s stubPromptRegistry) Get(slug string) (*prompt.Prompt, error) {
	if s.get == nil {
		return nil, fmt.Errorf("prompt %q not found", slug)
	}
	return s.get(slug)
}

func (s stubPromptRegistry) List() []*prompt.Prompt {
	if s.list == nil {
		return nil
	}
	return s.list()
}

func TestReviewPromptSetQuickMatchesCore(t *testing.T) {
	registry := stubPromptRegistry{}

	coreSet, err := reviewPromptSet("core", registry)
	require.NoError(t, err)

	quickSet, err := reviewPromptSet("quick", registry)
	require.NoError(t, err)

	require.Equal(t, coreSet, quickSet, "quick mode should return same prompts as core")
	require.Equal(t, []string{"name-availability", "name-phonetics", "name-suitability"}, quickSet)
}

func TestReviewPromptSetBrandSkipsMissingBrandPlan(t *testing.T) {
	registry := stubPromptRegistry{get: func(slug string) (*prompt.Prompt, error) {
		if slug == "brand-plan" {
			return nil, fmt.Errorf("prompt %q not found", slug)
		}
		return &prompt.Prompt{Config: prompt.Config{Slug: slug}}, nil
	}}

	set, err := reviewPromptSet("brand", registry)
	require.NoError(t, err)
	require.Equal(t, []string{"name-availability", "name-phonetics", "name-suitability", "brand-proposal"}, set)
}

func TestReviewPromptSetBrandIncludesBrandPlanWhenPresent(t *testing.T) {
	registry := stubPromptRegistry{get: func(slug string) (*prompt.Prompt, error) {
		return &prompt.Prompt{Config: prompt.Config{Slug: slug}}, nil
	}}

	set, err := reviewPromptSet("brand", registry)
	require.NoError(t, err)
	require.Equal(t, []string{"name-availability", "name-phonetics", "name-suitability", "brand-proposal", "brand-plan"}, set)
}

func TestRawFromAILinkErrorExtractsPayload(t *testing.T) {
	err := &ailink.RawResponseError{Err: errors.New("schema validation failed"), Raw: json.RawMessage(`{"bad": true}`)}
	require.Equal(t, json.RawMessage(`{"bad": true}`), rawFromAILinkError(err))
}

func TestAnalysisFromGenerateIncludeRawOnFailure(t *testing.T) {
	analysis := analysisFromGenerate(nil, &ailink.SearchError{Code: "AILINK_VALIDATION_ERROR", Message: "bad"}, json.RawMessage(`{"raw": true}`), includeRawOnFail)
	require.False(t, analysis.OK)
	require.NotNil(t, analysis.Error)
	require.Equal(t, json.RawMessage(`{"raw": true}`), analysis.Raw)
}

func TestReviewPromptSetFullMode(t *testing.T) {
	registry := stubPromptRegistry{
		list: func() []*prompt.Prompt {
			return []*prompt.Prompt{
				// Valid: only requires name
				{Config: prompt.Config{Slug: "name-availability", Input: prompt.InputSpec{RequiredVariables: []string{"name"}}}},
				// Valid: requires name and depth
				{Config: prompt.Config{Slug: "name-phonetics", Input: prompt.InputSpec{RequiredVariables: []string{"name", "depth"}}}},
				// Invalid: accepts images
				{Config: prompt.Config{Slug: "image-prompt", Input: prompt.InputSpec{AcceptsImages: true, RequiredVariables: []string{"name"}}}},
				// Invalid: requires extra variable
				{Config: prompt.Config{Slug: "comparison-prompt", Input: prompt.InputSpec{RequiredVariables: []string{"name", "competitor"}}}},
				// Valid: no required variables (empty)
				{Config: prompt.Config{Slug: "simple-prompt", Input: prompt.InputSpec{}}},
				// Nil prompt should be skipped
				nil,
			}
		},
	}

	set, err := reviewPromptSet("full", registry)
	require.NoError(t, err)
	// Results should be sorted and only include valid prompts
	require.Equal(t, []string{"name-availability", "name-phonetics", "simple-prompt"}, set)
}

func TestReviewPromptSetInvalidMode(t *testing.T) {
	registry := stubPromptRegistry{}

	_, err := reviewPromptSet("invalid-mode", registry)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported mode")
}

func TestReviewPromptSetModeNormalization(t *testing.T) {
	registry := stubPromptRegistry{}
	expected := []string{"name-availability", "name-phonetics", "name-suitability"}

	tests := []struct {
		name string
		mode string
	}{
		{"uppercase QUICK", "QUICK"},
		{"uppercase CORE", "CORE"},
		{"mixed case Quick", "Quick"},
		{"leading whitespace", "  quick"},
		{"trailing whitespace", "core  "},
		{"both whitespace", "  quick  "},
		{"empty defaults to core", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := reviewPromptSet(tt.mode, registry)
			require.NoError(t, err)
			require.Equal(t, expected, set)
		})
	}
}

func TestPromptSupportsNameOnly(t *testing.T) {
	tests := []struct {
		name     string
		prompt   *prompt.Prompt
		expected bool
	}{
		{
			name:     "nil prompt",
			prompt:   nil,
			expected: false,
		},
		{
			name:     "accepts images",
			prompt:   &prompt.Prompt{Config: prompt.Config{Input: prompt.InputSpec{AcceptsImages: true}}},
			expected: false,
		},
		{
			name:     "requires only name",
			prompt:   &prompt.Prompt{Config: prompt.Config{Input: prompt.InputSpec{RequiredVariables: []string{"name"}}}},
			expected: true,
		},
		{
			name:     "requires name and depth",
			prompt:   &prompt.Prompt{Config: prompt.Config{Input: prompt.InputSpec{RequiredVariables: []string{"name", "depth"}}}},
			expected: true,
		},
		{
			name:     "requires extra variable",
			prompt:   &prompt.Prompt{Config: prompt.Config{Input: prompt.InputSpec{RequiredVariables: []string{"name", "other"}}}},
			expected: false,
		},
		{
			name:     "empty required variables",
			prompt:   &prompt.Prompt{Config: prompt.Config{Input: prompt.InputSpec{RequiredVariables: []string{}}}},
			expected: true,
		},
		{
			name:     "whitespace in required variables ignored",
			prompt:   &prompt.Prompt{Config: prompt.Config{Input: prompt.InputSpec{RequiredVariables: []string{"name", "  ", "depth"}}}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := promptSupportsNameOnly(tt.prompt)
			require.Equal(t, tt.expected, result)
		})
	}
}
