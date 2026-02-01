//go:build docprims

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
	// Markdown is treated as plain text by ExtractContent (not routed to docprims)
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
		{"page.htm", true},
		{"data.xml", true},
		{"readme.md", false},
		{"code.go", false},
		{"config.json", false},
		{"noextension", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := IsDocprimsFormat(tc.path)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDocprimsVersion(t *testing.T) {
	version := DocprimsVersion()
	require.NotEmpty(t, version, "docprims version should not be empty")
	// Version should be semver-like
	require.Contains(t, version, ".", "version should contain dots")
}

func TestExtractContentDocx(t *testing.T) {
	// Minimal valid DOCX is complex (it's a ZIP archive with XML files)
	// This test verifies the error path when given invalid data
	_, err := ExtractContent("test.docx", []byte("not a valid docx"))
	require.Error(t, err, "invalid docx should return error")
}

func TestExtractContentHTML(t *testing.T) {
	html := []byte("<html><body><h1>Title</h1><p>Content here.</p></body></html>")
	content, err := ExtractContent("page.html", html)
	require.NoError(t, err)
	require.Contains(t, content, "Title")
	require.Contains(t, content, "Content here")
}
