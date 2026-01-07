//go:build cgo

package store

import (
	"context"
	"testing"

	"github.com/namelens/namelens/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpenLocalStore_ConfiguresSQLite(t *testing.T) {
	ctx := context.Background()

	cfg := config.StoreConfig{
		Driver: "libsql",
		Path:   "file:" + t.TempDir() + "/namelens.db",
	}

	store, err := Open(ctx, cfg)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	require.Equal(t, 1, store.DB.Stats().MaxOpenConnections)

	var journalMode string
	require.NoError(t, store.DB.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode))
	require.Contains(t, journalMode, "wal")

	var busyTimeout int
	require.NoError(t, store.DB.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout))
	require.GreaterOrEqual(t, busyTimeout, 1000)
}
