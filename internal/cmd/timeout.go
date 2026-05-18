package cmd

import "time"

// timeoutFlagToSec converts a `--timeout` cobra Duration flag value into the
// integer seconds expected by ailink request structs (TimeoutSec).
//
// Returns 0 when the flag is unset or non-positive, which is the sentinel
// ailink.Service.{Search,Generate,SearchBulk} use to fall back to
// ailink.default_timeout from config. A positive sub-second value rounds up
// to 1s so that a user-supplied "500ms" is honored as 1s rather than dropping
// to the config default (the rounding choice favors user intent over precision
// at the sub-second floor, which is below the practical LLM-call regime).
//
// Added in v0.2.5 for the `--timeout` flag on `check` and `generate`. See
// v0.3.x expert-call-reliability brief for the durable arc that will replace
// this single-knob timeout with a per-depth policy and streaming/retry logic.
func timeoutFlagToSec(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	sec := int(d.Seconds())
	if sec == 0 {
		return 1
	}
	return sec
}

// markImageTimeoutFallback is the last-resort per-image-call deadline for
// `mark` when neither the --timeout flag nor ailink.default_timeout has a
// usable value. Generous enough for real image generation latency on both
// xAI and OpenAI image APIs; tight enough that a stuck call doesn't hang
// the CLI for the registry's 5m maxTimeout ceiling.
const markImageTimeoutFallback = 120 * time.Second

// markImageTimeout resolves the per-image-call timeout for `mark`. Bypasses
// Service.Search/Generate (which performs this resolution internally for
// text calls), so we replicate the precedence here: explicit flag wins,
// then config default, then a generous package fallback. Added in v0.2.5
// for the `--timeout` flag on `mark`; see v0.3.x expert-call-reliability
// brief § 4b for the durable arc that moves this resolution into a single
// shared policy across all driver consumers.
func markImageTimeout(flag time.Duration, cfgDefault time.Duration) time.Duration {
	if flag > 0 {
		return flag
	}
	if cfgDefault > 0 {
		return cfgDefault
	}
	return markImageTimeoutFallback
}
