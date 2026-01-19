// Package config provides centralized configuration management for NameLens.
// It implements the three-layer config pattern using gofulmen/config:
// Layer 1: Crucible defaults (config/namelens/v0/namelens-defaults.yaml)
// Layer 2: User overrides (discovered via app identity)
// Layer 3: Environment variables and runtime overrides
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/fulmenhq/gofulmen/appidentity"
	gfconfig "github.com/fulmenhq/gofulmen/config"

	"github.com/fulmenhq/gofulmen/pathfinder"
	"github.com/fulmenhq/gofulmen/schema"
	"github.com/go-viper/mapstructure/v2"
	"github.com/namelens/namelens/internal/appid"
)

var (
	// appConfig holds the current application configuration
	appConfig   *Config
	configMu    sync.RWMutex
	appIdentity *appidentity.Identity
)

// findProjectRoot walks up from the current working directory to find the project root.
// It looks for project markers like go.mod or .git directory.
// This ensures config paths work correctly regardless of where the process is run from.
//
// This now uses gofulmen/pathfinder.FindRepositoryRoot() which provides:
// - Security boundaries (home directory ceiling, max depth protection)
// - Symlink loop detection
// - Cross-platform compatibility
// - Performance optimized (<30µs)
func findProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	markers := []string{"go.mod", ".git"}

	// CI-only boundary hint pattern:
	// - Treat CI workspace env vars as a boundary hint, not "the root".
	// - Still require repository markers.
	isCI := strings.EqualFold(strings.TrimSpace(os.Getenv("GITHUB_ACTIONS")), "true") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("CI")), "true")
	if isCI {
		boundaryKeys := []string{"FULMEN_WORKSPACE_ROOT", "GITHUB_WORKSPACE", "CI_PROJECT_DIR", "WORKSPACE"}
		for _, key := range boundaryKeys {
			boundary := strings.TrimSpace(os.Getenv(key))
			if boundary == "" {
				continue
			}
			boundary = filepath.Clean(boundary)
			if !filepath.IsAbs(boundary) {
				continue
			}
			st, err := os.Stat(boundary)
			if err != nil || !st.IsDir() {
				continue
			}
			// Only accept a boundary that contains the start path.
			if rel, err := filepath.Rel(boundary, cwd); err != nil || strings.HasPrefix(rel, "..") {
				continue
			}
			rootPath, err := pathfinder.FindRepositoryRoot(cwd, markers,
				pathfinder.WithBoundary(boundary),
				pathfinder.WithMaxDepth(20),
			)
			if err == nil {
				return rootPath, nil
			}
		}
	}

	rootPath, err := pathfinder.FindRepositoryRoot(cwd, markers, pathfinder.WithMaxDepth(10))
	if err != nil {
		return "", fmt.Errorf("project root not found: %w", err)
	}

	return rootPath, nil
}

// EnvVarSpec defines environment variable mappings for config fields
// following the pattern: {PREFIX}{NAME} maps to config path
type EnvVarSpec = gfconfig.EnvVarSpec

// Environment variable types
const (
	EnvString = gfconfig.EnvString
	EnvInt    = gfconfig.EnvInt
	EnvBool   = gfconfig.EnvBool
)

