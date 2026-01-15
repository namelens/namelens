package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/core"
	corestore "github.com/namelens/namelens/internal/core/store"
	"github.com/namelens/namelens/internal/output"
)

// compareRow holds extracted metrics for a single name.
type compareRow struct {
	Name              string              `json:"name"`
	Length            int                 `json:"length"`
	Availability      compareAvailability `json:"availability"`
	AvailabilityError string              `json:"availability_error,omitempty"`
	RiskLevel         string              `json:"risk_level,omitempty"`
	Phonetics         *comparePhonetics   `json:"phonetics,omitempty"`
	Suitability       *compareSuitability `json:"suitability,omitempty"`
}

type compareAvailability struct {
	Score   int `json:"score"`
	Total   int `json:"total"`
	Unknown int `json:"unknown"`
}

type comparePhonetics struct {
	OverallScore     int `json:"overall_score"`
	TypeabilityScore int `json:"typeability_score,omitempty"`
	CLISuitability   int `json:"cli_suitability,omitempty"`
}

type compareSuitability struct {
	OverallScore int    `json:"overall_score"`
	Rating       string `json:"rating,omitempty"`
}

var compareCmd = &cobra.Command{
	Use:   "compare <name1> <name2> [<name>...]",
	Short: "Compare candidate names side-by-side",
	Long:  "Compare multiple candidate names across availability, phonetics, and suitability in a compact table format for screening.",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runCompare,
}

func init() {
	rootCmd.AddCommand(compareCmd)

	compareCmd.Flags().String("profile", "startup", "Availability profile to use")
	compareCmd.Flags().String("mode", "", "Analysis mode: 'quick' for availability only, omit for full analysis with phonetics/suitability")
	compareCmd.Flags().String("output-format", "table", "Output format: table, json, markdown")
	compareCmd.Flags().String("out", "", "Write output to a file (default stdout)")
	compareCmd.Flags().String("out-dir", "", "Write output to a directory")
	_ = compareCmd.Flags().MarkHidden("out-dir") // compare outputs single table, not per-name files
	compareCmd.Flags().Bool("no-cache", false, "Skip cache lookup")
}

func runCompare(cmd *cobra.Command, args []string) error {
	names := args
	if len(names) < 2 {
		return errors.New("at least 2 names are required for comparison")
	}
	if len(names) > 20 {
		return errors.New("compare supports at most 20 names")
	}

	profileName, err := cmd.Flags().GetString("profile")
	if err != nil {
		return err
	}
	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		return err
	}

	// Validate mode
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode != "" && normalizedMode != "quick" {
		return fmt.Errorf("unsupported mode: %s (use 'quick' or omit for full analysis)", mode)
	}
	quickMode := normalizedMode == "quick"

	noCache, err := cmd.Flags().GetBool("no-cache")
	if err != nil {
		return err
	}

	format, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}
	outPath, _, err := resolveOutputTargets(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	store, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck

	cfg := config.GetConfig()
	if cfg == nil {
		return errors.New("config not loaded")
	}

	profile, err := resolveProfile(ctx, store, profileName, nil, nil, nil)
	if err != nil {
		return err
	}

	orchestrator := buildOrchestrator(cfg, store, !noCache)

	rows := make([]compareRow, 0, len(names))

	for _, name := range names {
		row := compareRow{
			Name:   name,
			Length: len(name),
		}

		// Run availability checks
		results, err := orchestrator.Check(ctx, name, profile)
		if err != nil {
			row.AvailabilityError = "error"
		} else {
			row.Availability = summarizeAvailability(results)
			// Derive risk level from availability results (no AI call needed)
			row.RiskLevel = deriveRiskLevel(results)
		}

		if !quickMode && row.AvailabilityError == "" {
			// Run phonetics analysis
			phonetics := runComparePhonetics(ctx, cfg, store, name, !noCache)
			if phonetics != nil {
				row.Phonetics = phonetics
			}

			// Run suitability analysis
			suitability := runCompareSuitability(ctx, cfg, store, name, !noCache)
			if suitability != nil {
				row.Suitability = suitability
			}
		}

		rows = append(rows, row)
	}

	sink, err := openSink(outPath)
	if err != nil {
		return err
	}
	defer sink.close() //nolint:errcheck

	return renderCompare(sink.writer, rows, format, quickMode)
}

func summarizeAvailability(results []*core.CheckResult) compareAvailability {
	var score, total, unknown int
	for _, r := range results {
		if r == nil {
			continue
		}
		total++
		switch r.Available {
		case core.AvailabilityAvailable:
			score++
		case core.AvailabilityUnknown, core.AvailabilityError, core.AvailabilityRateLimited:
			unknown++
		}
	}
	return compareAvailability{Score: score, Total: total, Unknown: unknown}
}

// deriveRiskLevel calculates risk from availability results without AI calls.
// Risk levels:
//   - "high": .com is taken or multiple key assets unavailable
//   - "medium": some assets taken but .com available
//   - "low": all or most assets available
func deriveRiskLevel(results []*core.CheckResult) string {
	if len(results) == 0 {
		return "unknown"
	}

	var comTaken, anyTaken, sawResult bool
	for _, r := range results {
		if r == nil {
			continue
		}
		sawResult = true
		if r.Available == core.AvailabilityTaken {
			anyTaken = true
			// Check if .com specifically is taken
			if r.CheckType == core.CheckTypeDomain && strings.HasSuffix(r.Name, ".com") {
				comTaken = true
			}
		}
	}

	if !sawResult {
		return "unknown"
	}
	if comTaken {
		return "high"
	}
	if anyTaken {
		return "medium"
	}
	return "low"
}

