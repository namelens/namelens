package context

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Corpus represents a structured context corpus for AI prompts.
type Corpus struct {
	Schema      string         `json:"$schema,omitempty"`
	Version     string         `json:"version"`
	GeneratedAt time.Time      `json:"generated_at"`
	Source      CorpusSource   `json:"source"`
	Budget      CorpusBudget   `json:"budget"`
	Manifest    CorpusManifest `json:"manifest"`
	Files       []FileInfo     `json:"files"`
	Excluded    []FileInfo     `json:"excluded,omitempty"`
	Content     []FileContent  `json:"content"`
}

// CorpusSource describes where the corpus was generated from.
type CorpusSource struct {
	Type string `json:"type"` // "directory", "files", "url"
	Path string `json:"path"`
}

// CorpusBudget describes the character budget usage.
type CorpusBudget struct {
	MaxChars  int `json:"max_chars"`
	UsedChars int `json:"used_chars"`
}

// CorpusManifest summarizes what was included/excluded.
type CorpusManifest struct {
	TotalFilesScanned int `json:"total_files_scanned"`
	FilesIncluded     int `json:"files_included"`
	FilesExcluded     int `json:"files_excluded"`
	FilesTruncated    int `json:"files_truncated"`
}

// FileContent holds the extracted text from a file.
type FileContent struct {
	File string `json:"file"`
	Text string `json:"text"`
}

// CorpusFromGatherResult converts a GatherResult to a Corpus.
func CorpusFromGatherResult(result *GatherResult, dir string, maxChars int) *Corpus {
	if result == nil {
		return &Corpus{Version: "1.0.0"}
	}

	// Extract content from the Context string
	// The Context is formatted as "--- File: path (class) ---\ncontent\n"
	content := parseContentFromContext(result.Context)

	return &Corpus{
		Schema:      "https://schemas.namelens.dev/context/v1.0.0.schema.json",
		Version:     "1.0.0",
		GeneratedAt: time.Now().UTC(),
		Source: CorpusSource{
			Type: "directory",
			Path: dir,
		},
		Budget: CorpusBudget{
			MaxChars:  maxChars,
			UsedChars: result.TotalChars,
		},
		Manifest: CorpusManifest{
			TotalFilesScanned: len(result.Included) + len(result.Excluded),
			FilesIncluded:     len(result.Included),
			FilesExcluded:     len(result.Excluded),
			FilesTruncated:    result.FilesTrimmed,
		},
		Files:    result.Included,
		Excluded: result.Excluded,
		Content:  content,
	}
}

// parseContentFromContext extracts file contents from the formatted context string.
func parseContentFromContext(ctx string) []FileContent {
	var content []FileContent
	if ctx == "" {
		return content
	}

	// Normalize: ensure context starts with newline for consistent splitting
	if strings.HasPrefix(ctx, "--- File: ") {
		ctx = "\n" + ctx
	}

	// Split by file headers
	parts := strings.Split(ctx, "\n--- File: ")
	for _, part := range parts {
		if part == "" {
			continue
		}

		// Find the header end
		headerEnd := strings.Index(part, " ---\n")
		if headerEnd == -1 {
			continue
		}

		// Extract filename (remove class info if present)
		header := part[:headerEnd]
		if idx := strings.Index(header, " ("); idx > 0 {
			header = header[:idx]
		}

		// Extract content
		text := strings.TrimSpace(part[headerEnd+5:])
		if text != "" {
			content = append(content, FileContent{
				File: header,
				Text: text,
			})
		}
	}

	return content
}

// ToJSON serializes the corpus to JSON.
func (c *Corpus) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// ToMarkdown formats the corpus as human-readable markdown.
func (c *Corpus) ToMarkdown() string {
	var b strings.Builder

	b.WriteString("# Context Corpus\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", c.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Source: %s\n", c.Source.Path))
	b.WriteString(fmt.Sprintf("Budget: %d/%d chars\n\n", c.Budget.UsedChars, c.Budget.MaxChars))

	// Manifest table
	b.WriteString("## Manifest\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Files scanned | %d |\n", c.Manifest.TotalFilesScanned))
	b.WriteString(fmt.Sprintf("| Files included | %d |\n", c.Manifest.FilesIncluded))
	b.WriteString(fmt.Sprintf("| Files excluded | %d |\n", c.Manifest.FilesExcluded))
	b.WriteString(fmt.Sprintf("| Files truncated | %d |\n\n", c.Manifest.FilesTruncated))

	// Included files table
	if len(c.Files) > 0 {
		b.WriteString("## Included Files\n\n")
		b.WriteString("| File | Class | Coverage | Chars |\n")
		b.WriteString("|------|-------|----------|-------|\n")
		for _, f := range c.Files {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n", f.Path, f.Class, f.Coverage, f.Chars))
		}
		b.WriteString("\n")
	}

	// Excluded files table
	if len(c.Excluded) > 0 {
		b.WriteString("## Excluded Files\n\n")
		b.WriteString("| File | Class | Reason |\n")
		b.WriteString("|------|-------|--------|\n")
		for _, f := range c.Excluded {
			b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", f.Path, f.Class, f.Reason))
		}
		b.WriteString("\n")
	}

	// Content
	b.WriteString("---\n\n## Content\n\n")
	for _, fc := range c.Content {
		// Find class for this file
		class := "unknown"
		for _, f := range c.Files {
			if f.Path == fc.File {
				class = f.Class
				break
			}
		}
		b.WriteString(fmt.Sprintf("### %s (%s)\n\n", fc.File, class))
		b.WriteString(fc.Text)
		b.WriteString("\n\n---\n\n")
	}

	return b.String()
}

// ToPromptContext formats the corpus for inclusion in an AI prompt.
// This is the format used when passing corpus to generate command.
func (c *Corpus) ToPromptContext() string {
	var b strings.Builder

	// Add manifest header so the model knows what's available
	b.WriteString("CONTEXT MANIFEST:\n")
	b.WriteString(fmt.Sprintf("- Source: %s\n", c.Source.Path))
	b.WriteString(fmt.Sprintf("- Included %d files (%d chars) from %d total\n",
		c.Manifest.FilesIncluded, c.Budget.UsedChars, c.Manifest.TotalFilesScanned))

	if c.Manifest.FilesTruncated > 0 {
		b.WriteString(fmt.Sprintf("- %d files were truncated\n", c.Manifest.FilesTruncated))
	}

	if len(c.Excluded) > 0 {
		b.WriteString(fmt.Sprintf("- %d files excluded (available on request)\n", len(c.Excluded)))
	}

	b.WriteString("\n--- INCLUDED CONTENT ---\n\n")

	for _, fc := range c.Content {
		// Find class for this file
		class := "unknown"
		coverage := "full"
		for _, f := range c.Files {
			if f.Path == fc.File {
				class = f.Class
				coverage = f.Coverage
				break
			}
		}
		b.WriteString(fmt.Sprintf("--- File: %s (%s, %s) ---\n", fc.File, class, coverage))
		b.WriteString(fc.Text)
		b.WriteString("\n\n")
	}

	return strings.TrimSpace(b.String())
}

// ParseCorpusJSON parses a JSON corpus.
func ParseCorpusJSON(data []byte) (*Corpus, error) {
	var c Corpus
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse corpus JSON: %w", err)
	}
	return &c, nil
}
