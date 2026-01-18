package ailink

import (
	"testing"

	"github.com/stretchr/testify/require"

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
