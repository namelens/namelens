package checker

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/namelens/namelens/internal/core"
)

const (
	whoisSource = "whois"
	dnsSource   = "dns"

	whoisIanaServer = "whois.iana.org"
	whoisPort       = "43"
	whoisMaxBytes   = 128 * 1024
)

// WhoisFallbackConfig controls WHOIS fallback behavior.
type WhoisFallbackConfig struct {
	Enabled           bool
	TLDs              []string
	RequireExplicit   bool
	CacheTTL          time.Duration
	Timeout           time.Duration
	Servers           map[string]string
	AvailablePatterns []string
	TakenPatterns     []string
}

// DNSFallbackConfig controls DNS fallback behavior.
type DNSFallbackConfig struct {
	Enabled  bool
	CacheTTL time.Duration
	Timeout  time.Duration
}

// WhoisClient performs WHOIS lookups.
type WhoisClient interface {
	Lookup(ctx context.Context, tld, domain string) (*WhoisResponse, error)
}

// WhoisResolver resolves the WHOIS server for a TLD.
type WhoisResolver interface {
	ResolveServer(ctx context.Context, tld string) (string, error)
}

// WhoisServerLookup performs a WHOIS lookup using a known server.
type WhoisServerLookup interface {
	LookupWithServer(ctx context.Context, server, domain string) (*WhoisResponse, error)
}

// WhoisResponse contains WHOIS response data.
type WhoisResponse struct {
	Server string
	Body   string
}

// DefaultWhoisClient is a TCP WHOIS client with optional server overrides.
type DefaultWhoisClient struct {
	Servers map[string]string
	Timeout time.Duration
}

// Lookup queries a WHOIS server for the given domain.
func (c *DefaultWhoisClient) Lookup(ctx context.Context, tld, domain string) (*WhoisResponse, error) {
	if strings.TrimSpace(domain) == "" {
		return nil, errors.New("whois domain is required")
	}

	server, err := c.ResolveServer(ctx, tld)
	if err != nil {
		return nil, err
	}

	body, err := queryWhois(ctx, server, domain, c.Timeout)
	if err != nil {
		return nil, err
	}

	return &WhoisResponse{Server: server, Body: body}, nil
}

// ResolveServer resolves the WHOIS server for a TLD.
func (c *DefaultWhoisClient) ResolveServer(ctx context.Context, tld string) (string, error) {
	tld = strings.ToLower(strings.TrimSpace(tld))
	if tld == "" {
		return "", errors.New("whois tld is required")
	}
	if c != nil && len(c.Servers) > 0 {
		if server := strings.TrimSpace(c.Servers[tld]); server != "" {
			return server, nil
		}
	}

	response, err := queryWhois(ctx, whoisIanaServer, tld, c.Timeout)
	if err != nil {
		return "", fmt.Errorf("whois iana query failed: %w", err)
	}

	for _, line := range strings.Split(response, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "refer:") || strings.HasPrefix(lower, "whois:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("no whois server for tld %s", tld)
}

// LookupWithServer queries a specific WHOIS server for a domain.
func (c *DefaultWhoisClient) LookupWithServer(ctx context.Context, server, domain string) (*WhoisResponse, error) {
	if strings.TrimSpace(domain) == "" {
		return nil, errors.New("whois domain is required")
	}
	body, err := queryWhois(ctx, server, domain, c.Timeout)
	if err != nil {
		return nil, err
	}
	return &WhoisResponse{Server: server, Body: body}, nil
}

func queryWhois(ctx context.Context, server, query string, timeout time.Duration) (string, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return "", errors.New("whois server is required")
	}

	dialer := &net.Dialer{}
	if timeout > 0 {
		dialer.Timeout = timeout
	}

	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(server, whoisPort))
	if err != nil {
		return "", fmt.Errorf("whois dial failed: %w", err)
	}
	defer conn.Close() // nolint:errcheck // best-effort cleanup on network connection

	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}

	if _, err := fmt.Fprintf(conn, "%s\r\n", query); err != nil {
		return "", fmt.Errorf("whois query failed: %w", err)
	}

	reader := bufio.NewReader(conn)
	limited := &io.LimitedReader{R: reader, N: whoisMaxBytes}
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("whois read failed: %w", err)
	}

	return string(body), nil
}

// WhoisPatterns provides match strings for availability parsing.
type WhoisPatterns struct {
	Available []string
	Taken     []string
}

func normalizeWhoisPatterns(cfg WhoisFallbackConfig) WhoisPatterns {
	available := cfg.AvailablePatterns
	taken := cfg.TakenPatterns
	if len(available) == 0 {
		available = []string{"no match", "not found", "no data found", "status: free"}
	}
	if len(taken) == 0 {
		taken = []string{"domain name:", "status: active", "registration status:", "created on"}
	}
	return WhoisPatterns{Available: available, Taken: taken}
}

func interpretWhois(body string, patterns WhoisPatterns) (core.Availability, string) {
	lower := strings.ToLower(body)
	for _, pattern := range patterns.Available {
		if pattern == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return core.AvailabilityAvailable, "whois not found"
		}
	}
	for _, pattern := range patterns.Taken {
		if pattern == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return core.AvailabilityTaken, "whois found"
		}
	}
	return core.AvailabilityUnknown, "whois ambiguous"
}

func whoisHash(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}
