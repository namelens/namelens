package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/ailink"
	ailinkctx "github.com/namelens/namelens/internal/ailink/context"
	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/observability"
	"go.uber.org/zap"
)

var generateCmd = &cobra.Command{
	Use:   "generate <concept>",
	Short: "Generate name alternatives",
	Long:  "Generate naming candidates from a product concept using AI",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringP("current-name", "n", "", "Current working name seeking alternatives")
	generateCmd.Flags().StringP("tagline", "t", "", "Product tagline/slogan")
	generateCmd.Flags().StringP("description", "d", "", "Inline product description")
	generateCmd.Flags().StringP("description-file", "f", "", "Read description from file (truncated to 2000 chars)")
	generateCmd.Flags().String("corpus", "", "Use pre-generated corpus file (JSON/markdown, or - for stdin)")
	generateCmd.Flags().StringP("scan-dir", "s", "", "Scan directory for context files (README.md, *.md, etc.)")
	generateCmd.Flags().Int("scan-budget", 32000, "Max characters to include from scanned files")
	generateCmd.Flags().StringP("constraints", "c", "", "Naming constraints/requirements")
	generateCmd.Flags().String("depth", "quick", "Generation depth: quick, deep")
	generateCmd.Flags().Bool("json", false, "Output raw JSON response")
	generateCmd.Flags().String("model", "", "Model override")
	generateCmd.Flags().String("prompt", "name-alternatives", "Prompt slug to use")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	concept := strings.TrimSpace(args[0])
	if concept == "" {
		return errors.New("concept is required")
	}

	currentName, _ := cmd.Flags().GetString("current-name")
	tagline, _ := cmd.Flags().GetString("tagline")
	description, _ := cmd.Flags().GetString("description")
	descriptionFile, _ := cmd.Flags().GetString("description-file")
	corpusPath, _ := cmd.Flags().GetString("corpus")
	scanDir, _ := cmd.Flags().GetString("scan-dir")
	scanBudget, _ := cmd.Flags().GetInt("scan-budget")
	constraints, _ := cmd.Flags().GetString("constraints")
	depth, _ := cmd.Flags().GetString("depth")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	modelOverride, _ := cmd.Flags().GetString("model")
	promptSlug, _ := cmd.Flags().GetString("prompt")

	// Build variables map - use both "concept" and "name" keys for flexibility
	// Different prompts may use different variable names for the main input
	variables := map[string]string{
		"concept": concept,
		"name":    concept, // Also set as "name" for prompts that use that variable
		"input":   concept, // Also set as "input" for generic usage
	}
	if currentName != "" {
		variables["current_name"] = currentName
	}
	if tagline != "" {
		variables["tagline"] = tagline
	}

	// Gather description from various sources (priority: inline > corpus > file > scan-dir)
	if description != "" {
		variables["description"] = description
	} else if corpusPath != "" {
		// Load pre-generated corpus
		corpus, err := loadCorpus(corpusPath)
		if err != nil {
			return fmt.Errorf("loading corpus: %w", err)
		}
		variables["description"] = corpus.ToPromptContext()
		if verbose {
			observability.CLILogger.Debug("Context loaded from corpus",
				zap.String("source", corpus.Source.Path),
				zap.Int("files", corpus.Manifest.FilesIncluded),
				zap.Int("chars", corpus.Budget.UsedChars))
		}
	} else if descriptionFile != "" {
		content, err := readTruncatedFile(descriptionFile, 2000)
		if err != nil {
			return fmt.Errorf("reading description file: %w", err)
		}
		variables["description"] = content
	} else if scanDir != "" {
		// Scan directory for context files
		cfg := ailinkctx.Config{
			Patterns: ailinkctx.DefaultPatterns,
			MaxChars: scanBudget,
		}
		result, err := ailinkctx.Gather(scanDir, cfg)
		if err != nil {
			return fmt.Errorf("scanning directory: %w", err)
		}
		if result.Context != "" {
			variables["description"] = result.Context
			if verbose {
				observability.CLILogger.Debug("Context gathered from directory",
					zap.String("dir", scanDir),
					zap.Strings("files", result.FilesUsed),
					zap.Int("chars", result.TotalChars),
					zap.Int("trimmed", result.FilesTrimmed),
					zap.Int("skipped", result.FilesSkipped))
			}
		}
	}
	if constraints != "" {
		variables["constraints"] = constraints
	}

	ctx := cmd.Context()
	cfg, err := config.Load(ctx)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Build service
	registry, err := buildPromptRegistry(cfg)
	if err != nil {
		return fmt.Errorf("loading prompts: %w", err)
	}
	promptDef, err := registry.Get(promptSlug)
	if err != nil {
		return fmt.Errorf("prompt not found: %w", err)
	}

	providers := ailink.NewRegistry(cfg.AILink)
	role := promptSlug
	resolved, err := providers.Resolve(role, promptDef, modelOverride)
	if err != nil {
		return fmt.Errorf("resolving provider: %w", err)
	}
	if strings.TrimSpace(resolved.Credential.APIKey) == "" {
		return errors.New("provider API key not configured")
	}

	catalog, err := buildSchemaCatalog()
	if err != nil {
		return fmt.Errorf("loading schemas: %w", err)
	}

	service := &ailink.Service{
		Providers: providers,
		Registry:  registry,
		Catalog:   catalog,
	}

	// Execute generation
	response, err := service.Generate(ctx, ailink.GenerateRequest{
		Role:       role,
		PromptSlug: promptSlug,
		Variables:  variables,
		Depth:      depth,
		Model:      modelOverride,
		UseTools:   true,
	})
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	// Output
	if jsonOutput {
		fmt.Println(string(response.Raw))
		return nil
	}

	return printGenerateResults(response.Raw, concept)
}

