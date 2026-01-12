package ailink

import "encoding/json"

// RawResponseError wraps an error with the raw response payload.
//
// This is useful when the model returned JSON that failed schema validation or decoding,
// but callers still want access to the raw payload for debugging.
type RawResponseError struct {
	Err error
	Raw json.RawMessage
}

func (e *RawResponseError) Error() string {
	if e == nil || e.Err == nil {
		return "ailink error"
	}
	return e.Err.Error()
}

func (e *RawResponseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
