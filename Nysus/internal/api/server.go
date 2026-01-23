// Package api implements the HTTP REST API for Nysus.
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/asgard/pandora/Nysus/internal/events"
	"github.com/asgard/pandora/internal/platform/db"
)

// Server represents the Nysus HTTP API server.
type Server struct {
	httpServer *http.Server
	pgDB       *db.PostgresDB
	mongoDB    *db.MongoDB
	eventBus   *events.EventBus
	wsHub      *WebSocketHub
}

// Config holds server configuration.
type Config struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultConfig returns default server configuration.
func DefaultConfig() Config {
	return Config{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// NewServer creates a new API server.
func NewServer(cfg Config, pgDB *db.PostgresDB, mongoDB *db.MongoDB, eventBus *events.EventBus) *Server {
	s := &Server{
		pgDB:     pgDB,
		mongoDB:  mongoDB,
		eventBus: eventBus,
		wsHub:    NewWebSocketHub(eventBus),
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpServer = &http.Server{
		Addr:         cfg.Addr,
		Handler:      s.middleware(mux),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// Start begins serving HTTP requests.
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	log.Printf("[API] Server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// registerRoutes sets up all API endpoints.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Auth endpoints
	mux.HandleFunc("/api/auth/signin", s.handleSignIn)
	mux.HandleFunc("/api/auth/signup", s.handleSignUp)
	mux.HandleFunc("/api/auth/signout", s.handleSignOut)
	mux.HandleFunc("/api/auth/refresh", s.handleRefreshToken)

	// User endpoints
	mux.HandleFunc("/api/user/profile", s.handleUserProfile)
	mux.HandleFunc("/api/user/subscription", s.handleUserSubscription)
	mux.HandleFunc("/api/user/notifications", s.handleNotificationSettings)

	// Subscription endpoints
	mux.HandleFunc("/api/subscriptions/plans", s.handleSubscriptionPlans)
	mux.HandleFunc("/api/subscriptions/checkout", s.handleCheckout)
	mux.HandleFunc("/api/subscriptions/portal", s.handleBillingPortal)
	mux.HandleFunc("/api/subscriptions/cancel", s.handleCancelSubscription)

	// Dashboard endpoints
	mux.HandleFunc("/api/dashboard/stats", s.handleDashboardStats)

	// Entity endpoints
	mux.HandleFunc("/api/alerts", s.handleAlerts)
	mux.HandleFunc("/api/missions", s.handleMissions)
	mux.HandleFunc("/api/satellites", s.handleSatellites)
	mux.HandleFunc("/api/hunoids", s.handleHunoids)
	mux.HandleFunc("/api/threats", s.handleThreats)

	// Streams endpoints (for Hubs)
	mux.HandleFunc("/api/streams", s.handleStreams)
	mux.HandleFunc("/api/streams/stats", s.handleStreamStats)
	mux.HandleFunc("/api/streams/featured", s.handleFeaturedStreams)
	mux.HandleFunc("/api/streams/search", s.handleStreamSearch)

	// WebSocket endpoint
	mux.HandleFunc("/ws/realtime", s.handleWebSocket)
}

// middleware applies common middleware to all requests.
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Request logging
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[API] %s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// JSON response helpers
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string, code string) {
	s.writeJSON(w, status, map[string]string{
		"message": message,
		"code":    code,
	})
}

// Health check handler
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := "healthy"
	pgHealth := "ok"
	mongoHealth := "ok"

	if err := s.pgDB.Health(ctx); err != nil {
		pgHealth = "error"
		status = "degraded"
	}

	if err := s.mongoDB.Health(ctx); err != nil {
		mongoHealth = "error"
		status = "degraded"
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   status,
		"postgres": pgHealth,
		"mongodb":  mongoHealth,
		"time":     time.Now().UTC(),
	})
}
