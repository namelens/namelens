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

const githubSource = "github"

// GitHubChecker performs availability checks against GitHub handles.
type GitHubChecker struct {
	Store       RegistryStore
	Client      *http.Client
	Limiter     *engine.RateLimiter
	CachePolicy CachePolicy
	UseCache    bool
	BaseURL     string
	Token       string
	ToolVersion string
	Clock       func() time.Time
}

// Check performs a GitHub handle availability check.
func (c *GitHubChecker) Check(ctx context.Context, name string) (*core.CheckResult, error) {
	if c == nil || c.Store == nil {
		return nil, errors.New("github checker is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	value := strings.TrimSpace(name)
	if value == "" {
		return nil, errors.New("handle name is required")
	}

	requestedAt := c.now()

	if c.UseCache {
		if cached, err := c.Store.GetCachedResult(ctx, value, core.CheckTypeGitHub, ""); err == nil && cached != nil {
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

	reqURL := baseURL.ResolveReference(&url.URL{Path: "/users/" + url.PathEscape(value)}).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token := strings.TrimSpace(c.Token); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

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
		result := c.result(value, core.AvailabilityAvailable, resp.StatusCode, "handle not found", nil, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	case http.StatusOK:
		extra := githubExtra(resp)
		result := c.result(value, core.AvailabilityTaken, resp.StatusCode, "handle found", extra, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	case http.StatusTooManyRequests, http.StatusForbidden:
		wait, extra := retryAfterHeader(resp)
		if c.Limiter != nil && endpoint != "" && wait > 0 {
			_ = c.Limiter.Record429(ctx, endpoint, wait)
		}
		result := c.result(value, core.AvailabilityRateLimited, resp.StatusCode, "github rate limited", extra, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	default:
		result := c.result(value, core.AvailabilityError, resp.StatusCode, "unexpected github response", nil, requestedAt, c.now(), baseURL.String())
		c.cacheResult(ctx, value, result)
		return result, nil
	}
}

// Type returns the checker type.
func (c *GitHubChecker) Type() core.CheckType {
	return core.CheckTypeGitHub
}

// SupportsName validates GitHub username constraints.
func (c *GitHubChecker) SupportsName(name string) bool {
	value := strings.TrimSpace(name)
	if value == "" || len(value) > 39 {
		return false
	}
	if strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return false
	}
	if strings.Contains(value, "--") {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9-]+$`, value)
	return matched
}

func (c *GitHubChecker) baseURL() *url.URL {
	if c != nil && c.BaseURL != "" {
		if parsed, err := url.Parse(c.BaseURL); err == nil {
			return parsed
		}
	}
	parsed, _ := url.Parse("https://api.github.com")
	return parsed
}

func (c *GitHubChecker) cacheResult(ctx context.Context, name string, result *core.CheckResult) {
	if c == nil || c.Store == nil || !c.UseCache || result == nil {
		return
	}

	ttl := cacheTTL(c.CachePolicy, result.Available)
	if ttl <= 0 {
		return
	}

	_ = c.Store.SetCachedResult(ctx, name, result, ttl)
}

func (c *GitHubChecker) result(name string, availability core.Availability, statusCode int, message string, extra map[string]any, requestedAt, resolvedAt time.Time, server string) *core.CheckResult {
	return &core.CheckResult{
		Name:       name,
		CheckType:  core.CheckTypeGitHub,
		Available:  availability,
		StatusCode: statusCode,
		Message:    message,
		ExtraData:  extra,
		Provenance: core.Provenance{
			CheckID:     uuid.New().String(),
			RequestedAt: requestedAt,
			ResolvedAt:  resolvedAt,
			Source:      githubSource,
			Server:      server,
			ToolVersion: c.ToolVersion,
		},
	}
}

func (c *GitHubChecker) now() time.Time {
	if c != nil && c.Clock != nil {
		return c.Clock()
	}
	return time.Now().UTC()
}

func githubExtra(resp *http.Response) map[string]any {
	if resp == nil || resp.Body == nil {
		return nil
	}

	var payload struct {
		Login   string `json:"login"`
		ID      int    `json:"id"`
		HTMLURL string `json:"html_url"`
		Type    string `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil
	}

	extra := map[string]any{}
	if payload.Login != "" {
		extra["login"] = payload.Login
	}
	if payload.ID != 0 {
		extra["id"] = payload.ID
	}
	if payload.HTMLURL != "" {
		extra["html_url"] = payload.HTMLURL
	}
	if payload.Type != "" {
		extra["type"] = payload.Type
	}

	if len(extra) == 0 {
		return nil
	}
	return extra
}
