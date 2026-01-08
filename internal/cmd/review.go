package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fulmenhq/gofulmen/ascii"
	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/ailink"
	"github.com/namelens/namelens/internal/ailink/prompt"
	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/core"
	"github.com/namelens/namelens/internal/output"
)

type includeRawMode string

const (
	includeRawNever  includeRawMode = "never"
	includeRawOnFail includeRawMode = "on-failure"
	includeRawAlways includeRawMode = "always"
)

type reviewResult struct {
	Name         string                    `json:"name"`
	Profile      string                    `json:"profile"`
	Mode         string                    `json:"mode"`
	Depth        string                    `json:"depth"`
	StartedAt    time.Time                 `json:"started_at"`
	CompletedAt  time.Time                 `json:"completed_at"`
	Availability reviewAvailability        `json:"availability"`
	Analyses     map[string]reviewAnalysis `json:"analyses"`
}

type reviewAvailability struct {
	Results     []*core.CheckResult `json:"results"`
	Score       int                 `json:"score"`
	Total       int                 `json:"total"`
	Unknown     int                 `json:"unknown"`
	CompletedAt time.Time           `json:"completed_at"`
}

type reviewAnalysis struct {
	OK    bool                `json:"ok"`
	Data  json.RawMessage     `json:"data,omitempty"`
	Error *ailink.SearchError `json:"error,omitempty"`
	Raw   json.RawMessage     `json:"raw,omitempty"`
}

var reviewCmd = &cobra.Command{
	Use:   "review <name>",
	Short: "Run a stitched name review workflow",
	Long:  "Review runs availability checks plus a mode-selected set of AILink analysis prompts.",
	Args:  cobra.ExactArgs(1),
	RunE:  runReview,
}

func init() {
	rootCmd.AddCommand(reviewCmd)

	reviewCmd.Flags().String("profile", "startup", "Availability profile to use")
	reviewCmd.Flags().String("mode", "core", "Review mode: core, brand, full")
	reviewCmd.Flags().String("depth", "quick", "Analysis depth: quick, deep")
	reviewCmd.Flags().String("output", "table", "Output format: table, json, markdown")
	reviewCmd.Flags().String("include-raw", string(includeRawOnFail), "Include raw analysis output: never, on-failure, always")
	reviewCmd.Flags().Bool("strict", false, "Return non-zero if any analysis fails")
	reviewCmd.Flags().Bool("no-cache", false, "Skip cache lookup")
}

