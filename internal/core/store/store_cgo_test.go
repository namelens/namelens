//go:build cgo

package store

import (
	"context"
	"testing"

	"github.com/namelens/namelens/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpenMemoryStore(t *testing.T) {
	ctx := context.Background()
	cfg := config.StoreConfig{
		Driver: "libsql",
		Path:   ":memory:",
	}

	store, err := Open(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, store)
	require.Equal(t, "libsql", store.Driver())
	require.NoError(t, store.Close())
}
