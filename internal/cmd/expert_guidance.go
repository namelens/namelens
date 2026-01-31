package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/namelens/namelens/internal/ailink"
)

// expertGuidanceShown tracks if the AI configuration warning has been shown
// this session to avoid repeating it.
var expertGuidanceShown bool

// isAIBackendConfigured checks if any AI provider has a valid API key configured.
func isAIBackendConfigured(cfg ailink.Config) bool {
	for _, provider := range cfg.Providers {
		if !provider.Enabled {
			continue
		}
		for _, cred := range provider.Credentials {
			if cred.Enabled && strings.TrimSpace(cred.APIKey) != "" {
				return true
			}
		}
	}
	return false
}

// showExpertGuidanceWarning prints a warning about limited analysis mode
// when no AI backend is configured. Shows once per session.
// Writes to stderr to avoid interfering with JSON/structured output.
func showExpertGuidanceWarning(cfg ailink.Config, w io.Writer) {
	if expertGuidanceShown {
		return
	}
	if isAIBackendConfigured(cfg) {
		return
	}

	if w == nil {
		w = os.Stderr
	}

	// Informational output to stderr - errors are best-effort
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Note: Running in limited analysis mode (no AI backend configured).")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "  Domain and registry checks show availability only, not commercial safety.")
	_, _ = fmt.Fprintln(w, "  Names may have trademark conflicts, active use, or brand confusion risks")
	_, _ = fmt.Fprintln(w, "  not detected by basic availability checks.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "  To enable comprehensive analysis, configure an AI backend:")
	_, _ = fmt.Fprintln(w, "    namelens config set ailink.providers.myai.ai_provider openai")
	_, _ = fmt.Fprintln(w, "    namelens config set ailink.providers.myai.credentials[0].api_key YOUR_KEY")
	_, _ = fmt.Fprintln(w, "    namelens config set ailink.default_provider myai")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "  Then use: namelens check <name> --expert")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "  See: https://namelens.dev/docs/configuration")
	_, _ = fmt.Fprintln(w, "")

	expertGuidanceShown = true
}

// showExpertTip prints a tip about using --expert flag after check results.
// Only shows when AI is configured but --expert was not used.
// Writes to stderr to avoid interfering with JSON/structured output.
func showExpertTip(cfg ailink.Config, expertUsed bool, w io.Writer) {
	if expertUsed {
		return
	}
	if !isAIBackendConfigured(cfg) {
		// Already showed the full warning, don't add more noise
		return
	}

	if w == nil {
		w = os.Stderr
	}

	// Informational output to stderr - errors are best-effort
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Tip: These results show availability only. For trademark, commercial use,")
	_, _ = fmt.Fprintln(w, "     and brand safety analysis, run with --expert flag.")
	_, _ = fmt.Fprintln(w, "")
}

// resetExpertGuidance resets the shown flag (for testing).
func resetExpertGuidance() {
	expertGuidanceShown = false
}
