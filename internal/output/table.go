package output

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/namelens/namelens/internal/core"
)

// TableFormatter renders results as an ASCII table.
type TableFormatter struct{}

// FormatBatch renders a batch result as a table.
func (f *TableFormatter) FormatBatch(result *core.BatchResult) (string, error) {
	if result == nil {
		return "", nil
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Type", "Name", "Status", "Notes"})

	for _, r := range result.Results {
		if r == nil {
			continue
		}
		t.AppendRow(table.Row{
			string(r.CheckType),
			displayName(r),
			statusLabel(r),
			formatNotes(r),
		})
	}

	if result.AILink != nil || result.AILinkError != nil {
		rowType, name, status, notes, ok := expertRow(result)
		if ok {
			t.AppendRow(table.Row{rowType, name, status, notes})
		}
	}

	if result.Total > 0 || result.Unknown > 0 {
		summary := fmt.Sprintf("%d/%d available", result.Score, result.Total)
		if result.Unknown > 0 {
			summary += fmt.Sprintf(", %d unknown", result.Unknown)
		}
		t.AppendFooter(table.Row{
			"",
			"",
			summary,
			"",
		})
	}

	rendered := t.Render()
	rendered += renderAnalysisSections(analysisSections(result), false)
	return rendered, nil
}
