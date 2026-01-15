package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/core"
)

func TestCargoCheckerAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"not found"}]}`))
	}))
	defer server.Close()

	checker := &CargoChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "nonexistent-crate")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityAvailable, result.Available)
	require.Equal(t, http.StatusNotFound, result.StatusCode)
}

func TestCargoCheckerTaken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"crate":{"name":"serde","max_version":"1.0.200","description":"A generic serialization/deserialization framework","repository":"https://github.com/serde-rs/serde"}}`))
	}))
	defer server.Close()

	checker := &CargoChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "serde")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityTaken, result.Available)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, "1.0.200", result.ExtraData["version"])
	require.Equal(t, "serde", result.ExtraData["name"])
}

func TestCargoCheckerSupportsName(t *testing.T) {
	checker := &CargoChecker{}

	tests := []struct {
		name     string
		expected bool
	}{
		{"serde", true},
		{"tokio-runtime", true},
		{"my_crate", true},
		{"Serde", true},
		{"a", true},
		{"a1", true},
		{"crate123", true},
		{"my-crate_v2", true},
		{"", false},
		{"1crate", false},
		{"-crate", false},
		{"_crate", false},
		{"crate/sub", false},
		{"crate.name", false},
		{"crate name", false},
		{string(make([]byte, 65)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checker.SupportsName(tt.name)
			require.Equal(t, tt.expected, got, "SupportsName(%q)", tt.name)
		})
	}
}

func TestCargoCheckerRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	checker := &CargoChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	result, err := checker.Check(context.Background(), "testcrate")
	require.NoError(t, err)
	require.Equal(t, core.AvailabilityRateLimited, result.Available)
	require.Equal(t, http.StatusTooManyRequests, result.StatusCode)
}

func TestCargoCheckerType(t *testing.T) {
	checker := &CargoChecker{}
	require.Equal(t, core.CheckTypeCargo, checker.Type())
}

func TestCargoCheckerRejectsInvalidName(t *testing.T) {
	requestMade := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &CargoChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	// Invalid names should be rejected without making HTTP request
	invalidNames := []string{"crate/sub", "1crate", "-crate", "_crate", "crate.name"}
	for _, name := range invalidNames {
		requestMade = false
		result, err := checker.Check(context.Background(), name)
		require.Error(t, err, "expected error for invalid name %q", name)
		require.Nil(t, result, "expected nil result for invalid name %q", name)
		require.False(t, requestMade, "expected no HTTP request for invalid name %q", name)
		require.Contains(t, err.Error(), "unsupported cargo crate name")
	}
}

func TestCargoCheckerProvenance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &CargoChecker{
		Store:       &stubRegistryStore{},
		Client:      server.Client(),
		BaseURL:     server.URL,
		ToolVersion: "1.2.3",
	}

	result, err := checker.Check(context.Background(), "testcrate")
	require.NoError(t, err)
	require.Equal(t, core.CheckTypeCargo, result.CheckType)
	require.Equal(t, "cargo", result.Provenance.Source)
	require.Equal(t, "1.2.3", result.Provenance.ToolVersion)
	require.NotEmpty(t, result.Provenance.CheckID)
	require.False(t, result.Provenance.RequestedAt.IsZero())
	require.False(t, result.Provenance.ResolvedAt.IsZero())
}

func TestCargoCheckerToolVersionDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify User-Agent header contains default version
		ua := r.Header.Get("User-Agent")
		require.Equal(t, "namelens/unknown", ua)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &CargoChecker{
		Store:   &stubRegistryStore{},
		Client:  server.Client(),
		BaseURL: server.URL,
		// ToolVersion intentionally not set
	}

	result, err := checker.Check(context.Background(), "testcrate")
	require.NoError(t, err)
	require.Equal(t, "unknown", result.Provenance.ToolVersion)
}
