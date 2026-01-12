package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fulmenhq/gofulmen/ascii"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/namelens/namelens/internal/ailink"
	"github.com/namelens/namelens/internal/ailink/prompt"
	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/core"
	corestore "github.com/namelens/namelens/internal/core/store"
	"github.com/namelens/namelens/internal/observability"
	"github.com/namelens/namelens/internal/output"
)

type includeRawMode string

const (
	includeRawNever  includeRawMode = "never"
	includeRawOnFail includeRawMode = "on-failure"
	includeRawAlways includeRawMode = "always"
)

func rawFromAILinkError(err error) json.RawMessage {
	var rawErr *ailink.RawResponseError
	if errors.As(err, &rawErr) && rawErr != nil && len(rawErr.Raw) > 0 {
		return rawErr.Raw
	}
	return nil
}

func runReviewSearch(ctx context.Context, cfg *config.Config, store *corestore.Store, name, depth, modelOverride, promptSlug string, useCache bool) (*ailink.SearchResponse, *ailink.SearchError, json.RawMessage) {
	if cfg == nil {
		return nil, &ailink.SearchError{Code: "AILINK_DISABLED", Message: "config not loaded"}, nil
	}

	promptSlug = strings.TrimSpace(promptSlug)
	if promptSlug == "" {
		promptSlug = "name-availability"
	}

	depth = strings.ToLower(strings.TrimSpace(depth))
	if depth == "" {
		depth = "quick"
	}

	registry, err := buildPromptRegistry(cfg)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to load prompts", Details: err.Error()}, nil
	}
	promptDef, err := registry.Get(promptSlug)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_PROMPT_NOT_FOUND", Message: err.Error()}, nil
	}

	providers := ailink.NewRegistry(cfg.AILink)
	role := strings.TrimSpace(cfg.Expert.Role)
	if role == "" {
		role = promptSlug
	}

	resolved, err := providers.Resolve(role, promptDef, modelOverride)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to resolve provider", Details: err.Error()}, nil
	}
	if strings.TrimSpace(resolved.Credential.APIKey) == "" {
		return nil, &ailink.SearchError{Code: "AILINK_NO_API_KEY", Message: "provider api key not configured", Details: resolved.ProviderID}, nil
	}

	cacheTTL := cfg.AILink.CacheTTL
	if useCache && store != nil && cacheTTL > 0 {
		entry, err := store.GetExpertCache(ctx, name, promptSlug, resolved.Model, resolved.BaseURL, depth)
		if err != nil {
			observability.CLILogger.Warn("Expert cache lookup failed", zap.Error(err))
		} else if entry != nil {
			response, err := decodeCachedExpert(entry.ResponseJSON)
			if err == nil {
				return response, nil, response.Raw
			}
			observability.CLILogger.Warn("Expert cache decode failed", zap.Error(err))
		}
	}

	catalog, err := buildSchemaCatalog()
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to load schemas", Details: err.Error()}, nil
	}

	svc := &ailink.Service{Providers: providers, Registry: registry, Catalog: catalog}
	response, err := svc.Search(ctx, ailink.SearchRequest{Role: role, Name: name, PromptSlug: promptSlug, Depth: depth, Model: modelOverride, UseTools: true})
	if err != nil {
		return nil, mapExpertError(err), rawFromAILinkError(err)
	}

	raw := json.RawMessage(response.Raw)
	if strings.TrimSpace(string(raw)) == "" {
		payload, err := json.Marshal(response)
		if err == nil {
			raw = payload
		}
	}

	if useCache && store != nil && cacheTTL > 0 {
		encoded := strings.TrimSpace(string(raw))
		if encoded != "" {
			if err := store.SetExpertCache(ctx, name, promptSlug, resolved.Model, resolved.BaseURL, depth, encoded, cacheTTL); err != nil {
				observability.CLILogger.Warn("Expert cache write failed", zap.Error(err))
			}
		}
	}

	response.Raw = append(response.Raw[:0], raw...)
	return response, nil, raw
}

