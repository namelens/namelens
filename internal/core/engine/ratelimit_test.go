package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/core"
)

type memoryRateStore struct {
	state map[string]*core.RateLimitState
}

func (m *memoryRateStore) GetRateLimit(ctx context.Context, endpoint string) (*core.RateLimitState, error) {
	if m.state == nil {
		return nil, nil
	}
	if val, ok := m.state[endpoint]; ok {
		return val, nil
	}
	return nil, nil
}

func (m *memoryRateStore) UpdateRateLimit(ctx context.Context, endpoint string, state *core.RateLimitState) error {
	if m.state == nil {
		m.state = make(map[string]*core.RateLimitState)
	}
	m.state[endpoint] = state
	return nil
}

func TestRateLimiterWindow(t *testing.T) {
	store := &memoryRateStore{}
	clock := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	limiter := &RateLimiter{
		Store: store,
		Limits: map[string]RateLimit{
			"rdap.example": {RequestsPerWindow: 1, WindowDuration: time.Minute},
		},
		Clock: func() time.Time { return clock },
	}

	allowed, _, err := limiter.Allow(context.Background(), "rdap.example")
	require.NoError(t, err)
	require.True(t, allowed)

	require.NoError(t, limiter.Record(context.Background(), "rdap.example"))

	allowed, wait, err := limiter.Allow(context.Background(), "rdap.example")
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, time.Minute, wait)
}

func TestRateLimiterBackoff(t *testing.T) {
	store := &memoryRateStore{}
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	limiter := &RateLimiter{
		Store: store,
		Clock: func() time.Time { return now },
	}

	require.NoError(t, limiter.Record429(context.Background(), "rdap.example", 30*time.Second))

	allowed, wait, err := limiter.Allow(context.Background(), "rdap.example")
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, 30*time.Second, wait)
}

func TestRateLimiterMargin(t *testing.T) {
	store := &memoryRateStore{}
	limiter := &RateLimiter{
		Store: store,
		Limits: map[string]RateLimit{
			"rdap.example": {RequestsPerWindow: 10, WindowDuration: time.Minute},
		},
		Clock: func() time.Time { return time.Now().UTC() },
	}

	limiter.ApplySafetyMargin(0.9)
	limit := limiter.getLimit("rdap.example")
	require.Equal(t, 9, limit.RequestsPerWindow)
}
