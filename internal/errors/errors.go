package errors

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fulmenhq/gofulmen/errors"
	"github.com/google/uuid"
	"github.com/namelens/namelens/internal/metrics"
	"github.com/namelens/namelens/internal/observability"
	"github.com/namelens/namelens/internal/server/middleware"
	"go.uber.org/zap"
)

// Error creation helpers for common error types

// User Errors (400-level)
func NewInvalidInputError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("bad_request", message)
}

func NewNotFoundError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("not_found", message)
}

func NewUnauthorizedError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("unauthorized", message)
}

func NewForbiddenError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("forbidden", message)
}

func NewMethodNotAllowedError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("method_not_allowed", message)
}

func NewConflictError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("conflict", message)
}

func NewValidationError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("validation_error", message)
}

// Server Errors (500-level)
func NewInternalError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("internal_error", message)
}

func NewDatabaseError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("database_error", message)
}

func NewExternalServiceError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("external_service_error", message)
}

func NewTimeoutError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("timeout", message)
}

// Application-Specific Errors
func NewDataProcessingError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("data_processing_error", message)
}

func NewConfigInvalidError(message string) *errors.ErrorEnvelope {
	return errors.NewErrorEnvelope("config_invalid", message)
}

// Wrap functions for existing errors
// These functions accept a context to extract correlation/trace IDs from the request context

