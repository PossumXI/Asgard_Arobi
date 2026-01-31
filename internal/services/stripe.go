package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v78"
	billingSession "github.com/stripe/stripe-go/v78/billingportal/session"
	checkoutSession "github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/invoice"
	"github.com/stripe/stripe-go/v78/subscription"
	"github.com/stripe/stripe-go/v78/webhook"
)

// StripeService handles Stripe payment operations.
type StripeService struct {
	subscriptionRepo *repositories.SubscriptionRepository
	userRepo         *repositories.UserRepository
	apiKey           string
	webhookSecret    string
}

// NewStripeService creates a new Stripe service.
func NewStripeService(
	subscriptionRepo *repositories.SubscriptionRepository,
	userRepo *repositories.UserRepository,
) *StripeService {
	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	if apiKey != "" {
		stripe.Key = apiKey
	}

	return &StripeService{
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
		apiKey:           apiKey,
		webhookSecret:    os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}
}

func getPlanPriceID(planID string) (string, bool) {
	switch planID {
	case "plan_observer":
		if val := os.Getenv("STRIPE_PRICE_OBSERVER"); val != "" {
			return val, true
		}
	case "plan_supporter":
		if val := os.Getenv("STRIPE_PRICE_SUPPORTER"); val != "" {
			return val, true
		}
	case "plan_commander":
		if val := os.Getenv("STRIPE_PRICE_COMMANDER"); val != "" {
			return val, true
		}
	}
	return "", false
}

// CreateCheckoutSession creates a Stripe checkout session.
func (s *StripeService) CreateCheckoutSession(userID, planID string) (map[string]interface{}, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("stripe is not configured")
	}

	// Get user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get or create Stripe customer
	customerID, err := s.getOrCreateCustomer(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create customer: %w", err)
	}

	// Get price ID for plan
	priceID, ok := getPlanPriceID(planID)
	if !ok {
		return nil, fmt.Errorf("invalid plan ID: %s", planID)
	}

	// Determine tier from plan ID
	tier := "observer"
	switch planID {
	case "plan_supporter":
		tier = "supporter"
	case "plan_commander":
		tier = "commander"
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(os.Getenv("STRIPE_SUCCESS_URL") + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(os.Getenv("STRIPE_CANCEL_URL")),
		Metadata: map[string]string{
			"user_id": userID,
			"plan_id": planID,
			"tier":    tier,
		},
	}

	sess, err := checkoutSession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return map[string]interface{}{
		"sessionId":  sess.ID,
		"sessionUrl": sess.URL,
	}, nil
}

// CreatePortalSession creates a Stripe customer portal session.
func (s *StripeService) CreatePortalSession(userID string) (map[string]interface{}, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("stripe is not configured")
	}

	// Get user subscription
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	if sub.StripeCustomerID.String == "" {
		return nil, fmt.Errorf("no Stripe customer ID found")
	}

	// Create portal session
	portalParams := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(sub.StripeCustomerID.String),
		ReturnURL: stripe.String(os.Getenv("STRIPE_PORTAL_RETURN_URL")),
	}

	portalSess, err := billingSession.New(portalParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create portal session: %w", err)
	}

	return map[string]interface{}{
		"portalUrl": portalSess.URL,
	}, nil
}

// CancelSubscription cancels a user's subscription at period end.
func (s *StripeService) CancelSubscription(userID string) error {
	if s.apiKey == "" {
		return fmt.Errorf("stripe is not configured")
	}

	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}
	if sub.StripeSubscriptionID.String == "" {
		return fmt.Errorf("no Stripe subscription ID found")
	}

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	updated, err := subscription.Update(sub.StripeSubscriptionID.String, params)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	sub.Status = "canceled"
	sub.CurrentPeriodEnd = sql.NullTime{Time: time.Unix(updated.CurrentPeriodEnd, 0).UTC(), Valid: true}
	sub.UpdatedAt = time.Now().UTC()
	if err := s.subscriptionRepo.Update(sub); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// ConstructWebhookEvent verifies and constructs a Stripe event from the request payload.
func (s *StripeService) ConstructWebhookEvent(payload []byte, signature string) (stripe.Event, error) {
	if s.webhookSecret == "" {
		return stripe.Event{}, fmt.Errorf("stripe webhook secret not configured")
	}
	return webhook.ConstructEvent(payload, signature, s.webhookSecret)
}

