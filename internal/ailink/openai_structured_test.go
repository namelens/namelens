package ailink

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink/driver"
	"github.com/namelens/namelens/internal/ailink/prompt"
)

func TestResponseFormatForProviderUsesJSONSchemaForOpenAIWhenSchemaPresent(t *testing.T) {
	resolved := &ResolvedProvider{Driver: &recordingDriver{name: "openai"}}
	def := &prompt.Prompt{Config: prompt.Config{Slug: "name-phonetics", ResponseSchema: map[string]any{"type": "object"}}}

	format := responseFormatForProvider(resolved, def)
	require.NotNil(t, format)
	require.Equal(t, "json_schema", format.Type)
	require.NotNil(t, format.JSONSchema)
	require.True(t, format.JSONSchema.Strict)
	require.Equal(t, map[string]any{"type": "object"}, format.JSONSchema.Schema)
}

func TestFallbackToJSONObjectResetsSchema(t *testing.T) {
	req := &driver.Request{ResponseFormat: &driver.ResponseFormat{Type: "json_schema", JSONSchema: &driver.JSONSchema{Name: "x", Strict: true, Schema: map[string]any{"type": "object"}}}}
	fallbackToJSONObject(req)
	require.NotNil(t, req.ResponseFormat)
	require.Equal(t, "json_object", req.ResponseFormat.Type)
	require.Nil(t, req.ResponseFormat.JSONSchema)
}
