package checker

import (
	"time"

	"github.com/namelens/namelens/internal/core"
)

// CachePolicy controls cache TTLs for check results.
type CachePolicy struct {
	AvailableTTL time.Duration
	TakenTTL     time.Duration
	ErrorTTL     time.Duration
}

func cachePolicyWithDefaults(policy CachePolicy) CachePolicy {
	if policy.AvailableTTL == 0 {
		policy.AvailableTTL = 5 * time.Minute
	}
	if policy.TakenTTL == 0 {
		policy.TakenTTL = time.Hour
	}
	if policy.ErrorTTL == 0 {
		policy.ErrorTTL = 30 * time.Second
	}
	return policy
}

func cacheTTL(policy CachePolicy, availability core.Availability) time.Duration {
	policy = cachePolicyWithDefaults(policy)

	switch availability {
	case core.AvailabilityAvailable:
		return policy.AvailableTTL
	case core.AvailabilityTaken:
		return policy.TakenTTL
	case core.AvailabilityError, core.AvailabilityRateLimited:
		return policy.ErrorTTL
	default:
		return policy.ErrorTTL
	}
}
