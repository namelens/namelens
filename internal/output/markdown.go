package output

import (
	"fmt"
	"strings"

	"github.com/namelens/namelens/internal/core"
)

// MarkdownFormatter renders results as a markdown table.
type MarkdownFormatter struct{}

// FormatBatch renders a batch result as Markdown.
func (f *MarkdownFormatter) FormatBatch(result *core.BatchResult) (string, error) {
	if result == nil {
		return "", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s availability\n\n", escapeMarkdownCell(result.Name))
	sb.WriteString("| Type | Name | Status | Notes |\n")
	sb.WriteString("|------|------|--------|-------|\n")

	for _, r := range result.Results {
		if r == nil {
			continue
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n",
			escapeMarkdownCell(string(r.CheckType)),
			escapeMarkdownCell(displayName(r)),
			escapeMarkdownCell(statusLabel(r)),
			escapeMarkdownCell(formatNotes(r)),
		)
	}

	if result.AILink != nil || result.AILinkError != nil {
		rowType, name, status, notes, ok := expertRow(result)
		if ok {
			fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n",
				escapeMarkdownCell(rowType),
				escapeMarkdownCell(name),
				escapeMarkdownCell(status),
				escapeMarkdownCell(notes),
			)
		}
	}

	if result.Total > 0 || result.Unknown > 0 {
		summary := fmt.Sprintf("%d/%d available", result.Score, result.Total)
		if result.Unknown > 0 {
			summary += fmt.Sprintf(", %d unknown", result.Unknown)
		}
		fmt.Fprintf(&sb, "\n**Score**: %s\n", summary)
	}

	sb.WriteString(renderAnalysisSections(analysisSections(result), true))
	return sb.String(), nil
}

func escapeMarkdownCell(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
