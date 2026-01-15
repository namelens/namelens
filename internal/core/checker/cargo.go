package checker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/namelens/namelens/internal/core"
	"github.com/namelens/namelens/internal/core/engine"
)

const cargoSource = "cargo"

// CargoChecker performs availability checks against crates.io.
type CargoChecker struct {
	Store       RegistryStore
	Client      *http.Client
	Limiter     *engine.RateLimiter
	CachePolicy CachePolicy
	UseCache    bool
	BaseURL     string
	ToolVersion string
	Clock       func() time.Time
}

// Check performs a crates.io availability check.
func (c *CargoChecker) Check(ctx context.Context, name string) (*core.CheckResult, error) {
	if c == nil || c.Store == nil {
		return nil, errors.New("cargo checker is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	value := strings.ToLower(strings.TrimSpace(name))
	if value == "" {
		return nil, errors.New("crate name is required")
	}
	if !c.SupportsName(value) {
		return nil, fmt.Errorf("unsupported cargo crate name: %q", name)
	}

	requestedAt := c.now()

	if c.UseCache {
		if cached, err := c.Store.GetCachedResult(ctx, value, core.CheckTypeCargo, ""); err == nil && cached != nil {
			cached.Name = value
			cached.Provenance.FromCache = true
			return cached, nil
		}
	}

	baseURL := c.baseURL()
	endpoint := baseURL.Hostname()

	if c.Limiter != nil && endpoint != "" {
		allowed, wait, err := c.Limiter.Allow(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		if !allowed {
			result := c.result(value, core.AvailabilityRateLimited, http.StatusTooManyRequests, fmt.Sprintf("rate limited, retry in %s", wait.Round(time.Second)), nil, requestedAt, c.now(), baseURL.String())
			c.cacheResult(ctx, value, result)
			return result, nil
		}
	}

	path := fmt.Sprintf("/api/v1/crates/%s", url.PathEscape(value))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL.ResolveReference(&url.URL{Path: path}).String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "namelens/"+c.toolVersion())

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	if c.Limiter != nil && endpoint != "" {
		if err := c.Limiter.Record(ctx, endpoint); err != nil {
			return nil, err
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		result := c.result(value, core.AvailabilityError, 0, err.Error(), nil, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	}
	defer resp.Body.Close() // nolint:errcheck // best-effort cleanup on HTTP response body

	switch resp.StatusCode {
	case http.StatusNotFound:
		result := c.result(value, core.AvailabilityAvailable, resp.StatusCode, "crate not found", nil, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	case http.StatusOK:
		extra := cargoExtra(resp)
		result := c.result(value, core.AvailabilityTaken, resp.StatusCode, "crate found", extra, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	case http.StatusTooManyRequests:
		wait, extra := retryAfterHeader(resp)
		if c.Limiter != nil && endpoint != "" && wait > 0 {
			_ = c.Limiter.Record429(ctx, endpoint, wait)
		}
		result := c.result(value, core.AvailabilityRateLimited, resp.StatusCode, "crates.io rate limited", extra, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	default:
		result := c.result(value, core.AvailabilityError, resp.StatusCode, "unexpected crates.io response", nil, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	}
}

// Type returns the checker type.
func (c *CargoChecker) Type() core.CheckType {
	return core.CheckTypeCargo
}

// SupportsName validates crate name constraints.
// Crate names must be 1-64 characters, alphanumeric plus - and _, starting with a letter.
func (c *CargoChecker) SupportsName(name string) bool {
	value := strings.TrimSpace(name)
	if value == "" || len(value) > 64 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_-]*$`, value)
	return matched
}

func (c *CargoChecker) baseURL() *url.URL {
	if c != nil && c.BaseURL != "" {
		if parsed, err := url.Parse(c.BaseURL); err == nil {
			return parsed
		}
	}
	parsed, _ := url.Parse("https://crates.io")
	return parsed
}

func (c *CargoChecker) cacheResult(ctx context.Context, name string, result *core.CheckResult) {
	if c == nil || c.Store == nil || !c.UseCache || result == nil {
		return
	}

	ttl := cacheTTL(c.CachePolicy, result.Available)
	if ttl <= 0 {
		return
	}

	_ = c.Store.SetCachedResult(ctx, name, result, ttl)
}

func (c *CargoChecker) result(name string, availability core.Availability, statusCode int, message string, extra map[string]any, requestedAt, resolvedAt time.Time, server string) *core.CheckResult {
	return &core.CheckResult{
		Name:       name,
		CheckType:  core.CheckTypeCargo,
		Available:  availability,
		StatusCode: statusCode,
		Message:    message,
		ExtraData:  extra,
		Provenance: core.Provenance{
			CheckID:     uuid.New().String(),
			RequestedAt: requestedAt,
			ResolvedAt:  resolvedAt,
			Source:      cargoSource,
			Server:      server,
			ToolVersion: c.toolVersion(),
		},
	}
}

func (c *CargoChecker) now() time.Time {
	if c != nil && c.Clock != nil {
		return c.Clock()
	}
	return time.Now().UTC()
}

func (c *CargoChecker) toolVersion() string {
	if c != nil && c.ToolVersion != "" {
		return c.ToolVersion
	}
	return "unknown"
}

func cargoExtra(resp *http.Response) map[string]any {
	if resp == nil || resp.Body == nil {
		return nil
	}

	var payload struct {
		Crate struct {
			Name        string `json:"name"`
			MaxVersion  string `json:"max_version"`
			Description string `json:"description"`
			Repository  string `json:"repository"`
		} `json:"crate"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil
	}

	extra := map[string]any{}
	if payload.Crate.Name != "" {
		extra["name"] = payload.Crate.Name
	}
	if payload.Crate.MaxVersion != "" {
		extra["version"] = payload.Crate.MaxVersion
	}
	if payload.Crate.Description != "" {
		extra["description"] = payload.Crate.Description
	}
	if payload.Crate.Repository != "" {
		extra["repository"] = payload.Crate.Repository
	}

	if len(extra) == 0 {
		return nil
	}
	return extra
}
