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
func AuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// No API key configured - allow all requests
			if cfg.APIKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if request is from localhost
			if cfg.AllowLocalhost && isLocalhost(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Validate API key
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "API key required for non-localhost requests")
				return
			}

			// Constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(providedKey), []byte(cfg.APIKey)) != 1 {
				writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isLocalhost checks if the request originated from localhost.
func isLocalhost(r *http.Request) bool {
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
