// Package repositories provides data access layer for database operations.
package repositories

import (
	"database/sql"
	"fmt"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// SubscriptionRepository handles subscription database operations.
type SubscriptionRepository struct {
	db *db.PostgresDB
}

// NewSubscriptionRepository creates a new subscription repository.
func NewSubscriptionRepository(pgDB *db.PostgresDB) *SubscriptionRepository {
	return &SubscriptionRepository{db: pgDB}
}

// GetByUserID retrieves a subscription by user ID.
func (r *SubscriptionRepository) GetByUserID(userID string) (*db.Subscription, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT id, user_id, stripe_subscription_id, stripe_customer_id,
		       tier, status, current_period_start, current_period_end,
		       created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	sub := &db.Subscription{}
	var stripeSubID sql.NullString
	var stripeCustID sql.NullString
	var tier sql.NullString
	var periodStart sql.NullTime
	var periodEnd sql.NullTime

	err = r.db.QueryRow(query, uid).Scan(
		&sub.ID,
		&sub.UserID,
		&stripeSubID,
		&stripeCustID,
		&tier,
		&sub.Status,
		&periodStart,
		&periodEnd,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Return a default free subscription
		return &db.Subscription{
			UserID: uid,
			Status: "active",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query subscription: %w", err)
	}

	sub.StripeSubscriptionID = stripeSubID
	sub.StripeCustomerID = stripeCustID
	sub.Tier = tier
	sub.CurrentPeriodStart = periodStart
	sub.CurrentPeriodEnd = periodEnd

	return sub, nil
}

// Create creates a new subscription.
func (r *SubscriptionRepository) Create(sub *db.Subscription) error {
	query := `
		INSERT INTO subscriptions (id, user_id, stripe_subscription_id, stripe_customer_id,
		                         tier, status, current_period_start, current_period_end,
		                         created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(query,
		sub.ID,
		sub.UserID,
		sub.StripeSubscriptionID,
		sub.StripeCustomerID,
		sub.Tier,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	return nil
}

// Update updates an existing subscription.
func (r *SubscriptionRepository) Update(sub *db.Subscription) error {
	query := `
		UPDATE subscriptions
		SET stripe_subscription_id = $3, stripe_customer_id = $4,
		    tier = $5, status = $6, current_period_start = $7,
		    current_period_end = $8, updated_at = $9
		WHERE id = $1 AND user_id = $2
	`

	_, err := r.db.Exec(query,
		sub.ID,
		sub.UserID,
		sub.StripeSubscriptionID,
		sub.StripeCustomerID,
		sub.Tier,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// GetByStripeSubscriptionID retrieves a subscription by Stripe subscription ID.
func (r *SubscriptionRepository) GetByStripeSubscriptionID(stripeSubID string) (*db.Subscription, error) {
	query := `
		SELECT id, user_id, stripe_subscription_id, stripe_customer_id,
		       tier, status, current_period_start, current_period_end,
		       created_at, updated_at
		FROM subscriptions
		WHERE stripe_subscription_id = $1
		LIMIT 1
	`

	sub := &db.Subscription{}
	var stripeSubscriptionID sql.NullString
	var stripeCustID sql.NullString
	var tier sql.NullString
	var periodStart sql.NullTime
	var periodEnd sql.NullTime

	err := r.db.QueryRow(query, stripeSubID).Scan(
		&sub.ID,
		&sub.UserID,
		&stripeSubscriptionID,
		&stripeCustID,
		&tier,
		&sub.Status,
		&periodStart,
		&periodEnd,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subscription not found for stripe_subscription_id: %s", stripeSubID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query subscription: %w", err)
	}

	sub.StripeSubscriptionID = stripeSubscriptionID
	sub.StripeCustomerID = stripeCustID
	sub.Tier = tier
	sub.CurrentPeriodStart = periodStart
	sub.CurrentPeriodEnd = periodEnd

	return sub, nil
}

// GetByStripeCustomerID retrieves a subscription by Stripe customer ID.
func (r *SubscriptionRepository) GetByStripeCustomerID(stripeCustomerID string) (*db.Subscription, error) {
	query := `
		SELECT id, user_id, stripe_subscription_id, stripe_customer_id,
		       tier, status, current_period_start, current_period_end,
		       created_at, updated_at
		FROM subscriptions
		WHERE stripe_customer_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	sub := &db.Subscription{}
	var stripeSubscriptionID sql.NullString
	var stripeCustID sql.NullString
	var tier sql.NullString
	var periodStart sql.NullTime
	var periodEnd sql.NullTime

	err := r.db.QueryRow(query, stripeCustomerID).Scan(
		&sub.ID,
		&sub.UserID,
		&stripeSubscriptionID,
		&stripeCustID,
		&tier,
		&sub.Status,
		&periodStart,
		&periodEnd,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subscription not found for stripe_customer_id: %s", stripeCustomerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query subscription: %w", err)
	}

	sub.StripeSubscriptionID = stripeSubscriptionID
	sub.StripeCustomerID = stripeCustID
	sub.Tier = tier
	sub.CurrentPeriodStart = periodStart
	sub.CurrentPeriodEnd = periodEnd

	return sub, nil
}
