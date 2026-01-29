package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscover(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "DECISIONS.md"), []byte("# Decisions"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.txt"), []byte("not matched"), 0644))

	files, err := Discover(dir, DefaultConfig())
	require.NoError(t, err)
	require.Len(t, files, 2)

	// README.md should come first (lower priority number)
	require.Equal(t, "README.md", files[0].Path)
	require.Equal(t, "DECISIONS.md", files[1].Path)
}

func TestDiscoverSubdirectory(t *testing.T) {
	dir := t.TempDir()

	// Create docs subdirectory
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Guide"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# README"), 0644))

	files, err := Discover(dir, DefaultConfig())
	require.NoError(t, err)
	require.Len(t, files, 2)

	// README should come before docs/*.md due to pattern order
	require.Equal(t, "README.md", files[0].Path)
	require.Equal(t, filepath.Join("docs", "guide.md"), files[1].Path)
}

func TestGather(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Project\n\nThis is the readme."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "DECISIONS.md"), []byte("# Decisions\n\nKey decisions here."), 0644))

	result, err := Gather(dir, DefaultConfig())
	require.NoError(t, err)
	require.Len(t, result.FilesUsed, 2)
	// Header now includes class name
	require.Contains(t, result.Context, "--- File: README.md (readme) ---")
	require.Contains(t, result.Context, "--- File: DECISIONS.md (decisions) ---")
	require.Contains(t, result.Context, "This is the readme")
	require.Equal(t, 0, result.FilesTrimmed)
	require.Equal(t, 0, result.FilesSkipped)

	// Check detailed file info
	require.Len(t, result.Included, 2)
	require.Equal(t, "readme", result.Included[0].Class)
	require.Equal(t, "full", result.Included[0].Coverage)
}

func TestGatherWithBudget(t *testing.T) {
	dir := t.TempDir()

	// Create a large file that exceeds budget
	largeContent := strings.Repeat("This is a test sentence. ", 1000)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(largeContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "DECISIONS.md"), []byte("Small file"), 0644))

	cfg := Config{
		Patterns: DefaultPatterns,
		MaxChars: 500, // Very small budget
	}

	result, err := Gather(dir, cfg)
	require.NoError(t, err)
	require.LessOrEqual(t, result.TotalChars, 600) // Some overhead allowed
	require.Contains(t, result.Context, "[... truncated ...]")
}

func TestGatherEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := Gather(dir, DefaultConfig())
	require.NoError(t, err)
	require.Empty(t, result.FilesUsed)
	require.Empty(t, result.Context)
}

func TestTruncateAtBoundary(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{
			name:   "no truncation needed",
			text:   "short text",
			maxLen: 100,
			want:   "short text",
		},
		{
			name:   "truncate at paragraph",
			text:   "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
			maxLen: 30,
			want:   "First paragraph.",
		},
		{
			name:   "truncate at line",
			text:   "Line one.\nLine two.\nLine three.",
			maxLen: 15,
			want:   "Line one.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateAtBoundary(tc.text, tc.maxLen)
			require.Equal(t, tc.want, got)
		})
	}
}
