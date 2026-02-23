package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink"
)

func TestApplyGenerateProviderOverrideSetsRoleRouting(t *testing.T) {
	cfg := ailink.Config{
		Providers: map[string]ailink.ProviderInstanceConfig{
			"namelens-openai": {Enabled: true},
		},
		Routing: map[string]string{
			"name-alternatives": "namelens-xai",
		},
	}

	out, err := applyGenerateProviderOverride(cfg, "name-alternatives", "namelens-openai")
	require.NoError(t, err)
	require.Equal(t, "namelens-openai", out.Routing["name-alternatives"])
	require.Equal(t, "namelens-xai", cfg.Routing["name-alternatives"], "source config must not be mutated")
}

func TestApplyGenerateProviderOverrideUnknownProvider(t *testing.T) {
	cfg := ailink.Config{
		Providers: map[string]ailink.ProviderInstanceConfig{
			"namelens-anthropic": {Enabled: true},
			"namelens-openai":    {Enabled: true},
		},
	}

	_, err := applyGenerateProviderOverride(cfg, "name-alternatives", "missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown provider "missing"`)
	require.Contains(t, err.Error(), "namelens-anthropic, namelens-openai")
}

func TestApplyGenerateProviderOverrideDisabledProvider(t *testing.T) {
	cfg := ailink.Config{
		Providers: map[string]ailink.ProviderInstanceConfig{
			"namelens-openai": {Enabled: false},
		},
	}

	_, err := applyGenerateProviderOverride(cfg, "name-alternatives", "namelens-openai")
	require.Error(t, err)
	require.Equal(t, `provider "namelens-openai" is disabled`, err.Error())
}

func TestApplyGenerateProviderOverrideEmptyProviderNoop(t *testing.T) {
	cfg := ailink.Config{
		Providers: map[string]ailink.ProviderInstanceConfig{
			"namelens-openai": {Enabled: true},
		},
	}

	out, err := applyGenerateProviderOverride(cfg, "name-alternatives", "")
	require.NoError(t, err)
	require.Equal(t, cfg, out)
}