func runComparePhonetics(ctx context.Context, cfg *config.Config, store *corestore.Store, name string, useCache bool) *comparePhonetics {
	vars := map[string]string{"name": name}
	raw, searchErr, _ := runReviewGenerate(ctx, cfg, store, "name-phonetics", name, "quick", "", vars, useCache)
	if searchErr != nil || len(raw) == 0 {
		return nil
	}

	return extractPhonetics(raw)
}

func runCompareSuitability(ctx context.Context, cfg *config.Config, store *corestore.Store, name string, useCache bool) *compareSuitability {
	vars := map[string]string{"name": name}
	raw, searchErr, _ := runReviewGenerate(ctx, cfg, store, "name-suitability", name, "quick", "", vars, useCache)
	if searchErr != nil || len(raw) == 0 {
		return nil
	}

	return extractSuitability(raw)
}

func extractPhonetics(raw json.RawMessage) *comparePhonetics {
	var data struct {
		OverallAssessment struct {
			CombinedScore    int `json:"combined_score"`
			TypeabilityScore int `json:"typeability_score"`
		} `json:"overall_assessment"`
		CLISuitability struct {
			Score int `json:"score"`
		} `json:"cli_suitability"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	if data.OverallAssessment.CombinedScore == 0 {
		return nil
	}
	return &comparePhonetics{
		OverallScore:     data.OverallAssessment.CombinedScore,
		TypeabilityScore: data.OverallAssessment.TypeabilityScore,
		CLISuitability:   data.CLISuitability.Score,
	}
}

func extractSuitability(raw json.RawMessage) *compareSuitability {
	var data struct {
		OverallSuitability struct {
			Score  int    `json:"score"`
			Rating string `json:"rating"`
		} `json:"overall_suitability"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	if data.OverallSuitability.Score == 0 && data.OverallSuitability.Rating == "" {
		return nil
	}
	return &compareSuitability{
		OverallScore: data.OverallSuitability.Score,
		Rating:       data.OverallSuitability.Rating,
	}
}

func renderCompare(w io.Writer, rows []compareRow, format output.Format, quickMode bool) error {
	switch format {
	case output.FormatJSON:
		payload, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(payload))
		return err
	case output.FormatMarkdown:
		return renderCompareMarkdown(w, rows, quickMode)
	default:
		return renderCompareTable(w, rows, quickMode)
	}
}

func renderCompareTable(w io.Writer, rows []compareRow, quickMode bool) error {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetStyle(table.StyleRounded)

	if quickMode {
		t.AppendHeader(table.Row{"Name", "Availability", "Length"})
		for _, row := range rows {
			t.AppendRow(table.Row{
				row.Name,
				formatAvailability(row),
				row.Length,
			})
		}
	} else {
		t.AppendHeader(table.Row{"Name", "Availability", "Risk", "Phonetics", "Suitability", "Length"})
		for _, row := range rows {
			t.AppendRow(table.Row{
				row.Name,
				formatAvailability(row),
				formatRisk(row),
				formatPhonetics(row),
				formatSuitability(row),
				row.Length,
			})
		}
	}

	t.Render()
	return nil
}

func renderCompareMarkdown(w io.Writer, rows []compareRow, quickMode bool) error {
	if quickMode {
		_, _ = fmt.Fprintln(w, "| Name | Availability | Length |")
		_, _ = fmt.Fprintln(w, "|------|--------------|--------|")
		for _, row := range rows {
			_, _ = fmt.Fprintf(w, "| %s | %s | %d |\n",
				row.Name, formatAvailability(row), row.Length)
		}
		return nil
	}

	_, _ = fmt.Fprintln(w, "| Name | Availability | Risk | Phonetics | Suitability | Length |")
	_, _ = fmt.Fprintln(w, "|------|--------------|------|-----------|-------------|--------|")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %d |\n",
			row.Name,
			formatAvailability(row),
			formatRisk(row),
			formatPhonetics(row),
			formatSuitability(row),
			row.Length)
	}
	return nil
}

// formatAvailability returns the availability display string.
// Shows "error" if availability check failed, otherwise "X/Y" with optional unknown count.
func formatAvailability(row compareRow) string {
	if row.AvailabilityError != "" {
		return row.AvailabilityError
	}
	avail := fmt.Sprintf("%d/%d", row.Availability.Score, row.Availability.Total)
	if row.Availability.Unknown > 0 {
		avail += fmt.Sprintf(" (%d?)", row.Availability.Unknown)
	}
	return avail
}

func formatRisk(row compareRow) string {
	if row.AvailabilityError != "" {
		return "-"
	}
	if row.RiskLevel == "" {
		return "-"
	}
	return row.RiskLevel
}

func formatPhonetics(row compareRow) string {
	if row.Phonetics == nil || row.Phonetics.OverallScore == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", row.Phonetics.OverallScore)
}

func formatSuitability(row compareRow) string {
	if row.Suitability == nil || row.Suitability.OverallScore == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", row.Suitability.OverallScore)
}
