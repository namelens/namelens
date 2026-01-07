package output

import (
	"fmt"
	"strings"

	"github.com/namelens/namelens/internal/core"
)

func expertRow(result *core.BatchResult) (string, string, string, string, bool) {
	if result == nil {
		return "", "", "", "", false
	}
	if result.AILinkError != nil {
		notes := result.AILinkError.Message
		if strings.TrimSpace(notes) == "" {
			notes = result.AILinkError.Details
		}
		return "expert", "ailink", "error", notes, true
	}
	if result.AILink == nil {
		return "", "", "", "", false
	}

	status := "risk: unknown"
	if level := strings.TrimSpace(result.AILink.RiskLevel); level != "" {
		status = "risk: " + level
	}

	notes := strings.TrimSpace(result.AILink.Summary)
	if notes == "" {
		if len(result.AILink.Raw) > 0 {
			notes = "expert analysis complete (see raw JSON in --output=json)"
		} else {
			notes = "expert analysis complete"
		}
	}

	return "expert", "ailink", status, notes, true
}

func displayName(result *core.CheckResult) string {
	if result == nil {
		return ""
	}

	name := strings.TrimSpace(result.Name)
	switch result.CheckType {
	case core.CheckTypeGitHub:
		if name == "" {
			return ""
		}
		return "@" + name
	case core.CheckTypeDomain:
		if name != "" {
			return name
		}
		if result.TLD != "" {
			return "." + result.TLD
		}
		return ""
	default:
		if name != "" {
			return name
		}
		return string(result.CheckType)
	}
}

func statusLabel(result *core.CheckResult) string {
	if result == nil {
		return "unknown"
	}

	switch result.Available {
	case core.AvailabilityAvailable:
		return "available"
	case core.AvailabilityTaken:
		return "taken"
	case core.AvailabilityRateLimited:
		return "rate limited"
	case core.AvailabilityUnsupported:
		return "unsupported"
	case core.AvailabilityError:
		return "error"
	default:
		return "unknown"
	}
}

func formatNotes(result *core.CheckResult) string {
	if result == nil {
		return ""
	}

	parts := []string{}
	if result.Message != "" && result.Available == core.AvailabilityError {
		parts = append(parts, result.Message)
	}
	if result.Available == core.AvailabilityRateLimited && result.ExtraData != nil {
		if retry, ok := result.ExtraData["retry_after"]; ok {
			parts = append(parts, fmt.Sprintf("retry: %v", retry))
		}
	}

	switch result.CheckType {
	case core.CheckTypeDomain:
		parts = append(parts, domainNotes(result)...)
	case core.CheckTypeNPM:
		parts = append(parts, npmNotes(result)...)
	case core.CheckTypePyPI:
		parts = append(parts, pypiNotes(result)...)
	case core.CheckTypeGitHub:
		parts = append(parts, githubNotes(result)...)
	}

	return strings.Join(parts, "; ")
}

func domainNotes(result *core.CheckResult) []string {
	if result == nil || result.ExtraData == nil {
		return nil
	}
	notes := []string{}
	if source, ok := result.ExtraData["resolution_source"]; ok {
		if value, ok := source.(string); ok && value != "" && value != "rdap" {
			notes = append(notes, fmt.Sprintf("source: %s", value))
		}
	}
	if expiration, ok := result.ExtraData["expiration"]; ok {
		notes = append(notes, fmt.Sprintf("exp: %v", expiration))
	}
	if registrar, ok := result.ExtraData["registrar"]; ok {
		notes = append(notes, fmt.Sprintf("registrar: %v", registrar))
	}
	return notes
}

func npmNotes(result *core.CheckResult) []string {
	if result == nil || result.ExtraData == nil {
		return nil
	}
	if latest, ok := result.ExtraData["latest_version"]; ok {
		return []string{fmt.Sprintf("latest: %v", latest)}
	}
	return nil
}

func pypiNotes(result *core.CheckResult) []string {
	if result == nil || result.ExtraData == nil {
		return nil
	}
	notes := []string{}
	if version, ok := result.ExtraData["version"]; ok {
		notes = append(notes, fmt.Sprintf("version: %v", version))
	}
	if summary, ok := result.ExtraData["summary"]; ok {
		notes = append(notes, fmt.Sprintf("summary: %v", summary))
	}
	return notes
}

func githubNotes(result *core.CheckResult) []string {
	if result == nil || result.ExtraData == nil {
		return nil
	}
	if url, ok := result.ExtraData["html_url"]; ok {
		return []string{fmt.Sprintf("url: %v", url)}
	}
	return nil
}
