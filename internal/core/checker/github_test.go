package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/core"
)

func TestGitHubCheckerAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &GitHubChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "example")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityAvailable, result.Available)
	require.Equal(t, http.StatusNotFound, result.StatusCode)
}

func TestGitHubCheckerTaken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"login":"example","id":123,"html_url":"https://github.com/example","type":"User"}`))
	}))
	defer server.Close()

	checker := &GitHubChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "example")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityTaken, result.Available)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, "https://github.com/example", result.ExtraData["html_url"])
}

func TestGitHubCheckerRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	checker := &GitHubChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "example")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityRateLimited, result.Available)
	require.Equal(t, http.StatusForbidden, result.StatusCode)
	require.Equal(t, "30", result.ExtraData["retry_after"])
}

func TestGitHubSupportsName(t *testing.T) {
	checker := &GitHubChecker{}
	require.True(t, checker.SupportsName("example"))
	require.False(t, checker.SupportsName("-bad"))
	require.False(t, checker.SupportsName("bad-"))
	require.False(t, checker.SupportsName("bad--name"))
	require.False(t, checker.SupportsName("bad_name"))
}
