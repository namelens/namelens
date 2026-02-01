package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/namelens/namelens/internal/ailink"
	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/core"
	"github.com/namelens/namelens/internal/core/checker"
	"github.com/namelens/namelens/internal/core/engine"
	"github.com/namelens/namelens/internal/core/store"
	"github.com/namelens/namelens/internal/observability"
	"github.com/namelens/namelens/internal/output"
)

var checkCmd = &cobra.Command{
	Use:   "check <name> [<name>...]",
	Short: "Check name availability",
	Long:  "Check if a name is available across domains, registries, and handles",
	Args:  cobra.ArbitraryArgs,
	RunE:  runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringSlice("tlds", []string{"com"}, "TLDs to check")
	checkCmd.Flags().StringSlice("registries", nil, "Registries to check (npm, pypi, cargo)")
	checkCmd.Flags().StringSlice("handles", nil, "Handles to check (github)")
	checkCmd.Flags().String("profile", "", "Use predefined profile")
	checkCmd.Flags().String("names-file", "", "Read names from file (one per line) or '-' for stdin")
	checkCmd.Flags().String("output-format", "table", "Output format: table, json, markdown")
	checkCmd.Flags().String("out", "", "Write output to a file (default stdout)")
	checkCmd.Flags().String("out-dir", "", "Write per-name outputs to a directory")
	checkCmd.Flags().Bool("no-cache", false, "Skip cache lookup")
	checkCmd.Flags().Bool("expert", false, "Include expert search backend")
	checkCmd.Flags().String("expert-depth", "quick", "Expert search depth: quick, deep")
	checkCmd.Flags().String("expert-model", "", "Expert model override")
	checkCmd.Flags().String("expert-prompt", "", "Expert prompt slug (defaults to config)")
	checkCmd.Flags().Bool("phonetics", false, "Analyze pronunciation and typeability")
	checkCmd.Flags().Bool("suitability", false, "Analyze cultural appropriateness")
	checkCmd.Flags().StringSlice("locales", nil, "Locales to analyze (comma-separated)")
	checkCmd.Flags().StringSlice("keyboards", nil, "Keyboard layouts for typeability analysis")
	checkCmd.Flags().String("sensitivity", "", "Suitability sensitivity: minimal, standard, strict")
}

