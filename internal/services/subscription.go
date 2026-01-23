// Package services provides business logic services for the API.
package services

import (
	"fmt"

	"github.com/asgard/pandora/internal/repositories"
)

// SubscriptionService handles subscription-related business logic.
type SubscriptionService struct {
	subscriptionRepo *repositories.SubscriptionRepository
	userRepo        *repositories.UserRepository
}

// NewSubscriptionService creates a new subscription service.
func NewSubscriptionService(subscriptionRepo *repositories.SubscriptionRepository, userRepo *repositories.UserRepository) *SubscriptionService {
	return &SubscriptionService{
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
	}
}

// GetPlans returns available subscription plans.
func (s *SubscriptionService) GetPlans() ([]map[string]interface{}, error) {
	plans := []map[string]interface{}{
		{
			"id":          "plan_observer",
			"name":        "Observer",
			"tier":        "observer",
			"price":       9.99,
			"interval":    "month",
			"features":    []string{"Live civilian feeds", "Basic alerts", "Dashboard access"},
			"highlighted": false,
		},
		{
			"id":          "plan_supporter",
			"name":        "Supporter",
			"tier":        "supporter",
			"price":       29.99,
			"interval":    "month",
			"features":    []string{"All Observer features", "Military feeds", "Priority alerts", "Mission tracking"},
			"highlighted": true,
		},
		{
			"id":          "plan_commander",
			"name":        "Commander",
			"tier":        "commander",
			"price":       99.99,
			"interval":    "month",
			"features":    []string{"All Supporter features", "Interstellar feeds", "Mission requests", "API access"},
			"highlighted": false,
		},
	}
	return plans, nil
}

// CreateCheckoutSession creates a Stripe checkout session.
func (s *SubscriptionService) CreateCheckoutSession(userID, planID string) (map[string]interface{}, error) {
	// In production, integrate with Stripe API
	// For now, return a mock session URL
	return map[string]interface{}{
		"sessionId": "mock_session_" + planID,
		"sessionUrl": "https://checkout.stripe.com/mock/" + planID,
	}, nil
}

// CreatePortalSession creates a Stripe customer portal session.
func (s *SubscriptionService) CreatePortalSession(userID string) (map[string]interface{}, error) {
	// In production, integrate with Stripe API
	return map[string]interface{}{
		"portalUrl": "https://billing.stripe.com/mock/" + userID,
	}, nil
}

// CancelSubscription cancels a user's subscription.
func (s *SubscriptionService) CancelSubscription(userID string) error {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	sub.Status = "cancelled"
	if err := s.subscriptionRepo.Update(sub); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	return nil
}

// ReactivateSubscription reactivates a cancelled subscription.
func (s *SubscriptionService) ReactivateSubscription(userID string) error {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	sub.Status = "active"
	if err := s.subscriptionRepo.Update(sub); err != nil {
		return fmt.Errorf("failed to reactivate subscription: %w", err)
	}

	return nil
}
