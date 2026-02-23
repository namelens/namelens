package prompt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	prompts, err := LoadDefaults()
	require.NoError(t, err)
	require.NotEmpty(t, prompts)

	reg, err := NewRegistry(prompts)
	require.NoError(t, err)

	prompt, err := reg.Get("name-availability")
	require.NoError(t, err)
	require.NotEmpty(t, prompt.Config.SystemTemplate)
}

func TestBrandPlanPromptUsesBrandPlanResponseSchema(t *testing.T) {
	prompts, err := LoadDefaults()
	require.NoError(t, err)

	reg, err := NewRegistry(prompts)
	require.NoError(t, err)

	brandPlan, err := reg.Get("brand-plan")
	require.NoError(t, err)

	ref, _ := brandPlan.Config.ResponseSchema["$ref"].(string)
	require.Equal(t, "ailink/v0/brand-plan-response", ref)
}