func runCheck(cmd *cobra.Command, args []string) error {
	namesFile, err := cmd.Flags().GetString("names-file")
	if err != nil {
		return err
	}
	names, err := resolveNames(args, namesFile)
	if err != nil {
		return err
	}

	tlds, err := cmd.Flags().GetStringSlice("tlds")
	if err != nil {
		return err
	}

	registries, err := cmd.Flags().GetStringSlice("registries")
	if err != nil {
		return err
	}

	handles, err := cmd.Flags().GetStringSlice("handles")
	if err != nil {
		return err
	}

	profileName, err := cmd.Flags().GetString("profile")
	if err != nil {
		return err
	}

	noCache, err := cmd.Flags().GetBool("no-cache")
	if err != nil {
		return err
	}
	expertEnabled, err := cmd.Flags().GetBool("expert")
	if err != nil {
		return err
	}
	expertDepth, err := cmd.Flags().GetString("expert-depth")
	if err != nil {
		return err
	}
	expertModel, err := cmd.Flags().GetString("expert-model")
	if err != nil {
		return err
	}
	expertPrompt, err := cmd.Flags().GetString("expert-prompt")
	if err != nil {
		return err
	}
	phoneticsEnabled, err := cmd.Flags().GetBool("phonetics")
	if err != nil {
		return err
	}
	suitabilityEnabled, err := cmd.Flags().GetBool("suitability")
	if err != nil {
		return err
	}
	localesRaw, err := cmd.Flags().GetStringSlice("locales")
	if err != nil {
		return err
	}
	keyboardsRaw, err := cmd.Flags().GetStringSlice("keyboards")
	if err != nil {
		return err
	}
	sensitivity, err := cmd.Flags().GetString("sensitivity")
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	startedAt := time.Now()
	store, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close() // nolint:errcheck // best-effort cleanup; errors logged internally

	cfg := config.GetConfig()
	if cfg == nil {
		return errors.New("config not loaded")
	}

	// Show guidance about AI backend if not configured
	showExpertGuidanceWarning(cfg.AILink, nil)

	profile, err := resolveProfile(ctx, store, profileName, tlds, registries, handles)
	if err != nil {
		return err
	}
	if len(profile.TLDs) == 0 && len(profile.Registries) == 0 && len(profile.Handles) == 0 {
		return errors.New("at least one check target is required")
	}

	orchestrator := buildOrchestrator(cfg, store, !noCache)

	locales := normalizeInputList(localesRaw)
	keyboards := normalizeInputList(keyboardsRaw)

	batches := make([]*core.BatchResult, 0, len(names))
	for _, name := range names {
		results, err := orchestrator.Check(ctx, name, profile)
		if err != nil {
			return err
		}

		var (
			expertResult    *ailink.SearchResponse
			expertError     *ailink.SearchError
			phoneticsResult json.RawMessage
			phoneticsError  *ailink.SearchError
			suitabilityRaw  json.RawMessage
			suitabilityErr  *ailink.SearchError
		)
		if expertEnabled || cfg.Expert.Enabled {
			expertResult, expertError = runExpert(ctx, cfg, store, name, expertDepth, expertModel, expertPrompt, !noCache)
		}
		if phoneticsEnabled {
			vars := map[string]string{"name": name}
			if len(locales) > 0 {
				vars["locales"] = strings.Join(locales, ", ")
			}
			if len(keyboards) > 0 {
				vars["keyboards"] = strings.Join(keyboards, ", ")
			}
			phoneticsResult, phoneticsError = runAnalysis(ctx, cfg, store, "name-phonetics", name, expertDepth, expertModel, vars, !noCache)
		}
		if suitabilityEnabled {
			vars := map[string]string{"name": name}
			if len(locales) > 0 {
				vars["locales"] = strings.Join(locales, ", ")
			}
			if trimmed := strings.TrimSpace(sensitivity); trimmed != "" {
				vars["sensitivity_level"] = trimmed
			}
			suitabilityRaw, suitabilityErr = runAnalysis(ctx, cfg, store, "name-suitability", name, expertDepth, expertModel, vars, !noCache)
		}

		batch := summarizeResults(name, results, expertResult, expertError, phoneticsResult, phoneticsError, suitabilityRaw, suitabilityErr)
		batches = append(batches, batch)
	}

	format, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}
	outPath, outDir, err := resolveOutputTargets(cmd)
	if err != nil {
		return err
	}

	var rendered string
	if len(batches) == 1 {
		rendered, err = output.NewFormatter(format).FormatBatch(batches[0])
	} else {
		rendered, err = output.FormatBatchList(format, batches)
	}
	if err != nil {
		return err
	}

	ext := outputExtension(format)
	if outDir != "" {
		outDir, err := ensureOutDir(outDir)
		if err != nil {
			return err
		}

		indexRendered := rendered
		if len(batches) == 1 {
			indexRendered, err = output.FormatBatchList(format, batches)
			if err != nil {
				return err
			}
		}

		indexPath := filepath.Join(outDir, fmt.Sprintf("check.index.%s", ext))
		indexSink, err := openSink(indexPath)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprint(indexSink.writer, indexRendered); err != nil {
			_ = indexSink.close()
			return err
		}
		if err := indexSink.close(); err != nil {
			return err
		}

		formatter := output.NewFormatter(format)
		for _, batch := range batches {
			if batch == nil {
				continue
			}
			fileName := sanitizeFilename(batch.Name)
			path := filepath.Join(outDir, fmt.Sprintf("%s.check.%s", fileName, ext))
			sink, err := openSink(path)
			if err != nil {
				return err
			}

			var content string
			if format == output.FormatJSON {
				payload, err := json.MarshalIndent(batch, "", "  ")
				if err != nil {
					_ = sink.close()
					return err
				}
				content = string(payload)
			} else {
				content, err = formatter.FormatBatch(batch)
				if err != nil {
					_ = sink.close()
					return err
				}
			}

			if strings.TrimSpace(content) != "" {
				if _, err := fmt.Fprint(sink.writer, content); err != nil {
					_ = sink.close()
					return err
				}
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
		if strings.TrimSpace(rendered) != "" {
			if _, err := fmt.Fprint(sink.writer, rendered); err != nil {
				_ = sink.close()
				return err
			}
		}
		if err := sink.close(); err != nil {
			return err
		}
		if format != output.FormatJSON && (outPath == "" || outPath == "-") {
			totalCount := 0
			for _, batch := range batches {
				if batch == nil {
					continue
				}
				totalCount += batch.Total
			}
			logThroughput(totalCount, startedAt)

			// Show tip about --expert if AI is configured but not used
			showExpertTip(cfg.AILink, expertEnabled || cfg.Expert.Enabled, nil)
		}
	}

	return nil
}

func validateName(name string) error {
	if len(name) < 1 || len(name) > 63 {
		return errors.New("name must be 1-63 characters")
	}

	matched, err := regexp.MatchString(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, name)
	if err != nil {
		return fmt.Errorf("name validation failed: %w", err)
	}
	if !matched {
		return errors.New("name must be lowercase alphanumeric with optional hyphens")
	}

	return nil
}

func normalizeTLDs(values []string) []string {
	seen := make(map[string]struct{})
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			tld := strings.ToLower(strings.TrimSpace(part))
			tld = strings.TrimPrefix(tld, ".")
			if tld == "" {
				continue
			}
			seen[tld] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for tld := range seen {
		result = append(result, tld)
	}
	if len(result) == 0 {
		return nil
	}

	sort.Strings(result)
	return result
}

func resolveGitHubToken() string {
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("NAMELENS_GITHUB_TOKEN"))
}

