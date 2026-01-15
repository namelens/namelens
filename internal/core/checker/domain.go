package checker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openrdap/rdap"

	"github.com/namelens/namelens/internal/core"
	"github.com/namelens/namelens/internal/core/engine"
)

const rdapSource = "rdap"

var defaultRDAPOverrides = map[string][]string{
	"app": {"https://pubapi.registry.google/rdap", "https://www.rdap.net/rdap"},
	"dev": {"https://pubapi.registry.google/rdap", "https://www.rdap.net/rdap"},
}

// DomainChecker performs RDAP availability checks for domains.
type DomainChecker struct {
	Store       DomainStore
	Client      *rdap.Client
	ToolVersion string
	Clock       func() time.Time
	Timeout     time.Duration
	Limiter     *engine.RateLimiter
	CachePolicy CachePolicy
	UseCache    bool
	Whois       WhoisClient
	WhoisCfg    WhoisFallbackConfig
	DNSCfg      DNSFallbackConfig

	// RDAPOverrides allows routing specific TLDs to known-good RDAP servers.
	// Keys are normalized TLDs without a leading dot.
	RDAPOverrides map[string][]string
}

// DomainStore combines bootstrap, cache, and rate limit persistence.
type DomainStore interface {
	BootstrapStore
	GetCachedResult(ctx context.Context, name string, checkType core.CheckType, tld string) (*core.CheckResult, error)
	SetCachedResult(ctx context.Context, name string, result *core.CheckResult, ttl time.Duration) error
	GetRateLimit(ctx context.Context, endpoint string) (*core.RateLimitState, error)
	UpdateRateLimit(ctx context.Context, endpoint string, state *core.RateLimitState) error
}

// Type returns the checker type.
func (d *DomainChecker) Type() core.CheckType {
	return core.CheckTypeDomain
}

// SupportsName returns true if the name looks like a domain.
func (d *DomainChecker) SupportsName(name string) bool {
	value := strings.TrimSpace(name)
	return value != "" && strings.Contains(value, ".")
}

