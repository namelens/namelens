package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
)

// AuthConfig configures API authentication.
type AuthConfig struct {
	// APIKey is the expected API key. If empty, authentication is disabled.
	APIKey string

	// AllowLocalhost allows unauthenticated access from localhost.
	AllowLocalhost bool
}

// AuthMiddleware creates middleware that validates API keys.
// Authentication is required for non-localhost requests when an API key is configured.
// If a key is provided in the request, it is always validated (even from localhost).
func AuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// No API key configured - allow all requests
			if cfg.APIKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check for provided API key
			providedKey := r.Header.Get("X-API-Key")

			// If a key is provided, always validate it (even from localhost)
			// This helps catch configuration errors during local development
			if providedKey != "" {
				if subtle.ConstantTimeCompare([]byte(providedKey), []byte(cfg.APIKey)) != 1 {
					writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "invalid API key")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// No key provided - allow localhost if configured
			if cfg.AllowLocalhost && isLocalhost(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Non-localhost without key
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "API key required for non-localhost requests")
		})
	}
}

// isLocalhost checks if the request originated from localhost.
func isLocalhost(r *http.Request) bool {
	// If request includes proxy forwarding headers, do not treat as localhost.
	// This prevents spoofing X-Forwarded-For to bypass auth when RealIP is enabled.
	if hasForwardedHeaders(r) {
		return false
	}

	// Get the remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// No port, use the whole address
		host = r.RemoteAddr
	}

	// Parse the IP
	ip := net.ParseIP(host)
	if ip == nil {
		// Not a valid IP, check if it's a hostname
		return strings.EqualFold(host, "localhost")
	}

	// Check for loopback addresses (127.0.0.1, ::1)
	return ip.IsLoopback()
}

func hasForwardedHeaders(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.TrimSpace(r.Header.Get("X-Forwarded-For")) != "" {
		return true
	}
	if strings.TrimSpace(r.Header.Get("X-Real-IP")) != "" {
		return true
	}
	if strings.TrimSpace(r.Header.Get("Forwarded")) != "" {
		return true
	}
	return false
}

// GenerateAPIKey generates a random API key with the nlcp_ prefix.
// The key is 32 random bytes encoded as hex (64 chars) plus the prefix.
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "nlcp_" + hex.EncodeToString(bytes), nil
}

// writeErrorResponse writes an API error response.
func writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Simple JSON encoding to avoid import cycles
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + message + `"}}`))
}
