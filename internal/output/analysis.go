package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/namelens/namelens/internal/core"
)

type analysisSection struct {
	Title string
	Lines []string
}

type phoneticsSummary struct {
	Syllables struct {
		Count     int    `json:"count"`
		Breakdown string `json:"breakdown"`
	} `json:"syllables"`
	Pronunciation struct {
		IPAPrimary string `json:"ipa_primary"`
	} `json:"pronunciation"`
	Typeability struct {
		OverallScore int `json:"overall_score"`
	} `json:"typeability"`
	CLISuitability struct {
		Score int `json:"score"`
	} `json:"cli_suitability"`
	OverallAssessment struct {
		Recommendation string `json:"recommendation"`
	} `json:"overall_assessment"`
}

type suitabilitySummary struct {
	OverallSuitability struct {
		Score   int    `json:"score"`
		Rating  string `json:"rating"`
		Summary string `json:"summary"`
	} `json:"overall_suitability"`
	RiskAssessment map[string]riskLevel `json:"risk_assessment"`
}

type riskLevel struct {
	Level string `json:"level"`
}

func analysisSections(result *core.BatchResult) []analysisSection {
	if result == nil {
		return nil
	}

	sections := make([]analysisSection, 0, 2)
	if section, ok := phoneticsSection(result); ok {
		sections = append(sections, section)
	}
	if section, ok := suitabilitySection(result); ok {
		sections = append(sections, section)
	}
	return sections
}

func phoneticsSection(result *core.BatchResult) (analysisSection, bool) {
	if result == nil {
		return analysisSection{}, false
	}
	if result.PhoneticsError != nil {
		message := strings.TrimSpace(result.PhoneticsError.Message)
		if message == "" {
			message = strings.TrimSpace(result.PhoneticsError.Details)
		}
		if message == "" {
			message = "analysis failed"
		}
		return analysisSection{
			Title: "Phonetics Analysis",
			Lines: []string{fmt.Sprintf("error: %s", message)},
		}, true
	}
	if len(result.Phonetics) == 0 {
		return analysisSection{}, false
	}

	var summary phoneticsSummary
	if err := json.Unmarshal(result.Phonetics, &summary); err != nil {
		return analysisSection{
			Title: "Phonetics Analysis",
			Lines: []string{"summary unavailable"},
		}, true
	}

	lines := make([]string, 0, 5)
	if summary.Syllables.Count > 0 {
		line := fmt.Sprintf("Syllables: %d", summary.Syllables.Count)
		if strings.TrimSpace(summary.Syllables.Breakdown) != "" {
			line += fmt.Sprintf(" (%s)", summary.Syllables.Breakdown)
		}
		lines = append(lines, line)
	}
	if strings.TrimSpace(summary.Pronunciation.IPAPrimary) != "" {
		lines = append(lines, fmt.Sprintf("Pronunciation: %s", summary.Pronunciation.IPAPrimary))
	}
	if summary.Typeability.OverallScore > 0 {
		lines = append(lines, fmt.Sprintf("Typeability: %d/100", summary.Typeability.OverallScore))
	}
	if summary.CLISuitability.Score > 0 {
		lines = append(lines, fmt.Sprintf("CLI suitability: %d/100", summary.CLISuitability.Score))
	}
	if strings.TrimSpace(summary.OverallAssessment.Recommendation) != "" {
		lines = append(lines, fmt.Sprintf("Notes: %s", summary.OverallAssessment.Recommendation))
	}
	if len(lines) == 0 {
		lines = append(lines, "analysis complete")
	}

	return analysisSection{Title: "Phonetics Analysis", Lines: lines}, true
}

func suitabilitySection(result *core.BatchResult) (analysisSection, bool) {
	if result == nil {
		return analysisSection{}, false
	}
	if result.SuitabilityError != nil {
		message := strings.TrimSpace(result.SuitabilityError.Message)
		if message == "" {
			message = strings.TrimSpace(result.SuitabilityError.Details)
		}
		if message == "" {
			message = "analysis failed"
		}
		return analysisSection{
			Title: "Suitability Analysis",
			Lines: []string{fmt.Sprintf("error: %s", message)},
		}, true
	}
	if len(result.Suitability) == 0 {
		return analysisSection{}, false
	}

	var summary suitabilitySummary
	if err := json.Unmarshal(result.Suitability, &summary); err != nil {
		return analysisSection{
			Title: "Suitability Analysis",
			Lines: []string{"summary unavailable"},
		}, true
	}

	lines := make([]string, 0, 4)
	if summary.OverallSuitability.Score > 0 || summary.OverallSuitability.Rating != "" {
		line := "Overall: "
		if summary.OverallSuitability.Score > 0 {
			line += fmt.Sprintf("%d/100", summary.OverallSuitability.Score)
		} else {
			line += "score unavailable"
		}
		if summary.OverallSuitability.Rating != "" {
			line += fmt.Sprintf(" (%s)", summary.OverallSuitability.Rating)
		}
		lines = append(lines, line)
	}

	riskLine := riskSummary(summary.RiskAssessment)
	if riskLine != "" {
		lines = append(lines, riskLine)
	}

	if strings.TrimSpace(summary.OverallSuitability.Summary) != "" {
		lines = append(lines, fmt.Sprintf("Notes: %s", summary.OverallSuitability.Summary))
	}
	if len(lines) == 0 {
		lines = append(lines, "analysis complete")
	}

	return analysisSection{Title: "Suitability Analysis", Lines: lines}, true
}

func riskSummary(levels map[string]riskLevel) string {
	if len(levels) == 0 {
		return ""
	}

	keys := make([]string, 0, len(levels))
	for key := range levels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	risks := make([]string, 0, len(keys))
	for _, key := range keys {
		level := strings.TrimSpace(levels[key].Level)
		if level == "" || strings.EqualFold(level, "clear") {
			continue
		}
		risks = append(risks, fmt.Sprintf("%s=%s", key, level))
	}
	if len(risks) == 0 {
		return "Risks: None identified"
	}
	return "Risks: " + strings.Join(risks, ", ")
}

func renderAnalysisSections(sections []analysisSection, markdown bool) string {
	if len(sections) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, section := range sections {
		if i > 0 {
			sb.WriteString("\n")
		}
		if markdown {
			sb.WriteString(fmt.Sprintf("\n\n### %s\n", section.Title))
			for _, line := range section.Lines {
				sb.WriteString(fmt.Sprintf("- %s\n", line))
			}
		} else {
			sb.WriteString(fmt.Sprintf("\n\n%s:\n", section.Title))
			for _, line := range section.Lines {
				sb.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}
	}
	return sb.String()
}
