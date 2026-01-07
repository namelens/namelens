package core

import "time"

// CheckType identifies the type of availability check.
type CheckType string

const (
	CheckTypeDomain CheckType = "domain"
	CheckTypeNPM    CheckType = "npm"
	CheckTypePyPI   CheckType = "pypi"
	CheckTypeGitHub CheckType = "github"
)

// Availability represents the availability state for a check.
type Availability int

const (
	AvailabilityUnknown     Availability = 0
	AvailabilityAvailable   Availability = 1
	AvailabilityTaken       Availability = 2
	AvailabilityError       Availability = 3
	AvailabilityRateLimited Availability = 4
	AvailabilityUnsupported Availability = 5
)

// Provenance captures metadata about how a check was resolved.
type Provenance struct {
	CheckID        string     `json:"check_id"`
	RequestedAt    time.Time  `json:"requested_at"`
	ResolvedAt     time.Time  `json:"resolved_at"`
	Source         string     `json:"source"`
	Server         string     `json:"server,omitempty"`
	FromCache      bool       `json:"from_cache"`
	CacheExpiresAt *time.Time `json:"cache_expires_at,omitempty"`
	ToolVersion    string     `json:"tool_version"`
}

// CheckResult reports availability and supporting context.
type CheckResult struct {
	Name       string         `json:"name"`
	CheckType  CheckType      `json:"check_type"`
	TLD        string         `json:"tld,omitempty"`
	Available  Availability   `json:"available"`
	StatusCode int            `json:"status_code,omitempty"`
	Message    string         `json:"message,omitempty"`
	ExtraData  map[string]any `json:"extra_data,omitempty"`
	Provenance Provenance     `json:"provenance"`
}