func runReviewGenerate(ctx context.Context, cfg *config.Config, store *corestore.Store, promptSlug, name, depth, modelOverride string, variables map[string]string, useCache bool) (json.RawMessage, *ailink.SearchError, json.RawMessage) {
	if cfg == nil {
		return nil, &ailink.SearchError{Code: "AILINK_DISABLED", Message: "config not loaded"}, nil
	}

	promptSlug = strings.TrimSpace(promptSlug)
	if promptSlug == "" {
		return nil, &ailink.SearchError{Code: "AILINK_PROMPT_NOT_FOUND", Message: "prompt slug is required"}, nil
	}

	depth = strings.ToLower(strings.TrimSpace(depth))
	if depth == "" {
		depth = "quick"
	}

	cleaned := make(map[string]string, len(variables)+1)
	for key, value := range variables {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		cleaned[key] = trimmed
	}
	if strings.TrimSpace(name) != "" {
		cleaned["name"] = strings.TrimSpace(name)
	}

	registry, err := buildPromptRegistry(cfg)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to load prompts", Details: err.Error()}, nil
	}
	promptDef, err := registry.Get(promptSlug)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_PROMPT_NOT_FOUND", Message: err.Error()}, nil
	}

	providers := ailink.NewRegistry(cfg.AILink)
	role := promptSlug

	resolved, err := providers.Resolve(role, promptDef, modelOverride)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to resolve provider", Details: err.Error()}, nil
	}
	if strings.TrimSpace(resolved.Credential.APIKey) == "" {
		return nil, &ailink.SearchError{Code: "AILINK_NO_API_KEY", Message: "provider api key not configured", Details: resolved.ProviderID}, nil
	}

	cacheTTL := cfg.AILink.CacheTTL
	cacheSlug := analysisCacheKey(promptSlug, cleaned)
	if useCache && store != nil && cacheTTL > 0 {
		entry, err := store.GetExpertCache(ctx, name, cacheSlug, resolved.Model, resolved.BaseURL, depth)
		if err != nil {
			observability.CLILogger.Warn("Expert cache lookup failed", zap.Error(err))
		} else if entry != nil {
			raw := json.RawMessage(entry.ResponseJSON)
			return raw, nil, raw
		}
	}

	catalog, err := buildSchemaCatalog()
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to load schemas", Details: err.Error()}, nil
	}

	svc := &ailink.Service{Providers: providers, Registry: registry, Catalog: catalog}
	response, err := svc.Generate(ctx, ailink.GenerateRequest{Role: role, PromptSlug: promptSlug, Variables: cleaned, Depth: depth, Model: modelOverride, UseTools: true})
	if err != nil {
		return nil, mapExpertError(err), rawFromAILinkError(err)
	}

	raw := response.Raw
	if useCache && store != nil && cacheTTL > 0 {
		encoded := strings.TrimSpace(string(raw))
		if encoded != "" {
			if err := store.SetExpertCache(ctx, name, cacheSlug, resolved.Model, resolved.BaseURL, depth, encoded, cacheTTL); err != nil {
				observability.CLILogger.Warn("Expert cache write failed", zap.Error(err))
			}
		}
	}

	return raw, nil, raw
}

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
	Use:   "review [<name>...]",
	Short: "Run a stitched name review workflow",
	Long:  "Review runs availability checks plus a mode-selected set of AILink analysis prompts.",
	Args:  cobra.ArbitraryArgs,
	RunE:  runReview,
}

func init() {
	rootCmd.AddCommand(reviewCmd)

	reviewCmd.Flags().String("profile", "startup", "Availability profile to use")
	reviewCmd.Flags().String("mode", "core", "Review mode: core, brand, full")
	reviewCmd.Flags().String("depth", "quick", "Analysis depth: quick, deep")
	reviewCmd.Flags().String("names-file", "", "Read names from file (one per line) or '-' for stdin")
	reviewCmd.Flags().String("output-format", "table", "Output format: table, json, markdown")
	reviewCmd.Flags().String("out", "", "Write output to a file (default stdout)")
	reviewCmd.Flags().String("out-dir", "", "Write per-name outputs to a directory")
	reviewCmd.Flags().String("include-raw", string(includeRawOnFail), "Include raw analysis output: never, on-failure, always")
	reviewCmd.Flags().Bool("strict", false, "Return non-zero if any analysis fails")
	reviewCmd.Flags().Bool("no-cache", false, "Skip cache lookup")
}

