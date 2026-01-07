package checker

import (
	"net/http"
	"time"
)

func retryAfterHeader(resp *http.Response) (time.Duration, map[string]any) {
	if resp == nil || resp.Header == nil {
		return 0, nil
	}

	retry := resp.Header.Get("Retry-After")
	if retry == "" {
		return 0, nil
	}

	if seconds, err := time.ParseDuration(retry + "s"); err == nil {
		return seconds, map[string]any{"retry_after": retry}
	}
	if parsed, err := http.ParseTime(retry); err == nil {
		return time.Until(parsed), map[string]any{"retry_after": retry}
	}

	return 0, map[string]any{"retry_after": retry}
}
