//go:build cgo

package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/core"
)

func TestProfileCRUD(t *testing.T) {
	ctx := context.Background()
	cfg := config.StoreConfig{
		Driver: "libsql",
		Path:   ":memory:",
	}

	store, err := Open(ctx, cfg)
	require.NoError(t, err)
	require.NoError(t, store.Migrate(ctx))
	defer store.Close() // nolint:errcheck // test cleanup

	require.NoError(t, store.SeedBuiltInProfiles(ctx))

	builtin, err := store.GetProfile(ctx, "startup")
	require.NoError(t, err)
	require.NotNil(t, builtin)
	require.True(t, builtin.IsBuiltin)

	custom := core.Profile{
		Name:        "custom",
		Description: "Custom profile",
		TLDs:        []string{"com"},
		Registries:  []string{"npm"},
		Handles:     []string{"github"},
	}
	require.NoError(t, store.UpsertProfile(ctx, custom, false, time.Now().UTC()))

	record, err := store.GetProfile(ctx, "custom")
	require.NoError(t, err)
	require.NotNil(t, record)
	require.False(t, record.IsBuiltin)
	require.Equal(t, custom.Description, record.Profile.Description)

	profiles, err := store.ListProfiles(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, profiles)
}
