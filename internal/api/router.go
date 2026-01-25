// Package api provides HTTP routing and handlers for the Nysus API server.
package api

import (
	"net/http"

	"github.com/asgard/pandora/internal/api/handlers"
	apimiddleware "github.com/asgard/pandora/internal/api/middleware"
	"github.com/asgard/pandora/internal/api/realtime"
	"github.com/asgard/pandora/internal/api/signaling"
	"github.com/asgard/pandora/internal/controlplane"
	realtimecore "github.com/asgard/pandora/internal/platform/realtime"
	"github.com/asgard/pandora/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Router sets up all API routes and handlers.
func NewRouter(
	authService *services.AuthService,
	userService *services.UserService,
	subscriptionService *services.SubscriptionService,
	dashboardService *services.DashboardService,
	streamService *services.StreamService,
	pricillaService *services.PricillaService,
	auditService *services.AuditService,
	eventBroadcaster *realtime.Broadcaster,
	signalingServer *signaling.Server,
	controlPlane *controlplane.UnifiedControlPlane,
) http.Handler {
	r := chi.NewRouter()
	apiRouter := chi.NewRouter()

	// Middleware
	apiRouter.Use(middleware.RequestID)
	apiRouter.Use(middleware.RealIP)
	apiRouter.Use(middleware.Logger)
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	adminHandler := handlers.NewAdminHandler(userService)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	streamHandler := handlers.NewStreamHandler(streamService)
	pricillaHandler := handlers.NewPricillaHandler(pricillaService)
	auditHandler := handlers.NewAuditHandler(auditService)

	// Initialize health handler
	healthHandler := handlers.NewHealthHandler()

	// Initialize control plane handler
	controlPlaneHandler := handlers.NewControlPlaneHandler(controlPlane)

	// API routes
	apiRouter.Route("/", func(r chi.Router) {
		// Health check
		r.Get("/health", healthHandler.Health)

		// Authentication routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signin", authHandler.SignIn)
			r.Post("/signup", authHandler.SignUp)
			r.Post("/signout", authHandler.SignOut)
			r.Post("/refresh", authHandler.RefreshToken)
			r.Post("/password-reset/request", authHandler.RequestPasswordReset)
			r.Post("/password-reset/confirm", authHandler.ResetPassword)
			r.Post("/verify-email", authHandler.VerifyEmail)
			r.Route("/fido2", func(r chi.Router) {
				r.Post("/register/start", authHandler.StartFido2Registration)
				r.Post("/register/complete", authHandler.CompleteFido2Registration)
				r.Post("/auth/start", authHandler.StartFido2Auth)
				r.Post("/auth/complete", authHandler.CompleteFido2Auth)
			})
		})

		// User routes (protected)
		r.Route("/user", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/profile", userHandler.GetProfile)
			r.Patch("/profile", userHandler.UpdateProfile)
			r.Get("/subscription", userHandler.GetSubscription)
			r.Patch("/notifications", userHandler.UpdateNotificationSettings)
		})

		// Subscription routes (protected)
		r.Route("/subscriptions", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/plans", subscriptionHandler.GetPlans)
			r.Post("/checkout", subscriptionHandler.CreateCheckoutSession)
			r.Post("/portal", subscriptionHandler.CreatePortalSession)
			r.Post("/cancel", subscriptionHandler.CancelSubscription)
			r.Post("/reactivate", subscriptionHandler.ReactivateSubscription)
		})

		// Webhook routes (no auth - verified by signature)
		r.Route("/webhooks", func(r chi.Router) {
			r.Post("/stripe", subscriptionHandler.HandleWebhook)
		})

		// Dashboard routes (protected)
		r.Route("/dashboard", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/stats", dashboardHandler.GetStats)
		})

		// Entity routes (protected)
		r.Route("/alerts", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/", dashboardHandler.GetAlerts)
			r.Get("/{id}", dashboardHandler.GetAlert)
		})

		r.Route("/missions", func(r chi.Router) {
			r.Use(apimiddleware.RequireAccessLevel(authService, realtimecore.AccessLevelMilitary))
			r.Get("/", dashboardHandler.GetMissions)
			r.Get("/{id}", dashboardHandler.GetMission)
		})

		r.Route("/satellites", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/", dashboardHandler.GetSatellites)
			r.Get("/{id}", dashboardHandler.GetSatellite)
		})

		r.Route("/hunoids", func(r chi.Router) {
			r.Use(apimiddleware.RequireAccessLevel(authService, realtimecore.AccessLevelMilitary))
			r.Get("/", dashboardHandler.GetHunoids)
			r.Get("/{id}", dashboardHandler.GetHunoid)
		})

		// Telemetry routes (protected)
		r.Route("/telemetry", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/satellite/{satelliteId}", dashboardHandler.GetSatelliteTelemetry)
			r.Get("/hunoid/{hunoidId}", dashboardHandler.GetHunoidTelemetry)
		})

		// Admin routes (protected)
		r.Route("/admin", func(r chi.Router) {
			r.Use(apimiddleware.RequireAccessLevel(authService, realtimecore.AccessLevelAdmin))
			r.Get("/users", adminHandler.ListUsers)
			r.Patch("/users/{userId}", adminHandler.UpdateUser)
		})

		// Stream routes
		// Public routes with optional auth for tier-based filtering
		r.Route("/streams", func(r chi.Router) {
			// Apply optional auth to all stream routes for tier-based filtering
			r.Use(authHandler.OptionalAuth)

			// Public endpoints - accessible by all, but content filtered by tier
			r.Get("/", streamHandler.GetStreams)
			r.Get("/stats", streamHandler.GetStreamStats)
			r.Get("/featured", streamHandler.GetFeaturedStreams)
			r.Get("/recent", streamHandler.GetRecentStreams)
			r.Get("/search", streamHandler.SearchStreams)
			r.Get("/{id}", streamHandler.GetStream)

			// Session creation requires authentication
			r.Route("/{id}/session", func(r chi.Router) {
				r.Use(authHandler.RequireAuth)
				r.Post("/", streamHandler.CreateStreamSession)
			})

			r.Route("/{id}/chat", func(r chi.Router) {
				r.Get("/", streamHandler.GetStreamChat)
				r.With(authHandler.RequireAuth).Post("/", streamHandler.SendStreamChat)
			})
		})

		// Protected stream routes requiring specific tiers
		r.Route("/streams/military", func(r chi.Router) {
			// Requires supporter tier or higher for military streams
			r.Use(apimiddleware.RequireTier(authService, "supporter"))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				// Forward to streams handler with military type filter
				q := r.URL.Query()
				q.Set("type", "military")
				r.URL.RawQuery = q.Encode()
				streamHandler.GetStreams(w, r)
			})
		})

		r.Route("/streams/interstellar", func(r chi.Router) {
			// Requires commander tier for interstellar streams
			r.Use(apimiddleware.RequireTier(authService, "commander"))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				// Forward to streams handler with interstellar type filter
				q := r.URL.Query()
				q.Set("type", "interstellar")
				r.URL.RawQuery = q.Encode()
				streamHandler.GetStreams(w, r)
			})
		})

		// Access level protected routes (for government/admin features)
		r.Route("/streams/admin", func(r chi.Router) {
			r.Use(apimiddleware.RequireAccessLevel(authService, realtimecore.AccessLevelAdmin))
			r.Get("/all", func(w http.ResponseWriter, r *http.Request) {
				// Admin can see all streams without filtering
				streamHandler.GetStreams(w, r)
			})
		})

		// Pricilla routes (protected)
		r.Route("/pricilla", func(r chi.Router) {
			r.Use(apimiddleware.RequireAccessLevel(authService, realtimecore.AccessLevelGovernment))
			r.Route("/missions", func(r chi.Router) {
				r.Get("/", pricillaHandler.HandleMissions)
				r.Post("/", pricillaHandler.HandleMissions)
				r.Get("/{id}", pricillaHandler.HandleMission)
			})
			r.Post("/payloads", pricillaHandler.HandlePayloads)
		})

		// Audit routes (protected)
		r.Route("/audit", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/logs", auditHandler.GetAuditLogs)
			r.Get("/logs/{id}", auditHandler.GetAuditLog)
			r.Get("/logs/component/{component}", auditHandler.GetAuditLogsByComponent)
			r.Get("/logs/user/{userId}", auditHandler.GetAuditLogsByUser)
			r.Get("/stats", auditHandler.GetAuditStats)
		})

		// Ethics routes (protected)
		r.Route("/ethics", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/decisions", auditHandler.GetEthicalDecisions)
			r.Get("/decisions/{id}", auditHandler.GetEthicalDecision)
			r.Get("/decisions/hunoid/{hunoidId}", auditHandler.GetEthicalDecisionsByHunoid)
			r.Get("/decisions/mission/{missionId}", auditHandler.GetEthicalDecisionsByMission)
			r.Get("/stats", auditHandler.GetEthicsStats)
		})

		// Control plane routes (protected, government/admin access)
		r.Route("/controlplane", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)

			// Status and health endpoints
			r.Get("/status", controlPlaneHandler.GetStatus)
			r.Get("/health", controlPlaneHandler.GetHealth)
			r.Get("/metrics", controlPlaneHandler.GetMetrics)

			// Events endpoints
			r.Get("/events", controlPlaneHandler.GetEvents)
			r.Get("/events/{id}", controlPlaneHandler.GetEvent)

			// Command endpoint
			r.Post("/command", controlPlaneHandler.PostCommand)

			// Systems management
			r.Get("/systems", controlPlaneHandler.GetSystems)
			r.Get("/systems/{id}", controlPlaneHandler.GetSystem)

			// Policy management
			r.Get("/policies", controlPlaneHandler.GetPolicies)
			r.Patch("/policies/{id}", controlPlaneHandler.PatchPolicy)

			// Active responses
			r.Get("/responses", controlPlaneHandler.GetResponses)
		})
	})

	r.Mount("/api", apiRouter)

	// WebSocket routes
	r.Route("/ws", func(r chi.Router) {
		r.Get("/realtime", func(w http.ResponseWriter, r *http.Request) {
			realtime.HandleWebSocket(w, r, eventBroadcaster)
		})
		r.Get("/signaling", func(w http.ResponseWriter, r *http.Request) {
			signalingServer.HandleWebSocket(w, r)
		})
	})

	return r
}