func logThroughput(count int, startedAt time.Time) {
	if count <= 0 {
		return
	}
	elapsed := time.Since(startedAt)
	if elapsed <= 0 {
		return
	}
	rate := float64(count) / elapsed.Seconds()
	observability.CLILogger.Info(
		"Check throughput",
		zap.Int("checks", count),
		zap.Duration("elapsed", elapsed),
		zap.Float64("rate_per_sec", rate),
	)
}

func buildOrchestrator(cfg *config.Config, store *store.Store, useCache bool) *engine.Orchestrator {
	limiter := &engine.RateLimiter{Store: store}
	limiter.ApplyOverrides(cfg.RateLimits)
	limiter.ApplySafetyMargin(cfg.RateLimitMargin)

	cachePolicy := checker.CachePolicy{
		AvailableTTL: cfg.Cache.AvailableTTL,
		TakenTTL:     cfg.Cache.TakenTTL,
		ErrorTTL:     cfg.Cache.ErrorTTL,
	}

	domainChecker := &checker.DomainChecker{
		Store:       store,
		ToolVersion: versionInfo.Version,
		Limiter:     limiter,
		CachePolicy: cachePolicy,
		UseCache:    useCache,
		WhoisCfg: checker.WhoisFallbackConfig{
			Enabled:           cfg.Domain.WhoisFallback.Enabled,
			TLDs:              cfg.Domain.WhoisFallback.TLDs,
			RequireExplicit:   cfg.Domain.WhoisFallback.RequireExplicit,
			CacheTTL:          cfg.Domain.WhoisFallback.CacheTTL,
			Timeout:           cfg.Domain.WhoisFallback.Timeout,
			Servers:           cfg.Domain.WhoisFallback.Servers,
			AvailablePatterns: cfg.Domain.WhoisFallback.AvailablePatterns,
			TakenPatterns:     cfg.Domain.WhoisFallback.TakenPatterns,
		},
		DNSCfg: checker.DNSFallbackConfig{
			Enabled:  cfg.Domain.DNSFallback.Enabled,
			CacheTTL: cfg.Domain.DNSFallback.CacheTTL,
			Timeout:  cfg.Domain.DNSFallback.Timeout,
		},
	}
	npmChecker := &checker.NPMChecker{
		Store:       store,
		ToolVersion: versionInfo.Version,
		Limiter:     limiter,
		CachePolicy: cachePolicy,
		UseCache:    useCache,
	}
	pypiChecker := &checker.PyPIChecker{
		Store:       store,
		ToolVersion: versionInfo.Version,
		Limiter:     limiter,
		CachePolicy: cachePolicy,
		UseCache:    useCache,
	}
	cargoChecker := &checker.CargoChecker{
		Store:       store,
		ToolVersion: versionInfo.Version,
		Limiter:     limiter,
		CachePolicy: cachePolicy,
		UseCache:    useCache,
	}
	githubChecker := &checker.GitHubChecker{
		Store:       store,
		ToolVersion: versionInfo.Version,
		Limiter:     limiter,
		Token:       resolveGitHubToken(),
		CachePolicy: cachePolicy,
		UseCache:    useCache,
	}

	return &engine.Orchestrator{
		Checkers: map[core.CheckType]engine.Checker{
			core.CheckTypeDomain: domainChecker,
		},
		RegistryCheckers: map[string]engine.Checker{
			"npm":   npmChecker,
			"pypi":  pypiChecker,
			"cargo": cargoChecker,
		},
		HandleCheckers: map[string]engine.Checker{
			"github": githubChecker,
		},
	}
}

