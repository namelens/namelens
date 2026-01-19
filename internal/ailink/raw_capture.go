package ailink

import (
	"encoding/json"
	"strings"
)

func truncateBytes(input []byte, max int) []byte {
	if max <= 0 {
		return nil
	}
	if len(input) <= max {
		return input
	}
	out := make([]byte, 0, max)
	out = append(out, input[:max]...)
	return out
}

func truncateJSONRaw(input json.RawMessage, max int) json.RawMessage {
	if max <= 0 {
		return nil
	}
	if len(input) <= max {
		return input
	}
	out := make(json.RawMessage, 0, max)
	out = append(out, input[:max]...)
	return out
}

func isRawCaptureEnabled(cfg Config, includeRaw bool) bool {
	if !includeRaw {
		return false
	}
	return cfg.Debug.CaptureRawEnabled
}

func rawLimit(cfg Config) int {
	if cfg.Debug.CaptureRawMaxBytes <= 0 {
		return 0
	}
	return cfg.Debug.CaptureRawMaxBytes
}

func safeOneLine(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}
