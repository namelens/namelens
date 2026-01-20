package ailink

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/fulmenhq/gofulmen/schema"
	"github.com/namelens/namelens/internal/ailink/prompt"
)

func openAISchemaForPrompt(def *prompt.Prompt, catalog *schema.Catalog) map[string]any {
	if def == nil {
		return nil
	}
	schemaDef := def.Config.ResponseSchema
	if len(schemaDef) == 0 {
		return nil
	}

	// Prompts commonly reference schemas by $ref (e.g. "ailink/v0/name-phonetics-response").
	// OpenAI cannot resolve these IDs; expand them to an inline schema document.
	ref, ok := schemaDef["$ref"].(string)
	if !ok || strings.TrimSpace(ref) == "" {
		return schemaDef
	}

	ref = strings.TrimSpace(ref)
	if catalog == nil {
		return nil
	}

	desc, err := catalog.GetSchema(ref)
	if err != nil {
		return nil
	}
	payload, err := os.ReadFile(desc.Path)
	if err != nil {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	return decoded
}