func summarizeResults(name string, results []*core.CheckResult, expert *ailink.SearchResponse, expertErr *ailink.SearchError, phonetics json.RawMessage, phoneticsErr *ailink.SearchError, suitability json.RawMessage, suitabilityErr *ailink.SearchError) *core.BatchResult {
	total := 0
	score := 0
	unknown := 0
	for _, result := range results {
		if result == nil {
			continue
		}
		// Count unknown/unsupported separately - they shouldn't affect the score denominator
		if result.Available == core.AvailabilityUnknown || result.Available == core.AvailabilityUnsupported {
			unknown++
			continue
		}
		total++
		if result.Available == core.AvailabilityAvailable {
			score++
		}
	}

	return &core.BatchResult{
		Name:             name,
		Results:          results,
		Score:            score,
		Total:            total,
		Unknown:          unknown,
		CompletedAt:      time.Now().UTC(),
		AILink:           expert,
		AILinkError:      expertErr,
		Phonetics:        phonetics,
		PhoneticsError:   phoneticsErr,
		Suitability:      suitability,
		SuitabilityError: suitabilityErr,
	}
}

func runExpert(ctx context.Context, cfg *config.Config, store *store.Store, name, depth, modelOverride, promptOverride string, useCache bool) (*ailink.SearchResponse, *ailink.SearchError) {
	if cfg == nil {
		return nil, &ailink.SearchError{Code: "AILINK_DISABLED", Message: "config not loaded"}
	}

	promptSlug := strings.TrimSpace(promptOverride)
	if promptSlug == "" {
		promptSlug = strings.TrimSpace(cfg.Expert.DefaultPrompt)
	}
	if promptSlug == "" {
		promptSlug = "name-availability"
	}

	depth = strings.ToLower(strings.TrimSpace(depth))
	if depth == "" {
		depth = "quick"
	}

	registry, err := buildPromptRegistry(cfg)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to load prompts", Details: err.Error()}
	}
	promptDef, err := registry.Get(promptSlug)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_PROMPT_NOT_FOUND", Message: err.Error()}
	}

	providers := ailink.NewRegistry(cfg.AILink)
	role := strings.TrimSpace(cfg.Expert.Role)
	if role == "" {
		role = promptSlug
	}

	resolved, err := providers.Resolve(role, promptDef, modelOverride)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to resolve provider", Details: err.Error()}
	}
	if strings.TrimSpace(resolved.Credential.APIKey) == "" {
		return nil, &ailink.SearchError{Code: "AILINK_NO_API_KEY", Message: "provider api key not configured", Details: resolved.ProviderID}
	}

	cacheTTL := cfg.AILink.CacheTTL
	if useCache && store != nil && cacheTTL > 0 {
		entry, err := store.GetExpertCache(ctx, name, promptSlug, resolved.Model, resolved.BaseURL, depth)
		if err != nil {
			observability.CLILogger.Warn("Expert cache lookup failed", zap.Error(err))
		} else if entry != nil {
			response, err := decodeCachedExpert(entry.ResponseJSON)
			if err == nil {
				return response, nil
			}
			observability.CLILogger.Warn("Expert cache decode failed", zap.Error(err))
		}
	}

	catalog, err := buildSchemaCatalog()
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to load schemas", Details: err.Error()}
	}

	service := &ailink.Service{
		Providers: providers,
		Registry:  registry,
		Catalog:   catalog,
	}

	response, err := service.Search(ctx, ailink.SearchRequest{
		Role:       role,
		Name:       name,
		PromptSlug: promptSlug,
		Depth:      depth,
		Model:      modelOverride,
		UseTools:   true,
	})
	if err != nil {
		return nil, mapExpertError(err)
	}

	if useCache && store != nil && cacheTTL > 0 {
		raw := strings.TrimSpace(string(response.Raw))
		if raw == "" {
			payload, err := json.Marshal(response)
			if err == nil {
				raw = string(payload)
			}
		}
		if raw != "" {
			if err := store.SetExpertCache(ctx, name, promptSlug, resolved.Model, resolved.BaseURL, depth, raw, cacheTTL); err != nil {
				observability.CLILogger.Warn("Expert cache write failed", zap.Error(err))
			}
		}
	}

	return response, nil
}