func runReview(cmd *cobra.Command, args []string) error {
	namesFile, err := cmd.Flags().GetString("names-file")
	if err != nil {
		return err
	}
	names, err := resolveNames(args, namesFile)
	if err != nil {
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

	format, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}
	outPath, outDir, err := resolveOutputTargets(cmd)
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

	registry, err := buildPromptRegistry(cfg)
	if err != nil {
		return err
	}

	promptSlugs, err := reviewPromptSet(mode, registry)
	if err != nil {
		return err
	}

	type reviewItem struct {
		result   *reviewResult
		batch    *core.BatchResult
		failed   int
		analyses map[string]reviewAnalysis
	}

	items := make([]reviewItem, 0, len(names))
	failedTotal := 0

	for _, name := range names {
		results, err := orchestrator.Check(ctx, name, profile)
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
				var raw json.RawMessage
				expertResult, expertError, raw = runReviewSearch(ctx, cfg, store, name, depth, "", slug, !noCache)

				a := reviewAnalysis{OK: expertError == nil}
				if expertError != nil {
					a.Error = expertError
				}
				if expertResult != nil {
					payload, _ := json.Marshal(expertResult)
					a.Data = json.RawMessage(payload)
				}
				if len(raw) > 0 {
					if rawMode == includeRawAlways || (rawMode == includeRawOnFail && expertError != nil) {
						a.Raw = raw
					}
				}
				analyses[slug] = a
			case "name-phonetics":
				vars := map[string]string{"name": name}
				phoneticsResult, phoneticsError, raw := runReviewGenerate(ctx, cfg, store, slug, name, depth, "", vars, !noCache)
				analyses[slug] = analysisFromGenerate(phoneticsResult, phoneticsError, raw, rawMode)
			case "name-suitability":
				vars := map[string]string{"name": name}
				suitabilityRaw, suitabilityErr, raw := runReviewGenerate(ctx, cfg, store, slug, name, depth, "", vars, !noCache)
				analyses[slug] = analysisFromGenerate(suitabilityRaw, suitabilityErr, raw, rawMode)
			default:
				vars := map[string]string{"name": name}
				data, errInfo, raw := runReviewGenerate(ctx, cfg, store, slug, name, depth, "", vars, !noCache)
				analyses[slug] = analysisFromGenerate(data, errInfo, raw, rawMode)
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
		failedTotal += failed
		items = append(items, reviewItem{result: review, batch: batch, failed: failed, analyses: analyses})
	}

	ext := outputExtension(format)

	renderOne := func(w io.Writer, item reviewItem) error {
		if w == nil || item.result == nil {
			return nil
		}

		switch format {
		case output.FormatJSON:
			payload, err := json.MarshalIndent(item.result, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(w, string(payload))
			return err
		case output.FormatMarkdown:
			rendered, err := output.NewFormatter(output.FormatMarkdown).FormatBatch(item.batch)
			if err != nil {
				return err
			}
			if len(names) > 1 {
				_, _ = fmt.Fprintf(w, "\n## %s\n\n", item.result.Name)
			}
			if rendered != "" {
				if _, err := fmt.Fprintln(w, rendered); err != nil {
					return err
				}
			}
			renderReviewExtrasMarkdown(w, item.analyses, []string{"name-availability", "name-phonetics", "name-suitability"})
			return nil
		default:
			if len(names) > 1 {
				_, _ = fmt.Fprint(w, ascii.DrawBox(item.result.Name, 0))
				_, _ = fmt.Fprintln(w)
			}
			rendered, err := output.NewFormatter(output.FormatTable).FormatBatch(item.batch)
			if err != nil {
				return err
			}
			if strings.TrimSpace(rendered) != "" {
				if _, err := fmt.Fprintln(w, rendered); err != nil {
					return err
				}
			}
			renderReviewExtrasTable(w, item.analyses, []string{"name-availability", "name-phonetics", "name-suitability"})
			return nil
		}
	}

	renderAll := func(w io.Writer) error {
		if format == output.FormatJSON {
			if len(items) == 1 {
				return renderOne(w, items[0])
			}
			results := make([]*reviewResult, 0, len(items))
			for _, item := range items {
				results = append(results, item.result)
			}
			payload, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(w, string(payload))
			return err
		}

		first := true
		for _, item := range items {
			if !first {
				_, _ = fmt.Fprintln(w)
			}
			first = false
			if err := renderOne(w, item); err != nil {
				return err
			}
		}
		return nil
	}

	if outDir != "" {
		outDir, err := ensureOutDir(outDir)
		if err != nil {
			return err
		}

		indexPath := filepath.Join(outDir, fmt.Sprintf("review.index.%s", ext))
		indexSink, err := openSink(indexPath)
		if err != nil {
			return err
		}
		if err := renderAll(indexSink.writer); err != nil {
			_ = indexSink.close()
			return err
		}
		if err := indexSink.close(); err != nil {
			return err
		}

		for _, item := range items {
			fileName := sanitizeFilename(item.result.Name)
			path := filepath.Join(outDir, fmt.Sprintf("%s.review.%s", fileName, ext))
			sink, err := openSink(path)
			if err != nil {
				return err
			}
			if err := renderOne(sink.writer, item); err != nil {
				_ = sink.close()
				return err
			}
			if err := sink.close(); err != nil {
				return err
			}
		}
	} else {
		sink, err := openSink(outPath)
		if err != nil {
			return err
		}
		if err := renderAll(sink.writer); err != nil {
			_ = sink.close()
			return err
		}
		if err := sink.close(); err != nil {
			return err
		}
	}

	if strict && failedTotal > 0 {
		return fmt.Errorf("review failed (%d analyses)", failedTotal)
	}
	return nil
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
		set := append(core, "brand-proposal")
		if _, err := registry.Get("brand-plan"); err == nil {
			set = append(set, "brand-plan")
		}
		return set, nil
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

func analysisFromGenerate(data json.RawMessage, errInfo *ailink.SearchError, raw json.RawMessage, rawMode includeRawMode) reviewAnalysis {
	a := reviewAnalysis{OK: errInfo == nil}
	if errInfo != nil {
		a.Error = errInfo
		if len(raw) > 0 {
			if rawMode == includeRawAlways || rawMode == includeRawOnFail {
				a.Raw = raw
			}
		}
		return a
	}
	a.Data = data
	if len(raw) > 0 && rawMode == includeRawAlways {
		a.Raw = raw
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
