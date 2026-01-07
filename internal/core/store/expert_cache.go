package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ExpertCacheEntry captures cached expert responses.
type ExpertCacheEntry struct {
	ResponseJSON string
	ExpiresAt    time.Time
}

// GetExpertCache returns a cached expert response if present and not expired.
func (s *Store) GetExpertCache(ctx context.Context, name, promptSlug, model, baseURL, depth string) (*ExpertCacheEntry, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	row := s.DB.QueryRowContext(ctx,
		`SELECT response_json, expires_at FROM expert_cache
		 WHERE name = ? AND prompt_slug = ? AND model = ? AND base_url = ? AND depth = ?`,
		name, promptSlug, model, baseURL, depth,
	)

	var (
		response string
		expires  int64
	)
	if err := row.Scan(&response, &expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	expiresAt := time.Unix(expires, 0).UTC()
	if time.Now().UTC().After(expiresAt) {
		return nil, nil
	}

	return &ExpertCacheEntry{ResponseJSON: response, ExpiresAt: expiresAt}, nil
}

// SetExpertCache stores an expert response with TTL.
func (s *Store) SetExpertCache(ctx context.Context, name, promptSlug, model, baseURL, depth, responseJSON string, ttl time.Duration) error {
	if s == nil || s.DB == nil {
		return errors.New("store is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ttl <= 0 {
		return nil
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO expert_cache (name, prompt_slug, model, base_url, depth, response_json, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(name, prompt_slug, model, base_url, depth)
		 DO UPDATE SET response_json = excluded.response_json,
		               created_at = excluded.created_at,
		               expires_at = excluded.expires_at`,
		name, promptSlug, model, baseURL, depth, responseJSON, now.Unix(), expiresAt.Unix(),
	)
	return err
}
