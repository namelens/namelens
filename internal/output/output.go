package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/namelens/namelens/internal/core"
)

// Format represents an output format.
type Format string

const (
	FormatTable    Format = "table"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

// Formatter renders batch results.
type Formatter interface {
	FormatBatch(result *core.BatchResult) (string, error)
}

// ParseFormat validates and normalizes a format string.
func ParseFormat(value string) (Format, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", string(FormatTable):
		return FormatTable, nil
	case string(FormatJSON):
		return FormatJSON, nil
	case string(FormatMarkdown):
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf("unsupported output format: %s", value)
	}
}

// NewFormatter returns a formatter for the requested format.
func NewFormatter(format Format) Formatter {
	switch format {
	case FormatJSON:
		return &JSONFormatter{Indent: true}
	case FormatMarkdown:
		return &MarkdownFormatter{}
	default:
		return &TableFormatter{}
	}
}

// FormatBatchList renders multiple batch results using the requested format.
func FormatBatchList(format Format, results []*core.BatchResult) (string, error) {
	if format == FormatJSON {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	formatter := NewFormatter(format)
	rendered := make([]string, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue
		}
		value, err := formatter.FormatBatch(result)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		rendered = append(rendered, value)
	}

	return strings.Join(rendered, "\n\n"), nil
}
