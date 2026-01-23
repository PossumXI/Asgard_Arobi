package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// NotificationSettings represents user notification preferences.
type NotificationSettings struct {
	EmailAlerts       bool `json:"emailAlerts"`
	PushNotifications bool `json:"pushNotifications"`
	WeeklyDigest      bool `json:"weeklyDigest"`
	SecurityAlerts    bool `json:"securityAlerts"`
	MissionUpdates    bool `json:"missionUpdates"`
	SystemStatus      bool `json:"systemStatus"`
}

// SubscriptionResponse represents subscription data.
type SubscriptionResponse struct {
	ID                 string  `json:"id"`
	UserID             string  `json:"userId"`
	Tier               string  `json:"tier"`
	Status             string  `json:"status"`
	CurrentPeriodStart string  `json:"currentPeriodStart"`
	CurrentPeriodEnd   string  `json:"currentPeriodEnd"`
	CancelAtPeriodEnd  bool    `json:"cancelAtPeriodEnd"`
	CreatedAt          string  `json:"createdAt"`
}

// handleUserProfile handles GET/PATCH /api/user/profile
func (s *Server) handleUserProfile(w http.ResponseWriter, r *http.Request) {
	// Extract user from token (simplified for demo)
	switch r.Method {
	case http.MethodGet:
		// Return demo user profile
		s.writeJSON(w, http.StatusOK, UserResponse{
			ID:               uuid.New().String(),
			Email:            "demo@asgard.dev",
			FullName:         "Demo User",
			SubscriptionTier: "supporter",
			IsGovernment:     false,
			EmailVerified:    true,
			CreatedAt:        time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			UpdatedAt:        time.Now().Format(time.RFC3339),
		})

	case http.MethodPatch:
		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
			return
		}

		// In production, update database
		s.writeJSON(w, http.StatusOK, UserResponse{
			ID:               uuid.New().String(),
			Email:            "demo@asgard.dev",
			FullName:         updates["fullName"].(string),
			SubscriptionTier: "supporter",
			IsGovernment:     false,
			EmailVerified:    true,
			CreatedAt:        time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			UpdatedAt:        time.Now().Format(time.RFC3339),
		})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
	}
}

// handleUserSubscription handles GET /api/user/subscription
func (s *Server) handleUserSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	now := time.Now().UTC()
	periodStart := now.Add(-15 * 24 * time.Hour)
	periodEnd := now.Add(15 * 24 * time.Hour)

	s.writeJSON(w, http.StatusOK, SubscriptionResponse{
		ID:                 uuid.New().String(),
		UserID:             uuid.New().String(),
		Tier:               "supporter",
		Status:             "active",
		CurrentPeriodStart: periodStart.Format(time.RFC3339),
		CurrentPeriodEnd:   periodEnd.Format(time.RFC3339),
		CancelAtPeriodEnd:  false,
		CreatedAt:          periodStart.Format(time.RFC3339),
	})
}

// handleNotificationSettings handles GET/PATCH /api/user/notifications
func (s *Server) handleNotificationSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, http.StatusOK, NotificationSettings{
			EmailAlerts:       true,
			PushNotifications: true,
			WeeklyDigest:      false,
			SecurityAlerts:    true,
			MissionUpdates:    true,
			SystemStatus:      true,
		})

	case http.MethodPatch:
		var settings NotificationSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
			return
		}
		s.writeJSON(w, http.StatusOK, settings)

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
	}
}

// SubscriptionPlan represents a subscription plan.
type SubscriptionPlan struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Tier          string   `json:"tier"`
	Price         int      `json:"price"`
	Interval      string   `json:"interval"`
	Features      []string `json:"features"`
	Highlighted   bool     `json:"highlighted,omitempty"`
	StripePriceID string   `json:"stripePriceId"`
}

// handleSubscriptionPlans handles GET /api/subscriptions/plans
func (s *Server) handleSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	plans := []SubscriptionPlan{
		{
			ID:       "observer-monthly",
			Name:     "Observer",
			Tier:     "observer",
			Price:    999, // $9.99
			Interval: "month",
			Features: []string{
				"Access to public satellite feeds",
				"Real-time alerts",
				"Basic dashboard",
				"Email support",
			},
			StripePriceID: "price_observer_monthly",
		},
		{
			ID:       "supporter-monthly",
			Name:     "Supporter",
			Tier:     "supporter",
			Price:    2999, // $29.99
			Interval: "month",
			Features: []string{
				"Everything in Observer",
				"HD streaming quality",
				"Historical data access",
				"API access",
				"Priority support",
			},
			Highlighted:   true,
			StripePriceID: "price_supporter_monthly",
		},
		{
			ID:       "commander-monthly",
			Name:     "Commander",
			Tier:     "commander",
			Price:    9999, // $99.99
			Interval: "month",
			Features: []string{
				"Everything in Supporter",
				"4K streaming",
				"Unlimited API calls",
				"Custom integrations",
				"Dedicated support",
				"Early access features",
			},
			StripePriceID: "price_commander_monthly",
		},
	}

	s.writeJSON(w, http.StatusOK, plans)
}

// handleCheckout handles POST /api/subscriptions/checkout
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	var req struct {
		PlanID string `json:"planId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// In production, this would create a Stripe checkout session
	s.writeJSON(w, http.StatusOK, map[string]string{
		"sessionId":  "cs_test_" + uuid.New().String()[:8],
		"sessionUrl": "https://checkout.stripe.com/pay/cs_test_example",
	})
}

// handleBillingPortal handles POST /api/subscriptions/portal
func (s *Server) handleBillingPortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	// In production, this would create a Stripe billing portal session
	s.writeJSON(w, http.StatusOK, map[string]string{
		"portalUrl": "https://billing.stripe.com/session/test_example",
	})
}

// handleCancelSubscription handles POST /api/subscriptions/cancel
func (s *Server) handleCancelSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Subscription will be cancelled at period end",
	})
}
