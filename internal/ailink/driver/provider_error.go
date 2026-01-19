package driver

import "fmt"

// ProviderError is returned when a provider responds with a non-2xx status.
//
// Drivers should populate RawResponse with the provider response body bytes.
// RawResponse must never include API keys.
type ProviderError struct {
	Provider    string
	StatusCode  int
	Message     string
	RawResponse []byte
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "provider error"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s request failed: status %d: %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s request failed: %s", e.Provider, e.Message)
}