func readTruncatedFile(path string, maxLen int) (result string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()

	if maxLen <= 0 {
		return "", nil
	}

	reader := bufio.NewReader(f)
	var builder strings.Builder
	builder.Grow(maxLen + 3)

	count := 0
	for count < maxLen+1 {
		r, _, readErr := reader.ReadRune()
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return "", readErr
		}
		if count < maxLen {
			builder.WriteRune(r)
		}
		count++
	}

	content := builder.String()
	if count > maxLen {
		content += "..."
	}
	return content, nil
}

// loadCorpus loads a corpus from a file path or stdin (if path is "-").
// It auto-detects JSON vs markdown format.
func loadCorpus(path string) (*ailinkctx.Corpus, error) {
	var data []byte
	var err error

	if path == "-" {
		// Read from stdin
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}
	}

	// Auto-detect format: JSON starts with { or whitespace then {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		return ailinkctx.ParseCorpusJSON(data)
	}

	// Assume markdown - convert to corpus
	return parseCorpusMarkdown(data)
}

// parseCorpusMarkdown extracts content from a markdown-formatted corpus.
// This is a simplified parser that extracts the content section.
func parseCorpusMarkdown(data []byte) (*ailinkctx.Corpus, error) {
	content := string(data)

	// For markdown corpus, we use the content directly as prompt context
	// The corpus command produces well-structured markdown that can be used as-is
	corpus := &ailinkctx.Corpus{
		Version: "1.0.0",
		Source: ailinkctx.CorpusSource{
			Type: "markdown",
			Path: "stdin",
		},
	}

	// Extract content section if present
	if idx := strings.Index(content, "## Content"); idx > 0 {
		// Use content from the Content section onwards
		corpus.Content = []ailinkctx.FileContent{
			{File: "corpus", Text: strings.TrimSpace(content[idx:])},
		}
	} else {
		// Use entire content
		corpus.Content = []ailinkctx.FileContent{
			{File: "corpus", Text: strings.TrimSpace(content)},
		}
	}

	return corpus, nil
}

func printGenerateResults(raw json.RawMessage, concept string) error {
	// Parse the JSON response
	var result struct {
		ConceptAnalysis struct {
			CoreFunction   string   `json:"core_function"`
			KeyThemes      []string `json:"key_themes"`
			TargetAudience string   `json:"target_audience"`
		} `json:"concept_analysis"`
		Candidates []struct {
			Name               string `json:"name"`
			Strategy           string `json:"strategy"`
			Rationale          string `json:"rationale"`
			Pronunciation      string `json:"pronunciation"`
			PotentialConflicts string `json:"potential_conflicts"`
			CLICommand         string `json:"cli_command"`
			Strength           string `json:"strength"`
		} `json:"candidates"`
		TopRecommendations []struct {
			Name string `json:"name"`
			Why  string `json:"why"`
		} `json:"top_recommendations"`
		NamingThemesExplored []string `json:"naming_themes_explored"`
		AvoidedPatterns      []string `json:"avoided_patterns"`
	}

	if err := json.Unmarshal(raw, &result); err != nil {
		// Fall back to raw output if parsing fails
		fmt.Println(string(raw))
		return nil
	}

	fmt.Printf("Generating name alternatives for: %s\n\n", concept)

	// Concept Analysis
	if result.ConceptAnalysis.CoreFunction != "" {
		fmt.Println("Concept Analysis:")
		fmt.Printf("  Core function: %s\n", result.ConceptAnalysis.CoreFunction)
		if len(result.ConceptAnalysis.KeyThemes) > 0 {
			fmt.Printf("  Key themes: %s\n", strings.Join(result.ConceptAnalysis.KeyThemes, ", "))
		}
		if result.ConceptAnalysis.TargetAudience != "" {
			fmt.Printf("  Target audience: %s\n", result.ConceptAnalysis.TargetAudience)
		}
		fmt.Println()
	}

	// Top Recommendations
	if len(result.TopRecommendations) > 0 {
		fmt.Println("Top Recommendations:")
		for i, rec := range result.TopRecommendations {
			fmt.Printf("  %d. %s - %s\n", i+1, rec.Name, rec.Why)
		}
		fmt.Println()
	}

	// All Candidates
	if len(result.Candidates) > 0 {
		fmt.Println("All Candidates:")
		fmt.Printf("  %-14s %-12s %-10s %s\n", "NAME", "STRATEGY", "STRENGTH", "CONFLICTS")
		for _, c := range result.Candidates {
			conflicts := c.PotentialConflicts
			if conflicts == "" {
				conflicts = "None found"
			}
			// Truncate conflicts for display
			if len(conflicts) > 40 {
				conflicts = conflicts[:37] + "..."
			}
			fmt.Printf("  %-14s %-12s %-10s %s\n", c.Name, c.Strategy, c.Strength, conflicts)
		}
		fmt.Println()
	}

	// Themes explored
	if len(result.NamingThemesExplored) > 0 {
		fmt.Printf("Themes explored: %s\n", strings.Join(result.NamingThemesExplored, ", "))
	}

	fmt.Println("\nRun 'namelens check <name>' to verify availability.")
	return nil
}
