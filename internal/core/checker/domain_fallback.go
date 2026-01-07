package checker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/namelens/namelens/internal/core"
)

func (d *DomainChecker) whoisAllowed(tld string) bool {
	if d == nil || !d.WhoisCfg.Enabled {
		return false
	}

	tld = strings.ToLower(strings.TrimSpace(tld))
	if tld == "" {
		return false
	}

	if len(d.WhoisCfg.TLDs) == 0 {
		return !d.WhoisCfg.RequireExplicit
	}

	for _, allowed := range d.WhoisCfg.TLDs {
		normalized := strings.TrimSpace(allowed)
		normalized = strings.TrimPrefix(normalized, ".")
		if strings.EqualFold(normalized, tld) {
			return true
		}
	}

	return !d.WhoisCfg.RequireExplicit
}

func (d *DomainChecker) cacheAllowed(source string, rdapAvailable, whoisAllowed, dnsAllowed bool) bool {
	switch source {
	case whoisSource:
		return !rdapAvailable && whoisAllowed
	case dnsSource:
		return !rdapAvailable && !whoisAllowed && dnsAllowed
	default:
		return rdapAvailable
	}
}

func cachedResolutionSource(result *core.CheckResult) string {
	if result == nil || result.ExtraData == nil {
		return rdapSource
	}
	if value, ok := result.ExtraData["resolution_source"]; ok {
		if source, ok := value.(string); ok && strings.TrimSpace(source) != "" {
			return strings.TrimSpace(source)
		}
	}
	return rdapSource
}

func (d *DomainChecker) checkWhois(ctx context.Context, name, tld string, requestedAt time.Time) *core.CheckResult {
	if d.WhoisCfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.WhoisCfg.Timeout)
		defer cancel()
	}

	client := d.Whois
	if client == nil {
		client = &DefaultWhoisClient{
			Servers: d.WhoisCfg.Servers,
			Timeout: d.WhoisCfg.Timeout,
		}
	}

	server := ""
	if resolver, ok := client.(WhoisResolver); ok {
		resolved, err := resolver.ResolveServer(ctx, tld)
		if err != nil {
			// No whois server for this TLD = no data, not an error
			return d.result(name, tld, core.AvailabilityUnknown, 0, err.Error(), nil, requestedAt, d.now(), whoisSource, "")
		}
		server = strings.TrimSpace(resolved)
	}

	endpoint := "whois"
	if server != "" {
		endpoint = "whois." + server
	}
	if d.Limiter != nil {
		allowed, wait, err := d.Limiter.Allow(ctx, endpoint)
		if err != nil {
			return d.result(name, tld, core.AvailabilityError, 0, err.Error(), nil, requestedAt, d.now(), whoisSource, "")
		}
		if !allowed {
			return d.result(name, tld, core.AvailabilityRateLimited, 429, fmt.Sprintf("whois rate limited, retry in %s", wait.Round(time.Second)), nil, requestedAt, d.now(), whoisSource, "")
		}
	}

	var (
		resp *WhoisResponse
		err  error
	)
	if server != "" {
		if lookupWithServer, ok := client.(WhoisServerLookup); ok {
			resp, err = lookupWithServer.LookupWithServer(ctx, server, name)
		} else {
			resp, err = client.Lookup(ctx, tld, name)
		}
	} else {
		resp, err = client.Lookup(ctx, tld, name)
	}
	if err != nil {
		// Treat server resolution failures as "no data" rather than errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "whois server") || strings.Contains(errMsg, "no whois server") {
			return d.result(name, tld, core.AvailabilityUnknown, 0, errMsg, nil, requestedAt, d.now(), whoisSource, "")
		}
		return d.result(name, tld, core.AvailabilityError, 0, errMsg, nil, requestedAt, d.now(), whoisSource, "")
	}
	if resp == nil {
		return d.result(name, tld, core.AvailabilityError, 0, "whois lookup failed", nil, requestedAt, d.now(), whoisSource, "")
	}

	if d.Limiter != nil {
		if err := d.Limiter.Record(ctx, endpoint); err != nil {
			return d.result(name, tld, core.AvailabilityError, 0, err.Error(), nil, requestedAt, d.now(), whoisSource, "")
		}
	}

	patterns := normalizeWhoisPatterns(d.WhoisCfg)
	availability, message := interpretWhois(resp.Body, patterns)
	extra := map[string]any{
		"whois_server":   resp.Server,
		"whois_raw_hash": whoisHash(resp.Body),
	}

	result := d.result(name, tld, availability, 0, message, extra, requestedAt, d.now(), whoisSource, resp.Server)
	return result
}

func (d *DomainChecker) checkDNS(ctx context.Context, name, tld string, requestedAt time.Time) *core.CheckResult {
	if d.DNSCfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.DNSCfg.Timeout)
		defer cancel()
	}

	records, err := net.DefaultResolver.LookupNS(ctx, name)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			extra := map[string]any{"dns_status": "nxdomain"}
			return d.result(name, tld, core.AvailabilityUnknown, 0, "dns nxdomain (non-authoritative)", extra, requestedAt, d.now(), dnsSource, "")
		}
		return d.result(name, tld, core.AvailabilityError, 0, fmt.Sprintf("dns lookup failed: %v", err), nil, requestedAt, d.now(), dnsSource, "")
	}

	if len(records) == 0 {
		extra := map[string]any{"dns_status": "no_records"}
		return d.result(name, tld, core.AvailabilityUnknown, 0, "dns no records (non-authoritative)", extra, requestedAt, d.now(), dnsSource, "")
	}

	extra := map[string]any{"dns_status": "records_present"}
	return d.result(name, tld, core.AvailabilityTaken, 0, "dns records present (non-authoritative)", extra, requestedAt, d.now(), dnsSource, "")
}