// Check performs a domain availability check using RDAP.
func (d *DomainChecker) Check(ctx context.Context, name string) (*core.CheckResult, error) {
	if d == nil || d.Store == nil {
		return nil, errors.New("domain checker is not configured")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	requestedAt := d.now()

	baseName, tld, err := splitDomain(name)
	if err != nil {
		return nil, err
	}

	servers, err := d.Store.GetRDAPServers(ctx, tld)
	if err != nil {
		return nil, err
	}

	if override := d.rdapOverrideServers(tld); len(override) > 0 {
		servers = override
	}

	rdapAvailable := len(servers) > 0
	whoisAllowed := d.whoisAllowed(tld)
	dnsAllowed := d.DNSCfg.Enabled

	if d.UseCache {
		if cached, err := d.Store.GetCachedResult(ctx, baseName, core.CheckTypeDomain, tld); err == nil && cached != nil {
			source := cachedResolutionSource(cached)
			if d.cacheAllowed(source, rdapAvailable, whoisAllowed, dnsAllowed) {
				cached.Name = name
				cached.Provenance.FromCache = true
				if cached.Provenance.Source == "" {
					cached.Provenance.Source = source
				}
				if cached.Provenance.Server == "" {
					if cached.ExtraData != nil {
						if value, ok := cached.ExtraData["resolution_server"]; ok {
							if server, ok := value.(string); ok && strings.TrimSpace(server) != "" {
								cached.Provenance.Server = server
							}
						}
					}
					if cached.Provenance.Server == "" && len(servers) > 0 {
						if serverURL, err := url.Parse(servers[0]); err == nil {
							cached.Provenance.Server = rdapDomainURL(serverURL, name)
						}

					}
				}
				if cached.Provenance.RequestedAt.IsZero() {
					cached.Provenance.RequestedAt = requestedAt
				}
				if cached.Provenance.CheckID == "" {
					cached.Provenance.CheckID = uuid.New().String()
				}
				if cached.Provenance.ToolVersion == "" {
					cached.Provenance.ToolVersion = d.ToolVersion
				}
				return cached, nil
			}
		}
	}

	if !rdapAvailable {
		if whoisAllowed {
			result := d.checkWhois(ctx, name, tld, requestedAt)
			d.cacheResult(ctx, baseName, result)
			return result, nil
		}
		if dnsAllowed {
			result := d.checkDNS(ctx, name, tld, requestedAt)
			d.cacheResult(ctx, baseName, result)
			return result, nil
		}
		return d.result(name, tld, core.AvailabilityUnsupported, 0, "no rdap server for tld", nil, requestedAt, d.now(), rdapSource, ""), nil
	}

	client := d.Client
	if client == nil {
		client = &rdap.Client{}
	}

	var lastResult *core.CheckResult
	for i, serverBase := range servers {
		serverURL, err := url.Parse(serverBase)
		if err != nil {
			return nil, fmt.Errorf("invalid rdap server url: %w", err)
		}
		endpoint := serverURL.Hostname()
		rdapRequestURL := rdapDomainURL(serverURL, name)

		if d.Limiter != nil && endpoint != "" {
			allowed, wait, err := d.Limiter.Allow(ctx, endpoint)
			if err != nil {
				return nil, err
			}
			if !allowed {
				lastResult = d.result(name, tld, core.AvailabilityRateLimited, 429, fmt.Sprintf("rate limited, retry in %s", wait.Round(time.Second)), nil, requestedAt, d.now(), rdapSource, rdapRequestURL)
				continue
			}
		}

		req := rdap.NewDomainRequest(name).WithServer(serverURL)
		if d.Timeout > 0 {
			req.Timeout = d.Timeout
		}
		req = req.WithContext(ctx)

		if d.Limiter != nil && endpoint != "" {
			if err := d.Limiter.Record(ctx, endpoint); err != nil {
				return nil, err
			}
		}

		resp, reqErr := client.Do(req)
		statusCode, server := responseStatus(resp, rdapRequestURL)

		if reqErr != nil {
			if isNotFound(reqErr) || statusCode == 404 {
				result := d.result(name, tld, core.AvailabilityAvailable, statusCode, "rdap not found", nil, requestedAt, d.now(), rdapSource, server)
				d.cacheResult(ctx, baseName, result)
				return result, nil
			}

			if statusCode == 429 {
				wait, extra := retryAfter(resp)
				if d.Limiter != nil && endpoint != "" && wait > 0 {
					_ = d.Limiter.Record429(ctx, endpoint, wait)
				}
				lastResult = d.result(name, tld, core.AvailabilityRateLimited, statusCode, "rdap rate limited", extra, requestedAt, d.now(), rdapSource, server)
				continue
			}

			if statusCode >= 500 && statusCode <= 599 {
				lastResult = d.result(name, tld, core.AvailabilityError, statusCode, "rdap server error", nil, requestedAt, d.now(), rdapSource, server)
				continue
			}

			lastResult = d.result(name, tld, core.AvailabilityError, statusCode, reqErr.Error(), nil, requestedAt, d.now(), rdapSource, server)
			continue
		}

		if domain, ok := resp.Object.(*rdap.Domain); ok {
			extra := domainExtra(domain)
			result := d.result(name, tld, core.AvailabilityTaken, statusCode, "domain found", extra, requestedAt, d.now(), rdapSource, server)
			d.cacheResult(ctx, baseName, result)
			return result, nil
		}

		lastResult = d.result(name, tld, core.AvailabilityUnknown, statusCode, "unexpected rdap response", nil, requestedAt, d.now(), rdapSource, server)

		if i == len(servers)-1 {
			break
		}
	}

	if lastResult == nil {
		lastResult = d.result(name, tld, core.AvailabilityError, 0, fmt.Sprintf("no rdap servers responded successfully (tried %d server(s))", len(servers)), nil, requestedAt, d.now(), rdapSource, "")
	}
	d.cacheResult(ctx, baseName, lastResult)
	return lastResult, nil
}

func (d *DomainChecker) result(name, tld string, availability core.Availability, statusCode int, message string, extra map[string]any, requestedAt, resolvedAt time.Time, source, server string) *core.CheckResult {
	if extra == nil {
		extra = map[string]any{}
	}
	if source != "" {
		extra["resolution_source"] = source
	}
	if strings.TrimSpace(server) != "" {
		extra["resolution_server"] = server
	}
	return &core.CheckResult{
		Name:       name,
		CheckType:  core.CheckTypeDomain,
		TLD:        tld,
		Available:  availability,
		StatusCode: statusCode,
		Message:    message,
		ExtraData:  extra,
		Provenance: core.Provenance{
			CheckID:     uuid.New().String(),
			RequestedAt: requestedAt,
			ResolvedAt:  resolvedAt,
			Source:      source,
			Server:      server,
			FromCache:   false,
			ToolVersion: d.ToolVersion,
		},
	}
}

func (d *DomainChecker) now() time.Time {
	if d != nil && d.Clock != nil {
		return d.Clock()
	}
	return time.Now().UTC()
}

func rdapDomainURL(server *url.URL, domain string) string {
	if server == nil {
		return ""
	}

	temp := *server
	temp.RawQuery = ""
	temp.Fragment = ""
	base := temp.String()
	if base == "" {
		return ""
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + "domain/" + strings.TrimSpace(domain)
}

func (d *DomainChecker) rdapOverrideServers(tld string) []string {
	normalized := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(tld, ".")))
	if normalized == "" {
		return nil
	}

	overrides := defaultRDAPOverrides
	if d != nil && d.RDAPOverrides != nil {
		overrides = d.RDAPOverrides
	}

	return overrides[normalized]
}

