package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/namelens/namelens/internal/core"
)

type RateLimitEntry struct {
	Endpoint string
	State    core.RateLimitState
}

type RateLimitQuery struct {
	All      bool
	Endpoint string
	Prefix   string
}

func (q RateLimitQuery) Validate() error {
	if q.All {
		return nil
	}
	if strings.TrimSpace(q.Endpoint) != "" {
		return nil
	}
	if strings.TrimSpace(q.Prefix) != "" {
		return nil
	}
	return errors.New("must specify --all, --endpoint, or --prefix")
}

func (q RateLimitQuery) whereClause() (string, []any, error) {
	if err := q.Validate(); err != nil {
		return "", nil, err
	}
	if q.All {
		return "", nil, nil
	}
	if endpoint := strings.TrimSpace(q.Endpoint); endpoint != "" {
		return "WHERE endpoint = ?", []any{endpoint}, nil
	}
	prefix := strings.TrimSpace(q.Prefix)
	if prefix == "" {
		return "", nil, errors.New("prefix is required")
	}
	return "WHERE endpoint LIKE ?", []any{prefix + "%"}, nil
}

func (s *Store) ListRateLimits(ctx context.Context, q RateLimitQuery) ([]RateLimitEntry, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	where, args, err := q.whereClause()
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT endpoint, request_count, window_start, backoff_until, last_429_at
		FROM rate_limits
		%s
		ORDER BY endpoint
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("list rate limits: %w", err)
	}
	defer rows.Close() // nolint:errcheck // best-effort cleanup

	entries := []RateLimitEntry{}
	for rows.Next() {
		var (
			endpoint     string
			requestCount int
			windowStart  int64
			backoffUntil sql.NullInt64
			last429At    sql.NullInt64
		)
		if err := rows.Scan(&endpoint, &requestCount, &windowStart, &backoffUntil, &last429At); err != nil {
			return nil, fmt.Errorf("scan rate limits: %w", err)
		}

		state := core.RateLimitState{
			RequestCount: requestCount,
			WindowStart:  time.Unix(windowStart, 0).UTC(),
		}
		if backoffUntil.Valid {
			value := time.Unix(backoffUntil.Int64, 0).UTC()
			state.BackoffUntil = &value
		}
		if last429At.Valid {
			value := time.Unix(last429At.Int64, 0).UTC()
			state.Last429At = &value
		}

		entries = append(entries, RateLimitEntry{Endpoint: endpoint, State: state})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list rate limits: %w", err)
	}

	return entries, nil
}

func (s *Store) CountRateLimits(ctx context.Context, q RateLimitQuery) (int, error) {
	if s == nil || s.DB == nil {
		return 0, errors.New("store is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	where, args, err := q.whereClause()
	if err != nil {
		return 0, err
	}

	row := s.DB.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*)
		FROM rate_limits
		%s
	`, where), args...)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count rate limits: %w", err)
	}
	return count, nil
}

func (s *Store) ResetRateLimits(ctx context.Context, q RateLimitQuery) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, errors.New("store is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	where, args, err := q.whereClause()
	if err != nil {
		return 0, err
	}

	result, err := s.DB.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM rate_limits
		%s
	`, where), args...)
	if err != nil {
		return 0, fmt.Errorf("reset rate limits: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("reset rate limits: %w", err)
	}
	return affected, nil
}