// Load loads configuration using the three-layer pattern:
// 1. Crucible defaults from config/namelens/v0/namelens-defaults.yaml
// 2. User overrides from XDG config paths
// 3. Environment variables and runtime overrides
//
// This function is safe to call multiple times (e.g., for config reload)
func Load(ctx context.Context, runtimeOverrides ...map[string]any) (*Config, error) {
	// Get app identity if not already loaded
	if appIdentity == nil {
		identity, err := appid.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load app identity: %w", err)
		}
		appIdentity = identity
	}

	// Find project root for absolute paths
	// This ensures config loading works from any working directory (including tests)
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	// Build layered config options
	// NameLens uses its own schema located in schemas/namelens/
	// Defaults are in config/namelens/v0/namelens-defaults.yaml
	// Using absolute paths ensures this works from any working directory
	catalog := schema.NewCatalog(filepath.Join(projectRoot, "schemas"))
	opts := gfconfig.LayeredConfigOptions{
		Category:     "namelens",
		Version:      "v0",
		DefaultsFile: "namelens-defaults.yaml",
		SchemaID:     "namelens/v0/config",
		UserPaths:    getUserConfigPaths(),
		Catalog:      catalog,
		DefaultsRoot: filepath.Join(projectRoot, "config"), // Absolute path for Layer 2 template
	}

	// Load environment variable overrides
	envOverrides, err := gfconfig.LoadEnvOverrides(getEnvSpecs())
	if err != nil {
		return nil, fmt.Errorf("failed to load environment overrides: %w", err)
	}

	if appIdentity != nil {
		prefix := appIdentity.EnvPrefix
		if !strings.HasSuffix(prefix, "_") {
			prefix += "_"
		}

		applyAILinkDynamicEnvOverrides(prefix, envOverrides)

		if value := strings.TrimSpace(os.Getenv(prefix + "RATE_LIMIT_MARGIN")); value != "" {
			margin, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid rate limit margin: %w", err)
			}
			envOverrides["rate_limit_margin"] = margin
		}
	}

	// Combine environment overrides with runtime overrides
	allOverrides := []map[string]any{envOverrides}
	allOverrides = append(allOverrides, runtimeOverrides...)

	// Load layered configuration
	merged, diagnostics, err := gfconfig.LoadLayeredConfig(opts, allOverrides...)
	if err != nil {
		return nil, fmt.Errorf("failed to load layered config: %w", err)
	}

	// Log validation diagnostics (warnings/errors from schema validation)
	// Note: We log these but don't fail hard to maintain flexibility
	for _, diag := range diagnostics {
		// TODO: Use logger when available
		fmt.Printf("Config validation: %s: %s\n", diag.Pointer, diag.Message)
	}

	// Unmarshal into typed config struct
	cfg := &Config{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           cfg,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.StringToFloat64HookFunc(),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(merged); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if strings.TrimSpace(cfg.Store.URL) == "" && strings.TrimSpace(cfg.Store.Path) == "" {
		cfg.Store.Path = defaultStorePath()
	}

	// Store the loaded config
	setConfig(cfg)

	return cfg, nil
}

// GetConfig returns the current application configuration (thread-safe)
func GetConfig() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return appConfig
}

// setConfig updates the current configuration (thread-safe)
func setConfig(cfg *Config) {
	configMu.Lock()
	defer configMu.Unlock()
	appConfig = cfg
}

// getUserConfigPaths returns the list of user config file paths to check
// Uses gofulmen/config for XDG-compliant path discovery
func getUserConfigPaths() []string {
	if appIdentity == nil {
		return []string{}
	}

	appName := appIdentity.ConfigName
	if strings.TrimSpace(appName) == "" {
		appName = appIdentity.BinaryName
	}
	if strings.TrimSpace(appName) == "" {
		appName = "namelens"
	}

	legacyNames := []string{}
	if appIdentity.BinaryName != "" && appIdentity.BinaryName != appName {
		legacyNames = append(legacyNames, appIdentity.BinaryName)
	}

	return gfconfig.GetAppConfigPaths(appName, legacyNames...)
}

