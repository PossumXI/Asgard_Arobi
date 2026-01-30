// Package api implements the HTTP REST API for Nysus.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/api/signaling"
	"github.com/asgard/pandora/internal/api/webrtc"
	"github.com/asgard/pandora/internal/nysus/events"
	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/platform/observability"
	"github.com/asgard/pandora/internal/platform/realtime"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/asgard/pandora/internal/services"
	"github.com/golang-jwt/jwt/v5"
	pionwebrtc "github.com/pion/webrtc/v4"
)

// Server represents the Nysus HTTP API server.
type Server struct {
	httpServer      *http.Server
	pgDB            *db.PostgresDB
	mongoDB         *db.MongoDB
	eventBus        *events.EventBus
	wsHub           *WebSocketHub
	natsBridge      *realtime.Bridge
	wsManager       *realtime.WebSocketManager
	accessRules     *realtime.AccessRules
	signalingServer *signaling.Server
	sfu             *webrtc.SFU
	streamService   *services.StreamService
	chatStore       *chatStore
	accessCodeService *services.AccessCodeService
	accessCodeCancel  context.CancelFunc
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
	// Initialize access rules
	accessRules := realtime.NewAccessRules()

	// Initialize WebSocket manager
	wsManager := realtime.NewWebSocketManager()

	// Try to initialize NATS bridge (optional - continues without if NATS unavailable)
	var natsBridge *realtime.Bridge
	natsConfig := realtime.DefaultBridgeConfig()
	if natsURI := getNATSURI(); natsURI != "" {
		natsConfig.NATSURL = natsURI
	}

	bridge, err := realtime.NewBridge(natsConfig, wsManager)
	if err != nil {
		log.Printf("[Nysus] NATS bridge unavailable: %v (continuing without real-time NATS events)", err)
	} else {
		natsBridge = bridge
		log.Println("[Nysus] NATS bridge initialized successfully")
	}

	// Initialize WebRTC SFU for streaming
	webrtcConfig := createWebRTCConfig()
	sfu := webrtc.NewSFU(webrtcConfig)
	log.Println("[Nysus] WebRTC SFU initialized")

	var streamService *services.StreamService
	var accessCodeService *services.AccessCodeService
	if pgDB != nil {
		streamRepo := repositories.NewStreamRepository(pgDB, mongoDB)
		streamService = services.NewStreamService(streamRepo)
		streamService.SetSFU(sfu)

		userRepo := repositories.NewUserRepository(pgDB)
		accessCodeRepo := repositories.NewAccessCodeRepository(pgDB)
		accessCodeService = services.NewAccessCodeService(accessCodeRepo, userRepo, services.NewEmailService())

		bootstrapAdminUser(pgDB)
	}

	// Initialize signaling server with the SFU and optional stream service.
	signalingServer := signaling.NewServer(streamService, sfu)
	log.Println("[Nysus] WebRTC signaling server initialized")

	s := &Server{
		pgDB:            pgDB,
		mongoDB:         mongoDB,
		eventBus:        eventBus,
		wsHub:           NewWebSocketHub(eventBus),
		natsBridge:      natsBridge,
		wsManager:       wsManager,
		accessRules:     accessRules,
		signalingServer: signalingServer,
		sfu:             sfu,
		streamService:   streamService,
		chatStore:       newChatStore(pgDB),
		accessCodeService: accessCodeService,
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

// createWebRTCConfig creates the WebRTC configuration with ICE servers.
func createWebRTCConfig() pionwebrtc.Configuration {
	config := pionwebrtc.Configuration{
		ICEServers: []pionwebrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun.l.google.com:19302",
					"stun:stun1.l.google.com:19302",
				},
			},
		},
	}

	// Add TURN server if configured
	if turnURL := os.Getenv("TURN_SERVER"); turnURL != "" {
		turnUsername := os.Getenv("TURN_USERNAME")
		turnPassword := os.Getenv("TURN_PASSWORD")
		config.ICEServers = append(config.ICEServers, pionwebrtc.ICEServer{
			URLs:       []string{turnURL},
			Username:   turnUsername,
			Credential: turnPassword,
		})
		log.Printf("[Nysus] TURN server configured: %s", turnURL)
	}

	return config
}

// getNATSURI returns the NATS URI from environment or default.
func getNATSURI() string {
	host := getEnvDefault("NATS_HOST", "localhost")
	port := getEnvDefault("NATS_PORT", "4222")
	return "nats://" + host + ":" + port
}

