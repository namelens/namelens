package ailink

import (
	"errors"
	"strings"

	"github.com/namelens/namelens/internal/ailink/driver"
	"github.com/namelens/namelens/internal/ailink/prompt"
)

func responseFormatForProvider(resolved *ResolvedProvider, def *prompt.Prompt) *driver.ResponseFormat {
	if def == nil {
		return &driver.ResponseFormat{Type: "json_object"}
	}

	if resolved != nil && resolved.Driver != nil && resolved.Driver.Name() == "openai" {
		if schema := def.Config.ResponseSchema; len(schema) > 0 {
			name := strings.TrimSpace(def.Config.Slug)
			if name == "" {
				name = "namelens_schema"
			}
			// OpenAI requires name to be alphanumeric/underscore.
			name = strings.NewReplacer("-", "_", ".", "_").Replace(name)
			return &driver.ResponseFormat{
				Type: "json_schema",
				JSONSchema: &driver.JSONSchema{
					Name:   name,
					Strict: true,
					Schema: schema,
				},
			}
		}
	}

	return &driver.ResponseFormat{Type: "json_object"}
}

func isOpenAIUnsupportedSchemaError(err error) bool {
	if err == nil {
		return false
	}
	var perr *driver.ProviderError
	if errors.As(err, &perr) && perr != nil && perr.StatusCode == 400 {
		msg := strings.ToLower(perr.Message)
		return strings.Contains(msg, "json_schema") || strings.Contains(msg, "response_format")
	}
	return false
}

func fallbackToJSONObject(req *driver.Request) {
	if req == nil {
		return
	}
	if req.ResponseFormat == nil {
		return
	}
	req.ResponseFormat = &driver.ResponseFormat{Type: "json_object"}
}