func runReview(cmd *cobra.Command, args []string) error {
	name := strings.ToLower(strings.TrimSpace(args[0]))
	if err := validateName(name); err != nil {
		return err
	}

	profileName, err := cmd.Flags().GetString("profile")
	if err != nil {
		return err
	}
	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		return err
	}
	depth, err := cmd.Flags().GetString("depth")
	if err != nil {
		return err
	}
	formatValue, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	includeRawValue, err := cmd.Flags().GetString("include-raw")
	if err != nil {
		return err
	}
	strict, err := cmd.Flags().GetBool("strict")
	if err != nil {
		return err
	}
	noCache, err := cmd.Flags().GetBool("no-cache")
	if err != nil {
		return err
	}

	format, err := output.ParseFormat(formatValue)
	if err != nil {
		return err
	}

	rawMode, err := parseIncludeRaw(includeRawValue)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	startedAt := time.Now()

	store, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close() // nolint:errcheck // best-effort cleanup

	cfg := config.GetConfig()
	if cfg == nil {
		return errors.New("config not loaded")
	}

	profile, err := resolveProfile(ctx, store, profileName, nil, nil, nil)
	if err != nil {
		return err
	}
	if len(profile.TLDs) == 0 && len(profile.Registries) == 0 && len(profile.Handles) == 0 {
		return errors.New("at least one check target is required")
	}

	orchestrator := buildOrchestrator(cfg, store, !noCache)
	results, err := orchestrator.Check(ctx, name, profile)
	if err != nil {
		return err
	}

	registry, err := buildPromptRegistry(cfg)
	if err != nil {
		return err
	}

	promptSlugs, err := reviewPromptSet(mode, registry)
	if err != nil {
		return err
	}

	analyses := make(map[string]reviewAnalysis, len(promptSlugs))

	var (
		expertResult    *ailink.SearchResponse
		expertError     *ailink.SearchError
		phoneticsResult json.RawMessage
		phoneticsError  *ailink.SearchError
		suitabilityRaw  json.RawMessage
		suitabilityErr  *ailink.SearchError
	)

	for _, slug := range promptSlugs {
		switch slug {
		case "name-availability":
			expertResult, expertError = runExpert(ctx, cfg, store, name, depth, "", slug, !noCache)
			a := reviewAnalysis{OK: expertError == nil}
			if expertError != nil {
				a.Error = expertError
			}
			if expertResult != nil {
				payload, _ := json.Marshal(expertResult)
				a.Data = json.RawMessage(payload)
				if rawMode == includeRawAlways {
					a.Raw = expertResult.Raw
				}
			}
			analyses[slug] = a
		case "name-phonetics":
			vars := map[string]string{"name": name}
			phoneticsResult, phoneticsError = runAnalysis(ctx, cfg, store, slug, name, depth, "", vars, !noCache)
			analyses[slug] = analysisFromGenerate(phoneticsResult, phoneticsError, rawMode)
		case "name-suitability":
			vars := map[string]string{"name": name}
			suitabilityRaw, suitabilityErr = runAnalysis(ctx, cfg, store, slug, name, depth, "", vars, !noCache)
			analyses[slug] = analysisFromGenerate(suitabilityRaw, suitabilityErr, rawMode)
		default:
			vars := map[string]string{"name": name}
			data, errInfo := runAnalysis(ctx, cfg, store, slug, name, depth, "", vars, !noCache)
			analyses[slug] = analysisFromGenerate(data, errInfo, rawMode)
		}
	}

	batch := summarizeResults(name, results, expertResult, expertError, phoneticsResult, phoneticsError, suitabilityRaw, suitabilityErr)

	availability := reviewAvailability{
		Results:     batch.Results,
		Score:       batch.Score,
		Total:       batch.Total,
		Unknown:     batch.Unknown,
		CompletedAt: batch.CompletedAt,
	}

	review := &reviewResult{
		Name:         name,
		Profile:      profileName,
		Mode:         strings.ToLower(strings.TrimSpace(mode)),
		Depth:        strings.ToLower(strings.TrimSpace(depth)),
		StartedAt:    startedAt.UTC(),
		CompletedAt:  time.Now().UTC(),
		Availability: availability,
		Analyses:     analyses,
	}

	failed := analysisFailures(analyses)
	if strict && failed > 0 {
		// Still render output first.
		defer func() {
			_ = failed
		}()
	}

	switch format {
	case output.FormatJSON:
		payload, err := json.MarshalIndent(review, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, string(payload))
		if err != nil {
			return err
		}
		if strict && failed > 0 {
			return fmt.Errorf("review failed (%d analyses)", failed)
		}
		return nil
	case output.FormatMarkdown:
		rendered, err := output.NewFormatter(output.FormatMarkdown).FormatBatch(batch)
		if err != nil {
			return err
		}
		if rendered != "" {
			if _, err := fmt.Fprintln(os.Stdout, rendered); err != nil {
				return err
			}
		}
		renderReviewExtrasMarkdown(os.Stdout, analyses, []string{"name-availability", "name-phonetics", "name-suitability"})
		if strict && failed > 0 {
			return fmt.Errorf("review failed (%d analyses)", failed)
		}
		return nil
	default:
		// Table
		rendered, err := output.NewFormatter(output.FormatTable).FormatBatch(batch)
		if err != nil {
			return err
		}
		if strings.TrimSpace(rendered) != "" {
			if _, err := fmt.Fprintln(os.Stdout, rendered); err != nil {
				return err
			}
		}
		renderReviewExtrasTable(os.Stdout, analyses, []string{"name-availability", "name-phonetics", "name-suitability"})
		if strict && failed > 0 {
			return fmt.Errorf("review failed (%d analyses)", failed)
		}
		return nil
	}
}

