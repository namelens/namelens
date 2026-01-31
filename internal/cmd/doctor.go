package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fulmenhq/gofulmen/foundry"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/fulmenhq/gofulmen/crucible"
	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/core/checker"
	errwrap "github.com/namelens/namelens/internal/errors"
	"github.com/namelens/namelens/internal/observability"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostic checks",
	Long:  "Run diagnostic checks on the system and suggest fixes for common issues.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		identity := GetAppIdentity()
		bannerName := "doctor"
		if identity != nil && identity.BinaryName != "" {
			bannerName = identity.BinaryName + " doctor"
		}
		observability.CLILogger.Info("=== " + bannerName + " ===")
		observability.CLILogger.Info("")
		observability.CLILogger.Info("Running diagnostic checks...")
		observability.CLILogger.Info("")

		allChecks := true
		totalChecks := 8

		// Check 1: Go version
		goVersion := runtime.Version()
		if goVersion >= "go1.23" {
			observability.CLILogger.Info(fmt.Sprintf("[1/%d] Checking Go version... ✅ %s", totalChecks, goVersion), zap.String("go_version", goVersion))
		} else {
			observability.CLILogger.Warn(fmt.Sprintf("[1/%d] Checking Go version... ⚠️  %s (recommended: go1.23+)", totalChecks, goVersion), zap.String("go_version", goVersion))
			allChecks = false
		}

		// Check 2: Crucible access
		version := crucible.GetVersion()
		if version.Crucible != "" {
			observability.CLILogger.Info(fmt.Sprintf("[2/%d] Checking Crucible access... ✅ v%s", totalChecks, version.Crucible), zap.String("crucible_version", version.Crucible))
		} else {
			observability.CLILogger.Error(fmt.Sprintf("[2/%d] Checking Crucible access... ❌ Cannot access Crucible", totalChecks))
			ExitWithCode(observability.CLILogger, foundry.ExitExternalServiceUnavailable, "Cannot access Crucible", errwrap.NewExternalServiceError("Crucible service unavailable"))
			allChecks = false
		}

		// Check 3: Gofulmen access
		if version.Gofulmen != "" {
			observability.CLILogger.Info(fmt.Sprintf("[3/%d] Checking Gofulmen access... ✅ v%s", totalChecks, version.Gofulmen), zap.String("gofulmen_version", version.Gofulmen))
		} else {
			observability.CLILogger.Error(fmt.Sprintf("[3/%d] Checking Gofulmen access... ❌ Cannot access Gofulmen", totalChecks))
			allChecks = false
		}

		// Check 4: Config directory
		configPath := config.DefaultConfigPath()
		if configPath == "" {
			observability.CLILogger.Error(fmt.Sprintf("[4/%d] Checking config directory... ❌ Cannot resolve config directory", totalChecks))
			ExitWithCode(observability.CLILogger, foundry.ExitFileNotFound, "Cannot resolve config directory", errwrap.NewInternalError("config directory not resolved"))
			allChecks = false
		} else {
			configDir := filepath.Dir(configPath)
			observability.CLILogger.Info(fmt.Sprintf("[4/%d] Checking config directory... ✅ %s", totalChecks, configDir), zap.String("config_dir", configDir))
		}

		// Check 5: Environment
		observability.CLILogger.Info(fmt.Sprintf("[5/%d] Checking environment... ✅ %s/%s", totalChecks, runtime.GOOS, runtime.GOARCH),
			zap.String("os", runtime.GOOS),
			zap.String("arch", runtime.GOARCH))

		// Check 6: Database
		cfg, cfgErr := config.Load(ctx)
		if cfgErr != nil {
			observability.CLILogger.Warn(fmt.Sprintf("[6/%d] Checking database... ⚠️  config not loaded", totalChecks), zap.Error(cfgErr))
			allChecks = false
		} else {
			if cfg.Store.URL != "" {
				observability.CLILogger.Info(fmt.Sprintf("[6/%d] Checking database... ✅ %s (remote)", totalChecks, cfg.Store.URL),
					zap.String("db_url", cfg.Store.URL))
				goto bootstrapCheck
			}

			dbPath := cfg.Store.Path
			if dbPath == "" {
				dbPath = config.DefaultStorePath()
			}
			// Resolve to absolute path for clarity
			absPath, _ := filepath.Abs(dbPath)
			if info, statErr := os.Stat(absPath); statErr == nil {
				sizeStr := formatFileSize(info.Size())
				observability.CLILogger.Info(fmt.Sprintf("[6/%d] Checking database... ✅ %s (%s)", totalChecks, absPath, sizeStr),
					zap.String("db_path", absPath),
					zap.Int64("db_size", info.Size()))
			} else if os.IsNotExist(statErr) {
				observability.CLILogger.Warn(fmt.Sprintf("[6/%d] Checking database... ⚠️  %s (not created yet)", totalChecks, absPath),
					zap.String("db_path", absPath))
			} else {
				observability.CLILogger.Warn(fmt.Sprintf("[6/%d] Checking database... ⚠️  %s (error: %v)", totalChecks, absPath, statErr),
					zap.String("db_path", absPath),
					zap.Error(statErr))
				allChecks = false
			}
		}

		// Check 7: Bootstrap cache
	bootstrapCheck:
		if cfgErr == nil {
			store, storeErr := openStore(ctx)
			if storeErr != nil {
				observability.CLILogger.Warn(fmt.Sprintf("[7/%d] Checking bootstrap cache... ⚠️  cannot open store", totalChecks), zap.Error(storeErr))
				allChecks = false
			} else {
				defer store.Close() //nolint:errcheck
				service := &checker.BootstrapService{Store: store}
				status, statusErr := service.Status(ctx)
				if statusErr != nil {
					observability.CLILogger.Warn(fmt.Sprintf("[7/%d] Checking bootstrap cache... ⚠️  cannot read status", totalChecks), zap.Error(statusErr))
					allChecks = false
				} else if status.TLDCount == 0 {
					observability.CLILogger.Warn(fmt.Sprintf("[7/%d] Checking bootstrap cache... ⚠️  empty (run 'namelens bootstrap update')", totalChecks))
				} else {
					ageStr := formatTimeAgo(status.FetchedAt)
					observability.CLILogger.Info(fmt.Sprintf("[7/%d] Checking bootstrap cache... ✅ %d TLDs (%s)", totalChecks, status.TLDCount, ageStr),
						zap.Int("tld_count", status.TLDCount),
						zap.Time("fetched_at", status.FetchedAt))
				}
			}
		} else {
			observability.CLILogger.Warn(fmt.Sprintf("[7/%d] Checking bootstrap cache... ⚠️  skipped (config not loaded)", totalChecks))
		}

		// Check 8: AI backend
		if cfgErr == nil {
			if isAIBackendConfigured(cfg.AILink) {
				observability.CLILogger.Info(fmt.Sprintf("[8/%d] Checking AI backend... ✅ configured", totalChecks))
			} else {
				observability.CLILogger.Warn(fmt.Sprintf("[8/%d] Checking AI backend... ⚠️  not configured (run 'namelens setup' or see docs)", totalChecks))
				observability.CLILogger.Info("       Expert analysis, name generation, and suitability checks require an AI backend.")
			}
		} else {
			observability.CLILogger.Warn(fmt.Sprintf("[8/%d] Checking AI backend... ⚠️  skipped (config not loaded)", totalChecks))
		}

		observability.CLILogger.Info("")
		if allChecks {
			appName := "namelens"
			if identity != nil && identity.BinaryName != "" {
				appName = identity.BinaryName
			}
			observability.CLILogger.Info(fmt.Sprintf("✅ All checks passed! Your %s installation is healthy.", appName))
		} else {
			observability.CLILogger.Warn("⚠️  Some checks failed. Review the output above for details.")
		}
		observability.CLILogger.Info("")
		observability.CLILogger.Info("=== End Diagnostics ===")
	},
}

