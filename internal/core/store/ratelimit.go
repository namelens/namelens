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

// GetRateLimit returns stored rate limit state for an endpoint.
func (s *Store) GetRateLimit(ctx context.Context, endpoint string) (*core.RateLimitState, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	var (
		requestCount int
		windowStart  int64
		backoffUntil sql.NullInt64
		last429At    sql.NullInt64
	)

	row := s.DB.QueryRowContext(ctx, `
		SELECT request_count, window_start, backoff_until, last_429_at
		FROM rate_limits
		WHERE endpoint = ?
	`, endpoint)

	if err := row.Scan(&requestCount, &windowStart, &backoffUntil, &last429At); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch rate limit: %w", err)
	}

	state := &core.RateLimitState{
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

	return state, nil
}

// UpdateRateLimit persists rate limit state for an endpoint.
func (s *Store) UpdateRateLimit(ctx context.Context, endpoint string, state *core.RateLimitState) error {
	if s == nil || s.DB == nil {
		return errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return errors.New("endpoint is required")
	}
	if state == nil {
		return errors.New("rate limit state is required")
	}

	var backoffUntil sql.NullInt64
	if state.BackoffUntil != nil {
		backoffUntil = sql.NullInt64{Int64: state.BackoffUntil.UTC().Unix(), Valid: true}
	}

	var last429At sql.NullInt64
	if state.Last429At != nil {
		last429At = sql.NullInt64{Int64: state.Last429At.UTC().Unix(), Valid: true}
	}

	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO rate_limits (endpoint, request_count, window_start, backoff_until, last_429_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET
			request_count = excluded.request_count,
			window_start = excluded.window_start,
			backoff_until = excluded.backoff_until,
			last_429_at = excluded.last_429_at
	`, endpoint, state.RequestCount, state.WindowStart.UTC().Unix(), backoffUntil, last429At)
	if err != nil {
		return fmt.Errorf("store rate limit: %w", err)
	}

	return nil
}
