package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/output"
)

type outputSink struct {
	writer io.Writer
	close  func() error
	path   string
}

func outputExtension(format output.Format) string {
	switch format {
	case output.FormatJSON:
		return "json"
	case output.FormatMarkdown:
		return "md"
	default:
		return "txt"
	}
}

var nonFilename = regexp.MustCompile(`[^a-z0-9._-]+`)

func sanitizeFilename(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	clean = nonFilename.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-.")
	if clean == "" {
		return "output"
	}
	return clean
}

func resolveOutputFormat(cmd *cobra.Command) (output.Format, error) {
	value, err := cmd.Flags().GetString("output-format")
	if err != nil {
		return "", err
	}
	return output.ParseFormat(value)
}

func resolveOutputTargets(cmd *cobra.Command) (outPath string, outDir string, err error) {
	outPath, err = cmd.Flags().GetString("out")
	if err != nil {
		return "", "", err
	}
	outDir, err = cmd.Flags().GetString("out-dir")
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(outPath) != "" && strings.TrimSpace(outDir) != "" {
		return "", "", fmt.Errorf("--out and --out-dir are mutually exclusive")
	}
	return strings.TrimSpace(outPath), strings.TrimSpace(outDir), nil
}

func openSink(path string) (*outputSink, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "-" {
		return &outputSink{writer: os.Stdout, close: func() error { return nil }, path: "-"}, nil
	}

	if err := os.MkdirAll(filepath.Dir(trimmed), 0755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.Create(trimmed)
	if err != nil {
		return nil, err
	}
	return &outputSink{writer: file, close: file.Close, path: trimmed}, nil
}

func ensureOutDir(dir string) (string, error) {
	clean := strings.TrimSpace(dir)
	if clean == "" {
		return "", nil
	}
	if err := os.MkdirAll(clean, 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return clean, nil
	}
	return abs, nil
}
