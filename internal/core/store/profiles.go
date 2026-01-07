package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/namelens/namelens/internal/core"
)

// SeedBuiltInProfiles ensures built-in profiles exist in the store.
func (s *Store) SeedBuiltInProfiles(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return errors.New("store is not initialized")
	}

	for _, profile := range core.BuiltInProfiles {
		if err := s.UpsertProfile(ctx, profile, true, time.Now().UTC()); err != nil {
			return err
		}
	}

	return nil
}

// UpsertProfile creates or updates a profile record.
func (s *Store) UpsertProfile(ctx context.Context, profile core.Profile, isBuiltin bool, updatedAt time.Time) error {
	if s == nil || s.DB == nil {
		return errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return errors.New("profile name is required")
	}
	profile.Name = name

	payload, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("encode profile: %w", err)
	}

	builtinValue := 0
	if isBuiltin {
		builtinValue = 1
	}

	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO profiles (name, config, is_builtin, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			config = excluded.config,
			is_builtin = excluded.is_builtin,
			updated_at = excluded.updated_at
	`, name, string(payload), builtinValue, updatedAt.UTC().Unix())
	if err != nil {
		return fmt.Errorf("store profile: %w", err)
	}

	return nil
}

// GetProfile returns a profile record by name.
func (s *Store) GetProfile(ctx context.Context, name string) (*core.ProfileRecord, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("profile name is required")
	}

	var (
		configJSON string
		isBuiltin  int
		updatedAt  sql.NullInt64
	)

	row := s.DB.QueryRowContext(ctx, `
		SELECT config, is_builtin, updated_at
		FROM profiles
		WHERE name = ?
	`, name)

	if err := row.Scan(&configJSON, &isBuiltin, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch profile: %w", err)
	}

	var profile core.Profile
	if err := json.Unmarshal([]byte(configJSON), &profile); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	if profile.Name == "" {
		profile.Name = name
	}

	record := &core.ProfileRecord{
		Profile:   profile,
		IsBuiltin: isBuiltin == 1,
	}
	if updatedAt.Valid {
		record.UpdatedAt = time.Unix(updatedAt.Int64, 0).UTC()
	}

	return record, nil
}

// ListProfiles returns all profiles ordered by name.
func (s *Store) ListProfiles(ctx context.Context) ([]core.ProfileRecord, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT name, config, is_builtin, updated_at
		FROM profiles
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}
	defer rows.Close() // nolint:errcheck // best-effort cleanup on SQL rows

	var records []core.ProfileRecord
	for rows.Next() {
		var (
			name       string
			configJSON string
			isBuiltin  int
			updatedAt  sql.NullInt64
		)
		if err := rows.Scan(&name, &configJSON, &isBuiltin, &updatedAt); err != nil {
			return nil, fmt.Errorf("list profiles: %w", err)
		}

		var profile core.Profile
		if err := json.Unmarshal([]byte(configJSON), &profile); err != nil {
			return nil, fmt.Errorf("decode profile: %w", err)
		}
		if profile.Name == "" {
			profile.Name = name
		}

		record := core.ProfileRecord{
			Profile:   profile,
			IsBuiltin: isBuiltin == 1,
		}
		if updatedAt.Valid {
			record.UpdatedAt = time.Unix(updatedAt.Int64, 0).UTC()
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}

	return records, nil
}
