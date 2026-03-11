package prompt

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fulmenhq/gofulmen/pathfinder"
	"github.com/fulmenhq/gofulmen/schema"
	"gopkg.in/yaml.v3"
)

//go:embed embedded/schemas/ailink/v0/prompt.schema.json
var embeddedPromptSchemaJSON []byte

var (
	standaloneOnce    sync.Once
	standaloneRoot    string
	standaloneRootErr error
)

const promptSchemaID = "ailink/v0/prompt"

// Load parses and validates a prompt definition from YAML bytes.
func Load(source string, data []byte) (*Prompt, error) {
	config, body, err := parseYAMLWithFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("parse prompt %s: %w", source, err)
	}

	if strings.TrimSpace(config.SystemTemplate) == "" {
		config.SystemTemplate = strings.TrimSpace(body)
	}

	if strings.TrimSpace(config.SystemTemplate) == "" {
		return nil, fmt.Errorf("prompt %s missing system_template", source)
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("validate prompt %s: %w", source, err)
	}

	return &Prompt{Config: config, Source: source}, nil
}

// LoadFromDir reads all prompt files (.md with YAML frontmatter) from a directory.
func LoadFromDir(dir string) ([]*Prompt, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("scan prompts: %w", err)
	}
	results := make([]*Prompt, 0, len(entries))
	for _, path := range entries {
		data, err := os.ReadFile(path) // #nosec G304 -- Prompt path is user-provided
		if err != nil {
			return nil, fmt.Errorf("read prompt %s: %w", path, err)
		}
		prompt, err := Load(path, data)
		if err != nil {
			return nil, err
		}
		results = append(results, prompt)
	}
	return results, nil
}

func parseYAMLWithFrontmatter(data []byte) (Config, string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return Config{}, "", fmt.Errorf("empty prompt")
	}

	lines := bufio.NewScanner(bytes.NewReader(trimmed))
	lines.Split(bufio.ScanLines)

	var (
		frontmatter []string
		body        []string
		inFront     bool
		headerSeen  bool
	)

	for lines.Scan() {
		line := lines.Text()
		switch {
		case !headerSeen && strings.TrimSpace(line) == "---":
			headerSeen = true
			inFront = true
		case headerSeen && inFront && strings.TrimSpace(line) == "---":
			inFront = false
		default:
			if inFront {
				frontmatter = append(frontmatter, line)
			} else {
				body = append(body, line)
			}
		}
	}
	if err := lines.Err(); err != nil {
		return Config{}, "", err
	}

	var cfg Config
	if headerSeen {
		if err := yaml.Unmarshal([]byte(strings.Join(frontmatter, "\n")), &cfg); err != nil {
			return Config{}, "", fmt.Errorf("invalid frontmatter: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(trimmed, &cfg); err != nil {
			return Config{}, "", fmt.Errorf("invalid yaml: %w", err)
		}
	}

	return cfg, strings.Join(body, "\n"), nil
}

func validateConfig(cfg Config) error {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	catalog, err := catalogForSchemas()
	if err != nil {
		return err
	}

	diagnostics, err := catalog.ValidateDataByID(promptSchemaID, payload)
	if err != nil {
		return err
	}
	if len(diagnostics) > 0 {
		return fmt.Errorf("schema validation failed: %s", diagnostics[0].Message)
	}
	return nil
}

func catalogForSchemas() (*schema.Catalog, error) {
	root, err := findRepoRoot()
	if err == nil {
		return schema.NewCatalog(filepath.Join(root, "schemas")), nil
	}

	fallback, fallbackErr := standaloneSchemaRoot()
	if fallbackErr != nil {
		return nil, fmt.Errorf("project root not found: %w; embedded fallback failed: %w", err, fallbackErr)
	}
	return schema.NewCatalog(fallback), nil
}

func standaloneSchemaRoot() (string, error) {
	standaloneOnce.Do(func() {
		root, err := os.MkdirTemp("", "namelens-prompt-schemas-*")
		if err != nil {
			standaloneRootErr = fmt.Errorf("create prompt schema temp dir: %w", err)
			return
		}

		target := filepath.Join(root, "ailink", "v0", "prompt.schema.json")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			standaloneRootErr = fmt.Errorf("create embedded schema dir: %w", err)
			return
		}
		if err := os.WriteFile(target, embeddedPromptSchemaJSON, 0o644); err != nil {
			standaloneRootErr = fmt.Errorf("write embedded prompt schema: %w", err)
			return
		}

		standaloneRoot = root
	})

	if standaloneRootErr != nil {
		return "", standaloneRootErr
	}
	return standaloneRoot, nil
}

// CleanupStandaloneSchemas removes temporary schema assets written to disk.
// The sync.Once guard is reset so subsequent calls re-extract if needed.
func CleanupStandaloneSchemas() error {
	if standaloneRoot != "" {
		if err := os.RemoveAll(standaloneRoot); err != nil {
			return fmt.Errorf("cleanup standalone prompt schema temp dir: %w", err)
		}
	}
	standaloneRoot = ""
	standaloneRootErr = nil
	standaloneOnce = sync.Once{}
	return nil
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	markers := []string{"go.mod", ".git"}

	if hint, ok := pathfinder.DetectCIBoundaryHint(cwd); ok {
		root, err := pathfinder.FindRepositoryRoot(
			cwd,
			markers,
			pathfinder.WithBoundary(hint.Boundary),
			pathfinder.WithMaxDepth(20),
		)
		if err == nil {
			return root, nil
		}
	}

	root, err := pathfinder.FindRepositoryRoot(cwd, markers, pathfinder.WithMaxDepth(10))
	if err != nil {
		return "", err
	}
	return root, nil
}
