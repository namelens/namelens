package driver

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TraceEntry represents a single request/response trace entry.
type TraceEntry struct {
	Timestamp   time.Time       `json:"timestamp"`
	Driver      string          `json:"driver"`
	Endpoint    string          `json:"endpoint"`
	Method      string          `json:"method"`
	Model       string          `json:"model,omitempty"`
	RequestBody json.RawMessage `json:"request_body,omitempty"`
	StatusCode  int             `json:"status_code,omitempty"`
	Response    json.RawMessage `json:"response,omitempty"`
	Error       string          `json:"error,omitempty"`
	DurationMs  int64           `json:"duration_ms"`
}

// Tracer records request/response traces to a file in NDJSON format.
type Tracer struct {
	file *os.File
	mu   sync.Mutex
}

var (
	globalTracer *Tracer
	tracerMu     sync.Mutex
)

// EnableTracing starts tracing to the specified file path.
// Returns a cleanup function that should be called to close the file.
func EnableTracing(path string) (func(), error) {
	tracerMu.Lock()
	defer tracerMu.Unlock()

	if globalTracer != nil {
		_ = globalTracer.Close()
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // #nosec G304 -- user-provided --trace file path
	if err != nil {
		return nil, fmt.Errorf("open trace file: %w", err)
	}

	globalTracer = &Tracer{file: f}
	return func() {
		tracerMu.Lock()
		defer tracerMu.Unlock()
		if globalTracer != nil {
			_ = globalTracer.Close()
			globalTracer = nil
		}
	}, nil
}

// DisableTracing stops tracing and closes the trace file.
func DisableTracing() {
	tracerMu.Lock()
	defer tracerMu.Unlock()
	if globalTracer != nil {
		_ = globalTracer.Close()
		globalTracer = nil
	}
}

// IsTracingEnabled returns true if tracing is active.
func IsTracingEnabled() bool {
	tracerMu.Lock()
	defer tracerMu.Unlock()
	return globalTracer != nil
}

// Trace records a trace entry if tracing is enabled.
func Trace(entry TraceEntry) {
	tracerMu.Lock()
	t := globalTracer
	tracerMu.Unlock()

	if t == nil {
		return
	}
	t.Write(entry)
}

// Write records a trace entry.
func (t *Tracer) Write(entry TraceEntry) {
	if t == nil || t.file == nil {
		return
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = t.file.Write(data)
	_, _ = t.file.Write([]byte("\n"))
}

// Sync flushes any buffered data to the trace file.
func (t *Tracer) Sync() error {
	if t == nil || t.file == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.file.Sync()
}

// Close syncs and closes the trace file.
func (t *Tracer) Close() error {
	if t == nil || t.file == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	// Sync before close to ensure all data is written
	_ = t.file.Sync()
	return t.file.Close()
}
