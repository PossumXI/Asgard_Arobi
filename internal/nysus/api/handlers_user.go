package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/asgard/pandora/internal/services"
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
	ID                 string `json:"id"`
	UserID             string `json:"userId"`
	Tier               string `json:"tier"`
	Status             string `json:"status"`
	CurrentPeriodStart string `json:"currentPeriodStart"`
	CurrentPeriodEnd   string `json:"currentPeriodEnd"`
	CancelAtPeriodEnd  bool   `json:"cancelAtPeriodEnd"`
	CreatedAt          string `json:"createdAt"`
}

// handleUserProfile handles GET/PATCH /api/user/profile
func (s *Server) handleUserProfile(w http.ResponseWriter, r *http.Request) {
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not configured", "DB_NOT_CONFIGURED")
		return
	}

	userID, err := s.requireUserID(r)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err.Error(), "UNAUTHORIZED")
		return
	}

	repo := repositories.NewUserRepository(s.pgDB)
	user, err := repo.GetByID(userID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "User not found", "USER_NOT_FOUND")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, http.StatusOK, buildUserResponse(user))

	case http.MethodPatch:
		var updates struct {
			FullName string `json:"fullName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
			return
		}

		if updates.FullName != "" {
			user.FullName = sql.NullString{String: updates.FullName, Valid: true}
		}

		if err := repo.Update(user); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to update user", "UPDATE_FAILED")
			return
		}

		s.writeJSON(w, http.StatusOK, buildUserResponse(user))

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
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not configured", "DB_NOT_CONFIGURED")
		return
	}

	userID, err := s.requireUserID(r)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err.Error(), "UNAUTHORIZED")
		return
	}

	subRepo := repositories.NewSubscriptionRepository(s.pgDB)
	sub, err := subRepo.GetByUserID(userID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Subscription not found", "SUBSCRIPTION_NOT_FOUND")
		return
	}

	id := ""
	if sub.ID != uuid.Nil {
		id = sub.ID.String()
	}
	tier := sub.Tier.String
	if tier == "" {
		tier = "free"
	}
	status := sub.Status
	if status == "" {
		status = "active"
	}
	periodStart := ""
	if sub.CurrentPeriodStart.Valid {
		periodStart = sub.CurrentPeriodStart.Time.Format(time.RFC3339)
	}
	periodEnd := ""
	if sub.CurrentPeriodEnd.Valid {
		periodEnd = sub.CurrentPeriodEnd.Time.Format(time.RFC3339)
	}
	createdAt := sub.CreatedAt.Format(time.RFC3339)

	s.writeJSON(w, http.StatusOK, SubscriptionResponse{
		ID:                 id,
		UserID:             userID,
		Tier:               tier,
		Status:             status,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		CancelAtPeriodEnd:  false,
		CreatedAt:          createdAt,
	})
}

// handleNotificationSettings handles GET/PATCH /api/user/notifications
func (s *Server) handleNotificationSettings(w http.ResponseWriter, r *http.Request) {
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not configured", "DB_NOT_CONFIGURED")
		return
	}

	userID, err := s.requireUserID(r)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err.Error(), "UNAUTHORIZED")
		return
	}

	repo := repositories.NewNotificationSettingsRepository(s.pgDB)

	switch r.Method {
	case http.MethodGet:
		settings, err := repo.GetByUserID(userID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to load notification settings", "SETTINGS_ERROR")
			return
		}
		s.writeJSON(w, http.StatusOK, NotificationSettings{
			EmailAlerts:       settings.EmailAlerts,
			PushNotifications: settings.PushNotifications,
			WeeklyDigest:      settings.WeeklyDigest,
			SecurityAlerts:    settings.SecurityAlerts,
			MissionUpdates:    settings.MissionUpdates,
			SystemStatus:      settings.SystemStatus,
		})

	case http.MethodPatch:
		var settings NotificationSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
			return
		}
		payload := &db.NotificationSettings{
			UserID:            uuid.MustParse(userID),
			EmailAlerts:       settings.EmailAlerts,
			PushNotifications: settings.PushNotifications,
			WeeklyDigest:      settings.WeeklyDigest,
			SecurityAlerts:    settings.SecurityAlerts,
			MissionUpdates:    settings.MissionUpdates,
			SystemStatus:      settings.SystemStatus,
			UpdatedAt:         time.Now().UTC(),
		}
		if err := repo.Upsert(payload); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to save settings", "SETTINGS_ERROR")
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

	plans, err := loadSubscriptionPlans()
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable, err.Error(), "PLANS_NOT_CONFIGURED")
		return
	}

	s.writeJSON(w, http.StatusOK, plans)
}

// handleCheckout handles POST /api/subscriptions/checkout
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not configured", "DB_NOT_CONFIGURED")
		return
	}

	userID, err := s.requireUserID(r)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err.Error(), "UNAUTHORIZED")
		return
	}

	var req struct {
		PlanID string `json:"planId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	stripeService := services.NewStripeService(
		repositories.NewSubscriptionRepository(s.pgDB),
		repositories.NewUserRepository(s.pgDB),
	)
	result, err := stripeService.CreateCheckoutSession(userID, req.PlanID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "CHECKOUT_FAILED")
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// handleBillingPortal handles POST /api/subscriptions/portal
func (s *Server) handleBillingPortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not configured", "DB_NOT_CONFIGURED")
		return
	}

	userID, err := s.requireUserID(r)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err.Error(), "UNAUTHORIZED")
		return
	}

	stripeService := services.NewStripeService(
		repositories.NewSubscriptionRepository(s.pgDB),
		repositories.NewUserRepository(s.pgDB),
	)
	result, err := stripeService.CreatePortalSession(userID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "PORTAL_FAILED")
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// handleCancelSubscription handles POST /api/subscriptions/cancel
func (s *Server) handleCancelSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not configured", "DB_NOT_CONFIGURED")
		return
	}

	userID, err := s.requireUserID(r)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err.Error(), "UNAUTHORIZED")
		return
	}

	stripeService := services.NewStripeService(
		repositories.NewSubscriptionRepository(s.pgDB),
		repositories.NewUserRepository(s.pgDB),
	)
	if err := stripeService.CancelSubscription(userID); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "CANCEL_FAILED")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Subscription will be cancelled at period end",
	})
}

