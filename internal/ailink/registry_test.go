package ailink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink/driver/anthropic"
	"github.com/namelens/namelens/internal/ailink/driver/openai"
	"github.com/namelens/namelens/internal/ailink/driver/xai"
	"github.com/namelens/namelens/internal/ailink/prompt"
)

func TestResolveModelPrefersProviderTierReasoningForDeep(t *testing.T) {
	providerCfg := ProviderInstanceConfig{Models: map[string]string{"default": "m-default", "reasoning": "m-reasoning"}}
	promptDef := &prompt.Prompt{Config: prompt.Config{ProviderHints: map[string]any{"preferred_models": []string{"prompt-model"}}}}

	model, err := resolveModel(providerCfg, promptDef, "", "deep")
	require.NoError(t, err)
	require.Equal(t, "m-reasoning", model)
}

func TestResolveModelFallsBackToDefaultWhenTierMissing(t *testing.T) {
	providerCfg := ProviderInstanceConfig{Models: map[string]string{"default": "m-default"}}

	model, err := resolveModel(providerCfg, nil, "", "deep")
	require.NoError(t, err)
	require.Equal(t, "m-default", model)
}

func TestResolveModelUsesFastTierForFastDepth(t *testing.T) {
	providerCfg := ProviderInstanceConfig{Models: map[string]string{"default": "m-default", "fast": "m-fast"}}

	model, err := resolveModel(providerCfg, nil, "", "fast")
	require.NoError(t, err)
	require.Equal(t, "m-fast", model)
}

func TestResolveModelUsesOverrideFirst(t *testing.T) {
	providerCfg := ProviderInstanceConfig{Models: map[string]string{"default": "m-default", "reasoning": "m-reasoning"}}

	model, err := resolveModel(providerCfg, nil, "override-model", "deep")
	require.NoError(t, err)
	require.Equal(t, "override-model", model)
}

func TestResolveModelFallsBackToPromptPreferredModels(t *testing.T) {
	providerCfg := ProviderInstanceConfig{}
	promptDef := &prompt.Prompt{Config: prompt.Config{ProviderHints: map[string]any{"preferred_models": []string{"prompt-model"}}}}

	model, err := resolveModel(providerCfg, promptDef, "", "")
	require.NoError(t, err)
	require.Equal(t, "prompt-model", model)
}

// TestDriverForUsesMaxTimeoutCeilingNotConfigDefault locks in the v0.2.5
// PR-2 layering fix. Driver-level `client.Timeout` must NOT be set to the
// config DefaultTimeout — doing so silently re-clamps the service-level
// context with a shorter duration whenever the service resolved a
// longer-than-default timeout (e.g. --timeout 240s with default 60s),
// capping the effective deadline at the config default and defeating the
// per-call override.
//
// The fix: driver client.Timeout is set to maxTimeout (5m), which matches
// the service-level clamp ceiling. Because context.WithTimeout only
// shortens, any service-level deadline ≤ maxTimeout passes through
// unmodified; bypass callers still get a 5m hard ceiling.
func TestDriverForUsesMaxTimeoutCeilingNotConfigDefault(t *testing.T) {
	// Use a config default that's distinguishable from both the package
	// fallback (60s) and maxTimeout (5m), so a regression to either value
	// is caught.
	const configDefault = 42 * time.Second

	cases := []struct {
		name         string
		aiProvider   string
		assertClient func(t *testing.T, d any)
	}{
		{
			name:       "xai",
			aiProvider: "xai",
			assertClient: func(t *testing.T, d any) {
				t.Helper()
				c, ok := d.(*xai.Client)
				require.True(t, ok, "expected *xai.Client, got %T", d)
				require.Equal(t, maxTimeout, c.Timeout, "xai client.Timeout should be maxTimeout, not config default (%s)", configDefault)
			},
		},
		{
			name:       "openai",
			aiProvider: "openai",
			assertClient: func(t *testing.T, d any) {
				t.Helper()
				c, ok := d.(*openai.Client)
				require.True(t, ok, "expected *openai.Client, got %T", d)
				require.Equal(t, maxTimeout, c.Timeout)
			},
		},
		{
			name:       "anthropic",
			aiProvider: "anthropic",
			assertClient: func(t *testing.T, d any) {
				t.Helper()
				c, ok := d.(*anthropic.Client)
				require.True(t, ok, "expected *anthropic.Client, got %T", d)
				require.Equal(t, maxTimeout, c.Timeout)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				DefaultProvider: "p",
				DefaultTimeout:  configDefault,
				Providers: map[string]ProviderInstanceConfig{
					"p": {
						Enabled:     true,
						AIProvider:  tc.aiProvider,
						Models:      map[string]string{"default": "m"},
						Credentials: []CredentialConfig{{APIKey: "k"}},
					},
				},
			}
			r := NewRegistry(cfg)
			d, err := r.driverFor("p", cfg.Providers["p"], cfg.Providers["p"].Credentials[0], "p0")
			require.NoError(t, err)
			tc.assertClient(t, d)
		})
	}
}