func parseIncludeRaw(value string) (includeRawMode, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", string(includeRawOnFail):
		return includeRawOnFail, nil
	case string(includeRawNever):
		return includeRawNever, nil
	case string(includeRawAlways):
		return includeRawAlways, nil
	default:
		return "", fmt.Errorf("unsupported include-raw value: %s", value)
	}
}

func reviewPromptSet(mode string, registry interface {
	List() []*prompt.Prompt
	Get(string) (*prompt.Prompt, error)
}) ([]string, error) {
	_ = registry
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "core"
	}

	// Core prompt set is stable and schema-backed.
	core := []string{"name-availability", "name-phonetics", "name-suitability"}

	switch mode {
	case "core":
		return core, nil
	case "brand":
		return append(core, "brand-proposal", "brand-plan"), nil
	case "full":
		// Best-effort: include prompts that only require `name`.
		prompts := registry.List()
		set := make([]string, 0, len(prompts))
		for _, p := range prompts {
			if p == nil {
				continue
			}
			if !promptSupportsNameOnly(p) {
				continue
			}
			set = append(set, p.Config.Slug)
		}
		sort.Strings(set)
		return set, nil
	default:
		return nil, fmt.Errorf("unsupported mode: %s", mode)
	}
}

func promptSupportsNameOnly(p *prompt.Prompt) bool {
	if p == nil {
		return false
	}
	if p.Config.Input.AcceptsImages {
		return false
	}

	for _, required := range p.Config.Input.RequiredVariables {
		r := strings.TrimSpace(required)
		if r == "" {
			continue
		}
		if r != "name" && r != "depth" {
			return false
		}
	}
	return true
}

func analysisFromGenerate(data json.RawMessage, errInfo *ailink.SearchError, rawMode includeRawMode) reviewAnalysis {
	a := reviewAnalysis{OK: errInfo == nil}
	if errInfo != nil {
		a.Error = errInfo
		return a
	}
	a.Data = data
	if rawMode == includeRawAlways {
		a.Raw = data
	}
	return a
}

func analysisFailures(analyses map[string]reviewAnalysis) int {
	count := 0
	for _, a := range analyses {
		if !a.OK {
			count++
		}
	}
	return count
}

func renderReviewExtrasTable(w io.Writer, analyses map[string]reviewAnalysis, base []string) {
	if w == nil {
		return
	}
	baseSet := make(map[string]struct{}, len(base))
	for _, s := range base {
		baseSet[s] = struct{}{}
	}

	extra := make([]string, 0)
	for slug := range analyses {
		if _, ok := baseSet[slug]; ok {
			continue
		}
		extra = append(extra, slug)
	}
	sort.Strings(extra)
	if len(extra) == 0 {
		return
	}

	lines := []string{"Additional analyses", ""}
	for _, slug := range extra {
		a := analyses[slug]
		status := "ok"
		if !a.OK {
			status = "error"
		}
		summary := extractSummary(a.Data)
		if summary != "" {
			lines = append(lines, fmt.Sprintf("%s: %s (%s)", slug, status, summary))
		} else {
			lines = append(lines, fmt.Sprintf("%s: %s", slug, status))
		}
	}

	_, _ = fmt.Fprint(w, ascii.DrawBox(strings.Join(lines, "\n"), 0))
}

func renderReviewExtrasMarkdown(w io.Writer, analyses map[string]reviewAnalysis, base []string) {
	if w == nil {
		return
	}
	baseSet := make(map[string]struct{}, len(base))
	for _, s := range base {
		baseSet[s] = struct{}{}
	}

	extra := make([]string, 0)
	for slug := range analyses {
		if _, ok := baseSet[slug]; ok {
			continue
		}
		extra = append(extra, slug)
	}
	sort.Strings(extra)
	if len(extra) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "\n## Additional analyses")
	for _, slug := range extra {
		a := analyses[slug]
		status := "ok"
		if !a.OK {
			status = "error"
		}
		summary := extractSummary(a.Data)
		if summary != "" {
			_, _ = fmt.Fprintf(w, "- `%s`: %s (%s)\n", slug, status, summary)
		} else {
			_, _ = fmt.Fprintf(w, "- `%s`: %s\n", slug, status)
		}
	}
}

func extractSummary(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return ""
	}
	if v, ok := obj["summary"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}
