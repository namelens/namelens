package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
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
	rateLimitResetOut      string
	rateLimitResetOutDir   string
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

		outPath := strings.TrimSpace(rateLimitResetOut)
		outDir := strings.TrimSpace(rateLimitResetOutDir)
		if outPath != "" && outDir != "" {
			return fmt.Errorf("--out and --out-dir are mutually exclusive")
		}
		ext := outputExtension(format)
		if outDir != "" {
			var err error
			outDir, err = ensureOutDir(outDir)
			if err != nil {
				return err
			}
			outPath = filepath.Join(outDir, fmt.Sprintf("rate-limit.reset.%s", ext))
		}
		sink, err := openSink(outPath)
		if err != nil {
			return err
		}
		defer func() { _ = sink.close() }()

		if rateLimitResetDryRun {
			return writeRateLimitResetResult(format, sink.writer, matched, 0, true)
		}

		deleted, err := db.ResetRateLimits(cmd.Context(), query)
		if err != nil {
			return err
		}

		return writeRateLimitResetResult(format, sink.writer, matched, deleted, false)
	},
}

func writeRateLimitResetResult(format output.Format, w io.Writer, matched int, deleted int64, dryRun bool) error {
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
		_, err = fmt.Fprintln(w, string(payload))
		return err
	}

	if dryRun {
		_, err := fmt.Fprintf(w, "Would delete %d rate limit entr(ies)\n", matched)
		return err
	}
	_, err := fmt.Fprintf(w, "Deleted %d/%d rate limit entr(ies)\n", deleted, matched)
	return err
}

func init() {
	rateLimitResetCmd.Flags().BoolVar(&rateLimitResetAll, "all", false, "Reset all endpoints")
	rateLimitResetCmd.Flags().StringVar(&rateLimitResetEndpoint, "endpoint", "", "Reset a single endpoint (exact match)")
	rateLimitResetCmd.Flags().StringVar(&rateLimitResetPrefix, "prefix", "", "Reset endpoints with matching prefix")
	rateLimitResetCmd.Flags().BoolVar(&rateLimitResetYes, "yes", false, "Confirm destructive reset")
	rateLimitResetCmd.Flags().BoolVar(&rateLimitResetDryRun, "dry-run", false, "Show what would be deleted")
	rateLimitResetCmd.Flags().StringVar(&rateLimitResetOutput, "output-format", string(output.FormatTable), "Output format: table|json")
	rateLimitResetCmd.Flags().StringVar(&rateLimitResetOut, "out", "", "Write output to a file (default stdout)")
	rateLimitResetCmd.Flags().StringVar(&rateLimitResetOutDir, "out-dir", "", "Write output to a directory")
}
