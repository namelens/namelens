package ailink

import (
	"context"
	"errors"
	"strings"

	"github.com/namelens/namelens/internal/ailink/driver"
)

func mapProviderError(err error) *SearchError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &SearchError{Code: "AILINK_PROVIDER_TIMEOUT", Message: "provider request timed out"}
	}

	var perr *driver.ProviderError
	if errors.As(err, &perr) && perr != nil {
		status := perr.StatusCode
		details := strings.TrimSpace(perr.Message)
		switch {
		case status == 401 || status == 403:
			return &SearchError{Code: "AILINK_PROVIDER_AUTH", Message: "provider authentication failed", Details: details}
		case status == 429:
			return &SearchError{Code: "AILINK_PROVIDER_RATE_LIMIT", Message: "provider rate limited", Details: details}
		case status >= 500 && status <= 599:
			return &SearchError{Code: "AILINK_PROVIDER_UNAVAILABLE", Message: "provider unavailable", Details: details}
		case status >= 400 && status <= 499:
			return &SearchError{Code: "AILINK_PROVIDER_BAD_REQUEST", Message: "provider rejected request", Details: details}
		default:
			return &SearchError{Code: "AILINK_PROVIDER_ERROR", Message: "provider request failed", Details: details}
		}
	}

	return &SearchError{Code: "AILINK_PROVIDER_ERROR", Message: "provider request failed", Details: err.Error()}
}
