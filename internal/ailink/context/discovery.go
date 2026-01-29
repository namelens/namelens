// Package context provides utilities for gathering context from project files
// to enrich AI prompts with relevant project information.
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultPatterns are the default file patterns to search for context.
// Ordered by priority - earlier patterns are included first.
var DefaultPatterns = []string{
	"README.md",
	"README.txt",
	"README",
	"ARCHITECTURE.md",
	"DESIGN.md",
	"DECISIONS.md",
	"STRUCTURE.md",
	"BOOTSTRAP.md",
	"CONCEPT.md",
	"OVERVIEW.md",
	"package.json",   // Node.js project metadata
	"Cargo.toml",     // Rust project metadata
	"go.mod",         // Go project metadata
	"pyproject.toml", // Python project metadata
	"docs/*.md",
	"doc/*.md",
	"*.md", // Catch-all for other markdown in root
}

// DefaultMaxChars is the default maximum characters to include in context.
// Roughly 8000 tokens at ~4 chars/token.
const DefaultMaxChars = 32000

// Config holds context discovery configuration.
type Config struct {
	Patterns []string // File patterns to search (globs)
	MaxChars int      // Maximum characters to include
}

// DefaultConfig returns the default discovery configuration.
func DefaultConfig() Config {
	return Config{
		Patterns: DefaultPatterns,
		MaxChars: DefaultMaxChars,
	}
}

// DiscoveredFile represents a file found during discovery.
type DiscoveredFile struct {
	Path     string // Relative path from scan root
	AbsPath  string // Absolute path
	Priority int    // Lower = higher priority (pattern index)
	Size     int64  // File size in bytes
}

