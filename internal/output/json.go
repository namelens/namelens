package output

import (
	"encoding/json"

	"github.com/namelens/namelens/internal/core"
)

// JSONFormatter renders results as JSON.
type JSONFormatter struct {
	Indent bool
}

// FormatBatch renders a batch result as JSON.
func (f *JSONFormatter) FormatBatch(result *core.BatchResult) (string, error) {
	if result == nil {
		return "", nil
	}

	var (
		data []byte
		err  error
	)

	if f.Indent {
		data, err = json.MarshalIndent(result, "", "  ")
	} else {
		data, err = json.Marshal(result)
	}
	if err != nil {
		return "", err
	}

	return string(data), nil
}
