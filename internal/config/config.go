package config

import (
	"time"

	"github.com/namelens/namelens/internal/ailink"
)

// Config represents the complete application configuration
// following the Fulmen Forge Workhorse Standard three-layer pattern:
// Layer 1: Crucible defaults (config/namelens/v0/namelens-defaults.yaml)
// Layer 2: User overrides (~/.config/namelens/namelens/config.yaml)
// Layer 3: Environment variables and runtime overrides
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Store   StoreConfig   `mapstructure:"store"`
	Cache   CacheConfig   `mapstructure:"cache"`
	Domain  DomainConfig  `mapstructure:"domain"`
	AILink  ailink.Config `mapstructure:"ailink"`
	Expert  ExpertConfig  `mapstructure:"expert"`
	Logging LoggingConfig `mapstructure:"logging"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Health  HealthConfig  `mapstructure:"health"`
	Debug   DebugConfig   `mapstructure:"debug"`
	Workers int           `mapstructure:"workers"`

	RateLimits      map[string]int `mapstructure:"rate_limits"`
	RateLimitMargin float64        `mapstructure:"rate_limit_margin"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// StoreConfig contains database configuration for libsql/Turso
type StoreConfig struct {
	Driver    string `mapstructure:"driver"`
	Path      string `mapstructure:"path"`
	URL       string `mapstructure:"url"`
	AuthToken string `mapstructure:"auth_token"`
}

// CacheConfig contains result cache TTL configuration.
type CacheConfig struct {
	AvailableTTL time.Duration `mapstructure:"available_ttl"`
	TakenTTL     time.Duration `mapstructure:"taken_ttl"`
	ErrorTTL     time.Duration `mapstructure:"error_ttl"`
}

// DomainConfig contains domain checker configuration.
type DomainConfig struct {
	WhoisFallback WhoisFallbackConfig `mapstructure:"whois_fallback"`
	DNSFallback   DNSFallbackConfig   `mapstructure:"dns_fallback"`
}

// WhoisFallbackConfig configures RDAP fallback behavior.
type WhoisFallbackConfig struct {
	Enabled           bool              `mapstructure:"enabled"`
	TLDs              []string          `mapstructure:"tlds"`
	RequireExplicit   bool              `mapstructure:"require_explicit"`
	CacheTTL          time.Duration     `mapstructure:"cache_ttl"`
	Timeout           time.Duration     `mapstructure:"timeout"`
	Servers           map[string]string `mapstructure:"servers"`
	AvailablePatterns []string          `mapstructure:"available_patterns"`
	TakenPatterns     []string          `mapstructure:"taken_patterns"`
}

// DNSFallbackConfig configures DNS-based fallback checks.
type DNSFallbackConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	CacheTTL time.Duration `mapstructure:"cache_ttl"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// ExpertConfig contains NameLens expert feature settings.
//
// Provider credentials and routing live under `ailink.*`.
type ExpertConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Role          string `mapstructure:"role"`
	DefaultPrompt string `mapstructure:"default_prompt"`
}

// LoggingConfig contains logging configuration
// Supports progressive logging profiles per Fulmen Forge Workhorse Standard:
// - SIMPLE: Console output only, minimal configuration (CLI tools)
// - STRUCTURED: Structured sinks, correlation IDs (API services)
// - ENTERPRISE: Multiple sinks, middleware, throttling, policy enforcement (production)
type LoggingConfig struct {
	// Level controls the minimum log level
	// Valid values: trace, debug, info, warn, error
	Level string `mapstructure:"level"`

	// Profile selects the logging complexity level
	// Valid values: SIMPLE, STRUCTURED, ENTERPRISE
	// See: gofulmen/docs/crucible-go/standards/observability/logging.md
	Profile string `mapstructure:"profile"`
}

// MetricsConfig contains Prometheus metrics configuration
type MetricsConfig struct {
	// Enabled controls whether metrics are exposed
	Enabled bool `mapstructure:"enabled"`

	// Port is the dedicated metrics endpoint port (Prometheus format)
	// Metrics are also available at the main HTTP port in JSON format
	Port int `mapstructure:"port"`
}

// HealthConfig contains health check configuration
type HealthConfig struct {
	// Enabled controls whether health endpoints are exposed
	Enabled bool `mapstructure:"enabled"`
}

// DebugConfig contains debug and profiling configuration
type DebugConfig struct {
	// Enabled controls whether debug mode is active
	Enabled bool `mapstructure:"enabled"`

	// PprofEnabled controls whether pprof endpoints are exposed
	// WARNING: Only enable in development/staging environments
	PprofEnabled bool `mapstructure:"pprof_enabled"`
}