// getEnvSpecs returns environment variable specifications for config mapping
// Maps {PREFIX}{NAME} environment variables to config paths
func getEnvSpecs() []EnvVarSpec {
	if appIdentity == nil {
		return []EnvVarSpec{}
	}

	prefix := appIdentity.EnvPrefix
	if !strings.HasSuffix(prefix, "_") {
		prefix += "_"
	}

	return []EnvVarSpec{
		// Server config
		{Name: prefix + "HOST", Path: []string{"server", "host"}, Type: EnvString},
		{Name: prefix + "PORT", Path: []string{"server", "port"}, Type: EnvInt},
		// Duration fields are parsed as strings and converted by mapstructure decode hook
		{Name: prefix + "READ_TIMEOUT", Path: []string{"server", "read_timeout"}, Type: EnvString},
		{Name: prefix + "WRITE_TIMEOUT", Path: []string{"server", "write_timeout"}, Type: EnvString},
		{Name: prefix + "IDLE_TIMEOUT", Path: []string{"server", "idle_timeout"}, Type: EnvString},
		{Name: prefix + "SHUTDOWN_TIMEOUT", Path: []string{"server", "shutdown_timeout"}, Type: EnvString},

		// Logging config (REQUIRED per Workhorse Standard)
		{Name: prefix + "LOG_LEVEL", Path: []string{"logging", "level"}, Type: EnvString},
		{Name: prefix + "LOG_PROFILE", Path: []string{"logging", "profile"}, Type: EnvString},

		// Store config
		{Name: prefix + "DB_DRIVER", Path: []string{"store", "driver"}, Type: EnvString},
		{Name: prefix + "DB_PATH", Path: []string{"store", "path"}, Type: EnvString},
		{Name: prefix + "DB_URL", Path: []string{"store", "url"}, Type: EnvString},
		{Name: prefix + "DB_AUTH_TOKEN", Path: []string{"store", "auth_token"}, Type: EnvString},

		// Domain fallback config
		{Name: prefix + "DOMAIN_WHOIS_FALLBACK_ENABLED", Path: []string{"domain", "whois_fallback", "enabled"}, Type: EnvBool},
		{Name: prefix + "DOMAIN_WHOIS_FALLBACK_TLDS", Path: []string{"domain", "whois_fallback", "tlds"}, Type: EnvString},
		{Name: prefix + "DOMAIN_WHOIS_FALLBACK_REQUIRE_EXPLICIT", Path: []string{"domain", "whois_fallback", "require_explicit"}, Type: EnvBool},
		{Name: prefix + "DOMAIN_WHOIS_FALLBACK_CACHE_TTL", Path: []string{"domain", "whois_fallback", "cache_ttl"}, Type: EnvString},
		{Name: prefix + "DOMAIN_WHOIS_FALLBACK_TIMEOUT", Path: []string{"domain", "whois_fallback", "timeout"}, Type: EnvString},

		{Name: prefix + "DOMAIN_DNS_FALLBACK_ENABLED", Path: []string{"domain", "dns_fallback", "enabled"}, Type: EnvBool},
		{Name: prefix + "DOMAIN_DNS_FALLBACK_CACHE_TTL", Path: []string{"domain", "dns_fallback", "cache_ttl"}, Type: EnvString},
		{Name: prefix + "DOMAIN_DNS_FALLBACK_TIMEOUT", Path: []string{"domain", "dns_fallback", "timeout"}, Type: EnvString},

		// AILink config
		{Name: prefix + "AILINK_DEFAULT_PROVIDER", Path: []string{"ailink", "default_provider"}, Type: EnvString},
		{Name: prefix + "AILINK_DEFAULT_TIMEOUT", Path: []string{"ailink", "default_timeout"}, Type: EnvString},
		{Name: prefix + "AILINK_CACHE_TTL", Path: []string{"ailink", "cache_ttl"}, Type: EnvString},
		{Name: prefix + "AILINK_PROMPTS_DIR", Path: []string{"ailink", "prompts_dir"}, Type: EnvString},
		{Name: prefix + "AILINK_DEBUG_CAPTURE_RAW_ENABLED", Path: []string{"ailink", "debug", "capture_raw_enabled"}, Type: EnvBool},
		{Name: prefix + "AILINK_DEBUG_CAPTURE_RAW_MAX_BYTES", Path: []string{"ailink", "debug", "capture_raw_max_bytes"}, Type: EnvInt},

		// Expert feature config
		{Name: prefix + "EXPERT_ENABLED", Path: []string{"expert", "enabled"}, Type: EnvBool},
		{Name: prefix + "EXPERT_ROLE", Path: []string{"expert", "role"}, Type: EnvString},
		{Name: prefix + "EXPERT_DEFAULT_PROMPT", Path: []string{"expert", "default_prompt"}, Type: EnvString},

		// Metrics config
		{Name: prefix + "METRICS_ENABLED", Path: []string{"metrics", "enabled"}, Type: EnvBool},
		{Name: prefix + "METRICS_PORT", Path: []string{"metrics", "port"}, Type: EnvInt},

		// Health config
		{Name: prefix + "HEALTH_ENABLED", Path: []string{"health", "enabled"}, Type: EnvBool},

		// Debug config
		{Name: prefix + "DEBUG_ENABLED", Path: []string{"debug", "enabled"}, Type: EnvBool},
		{Name: prefix + "DEBUG_PPROF_ENABLED", Path: []string{"debug", "pprof_enabled"}, Type: EnvBool},

		// Workers
		{Name: prefix + "WORKERS", Path: []string{"workers"}, Type: EnvInt},
	}
}

