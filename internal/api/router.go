// Package api provides HTTP routing and handlers for the Nysus API server.
package api

import (
	"net/http"

	"github.com/asgard/pandora/internal/api/handlers"
	"github.com/asgard/pandora/internal/api/realtime"
	"github.com/asgard/pandora/internal/api/signaling"
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
	eventBroadcaster *realtime.Broadcaster,
	signalingServer *signaling.Server,
) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
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
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	streamHandler := handlers.NewStreamHandler(streamService)

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})

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
			r.Use(authHandler.RequireAuth)
			r.Get("/", dashboardHandler.GetMissions)
			r.Get("/{id}", dashboardHandler.GetMission)
		})

		r.Route("/satellites", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/", dashboardHandler.GetSatellites)
			r.Get("/{id}", dashboardHandler.GetSatellite)
		})

		r.Route("/hunoids", func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/", dashboardHandler.GetHunoids)
			r.Get("/{id}", dashboardHandler.GetHunoid)
		})

		// Stream routes
		r.Route("/streams", func(r chi.Router) {
			r.Get("/", streamHandler.GetStreams)
			r.Get("/stats", streamHandler.GetStreamStats)
			r.Get("/featured", streamHandler.GetFeaturedStreams)
			r.Get("/search", streamHandler.SearchStreams)
			r.Get("/{id}", streamHandler.GetStream)
			r.Route("/{id}", func(r chi.Router) {
				r.Use(authHandler.OptionalAuth)
				r.Post("/session", streamHandler.CreateStreamSession)
			})
		})
	})

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
