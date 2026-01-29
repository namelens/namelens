package context

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		// Case variations - should all match (case-insensitive)
		{"README.md", "readme"},
		{"readme.md", "readme"},
		{"Readme.md", "readme"},
		{"README.txt", "readme"},
		{"readme.txt", "readme"},
		// Architecture variations
		{"ARCHITECTURE.md", "architecture"},
		{"architecture.md", "architecture"},
		{"DESIGN.md", "architecture"},
		{"Design.md", "architecture"},
		{"STRUCTURE.md", "architecture"},
		// Decisions
		{"DECISIONS.md", "decisions"},
		{"decisions.md", "decisions"},
		// Planning
		{"BOOTSTRAP.md", "planning"},
		{"Bootstrap.md", "planning"},
		{"PLAN.md", "planning"},
		{"VISION.md", "planning"},
		// Project metadata (typically lowercase)
		{"package.json", "project_metadata"},
		{"Package.json", "project_metadata"},
		{"Cargo.toml", "project_metadata"},
		{"cargo.toml", "project_metadata"},
		{"go.mod", "project_metadata"},
		{"Go.mod", "project_metadata"},
		{"pyproject.toml", "project_metadata"},
		// Code
		{"main.go", "code"},
		{"Main.Go", "code"},
		{"lib.rs", "code"},
		{"random.txt", ""}, // No match
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			class := ClassifyFile(tc.path, nil)
			if tc.want == "" {
				require.Nil(t, class)
			} else {
				require.NotNil(t, class)
				require.Equal(t, tc.want, class.Name)
			}
		})
	}
}

func TestClassifyFiles(t *testing.T) {
	files := []DiscoveredFile{
		{Path: "README.md"},
		{Path: "DECISIONS.md"},
		{Path: "main.go"},
		{Path: "unknown.xyz"},
	}

	classified := ClassifyFiles(files, nil)
	require.Len(t, classified, 4)

	require.Equal(t, "readme", classified[0].Class.Name)
	require.Equal(t, "decisions", classified[1].Class.Name)
	require.Equal(t, "code", classified[2].Class.Name)
	require.Nil(t, classified[3].Class)
}

func TestAllocateBudget(t *testing.T) {
	files := []ClassifiedFile{
		{DiscoveredFile: DiscoveredFile{Path: "README.md"}, Class: &DefaultClasses[0]},    // readme: 25%
		{DiscoveredFile: DiscoveredFile{Path: "DESIGN.md"}, Class: &DefaultClasses[1]},    // architecture: 20%
		{DiscoveredFile: DiscoveredFile{Path: "DECISIONS.md"}, Class: &DefaultClasses[2]}, // decisions: 20%
	}

	allocated := AllocateBudget(files, 10000)

	// Each file should have budget based on its class share
	require.Equal(t, 2500, allocated[0].BudgetAlloc) // 25% of 10000
	require.Equal(t, 2000, allocated[1].BudgetAlloc) // 20% of 10000
	require.Equal(t, 2000, allocated[2].BudgetAlloc) // 20% of 10000
}

func TestAllocateBudgetMultipleFilesPerClass(t *testing.T) {
	// Two files in the same class should split the class budget
	files := []ClassifiedFile{
		{DiscoveredFile: DiscoveredFile{Path: "README.md"}, Class: &DefaultClasses[0]},
		{DiscoveredFile: DiscoveredFile{Path: "README.txt"}, Class: &DefaultClasses[0]},
	}

	allocated := AllocateBudget(files, 10000)

	// 25% of 10000 = 2500, split between 2 files = 1250 each
	require.Equal(t, 1250, allocated[0].BudgetAlloc)
	require.Equal(t, 1250, allocated[1].BudgetAlloc)
}

func TestFilterByBudget(t *testing.T) {
	codeClass := &FileClass{Name: "code", MaxBudgetShare: 0.0}
	readmeClass := &FileClass{Name: "readme", MaxBudgetShare: 0.25}

	files := []ClassifiedFile{
		{DiscoveredFile: DiscoveredFile{Path: "README.md"}, Class: readmeClass, BudgetAlloc: 2500},
		{DiscoveredFile: DiscoveredFile{Path: "main.go"}, Class: codeClass, BudgetAlloc: 0},
	}

	filtered := FilterByBudget(files)
	require.Len(t, filtered, 1)
	require.Equal(t, "README.md", filtered[0].Path)
}
