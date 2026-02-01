//go:build docprims

package context

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/3leaps/docprims/bindings/go/docprims"
)

// docprimsFormats lists extensions handled by docprims.
var docprimsFormats = map[string]bool{
	".docx": true,
	".xlsx": true,
	".pptx": true,
	".html": true,
	".htm":  true,
	".xml":  true,
	// Note: .md is handled by docprims but we prefer plain text for markdown
}

// ExtractContent extracts text content from a file.
// For Office documents and HTML, uses docprims; for plain text, returns as-is.
func ExtractContent(path string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if docprimsFormats[ext] {
		return extractWithDocprims(path, data)
	}

	// Plain text fallback
	return string(data), nil
}

// extractWithDocprims uses the docprims library for document extraction.
func extractWithDocprims(path string, data []byte) (string, error) {
	// docprims uses URI to detect format from extension
	uri := fmt.Sprintf("mem://%s", filepath.Base(path))

	// Configure limits appropriate for context gathering
	opts := &docprims.Options{
		Limits: &docprims.Limits{
			MaxInputBytes:  50 * 1024 * 1024, // 50MB input
			MaxOutputBytes: 10 * 1024 * 1024, // 10MB output
			MaxBlocks:      5000,
		},
	}

	result, err := docprims.ExtractBytes(uri, data, opts)
	if err != nil {
		return "", fmt.Errorf("docprims extraction failed: %w", err)
	}

	return parseDocprimsText(result)
}

// docprimsResult represents the JSON output from docprims.
type docprimsResult struct {
	Document struct {
		Text    string `json:"text"`
		Quality struct {
			Status string `json:"status"`
			Reason string `json:"reason,omitempty"`
		} `json:"quality"`
	} `json:"document"`
}

// parseDocprimsText extracts the text field from docprims JSON output.
func parseDocprimsText(jsonData []byte) (string, error) {
	var result docprimsResult
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return "", fmt.Errorf("parsing docprims output: %w", err)
	}
	return result.Document.Text, nil
}

// IsDocprimsFormat checks if a file extension is handled by docprims.
func IsDocprimsFormat(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return docprimsFormats[ext]
}

// DocprimsVersion returns the version of the docprims library.
func DocprimsVersion() string {
	return docprims.Version()
}