// HandleWebhook processes Stripe webhook events.
func (s *StripeService) HandleWebhook(event stripe.Event) error {
	if s.apiKey == "" {
		return fmt.Errorf("stripe is not configured")
	}

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(event)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(event)
	case "invoice.payment_succeeded":
		return s.handlePaymentSucceeded(event)
	case "invoice.payment_failed":
		return s.handlePaymentFailed(event)
	default:
		return nil // Ignore unknown events
	}
}

// getOrCreateCustomer gets or creates a Stripe customer.
func (s *StripeService) getOrCreateCustomer(user *db.User) (string, error) {
	// Check if user already has a Stripe customer ID
	sub, err := s.subscriptionRepo.GetByUserID(user.ID.String())
	if err == nil && sub.StripeCustomerID.String != "" {
		return sub.StripeCustomerID.String, nil
	}

	// Create new customer
	params := &stripe.CustomerParams{
		Email: stripe.String(user.Email),
		Metadata: map[string]string{
			"user_id": user.ID.String(),
		},
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", err
	}

	return cust.ID, nil
}

// handleCheckoutCompleted handles checkout.session.completed event.
func (s *StripeService) handleCheckoutCompleted(event stripe.Event) error {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return fmt.Errorf("failed to parse checkout session: %w", err)
	}

	userID := sess.Metadata["user_id"]
	tier := sess.Metadata["tier"]

	// Get subscription from Stripe
	subscriptionID := sess.Subscription.ID
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Create or update subscription in database
	uid, _ := uuid.Parse(userID)
	dbSub := &db.Subscription{
		ID:                   uuid.New(),
		UserID:               uid,
		StripeSubscriptionID: sql.NullString{String: subscriptionID, Valid: true},
		StripeCustomerID:     sql.NullString{String: sess.Customer.ID, Valid: true},
		Tier:                 sql.NullString{String: tier, Valid: true},
		Status:               string(sub.Status),
		CurrentPeriodStart:   sql.NullTime{Time: time.Unix(sub.CurrentPeriodStart, 0), Valid: true},
		CurrentPeriodEnd:     sql.NullTime{Time: time.Unix(sub.CurrentPeriodEnd, 0), Valid: true},
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	existing, _ := s.subscriptionRepo.GetByUserID(userID)
	if existing != nil && existing.ID != uuid.Nil {
		dbSub.ID = existing.ID
		return s.subscriptionRepo.Update(dbSub)
	}

	return s.subscriptionRepo.Create(dbSub)
}

