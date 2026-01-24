// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/asgard/pandora/internal/services"
	"github.com/stripe/stripe-go/v78/webhook"
)

// SubscriptionHandler handles subscription endpoints.
type SubscriptionHandler struct {
	subscriptionService *services.SubscriptionService
}

// NewSubscriptionHandler creates a new subscription handler.
func NewSubscriptionHandler(subscriptionService *services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subscriptionService: subscriptionService}
}

// GetPlans handles GET /api/subscriptions/plans
func (h *SubscriptionHandler) GetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.subscriptionService.GetPlans()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, plans)
}

// CreateCheckoutSession handles POST /api/subscriptions/checkout
func (h *SubscriptionHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		PlanID string `json:"planId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	session, err := h.subscriptionService.CreateCheckoutSession(userID, req.PlanID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, session)
}

// CreatePortalSession handles POST /api/subscriptions/portal
func (h *SubscriptionHandler) CreatePortalSession(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	session, err := h.subscriptionService.CreatePortalSession(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, session)
}

// CancelSubscription handles POST /api/subscriptions/cancel
func (h *SubscriptionHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.subscriptionService.CancelSubscription(userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Subscription cancelled"})
}

// ReactivateSubscription handles POST /api/subscriptions/reactivate
func (h *SubscriptionHandler) ReactivateSubscription(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.subscriptionService.ReactivateSubscription(userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Subscription reactivated"})
}

// HandleWebhook handles POST /api/webhooks/stripe
func (h *SubscriptionHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Error reading request body", "invalid_body")
		return
	}

	// Get the webhook secret from environment
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		jsonError(w, http.StatusInternalServerError, "Webhook secret not configured", "config_error")
		return
	}

	// Verify the webhook signature
	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sigHeader, webhookSecret)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid webhook signature", "invalid_signature")
		return
	}

	// Route to the appropriate handler based on event type
	if err := h.subscriptionService.HandleStripeWebhook(event); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "webhook_error")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"received": "true"})
}
