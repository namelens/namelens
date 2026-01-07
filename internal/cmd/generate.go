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
	"github.com/namelens/namelens/internal/config"
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
	if description != "" {
		variables["description"] = description
	} else if descriptionFile != "" {
		content, err := readTruncatedFile(descriptionFile, 2000)
		if err != nil {
			return fmt.Errorf("reading description file: %w", err)
		}
		variables["description"] = content
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
