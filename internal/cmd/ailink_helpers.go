package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fulmenhq/gofulmen/pathfinder"
	"github.com/fulmenhq/gofulmen/schema"
	"github.com/namelens/namelens/internal/ailink/prompt"
	"github.com/namelens/namelens/internal/config"
)

func buildPromptRegistry(cfg *config.Config) (prompt.Registry, error) {
	defaults, err := prompt.LoadDefaults()
	if err != nil {
		return nil, err
	}

	merged := make(map[string]*prompt.Prompt, len(defaults))
	for _, p := range defaults {
		if p == nil {
			continue
		}
		merged[p.Config.Slug] = p
	}

	if cfg != nil {
		dir := strings.TrimSpace(cfg.AILink.PromptsDir)
		if dir != "" {
			overrides, err := prompt.LoadFromDir(dir)
			if err != nil {
				return nil, err
			}
			for _, p := range overrides {
				if p == nil {
					continue
				}
				merged[p.Config.Slug] = p
			}
		}
	}

	prompts := make([]*prompt.Prompt, 0, len(merged))
	for _, p := range merged {
		prompts = append(prompts, p)
	}
	return prompt.NewRegistry(prompts)
}

func buildSchemaCatalog() (*schema.Catalog, error) {
	root, err := findRepoRoot()
	if err != nil {
		return nil, err
	}
	return schema.NewCatalog(filepath.Join(root, "schemas")), nil
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	markers := []string{"go.mod", ".git"}

	// CI-only boundary hint pattern:
	// Treat CI workspace env vars as a boundary hint to prevent escaping container roots.
	isCI := strings.EqualFold(strings.TrimSpace(os.Getenv("GITHUB_ACTIONS")), "true") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("CI")), "true")
	if isCI {
		boundaryKeys := []string{"FULMEN_WORKSPACE_ROOT", "GITHUB_WORKSPACE", "CI_PROJECT_DIR", "WORKSPACE"}
		for _, key := range boundaryKeys {
			boundary := strings.TrimSpace(os.Getenv(key))
			if boundary == "" {
				continue
			}
			boundary = filepath.Clean(boundary)
			if !filepath.IsAbs(boundary) {
				continue
			}
			st, err := os.Stat(boundary)
			if err != nil || !st.IsDir() {
				continue
			}
			// Only accept a boundary that contains the start path.
			if rel, err := filepath.Rel(boundary, cwd); err != nil || strings.HasPrefix(rel, "..") {
				continue
			}
			rootPath, err := pathfinder.FindRepositoryRoot(cwd, markers,
				pathfinder.WithBoundary(boundary),
				pathfinder.WithMaxDepth(20),
			)
			if err == nil {
				return rootPath, nil
			}
		}
	}

	root, err := pathfinder.FindRepositoryRoot(cwd, markers, pathfinder.WithMaxDepth(10))
	if err != nil {
		return "", fmt.Errorf("project root not found: %w", err)
	}
	return root, nil
}
