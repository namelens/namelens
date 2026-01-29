package context

import (
	"path/filepath"
	"strings"
)

// FileClass defines a category of files with associated discovery and budget rules.
type FileClass struct {
	Name           string   // Identifier: "readme", "architecture", etc.
	Patterns       []string // Glob patterns that match this class
	Priority       int      // Lower = more important (included first)
	MaxBudgetShare float64  // 0.0-1.0, max fraction of total budget
	ExtractMode    ExtractMode
}

// ExtractMode determines how content is extracted from files.
type ExtractMode string

const (
	// ExtractFull reads the entire file content.
	ExtractFull ExtractMode = "full"
	// ExtractMetadata extracts only key metadata (for package files).
	ExtractMetadata ExtractMode = "metadata"
	// ExtractSummary extracts first N lines or sections (future).
	ExtractSummary ExtractMode = "summary"
)

// DefaultClasses defines the standard file classification hierarchy.
// Order matters: earlier classes have higher priority.
var DefaultClasses = []FileClass{
	{
		Name:           "readme",
		Patterns:       []string{"README.md", "README.txt", "README", "readme.md"},
		Priority:       1,
		MaxBudgetShare: 0.25,
		ExtractMode:    ExtractFull,
	},
	{
		Name:           "architecture",
		Patterns:       []string{"ARCHITECTURE.md", "ARCHITECTURE.txt", "DESIGN.md", "STRUCTURE.md"},
		Priority:       2,
		MaxBudgetShare: 0.20,
		ExtractMode:    ExtractFull,
	},
	{
		Name:           "decisions",
		Patterns:       []string{"DECISIONS.md", "DECISIONS.txt", "ADR-*.md", "adr-*.md"},
		Priority:       3,
		MaxBudgetShare: 0.20,
		ExtractMode:    ExtractFull,
	},
	{
		Name:           "planning",
		Patterns:       []string{"BOOTSTRAP.md", "PLAN.md", "CONCEPT.md", "VISION.md", "OVERVIEW.md"},
		Priority:       4,
		MaxBudgetShare: 0.15,
		ExtractMode:    ExtractFull,
	},
	{
		Name:           "project_metadata",
		Patterns:       []string{"package.json", "Cargo.toml", "go.mod", "pyproject.toml"},
		Priority:       5,
		MaxBudgetShare: 0.05,
		ExtractMode:    ExtractMetadata,
	},
	{
		Name:           "general_docs",
		Patterns:       []string{"docs/*.md", "doc/*.md", "*.md"},
		Priority:       6,
		MaxBudgetShare: 0.15,
		ExtractMode:    ExtractFull,
	},
	{
		Name:           "code",
		Patterns:       []string{"*.go", "*.rs", "*.py", "*.ts", "*.js"},
		Priority:       10,
		MaxBudgetShare: 0.0, // Skip code by default
		ExtractMode:    ExtractFull,
	},
}

// ClassifiedFile extends DiscoveredFile with classification information.
type ClassifiedFile struct {
	DiscoveredFile
	Class       *FileClass
	BudgetAlloc int // Allocated budget in characters
}

// ClassifyFile determines which class a file belongs to.
// Matching is case-insensitive to handle varying conventions (README.md vs readme.md).
// Returns nil if no class matches.
func ClassifyFile(path string, classes []FileClass) *FileClass {
	if classes == nil {
		classes = DefaultClasses
	}

	baseName := filepath.Base(path)
	baseNameLower := strings.ToLower(baseName)
	pathLower := strings.ToLower(path)

	for i := range classes {
		class := &classes[i]
		for _, pattern := range class.Patterns {
			patternLower := strings.ToLower(pattern)

			// Try exact match on basename first (case-insensitive)
			if matched, _ := filepath.Match(patternLower, baseNameLower); matched {
				return class
			}
			// Try match on full path for patterns with /
			if strings.Contains(pattern, "/") || strings.Contains(pattern, "*") {
				if matched, _ := filepath.Match(patternLower, pathLower); matched {
					return class
				}
			}
		}
	}

	return nil
}

// ClassifyFiles classifies a list of discovered files.
func ClassifyFiles(files []DiscoveredFile, classes []FileClass) []ClassifiedFile {
	if classes == nil {
		classes = DefaultClasses
	}

	result := make([]ClassifiedFile, 0, len(files))
	for _, f := range files {
		class := ClassifyFile(f.Path, classes)
		result = append(result, ClassifiedFile{
			DiscoveredFile: f,
			Class:          class,
		})
	}

	return result
}

// AllocateBudget distributes the total budget across classified files.
// Returns files with BudgetAlloc set, sorted by priority.
func AllocateBudget(files []ClassifiedFile, totalBudget int) []ClassifiedFile {
	if totalBudget <= 0 || len(files) == 0 {
		return files
	}

	// Group files by class
	byClass := make(map[string][]int) // class name -> indices
	for i, f := range files {
		className := "unknown"
		if f.Class != nil {
			className = f.Class.Name
		}
		byClass[className] = append(byClass[className], i)
	}

	// Calculate budget per class
	classBudgets := make(map[string]int)
	for className, indices := range byClass {
		// Find the class definition
		maxShare := 0.1 // Default 10% for unknown
		for _, f := range files {
			if f.Class != nil && f.Class.Name == className {
				maxShare = f.Class.MaxBudgetShare
				break
			}
		}

		// Allocate budget for this class
		classBudget := int(float64(totalBudget) * maxShare)
		if classBudget > 0 {
			classBudgets[className] = classBudget
		}

		// Distribute evenly among files in this class
		if len(indices) > 0 && classBudget > 0 {
			perFile := classBudget / len(indices)
			for _, idx := range indices {
				files[idx].BudgetAlloc = perFile
			}
		}
	}

	return files
}

// FilterByBudget removes files with zero budget allocation.
func FilterByBudget(files []ClassifiedFile) []ClassifiedFile {
	result := make([]ClassifiedFile, 0, len(files))
	for _, f := range files {
		if f.BudgetAlloc > 0 || (f.Class != nil && f.Class.MaxBudgetShare > 0) {
			result = append(result, f)
		}
	}
	return result
}
