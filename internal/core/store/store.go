package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/tursodatabase/go-libsql"

	"github.com/namelens/namelens/internal/config"
)

const driverLibsql = "libsql"

// Store wraps the database connection for NameLens.
type Store struct {
	DB     *sql.DB
	driver string
}

// Open initializes a store connection using the provided configuration.
func Open(ctx context.Context, cfg config.StoreConfig) (*Store, error) {
	driver := strings.TrimSpace(cfg.Driver)
	if driver == "" {
		driver = driverLibsql
	}

	if ctx == nil {
		ctx = context.Background()
	}

	switch driver {
	case driverLibsql:
		dsn, err := buildLibsqlDSN(cfg)
		if err != nil {
			return nil, err
		}

		db, err := sql.Open(driverLibsql, dsn)
		if err != nil {
			return nil, fmt.Errorf("open libsql store: %w", err)
		}
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("ping libsql store: %w", err)
		}

		if err := configureLocalSQLite(ctx, db, dsn, cfg); err != nil {
			_ = db.Close()
			return nil, err
		}

		return &Store{DB: db, driver: driver}, nil
	default:
		return nil, fmt.Errorf("unsupported store driver: %s", driver)
	}
}

// Close releases database resources.
func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

// Driver returns the configured store driver.
func (s *Store) Driver() string {
	if s == nil {
		return ""
	}
	return s.driver
}

func buildLibsqlDSN(cfg config.StoreConfig) (string, error) {
	if dsn := strings.TrimSpace(cfg.URL); dsn != "" {
		return addAuthToken(dsn, cfg.AuthToken)
	}

	path := strings.TrimSpace(cfg.Path)
	if path == "" {
		return "", errors.New("store path or url is required")
	}

	if path == ":memory:" {
		return path, nil
	}

	if strings.HasPrefix(path, "file:") {
		localPath, err := extractFilePath(path)
		if err != nil {
			return "", err
		}
		if err := ensureStoreDir(localPath); err != nil {
			return "", err
		}
		return path, nil
	}

	if strings.HasPrefix(path, "libsql:") {
		return path, nil
	}

	if err := ensureStoreDir(path); err != nil {
		return "", err
	}
	return "file:" + filepath.Clean(path), nil
}

func addAuthToken(dsn string, token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return dsn, nil
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("invalid store url: %w", err)
	}

	query := parsed.Query()
	if query.Get("authToken") == "" {
		query.Set("authToken", token)
		parsed.RawQuery = query.Encode()
	}

	return parsed.String(), nil
}

func extractFilePath(dsn string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("invalid store path: %w", err)
	}

	if parsed.Path != "" {
		return strings.TrimPrefix(parsed.Path, "//"), nil
	}

	return strings.TrimPrefix(parsed.Opaque, "//"), nil
}

func configureLocalSQLite(ctx context.Context, db *sql.DB, dsn string, cfg config.StoreConfig) error {
	if db == nil {
		return errors.New("store connection is nil")
	}
	if strings.TrimSpace(cfg.URL) != "" {
		return nil
	}
	if dsn == ":memory:" {
		return nil
	}
	if !strings.HasPrefix(dsn, "file:") {
		return nil
	}

	// SQLite is embedded; keep a single connection and use WAL to reduce lock contention.
	// This makes concurrent CLI checks behave predictably.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var busyTimeout int
	if err := db.QueryRowContext(ctx, "PRAGMA busy_timeout=5000").Scan(&busyTimeout); err != nil {
		return fmt.Errorf("set busy timeout: %w", err)
	}

	var journalMode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return fmt.Errorf("read journal mode: %w", err)
	}
	if strings.Contains(strings.ToLower(journalMode), "wal") {
		return nil
	}

	for attempt := 0; ; attempt++ {
		err := db.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&journalMode)
		if err == nil {
			break
		}

		message := strings.ToLower(err.Error())
		locked := strings.Contains(message, "database is locked")
		if !locked {
			return fmt.Errorf("enable WAL mode: %w", err)
		}
		if ctx.Err() != nil {
			// Best-effort: if another process has the DB, don't fail the CLI.
			return nil
		}

		// Back off briefly and retry within the 5s window.
		sleep := 50 * time.Millisecond
		switch {
		case attempt >= 5:
			sleep = 500 * time.Millisecond
		case attempt >= 2:
			sleep = 200 * time.Millisecond
		}
		time.Sleep(sleep)
	}

	return nil
}

func ensureStoreDir(path string) error {
	if strings.TrimSpace(path) == "" || path == ":memory:" {
		return nil
	}

	dir := filepath.Dir(filepath.Clean(path))
	if dir == "." || dir == string(filepath.Separator) {
		return nil
	}

	// #nosec G301 -- data directories use 0755 for multi-user access compatibility
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create store directory: %w", err)
	}
	return nil
}
