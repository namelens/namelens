package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/core/store"
	"github.com/namelens/namelens/internal/output"
)

var (
	rateLimitResetAll      bool
	rateLimitResetEndpoint string
	rateLimitResetPrefix   string
	rateLimitResetYes      bool
	rateLimitResetDryRun   bool
	rateLimitResetOutput   string
)

var rateLimitResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset stored rate limit state",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(rateLimitResetOutput)
		if err != nil {
			return err
		}
		if format != output.FormatJSON && format != output.FormatTable {
			return fmt.Errorf("unsupported output format: %s", format)
		}

		query := store.RateLimitQuery{
			All:      rateLimitResetAll,
			Endpoint: strings.TrimSpace(rateLimitResetEndpoint),
			Prefix:   strings.TrimSpace(rateLimitResetPrefix),
		}
		if err := query.Validate(); err != nil {
			return err
		}

		if query.All && !rateLimitResetYes && !rateLimitResetDryRun {
			return errors.New("--all requires --yes (or use --dry-run)")
		}

		db, err := openStore(cmd.Context())
		if err != nil {
			return err
		}
		defer db.Close() // nolint:errcheck // best-effort cleanup

		matched, err := db.CountRateLimits(cmd.Context(), query)
		if err != nil {
			return err
		}

		if rateLimitResetDryRun {
			return writeRateLimitResetResult(format, matched, 0, true)
		}

		deleted, err := db.ResetRateLimits(cmd.Context(), query)
		if err != nil {
			return err
		}

		return writeRateLimitResetResult(format, matched, deleted, false)
	},
}

func writeRateLimitResetResult(format output.Format, matched int, deleted int64, dryRun bool) error {
	result := map[string]any{
		"matched": matched,
		"deleted": deleted,
		"dry_run": dryRun,
	}

	if format == output.FormatJSON {
		payload, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, string(payload))
		return err
	}

	if dryRun {
		_, err := fmt.Fprintf(os.Stdout, "Would delete %d rate limit entr(ies)\n", matched)
		return err
	}
	_, err := fmt.Fprintf(os.Stdout, "Deleted %d/%d rate limit entr(ies)\n", deleted, matched)
	return err
}

func init() {
	rateLimitResetCmd.Flags().BoolVar(&rateLimitResetAll, "all", false, "Reset all endpoints")
	rateLimitResetCmd.Flags().StringVar(&rateLimitResetEndpoint, "endpoint", "", "Reset a single endpoint (exact match)")
	rateLimitResetCmd.Flags().StringVar(&rateLimitResetPrefix, "prefix", "", "Reset endpoints with matching prefix")
	rateLimitResetCmd.Flags().BoolVar(&rateLimitResetYes, "yes", false, "Confirm destructive reset")
	rateLimitResetCmd.Flags().BoolVar(&rateLimitResetDryRun, "dry-run", false, "Show what would be deleted")
	rateLimitResetCmd.Flags().StringVar(&rateLimitResetOutput, "output", string(output.FormatTable), "Output format: table|json")
}
