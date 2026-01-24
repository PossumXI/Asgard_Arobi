// Package services provides business logic services for the API.
package services

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
)

// UserService handles user-related business logic.
type UserService struct {
	userRepo         *repositories.UserRepository
	subscriptionRepo *repositories.SubscriptionRepository
	notificationRepo *repositories.NotificationSettingsRepository
}

// NewUserService creates a new user service.
func NewUserService(
	userRepo *repositories.UserRepository,
	subscriptionRepo *repositories.SubscriptionRepository,
	notificationRepo *repositories.NotificationSettingsRepository,
) *UserService {
	return &UserService{
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		notificationRepo: notificationRepo,
	}
}

// GetProfile retrieves a user's profile.
func (s *UserService) GetProfile(userID string) (*db.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// UpdateProfile updates a user's profile.
func (s *UserService) UpdateProfile(userID string, updates map[string]interface{}) (*db.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Apply updates
	if fullName, ok := updates["fullName"].(string); ok {
		user.FullName = sql.NullString{String: fullName, Valid: fullName != ""}
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// GetSubscription retrieves a user's subscription.
func (s *UserService) GetSubscription(userID string) (*db.Subscription, error) {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return sub, nil
}

// UpdateNotificationSettings updates a user's notification settings.
func (s *UserService) UpdateNotificationSettings(userID string, settings map[string]interface{}) error {
	if s.notificationRepo == nil {
		return fmt.Errorf("notification settings repository not configured")
	}

	current, err := s.notificationRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to load notification settings: %w", err)
	}

	if value, ok := settings["emailAlerts"].(bool); ok {
		current.EmailAlerts = value
	}
	if value, ok := settings["pushNotifications"].(bool); ok {
		current.PushNotifications = value
	}
	if value, ok := settings["weeklyDigest"].(bool); ok {
		current.WeeklyDigest = value
	}
	if value, ok := settings["securityAlerts"].(bool); ok {
		current.SecurityAlerts = value
	}
	if value, ok := settings["missionUpdates"].(bool); ok {
		current.MissionUpdates = value
	}
	if value, ok := settings["systemStatus"].(bool); ok {
		current.SystemStatus = value
	}

	if err := s.notificationRepo.Upsert(current); err != nil {
		return fmt.Errorf("failed to update notification settings: %w", err)
	}

	return nil
}

// AdminUserUpdate describes admin-managed user fields.
type AdminUserUpdate struct {
	FullName         *string
	SubscriptionTier *string
	IsGovernment     *bool
}

// ListUsers returns recent users for admin dashboards.
func (s *UserService) ListUsers(limit int) ([]*db.User, error) {
	return s.userRepo.ListUsers(limit)
}

// UpdateAdminUser updates selected fields on a user for admin workflows.
func (s *UserService) UpdateAdminUser(userID string, updates AdminUserUpdate) (*db.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if updates.FullName != nil {
		name := strings.TrimSpace(*updates.FullName)
		user.FullName = sql.NullString{String: name, Valid: name != ""}
	}

	if updates.SubscriptionTier != nil {
		tier := strings.ToLower(strings.TrimSpace(*updates.SubscriptionTier))
		if tier == "free" {
			tier = "observer"
		}
		switch tier {
		case "observer", "supporter", "commander":
			user.SubscriptionTier = tier
		default:
			return nil, fmt.Errorf("invalid subscription tier")
		}
	}

	if updates.IsGovernment != nil {
		user.IsGovernment = *updates.IsGovernment
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}
