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
	get func(string) (*prompt.Prompt, error)
}

func (s stubPromptRegistry) Get(slug string) (*prompt.Prompt, error) {
	if s.get == nil {
		return nil, fmt.Errorf("prompt %q not found", slug)
	}
	return s.get(slug)
}

func (s stubPromptRegistry) List() []*prompt.Prompt {
	return nil
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
