package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
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
	generateCmd.Flags().StringP("description-file", "f", "", "Read description from file")
	generateCmd.Flags().Int("description-budget", 32000, "Max characters to include from description file")
	generateCmd.Flags().String("corpus", "", "Use pre-generated corpus file (JSON/markdown, or - for stdin)")
	generateCmd.Flags().StringP("scan-dir", "s", "", "Scan directory for context files (README.md, *.md, etc.)")
	generateCmd.Flags().Int("scan-budget", 32000, "Max characters to include from scanned files")
	generateCmd.Flags().StringP("constraints", "c", "", "Naming constraints/requirements")
	generateCmd.Flags().String("depth", "quick", "Generation depth: quick, deep")
	generateCmd.Flags().Bool("json", false, "Output raw JSON response")
	generateCmd.Flags().String("model", "", "Model override")
	generateCmd.Flags().String("prompt", "name-alternatives", "Prompt slug to use")
	generateCmd.Flags().String("provider", "", "Override provider for this run (must match an ailink.providers key)")
	generateCmd.Flags().Duration("timeout", 0, "Per-call ailink timeout (e.g. 180s, 3m); overrides ailink.default_timeout from config. 0 = use config default.")
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
	descriptionBudget, _ := cmd.Flags().GetInt("description-budget")
	constraints, _ := cmd.Flags().GetString("constraints")
	depth, _ := cmd.Flags().GetString("depth")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	modelOverride, _ := cmd.Flags().GetString("model")
	promptSlug, _ := cmd.Flags().GetString("prompt")
	providerOverride, _ := cmd.Flags().GetString("provider")
	timeoutFlag, _ := cmd.Flags().GetDuration("timeout")

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
		content, err := readTruncatedFile(descriptionFile, descriptionBudget)
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
	if strings.TrimSpace(providerOverride) != "" {
		ailinkCfg, err := applyGenerateProviderOverride(cfg.AILink, role, providerOverride)
		if err != nil {
			return err
		}
		providers = ailink.NewRegistry(ailinkCfg)
	}

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
		TimeoutSec: timeoutFlagToSec(timeoutFlag),
	})
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	// Output
	if jsonOutput {
		fmt.Println(string(response.Raw))
		return nil
	}

	return printGenerateResults(response.Raw, concept, promptDef.Config.Slug, promptDef.Config.ResponseSchema)
}

func applyGenerateProviderOverride(cfg ailink.Config, role, providerID string) (ailink.Config, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		return cfg, fmt.Errorf("provider override requires a role")
	}

	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return cfg, nil
	}

	providerCfg, ok := cfg.Providers[providerID]
	if !ok {
		return cfg, fmt.Errorf("unknown provider %q (valid: %s)", providerID, strings.Join(configuredProviderIDs(cfg.Providers), ", "))
	}
	if !providerCfg.Enabled {
		return cfg, fmt.Errorf("provider %q is disabled", providerID)
	}

	out := cfg
	out.Routing = cloneRoutingMap(cfg.Routing)
	out.Routing[role] = providerID
	return out, nil
}

