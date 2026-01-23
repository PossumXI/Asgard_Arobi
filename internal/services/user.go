// Package services provides business logic services for the API.
package services

import (
	"fmt"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
)

// UserService handles user-related business logic.
type UserService struct {
	userRepo         *repositories.UserRepository
	subscriptionRepo *repositories.SubscriptionRepository
}

// NewUserService creates a new user service.
func NewUserService(userRepo *repositories.UserRepository, subscriptionRepo *repositories.SubscriptionRepository) *UserService {
	return &UserService{
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
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
		user.FullName = &fullName
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
	// In production, store in a user_settings table
	// For now, this is a placeholder
	return nil
}
