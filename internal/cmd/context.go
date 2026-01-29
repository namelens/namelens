package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	ailinkctx "github.com/namelens/namelens/internal/ailink/context"
	"github.com/namelens/namelens/internal/observability"
	"go.uber.org/zap"
)

var contextCmd = &cobra.Command{
	Use:   "context <directory>",
	Short: "Generate context corpus from directory",
	Long: `Scan a directory and produce a structured corpus for AI prompts.

The corpus includes file classification, budget allocation, and content extraction.
Output can be JSON (schema-backed) or Markdown (human-readable).

Examples:
  # Generate JSON corpus
  namelens context ./planning > corpus.json

  # Generate Markdown corpus
  namelens context ./planning --output=markdown > corpus.md

  # Show manifest only (no content)
  namelens context ./planning --manifest-only

  # Use with generate command
  namelens context ./planning | namelens generate "concept" --corpus=-`,
	Args: cobra.ExactArgs(1),
	RunE: runContext,
}

func init() {
	rootCmd.AddCommand(contextCmd)

	contextCmd.Flags().StringP("output", "o", "json", "Output format: json, markdown, prompt")
	contextCmd.Flags().Int("budget", 32000, "Max characters to include")
	contextCmd.Flags().Bool("manifest-only", false, "Output manifest without content")
	contextCmd.Flags().StringSlice("include", nil, "Additional patterns to include")
	contextCmd.Flags().StringSlice("exclude", nil, "Patterns to exclude")
}

func runContext(cmd *cobra.Command, args []string) error {
	dir := args[0]

	outputFormat, _ := cmd.Flags().GetString("output")
	budget, _ := cmd.Flags().GetInt("budget")
	manifestOnly, _ := cmd.Flags().GetBool("manifest-only")
	includePatterns, _ := cmd.Flags().GetStringSlice("include")
	excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")

	// Build config
	cfg := ailinkctx.Config{
		Patterns: ailinkctx.DefaultPatterns,
		MaxChars: budget,
	}

	// Add additional include patterns
	if len(includePatterns) > 0 {
		cfg.Patterns = append(cfg.Patterns, includePatterns...)
	}

	// Note: exclude patterns would need to be implemented in Gather
	_ = excludePatterns // TODO: implement exclude filtering

	// Gather context
	result, err := ailinkctx.Gather(dir, cfg)
	if err != nil {
		return fmt.Errorf("gathering context: %w", err)
	}

	if verbose {
		observability.CLILogger.Debug("Context gathered",
			zap.String("dir", dir),
			zap.Int("files_included", len(result.Included)),
			zap.Int("files_excluded", len(result.Excluded)),
			zap.Int("chars", result.TotalChars))
	}

	// Convert to corpus
	corpus := ailinkctx.CorpusFromGatherResult(result, dir, budget)

	// If manifest-only, clear content
	if manifestOnly {
		corpus.Content = nil
	}

	// Output in requested format
	switch outputFormat {
	case "json":
		data, err := corpus.ToJSON()
		if err != nil {
			return fmt.Errorf("serializing corpus: %w", err)
		}
		fmt.Println(string(data))

	case "markdown", "md":
		fmt.Println(corpus.ToMarkdown())

	case "prompt":
		// Output format suitable for direct inclusion in prompts
		fmt.Println(corpus.ToPromptContext())

	default:
		return fmt.Errorf("unknown output format: %s (use json, markdown, or prompt)", outputFormat)
	}

	return nil
}
