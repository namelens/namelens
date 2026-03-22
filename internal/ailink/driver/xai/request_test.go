package xai

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
)

func TestBuildResponsesRequestIncludesStructuredOutputFormat(t *testing.T) {
	req := &driver.Request{
		Model: "grok-4-1-fast-reasoning",
		Messages: []content.Message{
			{Role: "system", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "sys"}}},
			{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "usr"}}},
		},
		SearchParameters: &driver.SearchParameters{Sources: []driver.Source{{Type: "web"}}},
		ResponseFormat: &driver.ResponseFormat{
			Type: "json_schema",
			JSONSchema: &driver.JSONSchema{
				Name:   "short_domain_finder",
				Strict: true,
				Schema: map[string]any{"type": "object", "required": []any{"candidates"}},
			},
		},
	}

	payload, err := buildResponsesRequest(req)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Len(t, payload.Tools, 1)
	require.Equal(t, "web_search", payload.Tools[0].Type)
	require.NotNil(t, payload.Text)
	require.NotNil(t, payload.Text.Format)
	require.Equal(t, "json_schema", payload.Text.Format.Type)
	require.NotNil(t, payload.Text.Format.JSONSchema)
	require.Equal(t, "short_domain_finder", payload.Text.Format.JSONSchema.Name)
	require.True(t, payload.Text.Format.JSONSchema.Strict)
	require.Equal(t, map[string]any{"type": "object", "required": []any{"candidates"}}, payload.Text.Format.JSONSchema.Schema)
}

func TestBuildResponsesRequestIncludesJSONObjectFormat(t *testing.T) {
	req := &driver.Request{
		Model: "grok-4-1-fast-reasoning",
		Messages: []content.Message{
			{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: "usr"}}},
		},
		SearchParameters: &driver.SearchParameters{Sources: []driver.Source{{Type: "web"}}},
		ResponseFormat:   &driver.ResponseFormat{Type: "json_object"},
	}

	payload, err := buildResponsesRequest(req)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.NotNil(t, payload.Text)
	require.NotNil(t, payload.Text.Format)
	require.Equal(t, "json_object", payload.Text.Format.Type)
	require.Nil(t, payload.Text.Format.JSONSchema)
}
