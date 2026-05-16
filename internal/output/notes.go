package output

import (
	"fmt"
	"strings"

	"github.com/namelens/namelens/internal/core"
)

// expertRow surfaces the expert digest for a batch as a (rowType, name, status,
// notes, ok) tuple consumed by the markdown and table renderers.
//
// In v0.2.5 the synthesis moved into core.NewExpertDigest so JSON output gains
// the same digest the markdown row shows. This function now reads the
// pre-populated digest and falls back to synthesizing one in-place for any
// caller that constructs a BatchResult without going through the standard
// producer in internal/cmd/check.go (notably hand-built BatchResults in tests).
func expertRow(result *core.BatchResult) (string, string, string, string, bool) {
	if result == nil {
		return "", "", "", "", false
	}
	digest := result.Expert
	if digest == nil {
		digest = core.NewExpertDigest(result.Name, result.Results, result.AILink, result.AILinkError)
	}
	if digest == nil {
		return "", "", "", "", false
	}
	return "expert", digest.Name, digest.Status, digest.Notes, true
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
