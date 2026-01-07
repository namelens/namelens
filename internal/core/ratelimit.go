package core

import "time"

// RateLimitState captures per-endpoint rate limiting state.
type RateLimitState struct {
	RequestCount int
	WindowStart  time.Time
	BackoffUntil *time.Time
	Last429At    *time.Time
}
