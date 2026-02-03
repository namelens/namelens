package server

import (
	"context"
	"os"

	"github.com/fulmenhq/gofulmen/signals"
	"github.com/go-chi/chi/v5"

	"github.com/namelens/namelens/internal/api"
	"github.com/namelens/namelens/internal/appid"
	"go.uber.org/zap"

	"github.com/namelens/namelens/internal/observability"
	"github.com/namelens/namelens/internal/server/handlers"
)

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes() {
	// Standard health endpoints per Workhorse §9
	s.router.Get("/health", handlers.HealthHandler)
	s.router.Get("/health/live", handlers.LivenessHandler)
	s.router.Get("/health/ready", handlers.ReadinessHandler)
	s.router.Get("/health/startup", handlers.StartupHandler)

	// Version endpoint
	s.router.Get("/version", handlers.VersionHandler)

	// Metrics endpoint (in server package to access HandleError)
	s.router.Get("/metrics", MetricsHandler)

	// Admin signal endpoint (optional, requires NAMELENS_ADMIN_TOKEN)
	s.registerAdminEndpoint()
}

// registerAPIRoutes registers the control plane API routes
func (s *Server) registerAPIRoutes(authConfig api.AuthConfig) {
	if s.apiServer == nil {
		return
	}

	// Mount API routes with authentication middleware
	s.router.Group(func(r chi.Router) {
		// Apply auth middleware to API routes
		r.Use(api.AuthMiddleware(authConfig))

		// Mount the generated API handlers
		// Note: /health is already handled by existing health handlers
		// So we only mount /v1/* endpoints here
		r.Post("/v1/check", s.apiServer.CheckName)
		r.Post("/v1/compare", s.apiServer.CompareCandidates)
		r.Get("/v1/profiles", s.apiServer.ListProfiles)
		r.Get("/v1/status", s.apiServer.GetStatus)
	})

	logger := observability.ServerLogger
	if logger != nil {
		logger.Info("Control plane API routes registered",
			zap.Bool("auth_required", authConfig.APIKey != ""),
			zap.Bool("localhost_allowed", authConfig.AllowLocalhost))
	}
}

// registerAdminEndpoint optionally registers the admin signal endpoint
func (s *Server) registerAdminEndpoint() {
	// Get admin token from environment (identity-aware)
	ctx := context.Background()
	identity, _ := appid.Get(ctx)
	envPrefix := "WORKHORSE_"
	if identity != nil && identity.EnvPrefix != "" {
		envPrefix = identity.EnvPrefix
	}

	adminToken := os.Getenv(envPrefix + "ADMIN_TOKEN")
	logger := observability.ServerLogger

	if adminToken == "" {
		if logger != nil {
			logger.Debug("Admin signal endpoint disabled (no " + envPrefix + "ADMIN_TOKEN set)")
		}
		return
	}

	// Create HTTP signal handler with bearer token auth and rate limiting
	handler := signals.NewHTTPHandler(signals.HTTPConfig{
		TokenAuth: adminToken,
		RateLimit: 10,  // 10 requests per minute
		RateBurst: 5,   // burst size
		Manager:   nil, // use default global manager
	})

	// Register admin endpoint
	s.router.Post("/admin/signal", handler.ServeHTTP)

	if logger != nil {
		logger.Info("Admin signal endpoint enabled",
			zap.String("path", "/admin/signal"),
			zap.String("auth", "bearer token"),
			zap.String("rate_limit", "10/min, burst 5"))
		logger.Warn("Admin endpoint enabled - ensure this server is not exposed to public internet")
	}
}
