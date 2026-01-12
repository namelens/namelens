package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/core"
)

type stubBootstrapStore struct {
	servers map[string][]string
	cached  map[string]*core.CheckResult
}

func (s *stubBootstrapStore) SetRDAPServers(ctx context.Context, tld string, servers []string, updatedAt time.Time) error {
	if s.servers == nil {
		s.servers = make(map[string][]string)
	}
	s.servers[tld] = servers
	return nil
}

func (s *stubBootstrapStore) GetRDAPServers(ctx context.Context, tld string) ([]string, error) {
	if s.servers == nil {
		return nil, nil
	}
	return s.servers[tld], nil
}

func (s *stubBootstrapStore) SetBootstrapMeta(ctx context.Context, key, value string) error {
	return nil
}

func (s *stubBootstrapStore) GetBootstrapMeta(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (s *stubBootstrapStore) CountBootstrapTLDs(ctx context.Context) (int, error) {
	return len(s.servers), nil
}

func (s *stubBootstrapStore) GetCachedResult(ctx context.Context, name string, checkType core.CheckType, tld string) (*core.CheckResult, error) {
	if s.cached == nil {
		return nil, nil
	}
	key := name + "|" + string(checkType) + "|" + tld
	return s.cached[key], nil
}

func (s *stubBootstrapStore) SetCachedResult(ctx context.Context, name string, result *core.CheckResult, ttl time.Duration) error {
	return nil
}

func (s *stubBootstrapStore) GetRateLimit(ctx context.Context, endpoint string) (*core.RateLimitState, error) {
	return nil, nil
}

func (s *stubBootstrapStore) UpdateRateLimit(ctx context.Context, endpoint string, state *core.RateLimitState) error {
	return nil
}

func TestDomainCheckerAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := &stubBootstrapStore{servers: map[string][]string{"com": {server.URL}}}
	checker := &DomainChecker{Store: store, UseCache: true}

	result, err := checker.Check(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityAvailable, result.Available)
	require.Equal(t, http.StatusNotFound, result.StatusCode)
}

func TestDomainCheckerTaken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
  "objectClassName": "domain",
  "ldhName": "example.com",
  "status": ["active"],
  "events": [{"eventAction": "expiration", "eventDate": "2025-12-26T00:00:00Z"}]
}`))
	}))
	defer server.Close()

	store := &stubBootstrapStore{servers: map[string][]string{"com": {server.URL}}}
	checker := &DomainChecker{Store: store, UseCache: true}

	result, err := checker.Check(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityTaken, result.Available)
	require.Equal(t, http.StatusOK, result.StatusCode)
}

func TestDomainCheckerRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	store := &stubBootstrapStore{servers: map[string][]string{"com": {server.URL}}}
	checker := &DomainChecker{Store: store, UseCache: true}

	result, err := checker.Check(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityRateLimited, result.Available)
	require.Equal(t, http.StatusTooManyRequests, result.StatusCode)
}

func TestDomainCheckerUnsupported(t *testing.T) {
	store := &stubBootstrapStore{}
	checker := &DomainChecker{Store: store, UseCache: true}

	result, err := checker.Check(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityUnsupported, result.Available)
}

func TestDomainCheckerRDAPOverrideAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := &stubBootstrapStore{}
	checker := &DomainChecker{
		Store:         store,
		UseCache:      true,
		RDAPOverrides: map[string][]string{"dev": {server.URL}},
	}

	result, err := checker.Check(context.Background(), "example.dev")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityAvailable, result.Available)
	require.Equal(t, http.StatusNotFound, result.StatusCode)
	require.Equal(t, rdapSource, result.Provenance.Source)
	require.Equal(t, server.URL+"/domain/example.dev", result.Provenance.Server)
}

func TestDomainCheckerRDAPOverrideTaken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
  "objectClassName": "domain",
  "ldhName": "example.app",
  "status": ["active"],
  "events": [{"eventAction": "expiration", "eventDate": "2025-12-26T00:00:00Z"}]
}`))
	}))
	defer server.Close()

	store := &stubBootstrapStore{}
	checker := &DomainChecker{
		Store:         store,
		UseCache:      true,
		RDAPOverrides: map[string][]string{"app": {server.URL}},
	}

	result, err := checker.Check(context.Background(), "example.app")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityTaken, result.Available)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, rdapSource, result.Provenance.Source)
	require.Equal(t, server.URL+"/domain/example.app", result.Provenance.Server)
}

func TestDomainCheckerRDAPOverrideFallbackServer(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer fallback.Close()

	store := &stubBootstrapStore{}
	checker := &DomainChecker{
		Store:         store,
		UseCache:      true,
		RDAPOverrides: map[string][]string{"dev": {primary.URL, fallback.URL}},
	}

	result, err := checker.Check(context.Background(), "example.dev")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityAvailable, result.Available)
	require.Equal(t, http.StatusNotFound, result.StatusCode)
	require.Equal(t, fallback.URL+"/domain/example.dev", result.Provenance.Server)
}

func TestDomainCheckerRDAPOverrideCacheProvenance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cached := &core.CheckResult{
		Name:      "example",
		CheckType: core.CheckTypeDomain,
		TLD:       "dev",
		Available: core.AvailabilityTaken,
		ExtraData: map[string]any{
			"resolution_source": rdapSource,
			"resolution_server": server.URL + "/domain/example.dev",
		},
	}

	store := &stubBootstrapStore{
		cached: map[string]*core.CheckResult{
			"example|domain|dev": cached,
		},
	}
	checker := &DomainChecker{
		Store:         store,
		UseCache:      true,
		ToolVersion:   "test",
		RDAPOverrides: map[string][]string{"dev": {server.URL}},
	}

	result, err := checker.Check(context.Background(), "example.dev")
	require.NoError(t, err)
	require.True(t, result.Provenance.FromCache)
	require.Equal(t, rdapSource, result.Provenance.Source)
	require.Equal(t, server.URL+"/domain/example.dev", result.Provenance.Server)
	require.NotEmpty(t, result.Provenance.CheckID)
	require.NotEmpty(t, result.Provenance.ToolVersion)
}

type stubWhoisClient struct {
	response *WhoisResponse
	err      error
}

func (s *stubWhoisClient) Lookup(ctx context.Context, tld, domain string) (*WhoisResponse, error) {
	return s.response, s.err
}

func TestDomainCheckerWhoisFallback(t *testing.T) {
	store := &stubBootstrapStore{}
	checker := &DomainChecker{
		Store:    store,
		UseCache: true,
		Whois: &stubWhoisClient{
			response: &WhoisResponse{
				Server: "whois.nic.io",
				Body:   "No match for domain \"example.io\".",
			},
		},
		WhoisCfg: WhoisFallbackConfig{
			Enabled:         true,
			TLDs:            []string{"io"},
			RequireExplicit: true,
		},
	}

	result, err := checker.Check(context.Background(), "example.io")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityAvailable, result.Available)
	require.Equal(t, whoisSource, result.Provenance.Source)
}
