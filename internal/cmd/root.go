package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fulmenhq/gofulmen/appidentity"
	gfconfig "github.com/fulmenhq/gofulmen/config"
	"github.com/fulmenhq/gofulmen/foundry"
	"github.com/fulmenhq/gofulmen/telemetry"

	"github.com/namelens/namelens/internal/ailink/driver"
	"github.com/namelens/namelens/internal/appid"
	"github.com/namelens/namelens/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/namelens/namelens/internal/observability"
)

var (
	cfgFile   string
	verbose   bool
	traceFile string

	// App identity loaded from .fulmen/app.yaml
	appIdentity *appidentity.Identity

	// Version info set by main package
	versionInfo struct {
		Version   string
		Commit    string
		BuildDate string
	}
)

// SetVersionInfo is called by main package to set version information
func SetVersionInfo(version, commit, buildDate string) {
	versionInfo.Version = version
	versionInfo.Commit = commit
	versionInfo.BuildDate = buildDate
}

// GetAppIdentity returns the loaded app identity (only valid after initConfig)
func GetAppIdentity() *appidentity.Identity {
	return appIdentity
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	// NOTE: initConfig() overwrites these from app identity.
	Use:   filepath.Base(os.Args[0]),
	Short: "A Fulmen workhorse application template",
	Long: `A production-ready Fulmen workhorse service template.

Use the subcommands to perform specific operations.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Disable global telemetry early to prevent config loading from emitting
	// metrics to stdout. Server mode will initialize proper telemetry later.
	disabledConfig := &telemetry.Config{Enabled: false}
	if sys, err := telemetry.NewSystem(disabledConfig); err == nil {
		telemetry.SetGlobalSystem(sys)
	}

	// Load app identity early for help text (before cobra processes --help)
	ctx := context.Background()
	if identity, err := appid.Get(ctx); err == nil && identity != nil {
		appIdentity = identity
		if identity.BinaryName != "" {
			rootCmd.Use = identity.BinaryName
		}
		if identity.Description != "" {
			rootCmd.Short = identity.Description
			rootCmd.Long = fmt.Sprintf("%s - %s\n\nUse the subcommands to perform specific operations.", identity.BinaryName, identity.Description)
		}
	}

	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (optional; defaults to app identity config path)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output (sets log level to debug)")
	rootCmd.PersistentFlags().StringVar(&traceFile, "trace", "", "trace AILink requests/responses to NDJSON file")

	// Bind flags to viper
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load app identity from .fulmen/app.yaml
	ctx := context.Background()
	identity, err := appid.Get(ctx)
	if err != nil {
		ExitWithCodeStderr(foundry.ExitFileNotFound, "Failed to load app identity from .fulmen/app.yaml", err)
	}
	appIdentity = identity

	// Update CLI help surfaces from app identity (CDRL-friendly)
	if identity != nil {
		if identity.BinaryName != "" {
			rootCmd.Use = identity.BinaryName
		}
		if identity.Description != "" {
			rootCmd.Short = identity.Description
			rootCmd.Long = fmt.Sprintf("%s - %s\n\nUse the subcommands to perform specific operations.", identity.BinaryName, identity.Description)
		}
		if f := rootCmd.PersistentFlags().Lookup("config"); f != nil && identity.ConfigName != "" {
			f.Usage = fmt.Sprintf("config file (default is $XDG_CONFIG_HOME/%s/config.yaml)", identity.ConfigName)
		}
	}

	// Initialize CLI logger early so we can use it in config loading
	observability.InitCLILogger(appIdentity.BinaryName, verbose)

	// Enable AILink tracing if requested
	if traceFile != "" {
		cleanup, err := driver.EnableTracing(traceFile)
		if err != nil {
			observability.CLILogger.Warn("Failed to enable tracing", zap.Error(err))
		} else {
			observability.CLILogger.Debug("AILink tracing enabled", zap.String("file", traceFile))
			// Note: cleanup is not called here since we want tracing for the entire session
			// The file will be closed when the process exits
			_ = cleanup
		}
	}

	if cfgFile != "" {
		// Use config file from flag
		viper.SetConfigFile(cfgFile)
	} else {
		appConfigDir := gfconfig.GetAppConfigDir(appIdentity.ConfigName)
		if appConfigDir == "" {
			if verbose {
				observability.CLILogger.Warn("Could not resolve XDG config directory, falling back to home directory")
			}
			// Fall back to home directory
			home, err := os.UserHomeDir()
			if err != nil {
				ExitWithCode(observability.CLILogger, foundry.ExitFileNotFound, "Could not find home directory", err)
			}
			viper.AddConfigPath(home)
			viper.SetConfigName("." + appIdentity.ConfigName)
		} else {
			viper.AddConfigPath(appConfigDir)
			viper.SetConfigName("config")

			if appIdentity.BinaryName != "" && appIdentity.BinaryName != appIdentity.ConfigName {
				legacyConfigDir := gfconfig.GetAppConfigDir(appIdentity.BinaryName)
				if legacyConfigDir != "" {
					viper.AddConfigPath(legacyConfigDir)
				}
			}
		}

		// Also search in current directory
		viper.AddConfigPath("./config")
		viper.SetConfigType("yaml")
	}

	// Read in environment variables with prefix from app identity
	viper.SetEnvPrefix(appIdentity.EnvPrefix)
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			observability.CLILogger.Debug("Using config file", zap.String("path", viper.ConfigFileUsed()))
		}
	} else {
		// It's OK if config file doesn't exist, we have defaults
		if verbose {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				observability.CLILogger.Debug("No config file found, using defaults and environment variables")
			} else {
				observability.CLILogger.Warn("Error reading config file", zap.Error(err))
			}
		}
	}

	// Set defaults
	setDefaults()
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")
	viper.SetDefault("server.shutdown_timeout", "10s")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.profile", "structured")

	// Store defaults
	viper.SetDefault("store.driver", "libsql")
	viper.SetDefault("store.path", config.DefaultStorePath())
	viper.SetDefault("store.url", "")
	viper.SetDefault("store.auth_token", "")

	// Cache defaults
	viper.SetDefault("cache.available_ttl", "5m")
	viper.SetDefault("cache.taken_ttl", "1h")
	viper.SetDefault("cache.error_ttl", "30s")

	// Rate limit overrides (optional)
	viper.SetDefault("rate_limits", map[string]int{})
	viper.SetDefault("rate_limit_margin", 0.9)

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9090)

	// Health check defaults
	viper.SetDefault("health.enabled", true)

	// Worker defaults
	viper.SetDefault("workers", 4)

	// Debug defaults
	viper.SetDefault("debug.enabled", false)
	viper.SetDefault("debug.pprof_enabled", false)
}