var (
	doctorInitForce     bool
	doctorInitExpertKey string
	doctorResetConfig   bool
	doctorResetData     bool
	doctorResetAll      bool
)

var doctorInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := config.DefaultConfigPath()
		if configPath == "" {
			return fmt.Errorf("config path not resolved")
		}

		if _, err := os.Stat(configPath); err == nil && !doctorInitForce {
			return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
		}

		expertKey := strings.TrimSpace(doctorInitExpertKey)
		if strings.EqualFold(expertKey, "prompt") {
			key, err := promptForValue("Enter expert API key (leave blank to skip): ")
			if err != nil {
				return err
			}
			expertKey = key
		}

		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return fmt.Errorf("create config directory: %w", err)
		}

		mode := os.FileMode(0644)
		if expertKey != "" {
			mode = 0600
		}

		if err := os.WriteFile(configPath, []byte(buildInitConfig(expertKey)), mode); err != nil {
			return fmt.Errorf("write config file: %w", err)
		}

		observability.CLILogger.Info("Config initialized", zap.String("path", configPath))
		return nil
	},
}

var doctorConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show configuration status and paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := config.DefaultConfigPath()
		configExists := fileExists(configPath)

		dataDir := config.DefaultDataDir()
		cacheDir := config.DefaultCacheDir()

		observability.CLILogger.Info("Configuration:")
		observability.CLILogger.Info(fmt.Sprintf("  Config file:   %s (%s)", configPath, existenceStatus(configExists)))
		if dataDir != "" {
			observability.CLILogger.Info(fmt.Sprintf("  Data directory: %s (%s)", dataDir, existenceStatus(fileExists(dataDir))))
		} else {
			observability.CLILogger.Info("  Data directory: (not resolved)")
		}
		if cacheDir != "" {
			observability.CLILogger.Info(fmt.Sprintf("  Cache directory: %s (%s)", cacheDir, existenceStatus(fileExists(cacheDir))))
		} else {
			observability.CLILogger.Info("  Cache directory: (not resolved)")
		}

		cfg, err := config.Load(cmd.Context())
		if err != nil {
			observability.CLILogger.Warn("Config load failed", zap.Error(err))
		} else {
			if cfg.Store.URL != "" {
				observability.CLILogger.Info(fmt.Sprintf("  Database:      %s (remote)", cfg.Store.URL))
			} else {
				dbPath := cfg.Store.Path
				if dbPath == "" {
					dbPath = config.DefaultStorePath()
				}
				absPath, _ := filepath.Abs(dbPath)
				if info, statErr := os.Stat(absPath); statErr == nil {
					observability.CLILogger.Info(fmt.Sprintf("  Database:      %s (%s)", absPath, formatFileSize(info.Size())))
				} else if os.IsNotExist(statErr) {
					observability.CLILogger.Info(fmt.Sprintf("  Database:      %s (not created yet)", absPath))
				} else {
					observability.CLILogger.Warn("Database status error", zap.String("db_path", absPath), zap.Error(statErr))
				}
			}

			store, storeErr := openStore(cmd.Context())
			if storeErr != nil {
				observability.CLILogger.Warn("Bootstrap cache: not initialized (cannot open store)", zap.Error(storeErr))
			} else {
				defer store.Close() //nolint:errcheck
				service := &checker.BootstrapService{Store: store}
				status, statusErr := service.Status(cmd.Context())
				if statusErr != nil {
					observability.CLILogger.Warn("Bootstrap cache: not initialized (status unavailable)", zap.Error(statusErr))
				} else if status.TLDCount == 0 {
					observability.CLILogger.Warn("Bootstrap cache: empty (run 'namelens bootstrap update')")
				} else {
					observability.CLILogger.Info(fmt.Sprintf("  Bootstrap cache: %d TLDs (%s)", status.TLDCount, formatTimeAgo(status.FetchedAt)))
				}
			}

			observability.CLILogger.Info("")
			observability.CLILogger.Info("Environment:")
			observability.CLILogger.Info("  NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY: " + envStatus("NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY"))

			observability.CLILogger.Info("")
			observability.CLILogger.Info("Effective Settings:")
			observability.CLILogger.Info(fmt.Sprintf("  expert.enabled: %t", cfg.Expert.Enabled))
			observability.CLILogger.Info(fmt.Sprintf("  domain.whois_fallback.enabled: %t", cfg.Domain.WhoisFallback.Enabled))
		}

		return nil
	},
}

var doctorResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset user configuration and/or data",
	RunE: func(cmd *cobra.Command, args []string) error {
		if doctorResetAll {
			doctorResetConfig = true
			doctorResetData = true
		}

		if !doctorResetConfig && !doctorResetData {
			return fmt.Errorf("specify --config, --data, or --all")
		}

		if doctorResetConfig {
			configPath := config.DefaultConfigPath()
			if configPath == "" {
				observability.CLILogger.Warn("Config path not resolved; skipping config reset")
			} else if err := os.Remove(configPath); err == nil {
				observability.CLILogger.Info("Config removed", zap.String("path", configPath))
			} else if os.IsNotExist(err) {
				observability.CLILogger.Info("Config already removed", zap.String("path", configPath))
			} else {
				return fmt.Errorf("remove config file: %w", err)
			}
		}

		if doctorResetData {
			cfg, err := config.Load(cmd.Context())
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if cfg.Store.URL != "" {
				return fmt.Errorf("remote store configured; database reset is not supported")
			}

			dbPath := cfg.Store.Path
			if dbPath == "" {
				dbPath = config.DefaultStorePath()
			}
			absPath, _ := filepath.Abs(dbPath)
			if err := os.Remove(absPath); err == nil {
				observability.CLILogger.Info("Database removed", zap.String("path", absPath))
			} else if os.IsNotExist(err) {
				observability.CLILogger.Info("Database already removed", zap.String("path", absPath))
			} else {
				return fmt.Errorf("remove database: %w", err)
			}
		}

		return nil
	},
}

var doctorValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the current config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := config.DefaultConfigPath()
		if configPath == "" {
			return fmt.Errorf("config path not resolved")
		}
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", configPath)
		}

		if _, err := config.Load(cmd.Context()); err != nil {
			return err
		}

		observability.CLILogger.Info("Config is valid", zap.String("path", configPath))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.AddCommand(doctorInitCmd)
	doctorCmd.AddCommand(doctorConfigCmd)
	doctorCmd.AddCommand(doctorResetCmd)
	doctorCmd.AddCommand(doctorValidateCmd)

	doctorInitCmd.Flags().BoolVar(&doctorInitForce, "force", false, "overwrite existing config file")
	doctorInitCmd.Flags().StringVar(&doctorInitExpertKey, "expert-key", "", "set expert api key or use 'prompt' to enter")

	doctorResetCmd.Flags().BoolVar(&doctorResetConfig, "config", false, "remove user config file")
	doctorResetCmd.Flags().BoolVar(&doctorResetData, "data", false, "remove local database")
	doctorResetCmd.Flags().BoolVar(&doctorResetAll, "all", false, "remove config and data")
}

// formatFileSize returns a human-readable file size
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

// formatTimeAgo returns a human-readable relative time
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func buildInitConfig(expertKey string) string {
	lines := []string{
		"# namelens config - created by 'namelens doctor init'",
		"domain:",
		"  whois_fallback:",
		"    enabled: true",
		"ailink:",
		"  default_provider: namelens-xai",
		"  providers:",
		"    namelens-xai:",
		"      enabled: true",
		"      ai_provider: xai",
		"      base_url: https://api.x.ai/v1",
		"      models:",
		"        default: grok-4-1-fast-reasoning",
		"      credentials:",
		"        - label: default",
		"          priority: 0",
	}

	if strings.TrimSpace(expertKey) != "" {
		lines = append(lines, fmt.Sprintf("          api_key: %q", expertKey))
	} else {
		lines = append(lines, "          # api_key: \"\"  # Set via NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY or uncomment")
	}

	lines = append(lines,
		"expert:",
		"  enabled: true",
		"  default_prompt: name-availability",
	)

	return strings.Join(lines, "\n") + "\n"
}

func promptForValue(prompt string) (string, error) {
	if _, err := fmt.Fprint(os.Stdout, prompt); err != nil {
		return "", err
	}
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func existenceStatus(exists bool) string {
	if exists {
		return "exists"
	}
	return "missing"
}

func envStatus(name string) string {
	if strings.TrimSpace(os.Getenv(name)) != "" {
		return "(set)"
	}
	return "(not set)"
}