func runAnalysis(ctx context.Context, cfg *config.Config, store *store.Store, promptSlug, name, depth, modelOverride string, variables map[string]string, useCache bool) (json.RawMessage, *ailink.SearchError) {
	if cfg == nil {
		return nil, &ailink.SearchError{Code: "AILINK_DISABLED", Message: "config not loaded"}
	}

	promptSlug = strings.TrimSpace(promptSlug)
	if promptSlug == "" {
		return nil, &ailink.SearchError{Code: "AILINK_PROMPT_NOT_FOUND", Message: "prompt slug is required"}
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
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to load prompts", Details: err.Error()}
	}
	promptDef, err := registry.Get(promptSlug)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_PROMPT_NOT_FOUND", Message: err.Error()}
	}

	providers := ailink.NewRegistry(cfg.AILink)
	role := promptSlug

	resolved, err := providers.Resolve(role, promptDef, modelOverride)
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to resolve provider", Details: err.Error()}
	}
	if strings.TrimSpace(resolved.Credential.APIKey) == "" {
		return nil, &ailink.SearchError{Code: "AILINK_NO_API_KEY", Message: "provider api key not configured", Details: resolved.ProviderID}
	}

	cacheTTL := cfg.AILink.CacheTTL
	cacheSlug := analysisCacheKey(promptSlug, cleaned)
	if useCache && store != nil && cacheTTL > 0 {
		entry, err := store.GetExpertCache(ctx, name, cacheSlug, resolved.Model, resolved.BaseURL, depth)
		if err != nil {
			observability.CLILogger.Warn("Expert cache lookup failed", zap.Error(err))
		} else if entry != nil {
			return json.RawMessage(entry.ResponseJSON), nil
		}
	}

	catalog, err := buildSchemaCatalog()
	if err != nil {
		return nil, &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "failed to load schemas", Details: err.Error()}
	}

	service := &ailink.Service{
		Providers: providers,
		Registry:  registry,
		Catalog:   catalog,
	}

	response, err := service.Generate(ctx, ailink.GenerateRequest{
		Role:       role,
		PromptSlug: promptSlug,
		Variables:  cleaned,
		Depth:      depth,
		Model:      modelOverride,
		UseTools:   true,
	})
	if err != nil {
		return nil, mapExpertError(err)
	}

	if useCache && store != nil && cacheTTL > 0 {
		raw := strings.TrimSpace(string(response.Raw))
		if raw != "" {
			if err := store.SetExpertCache(ctx, name, cacheSlug, resolved.Model, resolved.BaseURL, depth, raw, cacheTTL); err != nil {
				observability.CLILogger.Warn("Expert cache write failed", zap.Error(err))
			}
		}
	}

	return response.Raw, nil
}

