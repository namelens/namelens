package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	//go:embed embedded/namelens/v0/namelens-defaults.yaml
	embeddedDefaultsYAML []byte

	//go:embed embedded/schemas/namelens/v0/config.schema.json
	embeddedConfigSchemaJSON []byte

	standaloneAssetsOnce sync.Once
	standaloneRoot       string
	standaloneRootErr    error
)

// CleanupStandaloneAssets removes temporary standalone config assets written to disk.
// It is safe to call even when standalone assets were never initialized.
func CleanupStandaloneAssets() error {
	if standaloneRoot == "" {
		return nil
	}

	if err := os.RemoveAll(standaloneRoot); err != nil {
		return fmt.Errorf("cleanup standalone config temp dir: %w", err)
	}

	standaloneRoot = ""
	return nil
}

func standaloneAssetsRoot() (string, error) {
	standaloneAssetsOnce.Do(func() {
		root, err := os.MkdirTemp("", "namelens-embedded-config-*")
		if err != nil {
			standaloneRootErr = fmt.Errorf("create standalone config temp dir: %w", err)
			return
		}

		if err := writeEmbeddedAsset(root, "config/namelens/v0/namelens-defaults.yaml", embeddedDefaultsYAML); err != nil {
			standaloneRootErr = err
			return
		}

		if err := writeEmbeddedAsset(root, "schemas/namelens/v0/config.schema.json", embeddedConfigSchemaJSON); err != nil {
			standaloneRootErr = err
			return
		}

		standaloneRoot = root
	})

	if standaloneRootErr != nil {
		return "", standaloneRootErr
	}

	return standaloneRoot, nil
}

func writeEmbeddedAsset(root string, relativePath string, contents []byte) error {
	target := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create embedded asset dir for %s: %w", relativePath, err)
	}

	if err := os.WriteFile(target, contents, 0o644); err != nil {
		return fmt.Errorf("write embedded asset %s: %w", relativePath, err)
	}

	return nil
}
