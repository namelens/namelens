package ailink

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/fulmenhq/gofulmen/schema"
)

//go:embed embedded/schemas/ailink/v0/*.json
var embeddedSchemasFS embed.FS

var (
	standaloneSchemasOnce sync.Once
	standaloneSchemasRoot string
	standaloneSchemasErr  error
)

// StandaloneSchemaCatalog returns a schema.Catalog backed by embedded schemas
// extracted to a temp directory. Used as a fallback when the repo root is not
// available (standalone binary, OOB directory).
func StandaloneSchemaCatalog() (*schema.Catalog, error) {
	root, err := standaloneSchemasRootDir()
	if err != nil {
		return nil, err
	}
	return schema.NewCatalog(root), nil
}

// StandaloneSchemasRoot returns the temp directory containing extracted schemas.
func StandaloneSchemasRoot() (string, error) {
	return standaloneSchemasRootDir()
}

// CleanupStandaloneSchemas removes temporary schema assets written to disk.
// The sync.Once guard is reset so subsequent calls re-extract if needed.
func CleanupStandaloneSchemas() error {
	if standaloneSchemasRoot != "" {
		if err := os.RemoveAll(standaloneSchemasRoot); err != nil {
			return fmt.Errorf("cleanup standalone schema temp dir: %w", err)
		}
	}
	standaloneSchemasRoot = ""
	standaloneSchemasErr = nil
	standaloneSchemasOnce = sync.Once{}
	return nil
}

func standaloneSchemasRootDir() (string, error) {
	standaloneSchemasOnce.Do(func() {
		root, err := os.MkdirTemp("", "namelens-ailink-schemas-*")
		if err != nil {
			standaloneSchemasErr = fmt.Errorf("create ailink schema temp dir: %w", err)
			return
		}

		err = fs.WalkDir(embeddedSchemasFS, "embedded/schemas", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Strip the "embedded/schemas/" prefix to get the relative schema path.
			rel, err := filepath.Rel("embedded/schemas", path)
			if err != nil {
				return err
			}
			target := filepath.Join(root, rel)
			if d.IsDir() {
				return os.MkdirAll(target, 0o755)
			}
			data, err := embeddedSchemasFS.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read embedded schema %s: %w", path, err)
			}
			return os.WriteFile(target, data, 0o644)
		})
		if err != nil {
			standaloneSchemasErr = fmt.Errorf("extract embedded schemas: %w", err)
			return
		}

		standaloneSchemasRoot = root
	})

	if standaloneSchemasErr != nil {
		return "", standaloneSchemasErr
	}
	return standaloneSchemasRoot, nil
}
