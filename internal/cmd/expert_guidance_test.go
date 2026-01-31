package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/namelens/namelens/internal/ailink"
)

func TestIsAIBackendConfigured(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ailink.Config
		expected bool
	}{
		{
			name:     "empty config",
			cfg:      ailink.Config{},
			expected: false,
		},
		{
			name: "provider with empty credentials",
			cfg: ailink.Config{
				Providers: map[string]ailink.ProviderInstanceConfig{
					"test": {Enabled: true, Credentials: []ailink.CredentialConfig{}},
				},
			},
			expected: false,
		},
		{
			name: "provider with disabled credential",
			cfg: ailink.Config{
				Providers: map[string]ailink.ProviderInstanceConfig{
					"test": {
						Enabled: true,
						Credentials: []ailink.CredentialConfig{
							{Enabled: false, APIKey: "sk-test"},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "provider with empty API key",
			cfg: ailink.Config{
				Providers: map[string]ailink.ProviderInstanceConfig{
					"test": {
						Enabled: true,
						Credentials: []ailink.CredentialConfig{
							{Enabled: true, APIKey: ""},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "provider with whitespace API key",
			cfg: ailink.Config{
				Providers: map[string]ailink.ProviderInstanceConfig{
					"test": {
						Enabled: true,
						Credentials: []ailink.CredentialConfig{
							{Enabled: true, APIKey: "   "},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "disabled provider with valid key",
			cfg: ailink.Config{
				Providers: map[string]ailink.ProviderInstanceConfig{
					"test": {
						Enabled: false,
						Credentials: []ailink.CredentialConfig{
							{Enabled: true, APIKey: "sk-valid"},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "valid configuration",
			cfg: ailink.Config{
				Providers: map[string]ailink.ProviderInstanceConfig{
					"test": {
						Enabled: true,
						Credentials: []ailink.CredentialConfig{
							{Enabled: true, APIKey: "sk-valid"},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple providers one valid",
			cfg: ailink.Config{
				Providers: map[string]ailink.ProviderInstanceConfig{
					"disabled": {Enabled: false, Credentials: []ailink.CredentialConfig{{Enabled: true, APIKey: "sk-1"}}},
					"valid":    {Enabled: true, Credentials: []ailink.CredentialConfig{{Enabled: true, APIKey: "sk-2"}}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAIBackendConfigured(tt.cfg)
			if result != tt.expected {
				t.Errorf("isAIBackendConfigured() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShowExpertGuidanceWarning(t *testing.T) {
	// Reset state before test
	resetExpertGuidance()

	t.Run("shows warning when not configured", func(t *testing.T) {
		resetExpertGuidance()
		var buf bytes.Buffer
		cfg := ailink.Config{}

		showExpertGuidanceWarning(cfg, &buf)

		output := buf.String()
		if !strings.Contains(output, "limited analysis mode") {
			t.Error("expected warning message about limited analysis mode")
		}
		if !strings.Contains(output, "namelens config set") {
			t.Error("expected configuration instructions")
		}
	})

	t.Run("does not show when configured", func(t *testing.T) {
		resetExpertGuidance()
		var buf bytes.Buffer
		cfg := ailink.Config{
			Providers: map[string]ailink.ProviderInstanceConfig{
				"test": {
					Enabled:     true,
					Credentials: []ailink.CredentialConfig{{Enabled: true, APIKey: "sk-test"}},
				},
			},
		}

		showExpertGuidanceWarning(cfg, &buf)

		if buf.Len() > 0 {
			t.Errorf("expected no output when configured, got: %s", buf.String())
		}
	})

	t.Run("shows only once per session", func(t *testing.T) {
		resetExpertGuidance()
		var buf1, buf2 bytes.Buffer
		cfg := ailink.Config{}

		showExpertGuidanceWarning(cfg, &buf1)
		showExpertGuidanceWarning(cfg, &buf2)

		if buf1.Len() == 0 {
			t.Error("expected warning on first call")
		}
		if buf2.Len() > 0 {
			t.Error("expected no warning on second call")
		}
	})
}

func TestShowExpertTip(t *testing.T) {
	t.Run("shows tip when configured but expert not used", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := ailink.Config{
			Providers: map[string]ailink.ProviderInstanceConfig{
				"test": {
					Enabled:     true,
					Credentials: []ailink.CredentialConfig{{Enabled: true, APIKey: "sk-test"}},
				},
			},
		}

		showExpertTip(cfg, false, &buf)

		output := buf.String()
		if !strings.Contains(output, "--expert") {
			t.Error("expected tip about --expert flag")
		}
	})

	t.Run("does not show when expert used", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := ailink.Config{
			Providers: map[string]ailink.ProviderInstanceConfig{
				"test": {
					Enabled:     true,
					Credentials: []ailink.CredentialConfig{{Enabled: true, APIKey: "sk-test"}},
				},
			},
		}

		showExpertTip(cfg, true, &buf)

		if buf.Len() > 0 {
			t.Errorf("expected no output when expert used, got: %s", buf.String())
		}
	})

	t.Run("does not show when not configured", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := ailink.Config{}

		showExpertTip(cfg, false, &buf)

		if buf.Len() > 0 {
			t.Errorf("expected no output when not configured, got: %s", buf.String())
		}
	})
}