// handleSubscriptionUpdated handles customer.subscription.updated event.
func (s *StripeService) handleSubscriptionUpdated(event stripe.Event) error {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	// Find subscription by Stripe subscription ID
	sub, err := s.subscriptionRepo.GetByStripeSubscriptionID(subscription.ID)
	if err != nil {
		// Subscription might not exist yet, try to create it
		if subscription.Customer == nil {
			return fmt.Errorf("subscription has no customer")
		}

		// Find user by Stripe customer ID
		userSub, err := s.subscriptionRepo.GetByStripeCustomerID(subscription.Customer.ID)
		if err != nil {
			return fmt.Errorf("subscription not found: %w", err)
		}

		uid, _ := uuid.Parse(userSub.UserID.String())
		dbSub := &db.Subscription{
			ID:                   uuid.New(),
			UserID:               uid,
			StripeSubscriptionID: sql.NullString{String: subscription.ID, Valid: true},
			StripeCustomerID:     sql.NullString{String: subscription.Customer.ID, Valid: true},
			Tier:                 sql.NullString{String: extractTierFromSubscription(&subscription), Valid: true},
			Status:               string(subscription.Status),
			CurrentPeriodStart:   sql.NullTime{Time: time.Unix(subscription.CurrentPeriodStart, 0), Valid: true},
			CurrentPeriodEnd:     sql.NullTime{Time: time.Unix(subscription.CurrentPeriodEnd, 0), Valid: true},
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		return s.subscriptionRepo.Create(dbSub)
	}

	// Update existing subscription
	sub.Status = string(subscription.Status)
	sub.CurrentPeriodStart = sql.NullTime{Time: time.Unix(subscription.CurrentPeriodStart, 0), Valid: true}
	sub.CurrentPeriodEnd = sql.NullTime{Time: time.Unix(subscription.CurrentPeriodEnd, 0), Valid: true}
	sub.UpdatedAt = time.Now()

	// Update tier if changed
	if tier := extractTierFromSubscription(&subscription); tier != "" {
		sub.Tier = sql.NullString{String: tier, Valid: true}
		if err := s.updateUserTier(sub.UserID, tier); err != nil {
			return err
		}
	}

	return s.subscriptionRepo.Update(sub)
}

// handleSubscriptionDeleted handles customer.subscription.deleted event.
func (s *StripeService) handleSubscriptionDeleted(event stripe.Event) error {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	// Find subscription by Stripe subscription ID
	sub, err := s.subscriptionRepo.GetByStripeSubscriptionID(subscription.ID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	// Update status to cancelled
	sub.Status = "cancelled"
	sub.UpdatedAt = time.Now()
	if err := s.subscriptionRepo.Update(sub); err != nil {
		return err
	}
	return s.updateUserTier(sub.UserID, "free")
}

// handlePaymentSucceeded handles invoice.payment_succeeded event.
func (s *StripeService) handlePaymentSucceeded(event stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("failed to parse invoice: %w", err)
	}

	if invoice.Subscription == nil {
		return nil // Not a subscription invoice
	}

	// Find subscription and update status
	sub, err := s.subscriptionRepo.GetByStripeSubscriptionID(invoice.Subscription.ID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	// Update subscription status to active if payment succeeded
	if sub.Status != "active" {
		sub.Status = "active"
		sub.UpdatedAt = time.Now()
		if err := s.subscriptionRepo.Update(sub); err != nil {
			return err
		}
	}

	if tier := sub.Tier.String; tier != "" {
		if err := s.updateUserTier(sub.UserID, tier); err != nil {
			return err
		}
	}

	// Get user and send confirmation email
	user, err := s.userRepo.GetByID(sub.UserID.String())
	if err == nil && user != nil {
		emailService := NewEmailService()
		if tier := sub.Tier.String; tier != "" {
			if err := emailService.SendSubscriptionConfirmation(user.Email, tier); err != nil {
				// Log but don't fail the webhook
				fmt.Printf("Failed to send confirmation email: %v\n", err)
			}
		}
	}

	return nil
}

// extractTierFromSubscription extracts the tier from a Stripe subscription's price metadata.
func extractTierFromSubscription(sub *stripe.Subscription) string {
	if sub == nil || len(sub.Items.Data) == 0 {
		return ""
	}

	// Check subscription metadata first
	if tier, ok := sub.Metadata["tier"]; ok && tier != "" {
		return tier
	}

	// Try to extract tier from price ID
	item := sub.Items.Data[0]
	if item.Price == nil {
		return ""
	}

	priceID := item.Price.ID
	if priceID == mustPlanPriceID("plan_observer") {
		return "observer"
	}
	if priceID == mustPlanPriceID("plan_supporter") {
		return "supporter"
	}
	if priceID == mustPlanPriceID("plan_commander") {
		return "commander"
	}

	// Check price metadata as fallback
	if item.Price.Metadata != nil {
		if tier, ok := item.Price.Metadata["tier"]; ok {
			return tier
		}
	}

	return ""
}

func mustPlanPriceID(planID string) string {
	if priceID, ok := getPlanPriceID(planID); ok {
		return priceID
	}
	return ""
}

// handlePaymentFailed handles invoice.payment_failed event.
func (s *StripeService) handlePaymentFailed(event stripe.Event) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("failed to parse invoice: %w", err)
	}

	if inv.Subscription == nil {
		return nil // Not a subscription invoice
	}

	// Get full invoice from Stripe
	invoiceID := inv.ID
	invoice, err := invoice.Get(invoiceID, nil)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// Find subscription
	sub, err := s.subscriptionRepo.GetByStripeSubscriptionID(invoice.Subscription.ID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	// Update subscription status based on Stripe's retry logic
	// Stripe will retry payments, so we mark as past_due
	sub.Status = "past_due"
	sub.UpdatedAt = time.Now()
	if err := s.subscriptionRepo.Update(sub); err != nil {
		return err
	}

	// Get user and send notification email
	user, err := s.userRepo.GetByID(sub.UserID.String())
	if err == nil && user != nil {
		emailService := NewEmailService()
		subject := "Payment Failed - ASGARD Subscription"
		message := fmt.Sprintf("Your payment for ASGARD subscription has failed. Please update your payment method to continue service.")
		if err := emailService.SendEmail(user.Email, subject, message); err != nil {
			// Log but don't fail the webhook
			fmt.Printf("Failed to send payment failure email: %v\n", err)
		}
	}

	return nil
}

func (s *StripeService) updateUserTier(userID uuid.UUID, tier string) error {
	user, err := s.userRepo.GetByID(userID.String())
	if err != nil {
		return err
	}
	user.SubscriptionTier = tier
	return s.userRepo.Update(user)
}
