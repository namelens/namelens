package ailink

import "time"

// Config defines provider configuration for AILink.
//
// This is intentionally self-contained so it can later be extracted as a
// standalone library configuration subtree.
type Config struct {
	DefaultProvider string        `mapstructure:"default_provider"`
	DefaultTimeout  time.Duration `mapstructure:"default_timeout"`
	CacheTTL        time.Duration `mapstructure:"cache_ttl"`

	// PromptsDir allows applications to override the built-in prompt set.
	// Prompts are owned by the application, but must follow AILink prompt rules.
	PromptsDir string `mapstructure:"prompts_dir"`

	// Debug controls optional diagnostics like raw payload capture.
	Debug DebugConfig `mapstructure:"debug"`

	// Providers is a set of provider instances keyed by a user-defined id (slug).
	// Each instance declares its underlying provider type via AIProvider.
	Providers map[string]ProviderInstanceConfig `mapstructure:"providers"`

	Routing   map[string]string   `mapstructure:"routing"`
	Fallbacks map[string][]string `mapstructure:"fallbacks"`
}

type DebugConfig struct {
	CaptureRawEnabled  bool `mapstructure:"capture_raw_enabled"`
	CaptureRawMaxBytes int  `mapstructure:"capture_raw_max_bytes"`
}

// ProviderInstanceConfig defines a configured provider instance (e.g. "namelens-xai").
type ProviderInstanceConfig struct {
	Enabled bool `mapstructure:"enabled"`

	// AIProvider is the provider type/driver identifier (e.g. "xai", "openai", "anthropic").
	AIProvider string `mapstructure:"ai_provider"`

	// SelectionPolicy controls which credential is chosen.
	// Supported values: "priority" (default), "round_robin".
	SelectionPolicy string `mapstructure:"selection_policy"`

	// DefaultCredential, if set, forces selecting the matching credential label.
	// If missing/invalid, selection falls back to SelectionPolicy.
	DefaultCredential string `mapstructure:"default_credential"`

	BaseURL      string            `mapstructure:"base_url"`
	Models       map[string]string `mapstructure:"models"`
	Capabilities Capabilities      `mapstructure:"capabilities"`
	Roles        []string          `mapstructure:"roles"`

	Credentials []CredentialConfig `mapstructure:"credentials"`
}

// CredentialConfig is a single credential for a provider instance.
//
// Multiple credentials enable key rotation, future load balancing, and per-key rate limit handling.
type CredentialConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Label    string `mapstructure:"label"`
	APIKey   string `mapstructure:"api_key"`
	Priority int    `mapstructure:"priority"`
}

// Capabilities describes provider-level hints.
//
// Drivers may also expose capabilities at runtime; these flags are primarily for
// config-time intent and future routing logic.
type Capabilities struct {
	Tools     bool `mapstructure:"tools"`
	Images    bool `mapstructure:"images"`
	Streaming bool `mapstructure:"streaming"`
}
