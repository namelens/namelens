package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	tests := []struct {
		name       string
		config     AuthConfig
		apiKey     string
		remoteAddr string
		wantStatus int
	}{
		{
			name:       "no auth configured allows all",
			config:     AuthConfig{},
			apiKey:     "",
			remoteAddr: "192.168.1.1:12345",
			wantStatus: http.StatusOK,
		},
		{
			name: "localhost allowed without key",
			config: AuthConfig{
				APIKey:         "test-key",
				AllowLocalhost: true,
			},
			apiKey:     "",
			remoteAddr: "127.0.0.1:12345",
			wantStatus: http.StatusOK,
		},
		{
			name: "non-localhost requires key",
			config: AuthConfig{
				APIKey:         "test-key",
				AllowLocalhost: true,
			},
			apiKey:     "",
			remoteAddr: "192.168.1.1:12345",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "valid key allows access",
			config: AuthConfig{
				APIKey:         "test-key",
				AllowLocalhost: true,
			},
			apiKey:     "test-key",
			remoteAddr: "192.168.1.1:12345",
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid key rejected",
			config: AuthConfig{
				APIKey:         "test-key",
				AllowLocalhost: true,
			},
			apiKey:     "wrong-key",
			remoteAddr: "192.168.1.1:12345",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "ipv6 localhost allowed",
			config: AuthConfig{
				APIKey:         "test-key",
				AllowLocalhost: true,
			},
			apiKey:     "",
			remoteAddr: "[::1]:12345",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := AuthMiddleware(tt.config)
			wrapped := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Check prefix
	if !strings.HasPrefix(key, "nlcp_") {
		t.Errorf("expected key to start with nlcp_, got %s", key)
	}

	// Check length (prefix + 64 hex chars)
	if len(key) != 5+64 {
		t.Errorf("expected key length %d, got %d", 5+64, len(key))
	}

	// Generate another key and ensure they're different
	key2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("failed to generate second key: %v", err)
	}

	if key == key2 {
		t.Error("expected different keys")
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		remoteAddr string
		want       bool
	}{
		{"127.0.0.1:8080", true},
		{"127.0.0.1", true},
		{"[::1]:8080", true},
		{"::1", true},
		{"localhost:8080", true},
		{"192.168.1.1:8080", false},
		{"10.0.0.1:8080", false},
		{"example.com:8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.remoteAddr, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr

			got := isLocalhost(req)
			if got != tt.want {
				t.Errorf("isLocalhost(%q) = %v, want %v", tt.remoteAddr, got, tt.want)
			}
		})
	}
}
