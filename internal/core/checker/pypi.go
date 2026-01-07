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

const pypiSource = "pypi"

// PyPIChecker performs availability checks against PyPI.
type PyPIChecker struct {
	Store       RegistryStore
	Client      *http.Client
	Limiter     *engine.RateLimiter
	CachePolicy CachePolicy
	UseCache    bool
	BaseURL     string
	ToolVersion string
	Clock       func() time.Time
}

// Check performs a PyPI availability check.
func (c *PyPIChecker) Check(ctx context.Context, name string) (*core.CheckResult, error) {
	if c == nil || c.Store == nil {
		return nil, errors.New("pypi checker is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	value := strings.ToLower(strings.TrimSpace(name))
	if value == "" {
		return nil, errors.New("package name is required")
	}

	requestedAt := c.now()

	if c.UseCache {
		if cached, err := c.Store.GetCachedResult(ctx, value, core.CheckTypePyPI, ""); err == nil && cached != nil {
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

	path := fmt.Sprintf("/pypi/%s/json", url.PathEscape(value))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL.ResolveReference(&url.URL{Path: path}).String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

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
		result := c.result(value, core.AvailabilityAvailable, resp.StatusCode, "package not found", nil, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	case http.StatusOK:
		extra := pypiExtra(resp)
		result := c.result(value, core.AvailabilityTaken, resp.StatusCode, "package found", extra, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	case http.StatusTooManyRequests:
		wait, extra := retryAfterHeader(resp)
		if c.Limiter != nil && endpoint != "" && wait > 0 {
			_ = c.Limiter.Record429(ctx, endpoint, wait)
		}
		result := c.result(value, core.AvailabilityRateLimited, resp.StatusCode, "pypi rate limited", extra, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	default:
		result := c.result(value, core.AvailabilityError, resp.StatusCode, "unexpected pypi response", nil, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	}
}

// Type returns the checker type.
func (c *PyPIChecker) Type() core.CheckType {
	return core.CheckTypePyPI
}

// SupportsName validates PyPI project name constraints.
func (c *PyPIChecker) SupportsName(name string) bool {
	value := strings.TrimSpace(name)
	if value == "" || len(value) > 200 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9._-]*$`, value)
	return matched
}

func (c *PyPIChecker) baseURL() *url.URL {
	if c != nil && c.BaseURL != "" {
		if parsed, err := url.Parse(c.BaseURL); err == nil {
			return parsed
		}
	}
	parsed, _ := url.Parse("https://pypi.org")
	return parsed
}

func (c *PyPIChecker) cacheResult(ctx context.Context, name string, result *core.CheckResult) {
	if c == nil || c.Store == nil || !c.UseCache || result == nil {
		return
	}

	ttl := cacheTTL(c.CachePolicy, result.Available)
	if ttl <= 0 {
		return
	}

	_ = c.Store.SetCachedResult(ctx, name, result, ttl)
}

func (c *PyPIChecker) result(name string, availability core.Availability, statusCode int, message string, extra map[string]any, requestedAt, resolvedAt time.Time, server string) *core.CheckResult {
	return &core.CheckResult{
		Name:       name,
		CheckType:  core.CheckTypePyPI,
		Available:  availability,
		StatusCode: statusCode,
		Message:    message,
		ExtraData:  extra,
		Provenance: core.Provenance{
			CheckID:     uuid.New().String(),
			RequestedAt: requestedAt,
			ResolvedAt:  resolvedAt,
			Source:      pypiSource,
			Server:      server,
			ToolVersion: c.ToolVersion,
		},
	}
}

func (c *PyPIChecker) now() time.Time {
	if c != nil && c.Clock != nil {
		return c.Clock()
	}
	return time.Now().UTC()
}

func pypiExtra(resp *http.Response) map[string]any {
	if resp == nil || resp.Body == nil {
		return nil
	}

	var payload struct {
		Info struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Summary string `json:"summary"`
			Home    string `json:"home_page"`
		} `json:"info"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil
	}

	extra := map[string]any{}
	if payload.Info.Name != "" {
		extra["name"] = payload.Info.Name
	}
	if payload.Info.Version != "" {
		extra["version"] = payload.Info.Version
	}
	if payload.Info.Summary != "" {
		extra["summary"] = payload.Info.Summary
	}
	if payload.Info.Home != "" {
		extra["home_page"] = payload.Info.Home
	}

	if len(extra) == 0 {
		return nil
	}
	return extra
}
