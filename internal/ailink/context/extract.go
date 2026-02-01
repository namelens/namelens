//go:build !docprims

package context

import (
	"path/filepath"
	"strings"
)

// docprimsFormats lists extensions that would be handled by docprims if enabled.
// Without docprims, these formats are not supported for text extraction.
var docprimsFormats = map[string]bool{
	".docx": true,
	".xlsx": true,
	".pptx": true,
	".html": true,
	".htm":  true,
	".xml":  true,
}

// ExtractContent extracts text content from a file.
// Without docprims build tag, Office/HTML documents return an error.
func ExtractContent(path string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if docprimsFormats[ext] {
		// Without docprims, we can't extract these formats
		// Return empty string to signal unsupported (caller will skip)
		return "", nil
	}

	// Plain text fallback
	return string(data), nil
}

// IsDocprimsFormat checks if a file extension would be handled by docprims.
func IsDocprimsFormat(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return docprimsFormats[ext]
}

// DocprimsVersion returns empty string when docprims is not enabled.
func DocprimsVersion() string {
	return ""
}
