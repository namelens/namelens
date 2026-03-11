package ailink

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStandaloneSchemaCatalogOutsideRepo(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = CleanupStandaloneSchemas()
	})

	catalog, err := StandaloneSchemaCatalog()
	require.NoError(t, err, "StandaloneSchemaCatalog should succeed outside a repo")
	require.NotNil(t, catalog)

	// Verify the catalog can resolve a known schema ID.
	diagnostics, err := catalog.ValidateDataByID("ailink/v0/prompt", []byte(`{"slug":"test","system_template":"hello"}`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
}

func TestStandaloneSchemaCatalogValidatesResponseSchemas(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = CleanupStandaloneSchemas()
	})

	catalog, err := StandaloneSchemaCatalog()
	require.NoError(t, err)

	// Verify search-response schema is available (used by expert checks).
	diagnostics, err := catalog.ValidateDataByID("ailink/v0/search-response", []byte(`{
		"summary": "test",
		"risk_level": "low",
		"likely_available": true,
		"confidence": 0.9,
		"mentions": [],
		"insights": [],
		"recommendations": []
	}`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
}
