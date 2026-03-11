package prompt

import (
	"os"
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

func TestCatalogForSchemasFallsBackToEmbedded(t *testing.T) {
	// Simulate running from outside a git repo by changing to a temp directory.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = CleanupStandaloneSchemas()
	})

	catalog, err := catalogForSchemas()
	require.NoError(t, err, "catalogForSchemas should fall back to embedded schema outside a repo")
	require.NotNil(t, catalog)
}

func TestLoadDefaultsOutsideRepo(t *testing.T) {
	// Verify that LoadDefaults works from a directory with no .git or go.mod.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = CleanupStandaloneSchemas()
	})

	prompts, err := LoadDefaults()
	require.NoError(t, err, "LoadDefaults should succeed outside a repo via embedded schema fallback")
	require.NotEmpty(t, prompts)
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
