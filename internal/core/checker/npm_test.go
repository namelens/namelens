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

type stubRegistryStore struct {
	cached map[string]*core.CheckResult
}

func (s *stubRegistryStore) GetCachedResult(ctx context.Context, name string, checkType core.CheckType, tld string) (*core.CheckResult, error) {
	if s.cached == nil {
		return nil, nil
	}
	return s.cached[name+string(checkType)], nil
}

func (s *stubRegistryStore) SetCachedResult(ctx context.Context, name string, result *core.CheckResult, ttl time.Duration) error {
	if s.cached == nil {
		s.cached = make(map[string]*core.CheckResult)
	}
	s.cached[name+string(result.CheckType)] = result
	return nil
}

func (s *stubRegistryStore) GetRateLimit(ctx context.Context, endpoint string) (*core.RateLimitState, error) {
	return nil, nil
}

func (s *stubRegistryStore) UpdateRateLimit(ctx context.Context, endpoint string, state *core.RateLimitState) error {
	return nil
}

func TestNPMCheckerAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &NPMChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "example")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityAvailable, result.Available)
	require.Equal(t, http.StatusNotFound, result.StatusCode)
}

func TestNPMCheckerTaken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"example","dist-tags":{"latest":"1.2.3"}}`))
	}))
	defer server.Close()

	checker := &NPMChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "example")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityTaken, result.Available)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, "1.2.3", result.ExtraData["latest_version"])
}
