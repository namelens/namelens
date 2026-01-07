package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/fulmenhq/gofulmen/crucible"
	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/observability"
)

var envInfoCmd = &cobra.Command{
	Use:   "envinfo",
	Short: "Display environment information",
	Long:  "Display comprehensive environment, configuration, and version information.",
	Run: func(cmd *cobra.Command, args []string) {
		version := crucible.GetVersion()

		observability.CLILogger.Info("=== NameLens Environment Information ===")
		observability.CLILogger.Info("")

		// Application Info
		identity := GetAppIdentity()
		observability.CLILogger.Info("Application:")
		observability.CLILogger.Info("  Name:       " + identity.BinaryName)
		observability.CLILogger.Info("  Version:    " + versionInfo.Version)
		observability.CLILogger.Info("  Commit:     " + versionInfo.Commit)
		observability.CLILogger.Info("  Built:      " + versionInfo.BuildDate)
		observability.CLILogger.Info("")

		// SSOT Info
		observability.CLILogger.Info("SSOT:")
		observability.CLILogger.Info("  Gofulmen:   "+version.Gofulmen, zap.String("gofulmen_version", version.Gofulmen))
		observability.CLILogger.Info("  Crucible:   "+version.Crucible, zap.String("crucible_version", version.Crucible))
		observability.CLILogger.Info("")

		// Runtime Info
		observability.CLILogger.Info("Runtime:")
		observability.CLILogger.Info("  Go Version: "+runtime.Version(), zap.String("go_version", runtime.Version()))
		observability.CLILogger.Info("  GOOS:       "+runtime.GOOS, zap.String("goos", runtime.GOOS))
		observability.CLILogger.Info("  GOARCH:     "+runtime.GOARCH, zap.String("goarch", runtime.GOARCH))
		observability.CLILogger.Info(fmt.Sprintf("  NumCPU:     %d", runtime.NumCPU()), zap.Int("num_cpu", runtime.NumCPU()))
		observability.CLILogger.Info("")

		cfg, err := config.Load(cmd.Context())
		if err != nil {
			observability.CLILogger.Warn("Config load failed", zap.Error(err))
			return
		}

		// Configuration
		observability.CLILogger.Info("Configuration:")
		observability.CLILogger.Info("  Server Host:    "+cfg.Server.Host, zap.String("host", cfg.Server.Host))
		observability.CLILogger.Info(fmt.Sprintf("  Server Port:    %d", cfg.Server.Port), zap.Int("port", cfg.Server.Port))
		observability.CLILogger.Info("  Log Level:      "+cfg.Logging.Level, zap.String("log_level", cfg.Logging.Level))
		observability.CLILogger.Info("  Log Profile:    "+cfg.Logging.Profile, zap.String("log_profile", cfg.Logging.Profile))
		observability.CLILogger.Info("  DB Driver:      "+cfg.Store.Driver, zap.String("db_driver", cfg.Store.Driver))
		if strings.TrimSpace(cfg.Store.URL) != "" {
			observability.CLILogger.Info("  DB URL:         "+cfg.Store.URL, zap.String("db_url", cfg.Store.URL))
		} else {
			observability.CLILogger.Info("  DB Path:        "+cfg.Store.Path, zap.String("db_path", cfg.Store.Path))
		}
		observability.CLILogger.Info(fmt.Sprintf("  Metrics Port:   %d", cfg.Metrics.Port), zap.Int("metrics_port", cfg.Metrics.Port))
		observability.CLILogger.Info("  Config File:    "+config.DefaultConfigPath(), zap.String("config_file", config.DefaultConfigPath()))
		observability.CLILogger.Info("")

		// Domain Fallback Configuration
		observability.CLILogger.Info("Domain Fallback:")
		observability.CLILogger.Info(fmt.Sprintf("  Whois Enabled:  %t", cfg.Domain.WhoisFallback.Enabled), zap.Bool("whois_enabled", cfg.Domain.WhoisFallback.Enabled))
		if cfg.Domain.WhoisFallback.Enabled {
			observability.CLILogger.Info(fmt.Sprintf("  Require Explicit: %t", cfg.Domain.WhoisFallback.RequireExplicit))
			observability.CLILogger.Info("  Whois Timeout:  " + cfg.Domain.WhoisFallback.Timeout.String())
			if len(cfg.Domain.WhoisFallback.TLDs) > 0 {
				observability.CLILogger.Info(fmt.Sprintf("  Whois TLDs:     %v", cfg.Domain.WhoisFallback.TLDs))
			}
		}
		observability.CLILogger.Info(fmt.Sprintf("  DNS Enabled:    %t", cfg.Domain.DNSFallback.Enabled), zap.Bool("dns_enabled", cfg.Domain.DNSFallback.Enabled))
		observability.CLILogger.Info("")

		// AILink Provider Configuration
		observability.CLILogger.Info("AILink:")
		observability.CLILogger.Info("  Default Provider: " + cfg.AILink.DefaultProvider)
		observability.CLILogger.Info("  Default Timeout:  " + cfg.AILink.DefaultTimeout.String())
		providerID := strings.TrimSpace(cfg.AILink.DefaultProvider)
		if providerID == "" {
			providerID = "(unset)"
		}
		providerCfg, ok := cfg.AILink.Providers[providerID]
		if !ok {
			observability.CLILogger.Info(fmt.Sprintf("  %s: (not configured)", providerID))
		} else {
			observability.CLILogger.Info(fmt.Sprintf("  %s.enabled: %t", providerID, providerCfg.Enabled))
			observability.CLILogger.Info(fmt.Sprintf("  %s.ai_provider: %s", providerID, providerCfg.AIProvider))
			observability.CLILogger.Info(fmt.Sprintf("  %s.base_url: %s", providerID, providerCfg.BaseURL))
			observability.CLILogger.Info(fmt.Sprintf("  %s.model: %s", providerID, providerCfg.Models["default"]))
			if len(providerCfg.Credentials) > 0 && strings.TrimSpace(providerCfg.Credentials[0].APIKey) != "" {
				observability.CLILogger.Info(fmt.Sprintf("  %s.credentials[0].api_key: (set)", providerID))
			} else {
				observability.CLILogger.Info(fmt.Sprintf("  %s.credentials[0].api_key: (not set)", providerID))
			}
		}
		observability.CLILogger.Info("")

		// Expert Feature Configuration
		observability.CLILogger.Info("Expert:")
		observability.CLILogger.Info(fmt.Sprintf("  Enabled:          %t", cfg.Expert.Enabled), zap.Bool("expert_enabled", cfg.Expert.Enabled))
		observability.CLILogger.Info("  Role:             " + cfg.Expert.Role)
		observability.CLILogger.Info("  Default Prompt:   " + cfg.Expert.DefaultPrompt)
		observability.CLILogger.Info("")

		observability.CLILogger.Info("=== End Environment Information ===")
	},
}

func init() {
	rootCmd.AddCommand(envInfoCmd)
}