func (s *Server) requireUserID(r *http.Request) (string, error) {
	rawToken := extractToken(r)
	if rawToken == "" {
		return "", fmt.Errorf("missing token")
	}
	userID, _, _, _, err := parseJWTClaims(rawToken)
	if err != nil || userID == "" {
		return "", fmt.Errorf("invalid token")
	}
	return userID, nil
}

func buildUserResponse(user *db.User) UserResponse {
	fullName := ""
	if user.FullName.Valid {
		fullName = user.FullName.String
	}
	lastLogin := ""
	if user.LastLogin.Valid {
		lastLogin = user.LastLogin.Time.Format(time.RFC3339)
	}

	response := UserResponse{
		ID:               user.ID.String(),
		Email:            user.Email,
		FullName:         fullName,
		SubscriptionTier: user.SubscriptionTier,
		IsGovernment:     user.IsGovernment,
		EmailVerified:    user.EmailVerified,
		CreatedAt:        user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        user.UpdatedAt.Format(time.RFC3339),
	}
	if lastLogin != "" {
		response.LastLogin = &lastLogin
	}
	return response
}

func loadSubscriptionPlans() ([]SubscriptionPlan, error) {
	raw := os.Getenv("SUBSCRIPTION_PLANS_JSON")
	if raw == "" {
		return nil, fmt.Errorf("SUBSCRIPTION_PLANS_JSON is not configured")
	}
	var plans []SubscriptionPlan
	if err := json.Unmarshal([]byte(raw), &plans); err != nil {
		return nil, fmt.Errorf("invalid SUBSCRIPTION_PLANS_JSON: %w", err)
	}
	return plans, nil
}