func splitDomain(domain string) (string, string, error) {
	value := strings.TrimSpace(domain)
	if value == "" {
		return "", "", errors.New("domain is required")
	}

	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return "", "", errors.New("domain must include a tld")
	}

	base := strings.ToLower(strings.Join(parts[:len(parts)-1], "."))
	tld := strings.ToLower(parts[len(parts)-1])

	return base, tld, nil
}

func responseStatus(resp *rdap.Response, fallbackURL string) (int, string) {
	if resp == nil || len(resp.HTTP) == 0 || resp.HTTP[0] == nil || resp.HTTP[0].Response == nil {
		return 0, strings.TrimSpace(fallbackURL)
	}

	hrr := resp.HTTP[0].Response
	url := ""
	if resp.HTTP[0].URL != "" {
		url = resp.HTTP[0].URL
	}
	if strings.TrimSpace(url) == "" {
		url = strings.TrimSpace(fallbackURL)
	}

	return hrr.StatusCode, url
}

func retryAfter(resp *rdap.Response) (time.Duration, map[string]any) {
	if resp == nil || len(resp.HTTP) == 0 || resp.HTTP[0] == nil || resp.HTTP[0].Response == nil {
		return 0, nil
	}

	retry := resp.HTTP[0].Response.Header.Get("Retry-After")
	if retry == "" {
		return 0, nil
	}

	if seconds, err := strconv.Atoi(retry); err == nil {
		return time.Duration(seconds) * time.Second, map[string]any{"retry_after": retry}
	}
	if parsed, err := httpParseTime(retry); err == nil {
		return time.Until(parsed), map[string]any{"retry_after": retry}
	}

	return 0, map[string]any{"retry_after": retry}
}

func domainExtra(domain *rdap.Domain) map[string]any {
	if domain == nil {
		return nil
	}

	extra := map[string]any{}
	if len(domain.Status) > 0 {
		extra["status"] = domain.Status
	}

	registrar := findRegistrar(domain)
	if registrar != "" {
		extra["registrar"] = registrar
	}

	if expiry := findEventDate(domain.Events, "expiration"); expiry != "" {
		extra["expiration"] = expiry
	}

	return extra
}

func findRegistrar(domain *rdap.Domain) string {
	if domain == nil {
		return ""
	}

	for _, entity := range domain.Entities {
		for _, role := range entity.Roles {
			if role == "registrar" && entity.VCard != nil {
				return entity.VCard.Name()
			}
		}
	}

	return ""
}

func findEventDate(events []rdap.Event, action string) string {
	for _, event := range events {
		if event.Action == action {
			return event.Date
		}
	}
	return ""
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	clientErr, ok := err.(*rdap.ClientError)
	if !ok {
		return false
	}

	return clientErr.Type == rdap.ObjectDoesNotExist
}

func (d *DomainChecker) cacheResult(ctx context.Context, name string, result *core.CheckResult) {
	if d == nil || d.Store == nil || !d.UseCache || result == nil {
		return
	}

	ttl := d.cacheTTL(result)
	if ttl <= 0 {
		return
	}

	_ = d.Store.SetCachedResult(ctx, name, result, ttl)
}

func (d *DomainChecker) cacheTTL(result *core.CheckResult) time.Duration {
	if result == nil {
		return 0
	}
	switch result.Provenance.Source {
	case whoisSource:
		if d.WhoisCfg.CacheTTL > 0 {
			return d.WhoisCfg.CacheTTL
		}
	case dnsSource:
		if d.DNSCfg.CacheTTL > 0 {
			return d.DNSCfg.CacheTTL
		}
	}
	return cacheTTL(d.CachePolicy, result.Available)
}

func httpParseTime(value string) (time.Time, error) {
	return http.ParseTime(value)
}