func configuredProviderIDs(providers map[string]ailink.ProviderInstanceConfig) []string {
	ids := make([]string, 0, len(providers))
	for id := range providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func cloneRoutingMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func readTruncatedFile(path string, maxLen int) (result string, err error) {
	f, err := os.Open(path) // #nosec G304 -- user-provided --context-file path
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
		data, err = os.ReadFile(path) // #nosec G304 -- user-provided --corpus file path
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

func printGenerateResults(raw json.RawMessage, concept, promptSlug string, responseSchema map[string]any) error {
	rendered, err := renderGenerateResults(os.Stdout, raw, concept, promptSlug, responseSchemaRef(responseSchema))
	if err != nil {
		return err
	}
	if rendered {
		return nil
	}
	return printGenerateRawFallback(os.Stdout, raw, concept)
}

type generateNameAlternativesResult struct {
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

type generateSearchResponse struct {
	Summary          string                   `json:"summary"`
	LikelyAvailable  *bool                    `json:"likely_available,omitempty"`
	RiskLevel        string                   `json:"risk_level,omitempty"`
	Confidence       *float64                 `json:"confidence,omitempty"`
	BrandAssessment  generateBrandAssessment  `json:"brand_assessment,omitempty"`
	ConflictAnalysis generateConflictAnalysis `json:"conflict_analysis,omitempty"`
	DomainStrategy   generateDomainStrategy   `json:"domain_strategy,omitempty"`
	Insights         []string                 `json:"insights,omitempty"`
	Mentions         []generateMention        `json:"mentions,omitempty"`
	Recommendations  []string                 `json:"recommendations,omitempty"`
}

type generateBrandAssessment struct {
	Memorability    string `json:"memorability,omitempty"`
	DeveloperAppeal string `json:"developer_appeal,omitempty"`
	Pronunciation   string `json:"pronunciation,omitempty"`
	VisualPotential string `json:"visual_potential,omitempty"`
}

type generateConflictAnalysis struct {
	ExistingSoftware  []string `json:"existing_software,omitempty"`
	TrademarkConcerns string   `json:"trademark_concerns,omitempty"`
	SocialPresence    string   `json:"social_presence,omitempty"`
}

type generateDomainStrategy struct {
	RecommendedTLD string   `json:"recommended_tld,omitempty"`
	Rationale      string   `json:"rationale,omitempty"`
	Alternatives   []string `json:"alternatives,omitempty"`
}

type generateBrandPlanResponse struct {
	Summary              string                       `json:"summary"`
	LikelyAvailable      *bool                        `json:"likely_available,omitempty"`
	RiskLevel            string                       `json:"risk_level,omitempty"`
	BrandIdentity        generateBrandIdentity        `json:"brand_identity,omitempty"`
	ImmediateActions     generateImmediateActions     `json:"immediate_actions,omitempty"`
	LaunchChecklist      []generateLaunchPhase        `json:"launch_checklist,omitempty"`
	CompetitiveLandscape generateCompetitiveLandscape `json:"competitive_landscape,omitempty"`
	Insights             []string                     `json:"insights,omitempty"`
	Mentions             []generateMention            `json:"mentions,omitempty"`
	Recommendations      []string                     `json:"recommendations,omitempty"`
}

type generateBrandIdentity struct {
	PositioningStatement string   `json:"positioning_statement,omitempty"`
	TaglineOptions       []string `json:"tagline_options,omitempty"`
	BrandVoice           string   `json:"brand_voice,omitempty"`
	VisualDirection      string   `json:"visual_direction,omitempty"`
}

type generateImmediateActions struct {
	DomainsToRegister      []string `json:"domains_to_register,omitempty"`
	HandlesToClaim         []string `json:"handles_to_claim,omitempty"`
	TrademarkConsideration string   `json:"trademark_considerations,omitempty"`
}

type generateCompetitiveLandscape struct {
	SimilarTools                 []string `json:"similar_tools,omitempty"`
	DifferentiationOpportunities []string `json:"differentiation_opportunities,omitempty"`
}

type generateLaunchPhase struct {
	Phase    string   `json:"phase,omitempty"`
	Actions  []string `json:"actions,omitempty"`
	Priority string   `json:"priority,omitempty"`
}

type generateBulkSearchResponse struct {
	Summary string                     `json:"summary,omitempty"`
	Items   []generateBulkSearchResult `json:"items"`
}

type generateBulkSearchResult struct {
	Name            string            `json:"name"`
	Summary         string            `json:"summary"`
	LikelyAvailable *bool             `json:"likely_available,omitempty"`
	RiskLevel       string            `json:"risk_level,omitempty"`
	Confidence      *float64          `json:"confidence,omitempty"`
	Insights        []string          `json:"insights,omitempty"`
	Mentions        []generateMention `json:"mentions,omitempty"`
	Recommendations []string          `json:"recommendations,omitempty"`
}

type generateMention struct {
	Source      string `json:"source,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	Relevance   string `json:"relevance,omitempty"`
	Sentiment   string `json:"sentiment,omitempty"`
	Date        string `json:"date,omitempty"`
}

// errWriter wraps an io.Writer and captures the first write error so callers
// can drive sequences of fmt.Fprintf/Fprintln calls without checking each one.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) Fprintf(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func (ew *errWriter) Fprintln(args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintln(ew.w, args...)
}

func renderGenerateResults(w io.Writer, raw json.RawMessage, concept, promptSlug, schemaRef string) (bool, error) {
	switch strings.TrimSpace(schemaRef) {
	case "", "ailink/v0/name-alternatives-response":
		return renderNameAlternativesResults(w, raw, concept)
	case "ailink/v0/search-response":
		return renderSearchResponseResults(w, raw, concept, promptSlug)
	case "ailink/v0/brand-plan-response":
		return renderBrandPlanResults(w, raw, concept)
	case "ailink/v0/search-bulk-response":
		return renderBulkSearchResults(w, raw, concept)
	default:
		return false, nil
	}
}

func renderNameAlternativesResults(w io.Writer, raw json.RawMessage, concept string) (bool, error) {
	var result generateNameAlternativesResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return false, nil
	}

	if result.ConceptAnalysis.CoreFunction == "" && len(result.Candidates) == 0 && len(result.TopRecommendations) == 0 && len(result.NamingThemesExplored) == 0 && len(result.AvoidedPatterns) == 0 {
		return false, nil
	}

	ew := &errWriter{w: w}
	ew.Fprintf("Generating name alternatives for: %s\n\n", concept)

	if result.ConceptAnalysis.CoreFunction != "" {
		ew.Fprintln("Concept Analysis:")
		ew.Fprintf("  Core function: %s\n", result.ConceptAnalysis.CoreFunction)
		if len(result.ConceptAnalysis.KeyThemes) > 0 {
			ew.Fprintf("  Key themes: %s\n", strings.Join(result.ConceptAnalysis.KeyThemes, ", "))
		}
		if result.ConceptAnalysis.TargetAudience != "" {
			ew.Fprintf("  Target audience: %s\n", result.ConceptAnalysis.TargetAudience)
		}
		ew.Fprintln()
	}

	if len(result.TopRecommendations) > 0 {
		ew.Fprintln("Top Recommendations:")
		for i, rec := range result.TopRecommendations {
			ew.Fprintf("  %d. %s - %s\n", i+1, rec.Name, rec.Why)
		}
		ew.Fprintln()
	}

	if len(result.Candidates) > 0 {
		ew.Fprintln("All Candidates:")
		ew.Fprintf("  %-14s %-12s %-10s %s\n", "NAME", "STRATEGY", "STRENGTH", "CONFLICTS")
		for _, c := range result.Candidates {
			conflicts := c.PotentialConflicts
			if conflicts == "" {
				conflicts = "None found"
			}
			if len(conflicts) > 40 {
				conflicts = conflicts[:37] + "..."
			}
			ew.Fprintf("  %-14s %-12s %-10s %s\n", c.Name, c.Strategy, c.Strength, conflicts)
		}
		ew.Fprintln()
	}

	if len(result.NamingThemesExplored) > 0 {
		ew.Fprintf("Themes explored: %s\n", strings.Join(result.NamingThemesExplored, ", "))
	}

	writeGenerateFooter(ew)
	return true, ew.err
}

func renderSearchResponseResults(w io.Writer, raw json.RawMessage, concept, promptSlug string) (bool, error) {
	var result generateSearchResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return false, nil
	}
	if strings.TrimSpace(result.Summary) == "" {
		return false, nil
	}

	ew := &errWriter{w: w}
	ew.Fprintf("Brand assessment for: %s\n\n", concept)
	ew.Fprintf("Summary: %s\n", result.Summary)
	if status := formatGenerateAvailability(result.LikelyAvailable); status != "" {
		ew.Fprintf("Likely available: %s\n", status)
	}
	if risk := strings.TrimSpace(result.RiskLevel); risk != "" {
		ew.Fprintf("Risk level: %s\n", risk)
	}
	if confidence := formatConfidence(result.Confidence); confidence != "" {
		ew.Fprintf("Confidence: %s\n", confidence)
	}

	if strings.TrimSpace(promptSlug) == "brand-proposal" {
		writeBrandAssessmentSection(ew, result.BrandAssessment)
		writeConflictAnalysisSection(ew, result.ConflictAnalysis)
		writeDomainStrategySection(ew, result.DomainStrategy)
	}

	writeStringListSection(ew, "Insights", result.Insights)
	writeMentionSection(ew, "Mentions", result.Mentions)
	writeStringListSection(ew, "Recommendations", result.Recommendations)

	writeGenerateFooter(ew)
	return true, ew.err
}

func renderBrandPlanResults(w io.Writer, raw json.RawMessage, concept string) (bool, error) {
	var result generateBrandPlanResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return false, nil
	}
	if strings.TrimSpace(result.Summary) == "" {
		return false, nil
	}

	ew := &errWriter{w: w}
	ew.Fprintf("Brand plan for: %s\n\n", concept)
	ew.Fprintf("Summary: %s\n", result.Summary)
	if status := formatGenerateAvailability(result.LikelyAvailable); status != "" {
		ew.Fprintf("Likely available: %s\n", status)
	}
	if risk := strings.TrimSpace(result.RiskLevel); risk != "" {
		ew.Fprintf("Risk level: %s\n", risk)
	}

	if strings.TrimSpace(result.BrandIdentity.PositioningStatement) != "" || len(result.BrandIdentity.TaglineOptions) > 0 || strings.TrimSpace(result.BrandIdentity.BrandVoice) != "" || strings.TrimSpace(result.BrandIdentity.VisualDirection) != "" {
		ew.Fprintln()
		ew.Fprintln("Brand Identity:")
		if v := strings.TrimSpace(result.BrandIdentity.PositioningStatement); v != "" {
			ew.Fprintf("  Positioning: %s\n", v)
		}
		if len(result.BrandIdentity.TaglineOptions) > 0 {
			ew.Fprintf("  Taglines: %s\n", strings.Join(result.BrandIdentity.TaglineOptions, "; "))
		}
		if v := strings.TrimSpace(result.BrandIdentity.BrandVoice); v != "" {
			ew.Fprintf("  Voice: %s\n", v)
		}
		if v := strings.TrimSpace(result.BrandIdentity.VisualDirection); v != "" {
			ew.Fprintf("  Visual direction: %s\n", v)
		}
	}

	if len(result.ImmediateActions.DomainsToRegister) > 0 || len(result.ImmediateActions.HandlesToClaim) > 0 || strings.TrimSpace(result.ImmediateActions.TrademarkConsideration) != "" {
		ew.Fprintln()
		ew.Fprintln("Immediate Actions:")
		if len(result.ImmediateActions.DomainsToRegister) > 0 {
			ew.Fprintf("  Domains: %s\n", strings.Join(result.ImmediateActions.DomainsToRegister, ", "))
		}
		if len(result.ImmediateActions.HandlesToClaim) > 0 {
			ew.Fprintf("  Handles: %s\n", strings.Join(result.ImmediateActions.HandlesToClaim, ", "))
		}
		if v := strings.TrimSpace(result.ImmediateActions.TrademarkConsideration); v != "" {
			ew.Fprintf("  Trademark: %s\n", v)
		}
	}

	if len(result.LaunchChecklist) > 0 {
		ew.Fprintln()
		ew.Fprintln("Launch Checklist:")
		for _, phase := range result.LaunchChecklist {
			label := strings.TrimSpace(phase.Phase)
			if label == "" {
				label = "phase"
			}
			priority := strings.TrimSpace(phase.Priority)
			if priority != "" {
				label += " [" + priority + "]"
			}
			ew.Fprintf("  - %s\n", label)
			for _, action := range phase.Actions {
				if strings.TrimSpace(action) != "" {
					ew.Fprintf("    * %s\n", action)
				}
			}
		}
	}

	if len(result.CompetitiveLandscape.SimilarTools) > 0 || len(result.CompetitiveLandscape.DifferentiationOpportunities) > 0 {
		ew.Fprintln()
		ew.Fprintln("Competitive Landscape:")
		if len(result.CompetitiveLandscape.SimilarTools) > 0 {
			ew.Fprintf("  Similar tools: %s\n", strings.Join(result.CompetitiveLandscape.SimilarTools, ", "))
		}
		if len(result.CompetitiveLandscape.DifferentiationOpportunities) > 0 {
			ew.Fprintf("  Differentiation: %s\n", strings.Join(result.CompetitiveLandscape.DifferentiationOpportunities, "; "))
		}
	}

	writeStringListSection(ew, "Insights", result.Insights)
	writeMentionSection(ew, "Mentions", result.Mentions)
	writeStringListSection(ew, "Recommendations", result.Recommendations)

	writeGenerateFooter(ew)
	return true, ew.err
}

func renderBulkSearchResults(w io.Writer, raw json.RawMessage, concept string) (bool, error) {
	var result generateBulkSearchResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return false, nil
	}
	if len(result.Items) == 0 {
		return false, nil
	}

	ew := &errWriter{w: w}
	if strings.TrimSpace(concept) != "" {
		ew.Fprintf("Bulk name assessment for: %s\n", concept)
	} else {
		ew.Fprintln("Bulk name assessment")
	}
	if strings.TrimSpace(result.Summary) != "" {
		ew.Fprintf("\nSummary: %s\n", result.Summary)
	}

	for i, item := range result.Items {
		ew.Fprintln()
		ew.Fprintf("%d. %s\n", i+1, item.Name)
		ew.Fprintf("   %s\n", item.Summary)
		if status := formatGenerateAvailability(item.LikelyAvailable); status != "" {
			ew.Fprintf("   Likely available: %s\n", status)
		}
		if risk := strings.TrimSpace(item.RiskLevel); risk != "" {
			ew.Fprintf("   Risk level: %s\n", risk)
		}
		if confidence := formatConfidence(item.Confidence); confidence != "" {
			ew.Fprintf("   Confidence: %s\n", confidence)
		}
		writeIndentedStringListSection(ew, "Insights", item.Insights, "   ")
		writeIndentedStringListSection(ew, "Recommendations", item.Recommendations, "   ")
	}

	writeGenerateFooter(ew)
	return true, ew.err
}

func responseSchemaRef(responseSchema map[string]any) string {
	if len(responseSchema) == 0 {
		return ""
	}
	ref, _ := responseSchema["$ref"].(string)
	return strings.TrimSpace(ref)
}

func printGenerateRawFallback(w io.Writer, raw json.RawMessage, concept string) error {
	ew := &errWriter{w: w}
	if strings.TrimSpace(concept) != "" {
		ew.Fprintf("Generated output for: %s\n\n", concept)
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err == nil {
		ew.Fprintln(pretty.String())
	} else {
		ew.Fprintln(string(raw))
	}

	writeGenerateFooter(ew)
	return ew.err
}

func writeGenerateFooter(ew *errWriter) {
	ew.Fprintln("\nRun 'namelens check <name>' to verify availability.")
}

func writeStringListSection(ew *errWriter, title string, items []string) {
	writeIndentedStringListSection(ew, title, items, "")
}

func writeIndentedStringListSection(ew *errWriter, title string, items []string, indent string) {
	if len(items) == 0 {
		return
	}
	ew.Fprintln()
	if indent != "" {
		ew.Fprintf("%s%s:\n", indent, title)
		for _, item := range items {
			if strings.TrimSpace(item) != "" {
				ew.Fprintf("%s- %s\n", indent, item)
			}
		}
		return
	}
	ew.Fprintf("%s:\n", title)
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			ew.Fprintf("  - %s\n", item)
		}
	}
}

func writeMentionSection(ew *errWriter, title string, mentions []generateMention) {
	if len(mentions) == 0 {
		return
	}
	ew.Fprintln()
	ew.Fprintf("%s:\n", title)
	for _, mention := range mentions {
		parts := make([]string, 0, 4)
		if source := strings.TrimSpace(mention.Source); source != "" {
			parts = append(parts, source)
		}
		if relevance := strings.TrimSpace(mention.Relevance); relevance != "" {
			parts = append(parts, relevance)
		}
		if sentiment := strings.TrimSpace(mention.Sentiment); sentiment != "" {
			parts = append(parts, sentiment)
		}
		label := strings.Join(parts, "/")
		if label != "" {
			label = "[" + label + "] "
		}
		description := strings.TrimSpace(mention.Description)
		if description == "" {
			description = strings.TrimSpace(mention.URL)
		}
		if description == "" {
			continue
		}
		ew.Fprintf("  - %s%s\n", label, description)
		if url := strings.TrimSpace(mention.URL); url != "" {
			ew.Fprintf("    %s\n", url)
		}
	}
}

func writeBrandAssessmentSection(ew *errWriter, result generateBrandAssessment) {
	if strings.TrimSpace(result.Memorability) == "" && strings.TrimSpace(result.DeveloperAppeal) == "" && strings.TrimSpace(result.Pronunciation) == "" && strings.TrimSpace(result.VisualPotential) == "" {
		return
	}
	ew.Fprintln()
	ew.Fprintln("Brand Assessment:")
	if v := strings.TrimSpace(result.Memorability); v != "" {
		ew.Fprintf("  Memorability: %s\n", v)
	}
	if v := strings.TrimSpace(result.DeveloperAppeal); v != "" {
		ew.Fprintf("  Developer appeal: %s\n", v)
	}
	if v := strings.TrimSpace(result.Pronunciation); v != "" {
		ew.Fprintf("  Pronunciation: %s\n", v)
	}
	if v := strings.TrimSpace(result.VisualPotential); v != "" {
		ew.Fprintf("  Visual potential: %s\n", v)
	}
}

func writeConflictAnalysisSection(ew *errWriter, result generateConflictAnalysis) {
	if len(result.ExistingSoftware) == 0 && strings.TrimSpace(result.TrademarkConcerns) == "" && strings.TrimSpace(result.SocialPresence) == "" {
		return
	}
	ew.Fprintln()
	ew.Fprintln("Conflict Analysis:")
	if len(result.ExistingSoftware) > 0 {
		ew.Fprintf("  Existing software: %s\n", strings.Join(result.ExistingSoftware, ", "))
	}
	if v := strings.TrimSpace(result.TrademarkConcerns); v != "" {
		ew.Fprintf("  Trademark concerns: %s\n", v)
	}
	if v := strings.TrimSpace(result.SocialPresence); v != "" {
		ew.Fprintf("  Social presence: %s\n", v)
	}
}

func writeDomainStrategySection(ew *errWriter, result generateDomainStrategy) {
	if strings.TrimSpace(result.RecommendedTLD) == "" && strings.TrimSpace(result.Rationale) == "" && len(result.Alternatives) == 0 {
		return
	}
	ew.Fprintln()
	ew.Fprintln("Domain Strategy:")
	if v := strings.TrimSpace(result.RecommendedTLD); v != "" {
		ew.Fprintf("  Recommended TLD: %s\n", v)
	}
	if v := strings.TrimSpace(result.Rationale); v != "" {
		ew.Fprintf("  Rationale: %s\n", v)
	}
	if len(result.Alternatives) > 0 {
		ew.Fprintf("  Alternatives: %s\n", strings.Join(result.Alternatives, ", "))
	}
}

func formatGenerateAvailability(v *bool) string {
	if v == nil {
		return ""
	}
	if *v {
		return "yes"
	}
	return "no"
}

func formatConfidence(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.0f%%", *v*100)
}
