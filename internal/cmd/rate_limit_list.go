package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fulmenhq/gofulmen/ascii"
	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/core/store"
	"github.com/namelens/namelens/internal/output"
)

var (
	rateLimitListOutput string
	rateLimitListOut    string
	rateLimitListOutDir string
	rateLimitListAll    bool
	rateLimitListPrefix string
)

var rateLimitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored rate limit state",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(rateLimitListOutput)
		if err != nil {
			return err
		}
		if format != output.FormatJSON && format != output.FormatTable {
			return fmt.Errorf("unsupported output format: %s", format)
		}

		db, err := openStore(cmd.Context())
		if err != nil {
			return err
		}
		defer db.Close() // nolint:errcheck // best-effort cleanup

		query := store.RateLimitQuery{
			All:    rateLimitListAll,
			Prefix: strings.TrimSpace(rateLimitListPrefix),
		}
		if !query.All && query.Prefix == "" {
			query.All = true
		}

		entries, err := db.ListRateLimits(cmd.Context(), query)
		if err != nil {
			return err
		}

		outPath := strings.TrimSpace(rateLimitListOut)
		outDir := strings.TrimSpace(rateLimitListOutDir)
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
			outPath = filepath.Join(outDir, fmt.Sprintf("rate-limit.list.%s", ext))
		}

		sink, err := openSink(outPath)
		if err != nil {
			return err
		}
		defer func() { _ = sink.close() }()

		if format == output.FormatJSON {
			payload, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(sink.writer, string(payload))
			return err
		}

		lines := []string{"Rate Limits", ""}
		if len(entries) == 0 {
			lines = append(lines, "(no stored rate limit state)")
			_, _ = fmt.Fprint(sink.writer, ascii.DrawBox(strings.Join(lines, "\n"), 0))
			return nil
		}

		for _, entry := range entries {
			backoff := "-"
			if entry.State.BackoffUntil != nil {
				backoff = entry.State.BackoffUntil.UTC().Format(time.RFC3339)
			}
			lines = append(lines, fmt.Sprintf("%s: count=%d backoff_until=%s", entry.Endpoint, entry.State.RequestCount, backoff))
		}

		_, _ = fmt.Fprint(sink.writer, ascii.DrawBox(strings.Join(lines, "\n"), 0))
		return nil
	},
}

func init() {
	rateLimitListCmd.Flags().StringVar(&rateLimitListOutput, "output-format", string(output.FormatTable), "Output format: table|json")
	rateLimitListCmd.Flags().StringVar(&rateLimitListOut, "out", "", "Write output to a file (default stdout)")
	rateLimitListCmd.Flags().StringVar(&rateLimitListOutDir, "out-dir", "", "Write output to a directory")
	rateLimitListCmd.Flags().BoolVar(&rateLimitListAll, "all", false, "List all endpoints")
	rateLimitListCmd.Flags().StringVar(&rateLimitListPrefix, "prefix", "", "List endpoints with matching prefix")
}