func getEnvDefault(key, defaultVal string) string {
	if val := getEnv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnv(key string) string {
	return os.Getenv(key)
}

// Start begins serving HTTP requests.
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	// Start NATS bridge if available
	if s.natsBridge != nil {
		if err := s.natsBridge.Start(); err != nil {
			log.Printf("[Nysus] Failed to start NATS bridge: %v", err)
		} else {
			log.Println("[Nysus] NATS bridge started - real-time events active")
		}
	}

	if s.accessCodeService != nil {
		rotationCtx, cancel := context.WithCancel(context.Background())
		s.accessCodeCancel = cancel
		go s.accessCodeService.StartRotationLoop(rotationCtx)
		log.Println("[AccessCode] rotation loop started")
	}

	log.Printf("[API] Server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	// Stop NATS bridge
	if s.natsBridge != nil {
		s.natsBridge.Stop()
	}

	// Stop WebSocket manager
	if s.wsManager != nil {
		s.wsManager.Stop()
	}

	if s.accessCodeCancel != nil {
		s.accessCodeCancel()
	}

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
	mux.HandleFunc("/api/access-codes/validate", s.handleAccessCodeValidate)

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
	mux.HandleFunc("/api/streams/recent", s.handleRecentStreams)
	mux.HandleFunc("/api/streams/search", s.handleStreamSearch)
	mux.HandleFunc("/api/streams/", s.handleStreamRoutes)

	// Admin endpoints
	mux.HandleFunc("/api/admin/users", s.handleAdminUsers)
	mux.HandleFunc("/api/admin/users/", s.handleAdminUser)
	mux.HandleFunc("/api/admin/access-codes", s.handleAdminAccessCodes)
	mux.HandleFunc("/api/admin/access-codes/rotate", s.handleAdminAccessCodesRotate)
	mux.HandleFunc("/api/admin/access-codes/", s.handleAdminAccessCode)

	// Pricilla endpoints
	mux.HandleFunc("/api/pricilla/missions", s.handlePricillaMissions)
	mux.HandleFunc("/api/pricilla/missions/", s.handlePricillaMission)
	mux.HandleFunc("/api/pricilla/payloads", s.handlePricillaPayloads)

	// Control plane endpoints (admin/government)
	mux.HandleFunc("/api/controlplane/status", s.handleControlPlaneStatus)
	mux.HandleFunc("/api/controlplane/command", s.handleControlPlaneCommand)

	// WebSocket endpoints
	mux.HandleFunc("/ws", s.handleRealtimeWebSocket)       // Fallback for /ws without suffix
	mux.HandleFunc("/ws/realtime", s.handleWebSocket)
	mux.HandleFunc("/ws/events", s.handleRealtimeWebSocket)
	mux.HandleFunc("/ws/signaling", s.handleSignalingWebSocket)

	// Real-time stats endpoint
	mux.HandleFunc("/api/realtime/stats", s.handleRealtimeStats)

	// Prometheus metrics endpoint
	mux.Handle("/metrics", observability.Handler())
}

// middleware applies common middleware to all requests.
func (s *Server) middleware(next http.Handler) http.Handler {
	// Wrap with metrics middleware first
	handler := observability.HTTPMiddleware(next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || strings.HasPrefix(r.URL.Path, "/ws/") {
			next.ServeHTTP(w, r)
			return
		}

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
		handler.ServeHTTP(w, r)
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
	natsHealth := "ok"

	if s.pgDB != nil {
		if err := s.pgDB.Health(ctx); err != nil {
			pgHealth = "error"
			status = "degraded"
		}
	} else {
		pgHealth = "not_connected"
	}

	if s.mongoDB != nil {
		if err := s.mongoDB.Health(ctx); err != nil {
			mongoHealth = "error"
			status = "degraded"
		}
	} else {
		mongoHealth = "not_connected"
	}

	if s.natsBridge != nil {
		if !s.natsBridge.IsConnected() {
			natsHealth = "disconnected"
			// NATS is optional, so don't degrade overall status
		}
	} else {
		natsHealth = "not_configured"
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   status,
		"postgres": pgHealth,
		"mongodb":  mongoHealth,
		"nats":     natsHealth,
		"time":     time.Now().UTC(),
	})
}

// handleRealtimeWebSocket handles WebSocket connections for NATS-bridged events.
func (s *Server) handleRealtimeWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, accessLevel := s.resolveRealtimeAccess(r)
	s.wsManager.HandleWebSocket(w, r, userID, accessLevel)
}