// appNamesForPaths returns the config name and binary name from app identity,
// falling back to "namelens" if not set.
func appNamesForPaths() (configName string, binaryName string) {
	configName = "namelens"
	binaryName = "namelens"
	if appIdentity == nil {
		return configName, binaryName
	}

	if strings.TrimSpace(appIdentity.ConfigName) != "" {
		configName = appIdentity.ConfigName
	}
	if strings.TrimSpace(appIdentity.BinaryName) != "" {
		binaryName = appIdentity.BinaryName
	}
	return configName, binaryName
}

// DefaultConfigPath returns the XDG-compliant path to the user config file.
func DefaultConfigPath() string {
	configName, _ := appNamesForPaths()
	configDir := gfconfig.GetAppConfigDir(configName)
	if strings.TrimSpace(configDir) == "" {
		return ""
	}
	return filepath.Join(configDir, "config.yaml")
}

// DefaultDataDir returns the XDG-compliant data directory for the app.
func DefaultDataDir() string {
	configName, _ := appNamesForPaths()
	return gfconfig.GetAppDataDir(configName)
}

// DefaultCacheDir returns the XDG-compliant cache directory for the app.
func DefaultCacheDir() string {
	configName, _ := appNamesForPaths()
	return gfconfig.GetAppCacheDir(configName)
}

// DefaultStorePath returns the XDG-compliant path to the database file.
func DefaultStorePath() string {
	configName, binaryName := appNamesForPaths()
	dataDir := gfconfig.GetAppDataDir(configName)
	if strings.TrimSpace(dataDir) == "" {
		return "./" + binaryName + ".db"
	}
	return filepath.Join(dataDir, binaryName+".db")
}

// defaultStorePath is an unexported alias for internal use.
func defaultStorePath() string {
	return DefaultStorePath()
}

func applyAILinkDynamicEnvOverrides(prefix string, envOverrides map[string]any) {
	providerPrefix := prefix + "AILINK_PROVIDERS_"
	routingPrefix := prefix + "AILINK_ROUTING_"

	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}

		switch {
		case strings.HasPrefix(key, providerPrefix):
			applyAILinkProviderOverride(envOverrides, key[len(providerPrefix):], value)
		case strings.HasPrefix(key, routingPrefix):
			applyAILinkRoutingOverride(envOverrides, key[len(routingPrefix):], value)
		}
	}
}

func applyAILinkRoutingOverride(envOverrides map[string]any, rawRole string, providerID string) {
	role := toSlug(rawRole)
	providerID = strings.TrimSpace(providerID)
	if role == "" || providerID == "" {
		return
	}

	ailink := ensureMap(envOverrides, "ailink")
	routing := ensureMap(ailink, "routing")
	routing[role] = providerID
}

