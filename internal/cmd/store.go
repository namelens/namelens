package cmd

import (
	"context"
	"fmt"

	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/core/store"
)

func openStore(ctx context.Context) (*store.Store, error) {
	cfg, err := config.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	db, err := store.Open(ctx, cfg.Store)
	if err != nil {
		return nil, err
	}

	if err := db.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := db.SeedBuiltInProfiles(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
