package engine

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/namelens/namelens/internal/core"
)

// RateLimiter enforces per-endpoint rate limits.
type RateLimiter struct {
	Store  RateLimitStore
	Limits map[string]RateLimit
	Clock  func() time.Time
	Margin float64
}

// RateLimit represents a rate limit window.
type RateLimit struct {
	RequestsPerWindow int
	WindowDuration    time.Duration
}

// RateLimitStore stores rate limit state.
type RateLimitStore interface {
	GetRateLimit(ctx context.Context, endpoint string) (*core.RateLimitState, error)
	UpdateRateLimit(ctx context.Context, endpoint string, state *core.RateLimitState) error
}

// DefaultLimits provides conservative defaults per endpoint.
var DefaultLimits = map[string]RateLimit{
	"rdap.verisign.com":  {RequestsPerWindow: 30, WindowDuration: time.Minute},
	"rdap.nic.google":    {RequestsPerWindow: 30, WindowDuration: time.Minute},
	"rdap.nic.io":        {RequestsPerWindow: 10, WindowDuration: 10 * time.Second},
	"whois":              {RequestsPerWindow: 30, WindowDuration: time.Hour},
	"registry.npmjs.org": {RequestsPerWindow: 100, WindowDuration: time.Minute},
	"pypi.org":           {RequestsPerWindow: 100, WindowDuration: time.Minute},
	"api.github.com":     {RequestsPerWindow: 60, WindowDuration: time.Hour},
}

// Allow checks if a request is allowed and returns wait duration if not.
func (r *RateLimiter) Allow(ctx context.Context, endpoint string) (bool, time.Duration, error) {
	if r == nil || r.Store == nil {
		return true, 0, nil
	}

	state, err := r.Store.GetRateLimit(ctx, endpoint)
	if err != nil {
		return true, 0, err
	}
	if state == nil {
		state = &core.RateLimitState{WindowStart: r.now()}
	}

	if state.BackoffUntil != nil && r.now().Before(*state.BackoffUntil) {
		return false, state.BackoffUntil.Sub(r.now()), nil
	}

	limit := r.getLimit(endpoint)
	windowEnd := state.WindowStart.Add(limit.WindowDuration)
	if r.now().After(windowEnd) {
		state.RequestCount = 0
		state.WindowStart = r.now()
	}

	if state.RequestCount >= limit.RequestsPerWindow {
		return false, windowEnd.Sub(r.now()), nil
	}

	return true, 0, nil
}

// Record increments the request count for an endpoint.
func (r *RateLimiter) Record(ctx context.Context, endpoint string) error {
	if r == nil || r.Store == nil {
		return nil
	}

	state, err := r.Store.GetRateLimit(ctx, endpoint)
	if err != nil {
		return err
	}
	if state == nil {
		state = &core.RateLimitState{WindowStart: r.now()}
	}

	state.RequestCount++
	if state.WindowStart.IsZero() {
		state.WindowStart = r.now()
	}

	return r.Store.UpdateRateLimit(ctx, endpoint, state)
}

// Record429 applies a backoff window from a 429 response.
func (r *RateLimiter) Record429(ctx context.Context, endpoint string, retryAfter time.Duration) error {
	if r == nil || r.Store == nil {
		return nil
	}

	state, err := r.Store.GetRateLimit(ctx, endpoint)
	if err != nil {
		return err
	}
	if state == nil {
		state = &core.RateLimitState{WindowStart: r.now()}
	}

	now := r.now()
	state.Last429At = &now
	if retryAfter > 0 {
		until := now.Add(retryAfter)
		state.BackoffUntil = &until
	}

	return r.Store.UpdateRateLimit(ctx, endpoint, state)
}

// ApplyOverrides merges per-endpoint request overrides (per minute).
func (r *RateLimiter) ApplyOverrides(overrides map[string]int) {
	if r == nil || len(overrides) == 0 {
		return
	}

	if r.Limits == nil {
		r.Limits = make(map[string]RateLimit, len(DefaultLimits))
		for key, limit := range DefaultLimits {
			r.Limits[key] = limit
		}
	}

	for endpoint, value := range overrides {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" || value <= 0 {
			continue
		}
		r.Limits[endpoint] = RateLimit{
			RequestsPerWindow: value,
			WindowDuration:    time.Minute,
		}
	}
}

// ApplySafetyMargin adjusts the effective request limits by a ratio (0-1].
func (r *RateLimiter) ApplySafetyMargin(margin float64) {
	if r == nil {
		return
	}
	if margin <= 0 || margin > 1 {
		return
	}
	r.Margin = margin
}

func (r *RateLimiter) getLimit(endpoint string) RateLimit {
	if r == nil {
		return RateLimit{RequestsPerWindow: 1, WindowDuration: time.Minute}
	}

	limits := r.Limits
	if limits == nil {
		limits = DefaultLimits
	}

	if limit, ok := limits[endpoint]; ok {
		return r.applyMargin(limit)
	}

	if strings.HasPrefix(endpoint, "whois.") {
		if limit, ok := limits["whois"]; ok {
			return r.applyMargin(limit)
		}
	}

	return r.applyMargin(RateLimit{RequestsPerWindow: 30, WindowDuration: time.Minute})
}

func (r *RateLimiter) now() time.Time {
	if r != nil && r.Clock != nil {
		return r.Clock()
	}
	return time.Now().UTC()
}

func (r *RateLimiter) applyMargin(limit RateLimit) RateLimit {
	if r == nil || r.Margin <= 0 || r.Margin > 1 {
		return limit
	}
	adjusted := int(math.Floor(float64(limit.RequestsPerWindow) * r.Margin))
	if adjusted < 1 {
		adjusted = 1
	}
	limit.RequestsPerWindow = adjusted
	return limit
}
