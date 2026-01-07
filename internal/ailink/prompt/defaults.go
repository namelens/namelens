package prompt

import (
	"embed"
	"fmt"
)

//go:embed prompts/*.md
var defaultPromptsFS embed.FS

// LoadDefaults loads the embedded prompt set.
func LoadDefaults() ([]*Prompt, error) {
	entries, err := defaultPromptsFS.ReadDir("prompts")
	if err != nil {
		return nil, fmt.Errorf("read embedded prompts: %w", err)
	}
	results := make([]*Prompt, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := defaultPromptsFS.ReadFile("prompts/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read embedded prompt %s: %w", entry.Name(), err)
		}
		prompt, err := Load(entry.Name(), data)
		if err != nil {
			return nil, err
		}
		results = append(results, prompt)
	}
	return results, nil
}

// DefaultRegistry builds a registry from embedded prompts.
func DefaultRegistry() (Registry, error) {
	prompts, err := LoadDefaults()
	if err != nil {
		return nil, err
	}
	return NewRegistry(prompts)
}