func analysisCacheKey(promptSlug string, variables map[string]string) string {
	if promptSlug == "" || len(variables) == 0 {
		return promptSlug
	}

	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, key := range keys {
		sb.WriteString(key)
		sb.WriteByte('=')
		sb.WriteString(variables[key])
		sb.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return fmt.Sprintf("%s:%x", promptSlug, sum[:8])
}

func mapExpertError(err error) *ailink.SearchError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &ailink.SearchError{Code: "AILINK_TIMEOUT", Message: "expert request timed out"}
	}

	if mapped := ailink.MapProviderError(err); mapped != nil {
		return mapped
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "schema validation failed"):
		return &ailink.SearchError{Code: "AILINK_VALIDATION_ERROR", Message: "expert response failed schema validation (try again with --trace <file> to capture the raw payload)", Details: message}
	case strings.Contains(message, "prompt") && strings.Contains(message, "not found"):
		return &ailink.SearchError{Code: "AILINK_PROMPT_NOT_FOUND", Message: message}
	default:
		return &ailink.SearchError{Code: "AILINK_API_ERROR", Message: "expert request failed", Details: message}
	}
}

func decodeCachedExpert(raw string) (*ailink.SearchResponse, error) {
	var parsed ailink.SearchResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, err
	}
	parsed.Raw = append(parsed.Raw[:0], raw...)
	return &parsed, nil
}

func resolveProfile(ctx context.Context, store interface {
	GetProfile(context.Context, string) (*core.ProfileRecord, error)
}, profileName string, tlds, registries, handles []string) (core.Profile, error) {
	name := strings.TrimSpace(profileName)
	if name == "" {
		return core.Profile{
			Name:       "custom",
			TLDs:       normalizeTLDs(tlds),
			Registries: normalizeList(registries),
			Handles:    normalizeList(handles),
		}, nil
	}

	record, err := store.GetProfile(ctx, name)
	if err != nil {
		return core.Profile{}, err
	}
	if record != nil {
		record.Profile.TLDs = normalizeTLDs(record.Profile.TLDs)
		record.Profile.Registries = normalizeList(record.Profile.Registries)
		record.Profile.Handles = normalizeList(record.Profile.Handles)
		return record.Profile, nil
	}

	if profile, ok := core.FindBuiltInProfile(name); ok {
		profile.TLDs = normalizeTLDs(profile.TLDs)
		profile.Registries = normalizeList(profile.Registries)
		profile.Handles = normalizeList(profile.Handles)
		return *profile, nil
	}

	return core.Profile{}, fmt.Errorf("profile %q not found", name)
}

func normalizeList(values []string) []string {
	seen := make(map[string]struct{})
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			item := strings.ToLower(strings.TrimSpace(part))
			if item == "" {
				continue
			}
			seen[item] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for item := range seen {
		result = append(result, item)
	}
	if len(result) == 0 {
		return nil
	}

	sort.Strings(result)
	return result
}

func normalizeInputList(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			item := strings.TrimSpace(part)
			if item == "" {
				continue
			}
			key := strings.ToLower(item)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
