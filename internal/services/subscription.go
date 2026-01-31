// Package services provides business logic services for the API.
package services

import (
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/repositories"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/subscription"
)

// SubscriptionService handles subscription-related business logic.
type SubscriptionService struct {
	subscriptionRepo *repositories.SubscriptionRepository
	userRepo         *repositories.UserRepository
	stripeService    *StripeService
}

// NewSubscriptionService creates a new subscription service.
func NewSubscriptionService(subscriptionRepo *repositories.SubscriptionRepository, userRepo *repositories.UserRepository) *SubscriptionService {
	stripeService := NewStripeService(subscriptionRepo, userRepo)
	return &SubscriptionService{
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
		stripeService:    stripeService,
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
	return s.stripeService.CreateCheckoutSession(userID, planID)
}

// CreatePortalSession creates a Stripe customer portal session.
func (s *SubscriptionService) CreatePortalSession(userID string) (map[string]interface{}, error) {
	return s.stripeService.CreatePortalSession(userID)
}

// CancelSubscription cancels a user's subscription.
func (s *SubscriptionService) CancelSubscription(userID string) error {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// If there's a Stripe subscription, cancel it at period end
	if sub.StripeSubscriptionID.Valid && sub.StripeSubscriptionID.String != "" {
		params := &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}
		_, err := subscription.Update(sub.StripeSubscriptionID.String, params)
		if err != nil {
			return fmt.Errorf("failed to cancel Stripe subscription: %w", err)
		}
	}

	// Update database status to canceling (will be cancelled at period end)
	sub.Status = "canceling"
	sub.UpdatedAt = time.Now()
	if err := s.subscriptionRepo.Update(sub); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// ReactivateSubscription reactivates a cancelled subscription.
func (s *SubscriptionService) ReactivateSubscription(userID string) error {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// If there's a Stripe subscription that's set to cancel, reactivate it
	if sub.StripeSubscriptionID.Valid && sub.StripeSubscriptionID.String != "" {
		params := &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(false),
		}
		_, err := subscription.Update(sub.StripeSubscriptionID.String, params)
		if err != nil {
			return fmt.Errorf("failed to reactivate Stripe subscription: %w", err)
		}
	}

	// Update database status back to active
	sub.Status = "active"
	sub.UpdatedAt = time.Now()
	if err := s.subscriptionRepo.Update(sub); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// ConstructWebhookEvent verifies and parses a Stripe webhook event.
func (s *SubscriptionService) ConstructWebhookEvent(payload []byte, signature string) (stripe.Event, error) {
	return s.stripeService.ConstructWebhookEvent(payload, signature)
}

// HandleWebhookEvent processes a verified Stripe event.
func (s *SubscriptionService) HandleWebhookEvent(event stripe.Event) error {
	return s.stripeService.HandleWebhook(event)
}

// HandleStripeWebhook processes Stripe webhook events.
func (s *SubscriptionService) HandleStripeWebhook(event stripe.Event) error {
	return s.stripeService.HandleWebhook(event)
}
