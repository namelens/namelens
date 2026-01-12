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

// GetCachedResult returns a cached check result if it is still valid.
func (s *Store) GetCachedResult(ctx context.Context, name string, checkType core.CheckType, tld string) (*core.CheckResult, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	keyName := strings.TrimSpace(name)
	if keyName == "" {
		return nil, errors.New("cache name is required")
	}

	tld = normalizeTLD(tld)

	var (
		extraJSON  sql.NullString
		message    sql.NullString
		checkedAt  int64
		expiresAt  int64
		available  int
		statusCode sql.NullInt64
	)

	row := s.DB.QueryRowContext(ctx, `
		SELECT available, status_code, message, extra_data, checked_at, expires_at
		FROM check_cache
		WHERE name = ? AND check_type = ? AND tld = ? AND expires_at > ?
	`, keyName, string(checkType), tld, time.Now().UTC().Unix())

	if err := row.Scan(&available, &statusCode, &message, &extraJSON, &checkedAt, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch cached result: %w", err)
	}

	var extra map[string]any
	if extraJSON.Valid && extraJSON.String != "" {
		if err := json.Unmarshal([]byte(extraJSON.String), &extra); err != nil {
			return nil, fmt.Errorf("decode cached result: %w", err)
		}
	}

	checked := time.Unix(checkedAt, 0).UTC()
	expires := time.Unix(expiresAt, 0).UTC()

	result := &core.CheckResult{
		Name:       keyName,
		CheckType:  checkType,
		TLD:        tld,
		Available:  core.Availability(available),
		StatusCode: int(statusCode.Int64),
		Message:    message.String,
		ExtraData:  extra,
		Provenance: core.Provenance{
			ResolvedAt:     checked,
			FromCache:      true,
			CacheExpiresAt: &expires,
		},
	}

	if result.ExtraData != nil {
		if value, ok := result.ExtraData["resolution_server"]; ok {
			if server, ok := value.(string); ok {
				result.Provenance.Server = strings.TrimSpace(server)
			}
		}
	}

	return result, nil
}

// SetCachedResult stores a check result with a TTL.
func (s *Store) SetCachedResult(ctx context.Context, name string, result *core.CheckResult, ttl time.Duration) error {
	if s == nil || s.DB == nil {
		return errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if ttl <= 0 || result == nil {
		return nil
	}

	keyName := strings.TrimSpace(name)
	if keyName == "" {
		return errors.New("cache name is required")
	}

	extraJSON, err := json.Marshal(result.ExtraData)
	if err != nil {
		return fmt.Errorf("encode cached result: %w", err)
	}

	now := time.Now().UTC()
	expires := now.Add(ttl)

	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO check_cache (name, check_type, tld, available, status_code, extra_data, message, checked_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name, check_type, tld) DO UPDATE SET
			available = excluded.available,
			status_code = excluded.status_code,
			extra_data = excluded.extra_data,
			message = excluded.message,
			checked_at = excluded.checked_at,
			expires_at = excluded.expires_at
	`, keyName, string(result.CheckType), normalizeTLD(result.TLD), int(result.Available), result.StatusCode, string(extraJSON), result.Message, now.Unix(), expires.Unix())
	if err != nil {
		return fmt.Errorf("store cached result: %w", err)
	}

	return nil
}