// handleSignalingWebSocket handles WebSocket connections for WebRTC signaling.
func (s *Server) handleSignalingWebSocket(w http.ResponseWriter, r *http.Request) {
	s.signalingServer.HandleWebSocket(w, r)
}

// handleRealtimeStats returns real-time infrastructure statistics.
func (s *Server) handleRealtimeStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"timestamp": time.Now().UTC(),
	}

	if s.natsBridge != nil {
		stats["nats"] = s.natsBridge.Stats()
	} else {
		stats["nats"] = map[string]interface{}{"status": "not_configured"}
	}

	if s.wsManager != nil {
		stats["websocket"] = s.wsManager.Stats()
	}

	if s.wsHub != nil {
		stats["legacy_websocket"] = map[string]interface{}{
			"active": true,
		}
	}

	// Add WebRTC SFU stats
	if s.sfu != nil {
		stats["webrtc_sfu"] = map[string]interface{}{
			"status": "active",
		}
	}

	s.writeJSON(w, http.StatusOK, stats)
}

func (s *Server) resolveRealtimeAccess(r *http.Request) (string, realtime.AccessLevel) {
	defaultAccess := realtime.AccessLevelFromString(r.URL.Query().Get("access"))
	if defaultAccess == "" {
		defaultAccess = realtime.AccessLevelPublic
	}

	token := extractToken(r)
	if token == "" {
		return "anonymous", defaultAccess
	}

	userID, role, tier, isGovernment, err := parseJWTClaims(token)
	if err != nil || userID == "" {
		return "anonymous", defaultAccess
	}

	// Prefer explicit role or tier from token, otherwise fall back to DB.
	if level := accessLevelFromToken(role, tier, isGovernment); level != "" {
		return userID, level
	}

	return userID, s.lookupUserAccessLevel(userID, defaultAccess)
}

func (s *Server) lookupUserAccessLevel(userID string, fallback realtime.AccessLevel) realtime.AccessLevel {
	if s.pgDB == nil {
		return fallback
	}

	repo := repositories.NewUserRepository(s.pgDB)
	user, err := repo.GetByID(userID)
	if err != nil {
		return fallback
	}

	if user.IsGovernment {
		return realtime.AccessLevelGovernment
	}

	switch strings.ToLower(user.SubscriptionTier) {
	case "commander":
		return realtime.AccessLevelMilitary
	case "supporter", "observer", "free":
		return realtime.AccessLevelCivilian
	default:
		return fallback
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	return ""
}

func parseJWTClaims(tokenString string) (string, string, string, bool, error) {
	secret := getJWTSecret()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil {
		return "", "", "", false, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", "", false, fmt.Errorf("invalid token claims")
	}

	userID, _ := claims["user_id"].(string)
	role, _ := claims["role"].(string)
	tier, _ := claims["subscription_tier"].(string)
	isGovernment, _ := claims["is_government"].(bool)

	return userID, role, tier, isGovernment, nil
}

func accessLevelFromToken(role, tier string, isGovernment bool) realtime.AccessLevel {
	if isGovernment {
		return realtime.AccessLevelGovernment
	}
	switch strings.ToLower(role) {
	case "admin":
		return realtime.AccessLevelAdmin
	case "military":
		return realtime.AccessLevelMilitary
	case "civilian", "user", "subscriber":
		return realtime.AccessLevelCivilian
	}

	switch strings.ToLower(tier) {
	case "commander":
		return realtime.AccessLevelMilitary
	case "supporter", "observer", "free":
		return realtime.AccessLevelCivilian
	}

	return ""
}

func getJWTSecret() []byte {
	secret := os.Getenv("ASGARD_JWT_SECRET")
	if len(secret) >= 32 {
		return []byte(secret)
	}
	// In development mode only, use a default (but log warning)
	if os.Getenv("ASGARD_ENV") == "development" {
		fmt.Println("[WARNING] Using default JWT secret - set ASGARD_JWT_SECRET in production!")
		return []byte("dev_jwt_secret_not_for_production_use")
	}
	panic("ASGARD_JWT_SECRET environment variable must be set (min 32 characters)")
}