func WrapInvalidInput(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("bad_request", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapNotFound(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("not_found", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapUnauthorized(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("unauthorized", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapForbidden(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("forbidden", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapConflict(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("conflict", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapValidationError(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("validation_error", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapInternal(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("internal_error", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapDatabaseError(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("database_error", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapExternalService(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("external_service_error", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapTimeout(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("timeout", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapDataProcessing(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("data_processing_error", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

func WrapConfigInvalid(ctx context.Context, err error, message string) *errors.ErrorEnvelope {
	envelope := errors.NewErrorEnvelope("config_invalid", message)
	envelope = envelope.WithCorrelationID(extractCorrelationID(ctx))
	envelope = envelope.WithTraceID(extractTraceID(ctx))
	envelope = withWrappedError(envelope, err)
	return envelope
}

// Helper functions for ID generation

// extractCorrelationID gets correlation ID from context, falls back to generating new UUID
func extractCorrelationID(ctx context.Context) string {
	if ctx != nil {
		if requestID := middleware.GetRequestID(ctx); requestID != "" {
			return requestID
		}
	}
	// Fallback: generate new UUID when context is nil or has no request ID
	return uuid.New().String()
}

// extractTraceID gets trace ID from context, falls back to generating new UUID
func extractTraceID(ctx context.Context) string {
	// TODO: Extract from OpenTelemetry or other tracing system when implemented
	// For now, use correlation ID as trace ID
	return extractCorrelationID(ctx)
}

// EnsureEnvelope normalizes any error into a gofulmen ErrorEnvelope.
func EnsureEnvelope(err error) *errors.ErrorEnvelope {
	if err == nil {
		env := errors.NewErrorEnvelope("internal_error", "unexpected nil error")
		env, _ = env.WithSeverity(errors.SeverityCritical)
		return env
	}

	if envelope, ok := err.(*errors.ErrorEnvelope); ok && envelope != nil {
		return envelope
	}

	env := errors.NewErrorEnvelope("internal_error", "unexpected error")
	env, _ = env.WithContext(map[string]interface{}{
		"wrapped_error": err.Error(),
	})
	env, _ = env.WithSeverity(errors.SeverityHigh)
	return env
}

// EnsureCorrelationID attaches a correlation ID to the envelope using the context when available.
func EnsureCorrelationID(envelope *errors.ErrorEnvelope, ctx context.Context) *errors.ErrorEnvelope {
	if envelope == nil {
		return nil
	}

	if envelope.CorrelationID != "" {
		return envelope
	}

	var correlationID string
	if ctx != nil {
		correlationID = middleware.GetRequestID(ctx)
	}

	if correlationID == "" {
		correlationID = "fallback-" + errors.GenerateCorrelationID()
	}

	return envelope.WithCorrelationID(correlationID)
}

// HTTPStatusFromEnvelope resolves the HTTP status code corresponding to an error envelope.
func HTTPStatusFromEnvelope(envelope *errors.ErrorEnvelope) int {
	if envelope == nil {
		return http.StatusInternalServerError
	}
	return HTTPStatusFromCode(envelope.Code)
}

// HTTPStatusFromCode resolves the HTTP status code corresponding to an error code.
// Supports both lowercase snake_case (preferred) and legacy SCREAMING_CASE codes.
func HTTPStatusFromCode(code string) int {
	switch code {
	case "bad_request", "validation_error":
		return http.StatusBadRequest
	case "not_found":
		return http.StatusNotFound
	case "unauthorized":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "method_not_allowed":
		return http.StatusMethodNotAllowed
	case "conflict":
		return http.StatusConflict
	case "timeout":
		return http.StatusGatewayTimeout
	case "external_service_error":
		return http.StatusBadGateway
	case "service_unavailable":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func withWrappedError(envelope *errors.ErrorEnvelope, err error) *errors.ErrorEnvelope {
	if envelope == nil || err == nil {
		return envelope
	}

	updated, updateErr := envelope.WithContext(map[string]interface{}{
		"wrapped_error": err.Error(),
	})
	if updateErr != nil {
		return envelope
	}
	return updated
}

// ResponseDetails constructs API-safe details map by merging envelope details and context.
func ResponseDetails(envelope *errors.ErrorEnvelope) map[string]interface{} {
	if envelope == nil {
		return nil
	}

	details := make(map[string]interface{})

	for key, value := range envelope.Details {
		details[key] = value
	}

	for key, value := range envelope.Context {
		if _, exists := details[key]; !exists {
			details[key] = value
		}
	}

	if len(details) == 0 {
		return nil
	}

	return details
}

// HTTPErrorDetail captures the error body returned to callers.
type HTTPErrorDetail struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

// HTTPErrorResponse wraps HTTPErrorDetail in the standard envelope structure.
type HTTPErrorResponse struct {
	Error HTTPErrorDetail `json:"error"`
}

// RespondWithError normalizes the supplied error and writes a JSON response.
func RespondWithError(w http.ResponseWriter, r *http.Request, err error) {
	RespondWithEnvelope(w, r, EnsureEnvelope(err))
}

// RespondWithEnvelope finalizes the provided envelope, logging and emitting metrics.
func RespondWithEnvelope(w http.ResponseWriter, r *http.Request, envelope *errors.ErrorEnvelope) {
	if w == nil {
		return
	}

	if r != nil {
		envelope = EnsureCorrelationID(envelope, r.Context())
	} else {
		envelope = EnsureCorrelationID(envelope, nil)
	}

	statusCode := HTTPStatusFromEnvelope(envelope)

	response := HTTPErrorResponse{
		Error: HTTPErrorDetail{
			Code:      envelope.Code,
			Message:   envelope.Message,
			Details:   ResponseDetails(envelope),
			RequestID: envelope.CorrelationID,
		},
	}

	logHTTPError(envelope, statusCode)
	emitErrorMetrics(r, envelope, statusCode)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

func logHTTPError(envelope *errors.ErrorEnvelope, statusCode int) {
	if observability.ServerLogger == nil || envelope == nil {
		return
	}

	fields := []zap.Field{
		zap.String("error_code", envelope.Code),
		zap.Int("http_status", statusCode),
	}

	if envelope.Severity != "" {
		fields = append(fields, zap.String("severity", string(envelope.Severity)))
	}

	for key, value := range envelope.Context {
		fields = append(fields, zap.Any(key, value))
	}

	if envelope.CorrelationID != "" {
		fields = append(fields, zap.String("request_id", envelope.CorrelationID))
	}

	switch envelope.Severity {
	case errors.SeverityCritical, errors.SeverityHigh:
		observability.ServerLogger.Error(envelope.Message, fields...)
	case errors.SeverityMedium:
		observability.ServerLogger.Warn(envelope.Message, fields...)
	default:
		observability.ServerLogger.Info(envelope.Message, fields...)
	}
}

func emitErrorMetrics(r *http.Request, envelope *errors.ErrorEnvelope, statusCode int) {
	if envelope == nil {
		return
	}

	metrics.RecordError(envelope.Code, statusCode)
	if r != nil {
		metrics.RecordErrorByEndpoint(r.URL.Path, envelope.Code)
	}
}
