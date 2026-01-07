package prompt

import (
	"fmt"
	"sort"
	"strings"
)

// Registry provides access to prompt definitions.
type Registry interface {
	Get(slug string) (*Prompt, error)
	List() []*Prompt
}

// InMemoryRegistry stores prompts by slug.
type InMemoryRegistry struct {
	prompts map[string]*Prompt
}

// NewRegistry builds a registry from prompts.
func NewRegistry(prompts []*Prompt) (*InMemoryRegistry, error) {
	reg := &InMemoryRegistry{prompts: make(map[string]*Prompt)}
	for _, prompt := range prompts {
		if prompt == nil {
			continue
		}
		slug := strings.TrimSpace(prompt.Config.Slug)
		if slug == "" {
			return nil, fmt.Errorf("prompt missing slug")
		}
		if _, ok := reg.prompts[slug]; ok {
			return nil, fmt.Errorf("duplicate prompt slug: %s", slug)
		}
		reg.prompts[slug] = prompt
	}
	return reg, nil
}

// Get returns the prompt for the slug.
func (r *InMemoryRegistry) Get(slug string) (*Prompt, error) {
	if r == nil {
		return nil, fmt.Errorf("prompt registry not configured")
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("prompt slug is required")
	}
	prompt, ok := r.prompts[slug]
	if !ok {
		return nil, fmt.Errorf("prompt %q not found", slug)
	}
	return prompt, nil
}

// List returns prompts sorted by slug.
func (r *InMemoryRegistry) List() []*Prompt {
	if r == nil {
		return nil
	}
	keys := make([]string, 0, len(r.prompts))
	for slug := range r.prompts {
		keys = append(keys, slug)
	}
	sort.Strings(keys)
	result := make([]*Prompt, 0, len(keys))
	for _, slug := range keys {
		result = append(result, r.prompts[slug])
	}
	return result
}
