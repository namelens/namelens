package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS bootstrap_tlds (
		tld TEXT PRIMARY KEY,
		rdap_urls TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS bootstrap_meta (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS check_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		check_type TEXT NOT NULL,
		tld TEXT,
		available INTEGER,
		status_code INTEGER,
		extra_data TEXT,
		message TEXT,
		checked_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL,
		UNIQUE(name, check_type, tld)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_check_cache_expires ON check_cache(expires_at);`,
	`CREATE INDEX IF NOT EXISTS idx_check_cache_lookup ON check_cache(name, check_type);`,
	`CREATE TABLE IF NOT EXISTS rate_limits (
		endpoint TEXT PRIMARY KEY,
		request_count INTEGER NOT NULL DEFAULT 0,
		window_start INTEGER NOT NULL,
		backoff_until INTEGER,
		last_429_at INTEGER
	);`,
	`CREATE TABLE IF NOT EXISTS profiles (
		name TEXT PRIMARY KEY,
		config TEXT NOT NULL,
		is_builtin INTEGER DEFAULT 0,
		updated_at INTEGER
	);`,
	`CREATE TABLE IF NOT EXISTS expert_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		prompt_slug TEXT NOT NULL,
		model TEXT NOT NULL,
		base_url TEXT NOT NULL,
		depth TEXT NOT NULL,
		response_json TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL,
		UNIQUE(name, prompt_slug, model, base_url, depth)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_expert_cache_expires ON expert_cache(expires_at);`,
}

// Migrate ensures the required database tables exist.
func (s *Store) Migrate(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	for _, stmt := range schemaStatements {
		if _, err := s.DB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store migration failed: %w", err)
		}
	}

	if err := s.ensureColumn(ctx, "check_cache", "message", "TEXT"); err != nil {
		return err
	}

	return nil
}

func (s *Store) ensureColumn(ctx context.Context, table, column, columnDef string) error {
	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("inspect %s schema: %w", table, err)
	}
	defer rows.Close() // nolint:errcheck // best-effort cleanup on SQL rows

	for rows.Next() {
		var (
			cid     int
			name    string
			colType string
			notNull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return fmt.Errorf("inspect %s columns: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("inspect %s columns: %w", table, err)
	}

	if _, err := s.DB.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnDef)); err != nil {
		return fmt.Errorf("add %s.%s column: %w", table, column, err)
	}

	return nil
}
