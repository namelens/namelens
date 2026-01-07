package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/core/checker"
	"github.com/namelens/namelens/internal/observability"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Manage RDAP bootstrap data",
}

var bootstrapUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Refresh RDAP bootstrap cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore(cmd.Context())
		if err != nil {
			return err
		}
		defer store.Close() // nolint:errcheck // best-effort cleanup; errors logged internally

		service := &checker.BootstrapService{Store: store}
		summary, err := service.Update(cmd.Context())
		if err != nil {
			return err
		}

		// Get database path for user info
		dbPath := getDBPath()

		observability.CLILogger.Info("Bootstrap cache updated",
			zap.Int("tld_count", summary.TLDCount),
			zap.String("version", summary.Version),
			zap.String("publication", formatTime(summary.Publication)),
			zap.String("fetched_at", formatTime(summary.FetchedAt)),
			zap.String("database", dbPath),
		)

		fmt.Printf("Fetched %d TLDs from IANA\n", summary.TLDCount)
		fmt.Printf("Database: %s\n", dbPath)
		return nil
	},
}

var bootstrapStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show bootstrap cache status",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore(cmd.Context())
		if err != nil {
			return err
		}
		defer store.Close() // nolint:errcheck // best-effort cleanup; errors logged internally

		service := &checker.BootstrapService{Store: store}
		status, err := service.Status(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Printf("Bootstrap cache: %d TLDs\n", status.TLDCount)
		fmt.Printf("Last updated: %s\n", formatTime(status.FetchedAt))
		if status.Publication.IsZero() {
			fmt.Printf("Publication: unknown\n")
		} else {
			fmt.Printf("Publication: %s\n", formatTime(status.Publication))
		}
		if status.Source != "" {
			fmt.Printf("Source: %s\n", status.Source)
		}
		if status.Version != "" {
			fmt.Printf("Version: %s\n", status.Version)
		}
		fmt.Printf("Database: %s\n", getDBPath())
		return nil
	},
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.UTC().Format(time.RFC3339)
}

// getDBPath returns the resolved database path from config
func getDBPath() string {
	cfg := config.GetConfig()
	if cfg == nil {
		return config.DefaultStorePath()
	}
	if cfg.Store.URL != "" {
		return cfg.Store.URL
	}
	dbPath := cfg.Store.Path
	if dbPath == "" {
		dbPath = config.DefaultStorePath()
	}
	if absPath, err := filepath.Abs(dbPath); err == nil {
		return absPath
	}
	return dbPath
}

func init() {
	bootstrapCmd.AddCommand(bootstrapUpdateCmd)
	bootstrapCmd.AddCommand(bootstrapStatusCmd)
	rootCmd.AddCommand(bootstrapCmd)
}
