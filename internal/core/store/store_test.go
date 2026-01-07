package store

import (
	"testing"

	"github.com/namelens/namelens/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBuildLibsqlDSN(t *testing.T) {
	t.Run("URLUsesRawValue", func(t *testing.T) {
		cfg := config.StoreConfig{
			URL:       "libsql://example.turso.io",
			AuthToken: "token123",
		}

		dsn, err := buildLibsqlDSN(cfg)
		require.NoError(t, err)
		require.Equal(t, "libsql://example.turso.io?authToken=token123", dsn)
	})

	t.Run("URLWithExistingQuery", func(t *testing.T) {
		cfg := config.StoreConfig{
			URL:       "libsql://example.turso.io?foo=bar",
			AuthToken: "token123",
		}

		dsn, err := buildLibsqlDSN(cfg)
		require.NoError(t, err)
		require.Equal(t, "libsql://example.turso.io?authToken=token123&foo=bar", dsn)
	})

	t.Run("PathWithFilePrefix", func(t *testing.T) {
		cfg := config.StoreConfig{Path: "file:./namelens.db"}

		dsn, err := buildLibsqlDSN(cfg)
		require.NoError(t, err)
		require.Equal(t, "file:./namelens.db", dsn)
	})

	t.Run("PathMissing", func(t *testing.T) {
		cfg := config.StoreConfig{}

		_, err := buildLibsqlDSN(cfg)
		require.Error(t, err)
	})

	t.Run("MemoryPath", func(t *testing.T) {
		cfg := config.StoreConfig{Path: ":memory:"}

		dsn, err := buildLibsqlDSN(cfg)
		require.NoError(t, err)
		require.Equal(t, ":memory:", dsn)
	})
}