// Discover finds files matching the configured patterns in the given directory.
func Discover(dir string, cfg Config) ([]DiscoveredFile, error) {
	if cfg.Patterns == nil {
		cfg.Patterns = DefaultPatterns
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}

	seen := make(map[string]bool)
	var files []DiscoveredFile

	for priority, pattern := range cfg.Patterns {
		matches, err := filepath.Glob(filepath.Join(absDir, pattern))
		if err != nil {
			continue // Invalid pattern, skip
		}

		for _, absPath := range matches {
			if seen[absPath] {
				continue
			}

			info, err := os.Stat(absPath)
			if err != nil || info.IsDir() {
				continue
			}

			relPath, _ := filepath.Rel(absDir, absPath)
			seen[absPath] = true
			files = append(files, DiscoveredFile{
				Path:     relPath,
				AbsPath:  absPath,
				Priority: priority,
				Size:     info.Size(),
			})
		}
	}

	// Sort by priority (pattern order), then by path for determinism
	sort.Slice(files, func(i, j int) bool {
		if files[i].Priority != files[j].Priority {
			return files[i].Priority < files[j].Priority
		}
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// FileInfo contains metadata about an included or excluded file.
type FileInfo struct {
	Path     string `json:"path"`
	Class    string `json:"class,omitempty"`
	Chars    int    `json:"chars"`
	Coverage string `json:"coverage"` // "full", "truncated", "metadata", "skipped"
	Reason   string `json:"reason,omitempty"`
}

// GatherResult contains the gathered context and metadata.
type GatherResult struct {
	Context      string     // Formatted context string
	FilesUsed    []string   // Relative paths of files included (for backward compat)
	TotalChars   int        // Total characters included
	FilesTrimmed int        // Number of files that were truncated
	FilesSkipped int        // Number of files skipped due to budget
	Included     []FileInfo // Detailed info about included files
	Excluded     []FileInfo // Detailed info about excluded files
}

// Gather reads files and assembles context within the character budget.
// Uses file classification to allocate budget intelligently.
func Gather(dir string, cfg Config) (*GatherResult, error) {
	if cfg.MaxChars <= 0 {
		cfg.MaxChars = DefaultMaxChars
	}

	files, err := Discover(dir, cfg)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return &GatherResult{}, nil
	}

	// Classify files and allocate budget
	classified := ClassifyFiles(files, nil)
	classified = AllocateBudget(classified, cfg.MaxChars)

	// Sort by class priority, then by path
	sort.Slice(classified, func(i, j int) bool {
		pi, pj := 100, 100
		if classified[i].Class != nil {
			pi = classified[i].Class.Priority
		}
		if classified[j].Class != nil {
			pj = classified[j].Class.Priority
		}
		if pi != pj {
			return pi < pj
		}
		return classified[i].Path < classified[j].Path
	})

	var builder strings.Builder
	var filesUsed []string
	var included []FileInfo
	var excluded []FileInfo
	remaining := cfg.MaxChars
	filesTrimmed := 0
	filesSkipped := 0

	// Reserve space for file headers (~60 chars each)
	headerBudget := len(classified) * 60
	if headerBudget > remaining/4 {
		headerBudget = remaining / 4
	}
	contentBudget := remaining - headerBudget

	for _, f := range classified {
		className := "unknown"
		extractMode := ExtractFull
		if f.Class != nil {
			className = f.Class.Name
			extractMode = f.Class.ExtractMode
			// Skip files with zero budget allocation
			if f.Class.MaxBudgetShare <= 0 {
				excluded = append(excluded, FileInfo{
					Path:     f.Path,
					Class:    className,
					Chars:    int(f.Size),
					Coverage: "skipped",
					Reason:   "class excluded",
				})
				filesSkipped++
				continue
			}
		}

		if contentBudget <= 100 {
			excluded = append(excluded, FileInfo{
				Path:     f.Path,
				Class:    className,
				Chars:    int(f.Size),
				Coverage: "skipped",
				Reason:   "budget exhausted",
			})
			filesSkipped++
			continue
		}

		content, err := os.ReadFile(f.AbsPath)
		if err != nil {
			continue
		}

		var text string
		coverage := "full"

		// Apply extraction mode
		if extractMode == ExtractMetadata {
			text = ExtractMetadataFromFile(content, f.Path)
			coverage = "metadata"
		} else {
			text = strings.TrimSpace(string(content))
		}

		if text == "" {
			continue
		}

		// Format header with class info
		header := fmt.Sprintf("\n--- File: %s (%s) ---\n", f.Path, className)

		// Check if we need to truncate
		if len(text) > contentBudget-len(header) {
			maxLen := contentBudget - len(header) - 50
			if maxLen > 0 {
				text = truncateAtBoundary(text, maxLen)
				text += "\n\n[... truncated ...]"
				coverage = "truncated"
				filesTrimmed++
			} else {
				excluded = append(excluded, FileInfo{
					Path:     f.Path,
					Class:    className,
					Chars:    len(text),
					Coverage: "skipped",
					Reason:   "insufficient budget",
				})
				filesSkipped++
				continue
			}
		}

		builder.WriteString(header)
		builder.WriteString(text)
		builder.WriteString("\n")

		filesUsed = append(filesUsed, f.Path)
		included = append(included, FileInfo{
			Path:     f.Path,
			Class:    className,
			Chars:    len(text),
			Coverage: coverage,
		})
		contentBudget -= len(header) + len(text) + 1
	}

	result := builder.String()
	return &GatherResult{
		Context:      strings.TrimSpace(result),
		FilesUsed:    filesUsed,
		TotalChars:   len(result),
		FilesTrimmed: filesTrimmed,
		FilesSkipped: filesSkipped,
		Included:     included,
		Excluded:     excluded,
	}, nil
}

// truncateAtBoundary truncates text at a paragraph or line boundary.
func truncateAtBoundary(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Try to find a paragraph break
	truncated := text[:maxLen]
	if idx := strings.LastIndex(truncated, "\n\n"); idx > maxLen/2 {
		return strings.TrimSpace(truncated[:idx])
	}

	// Fall back to line break
	if idx := strings.LastIndex(truncated, "\n"); idx > maxLen/2 {
		return strings.TrimSpace(truncated[:idx])
	}

	// Fall back to word break
	if idx := strings.LastIndex(truncated, " "); idx > maxLen/2 {
		return strings.TrimSpace(truncated[:idx])
	}

	return strings.TrimSpace(truncated)
}
