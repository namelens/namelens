package ailink

import (
	"path/filepath"
	"testing"

	"github.com/fulmenhq/gofulmen/schema"
	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink/prompt"
)

func TestOpenAISchemaForPromptExpandsRef(t *testing.T) {
	catalog := schema.NewCatalog(filepath.Join("..", "..", "schemas"))

	def := &prompt.Prompt{Config: prompt.Config{Slug: "name-phonetics", ResponseSchema: map[string]any{"$ref": "ailink/v0/name-phonetics-response"}}}
	result := openAISchemaForPrompt(def, catalog)
	require.NotNil(t, result)
	require.NotEmpty(t, result["$id"]) // schema file should include $id
}