func applyAILinkProviderOverride(envOverrides map[string]any, raw string, value string) {
	parts := strings.Split(strings.TrimSpace(raw), "_")
	if len(parts) < 2 {
		return
	}

	section := -1
	for i, part := range parts {
		switch part {
		case "ENABLED", "AI", "BASE", "MODELS", "CREDENTIALS":
			section = i
		}
		if section != -1 {
			break
		}
	}
	if section <= 0 {
		return
	}

	providerID := strings.ToLower(strings.Join(parts[:section], "-"))
	if providerID == "" {
		return
	}

	ailink := ensureMap(envOverrides, "ailink")
	providers := ensureMap(ailink, "providers")
	provider := ensureMap(providers, providerID)

	rest := parts[section:]
	switch {
	case len(rest) == 1 && rest[0] == "ENABLED":
		provider["enabled"] = strings.EqualFold(strings.TrimSpace(value), "true")
	case len(rest) == 1 && rest[0] == "SELECTION":
		// legacy guard: ignore
	case len(rest) == 2 && rest[0] == "AI" && rest[1] == "PROVIDER":
		provider["ai_provider"] = strings.ToLower(strings.TrimSpace(value))
	case len(rest) == 2 && rest[0] == "DEFAULT" && rest[1] == "CREDENTIAL":
		provider["default_credential"] = strings.TrimSpace(value)
	case len(rest) == 2 && rest[0] == "SELECTION" && rest[1] == "POLICY":
		provider["selection_policy"] = strings.ToLower(strings.TrimSpace(value))
	case len(rest) == 2 && rest[0] == "BASE" && rest[1] == "URL":
		provider["base_url"] = strings.TrimSpace(value)
	case len(rest) >= 2 && rest[0] == "MODELS":
		modelKey := strings.ToLower(strings.Join(rest[1:], "_"))
		models := ensureMap(provider, "models")
		models[modelKey] = strings.TrimSpace(value)
	case len(rest) >= 3 && rest[0] == "CREDENTIALS":
		idx, err := strconv.Atoi(rest[1])
		if err != nil || idx < 0 {
			return
		}
		field := strings.ToLower(strings.Join(rest[2:], "_"))
		if field == "" {
			return
		}

		creds := ensureSlice(provider, "credentials", idx+1)
		cred := ensureSliceMap(creds, idx)
		if field == "priority" {
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
				cred[field] = parsed
			} else {
				cred[field] = strings.TrimSpace(value)
			}
			return
		}
		if field == "enabled" {
			cred[field] = strings.EqualFold(strings.TrimSpace(value), "true")
			return
		}
		cred[field] = strings.TrimSpace(value)
	}
}

func ensureMap(parent map[string]any, key string) map[string]any {
	if parent == nil {
		return map[string]any{}
	}
	if existing, ok := parent[key]; ok {
		if typed, ok := existing.(map[string]any); ok {
			return typed
		}
	}
	next := map[string]any{}
	parent[key] = next
	return next
}

func ensureSlice(parent map[string]any, key string, length int) []any {
	var existing []any
	if raw, ok := parent[key]; ok {
		existing, _ = raw.([]any)
	}
	for len(existing) < length {
		existing = append(existing, map[string]any{})
	}
	parent[key] = existing
	return existing
}

func ensureSliceMap(slice []any, idx int) map[string]any {
	if idx < 0 || idx >= len(slice) {
		return map[string]any{}
	}
	if typed, ok := slice[idx].(map[string]any); ok {
		return typed
	}
	m := map[string]any{}
	slice[idx] = m
	return m
}

func toSlug(raw string) string {
	parts := strings.Split(strings.TrimSpace(raw), "_")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.ToLower(strings.TrimSpace(part))
		if p == "" {
			continue
		}
		clean = append(clean, p)
	}
	return strings.Join(clean, "-")
}
