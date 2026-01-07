package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/core"
)

func TestPyPICheckerAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &PyPIChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "example")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityAvailable, result.Available)
	require.Equal(t, http.StatusNotFound, result.StatusCode)
}

func TestPyPICheckerTaken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"info":{"name":"example","version":"1.0.0","summary":"Example","home_page":"https://example.com"}}`))
	}))
	defer server.Close()

	checker := &PyPIChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "example")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityTaken, result.Available)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, "1.0.0", result.ExtraData["version"])
}
