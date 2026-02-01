//go:build !docprims

package context

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractContentPlainText(t *testing.T) {
	content, err := ExtractContent("test.txt", []byte("hello world"))
	require.NoError(t, err)
	require.Equal(t, "hello world", content)
}

func TestExtractContentMarkdown(t *testing.T) {
	content, err := ExtractContent("readme.md", []byte("# Hello\n\nWorld"))
	require.NoError(t, err)
	require.Equal(t, "# Hello\n\nWorld", content)
}

func TestIsDocprimsFormat(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"document.docx", true},
		{"Document.DOCX", true},
		{"spreadsheet.xlsx", true},
		{"slides.pptx", true},
		{"page.html", true},
		{"readme.md", false},
		{"code.go", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := IsDocprimsFormat(tc.path)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDocprimsVersionEmpty(t *testing.T) {
	// Without docprims build tag, version is empty
	version := DocprimsVersion()
	require.Empty(t, version)
}

func TestExtractContentDocxUnsupported(t *testing.T) {
	// Without docprims, Office formats return empty (unsupported)
	content, err := ExtractContent("test.docx", []byte("data"))
	require.NoError(t, err)
	require.Empty(t, content, "without docprims, Office formats should return empty")
}
