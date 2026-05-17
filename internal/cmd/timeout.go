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
