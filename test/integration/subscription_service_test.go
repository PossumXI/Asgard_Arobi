package integration_test

import (
	"testing"

	"github.com/asgard/pandora/internal/services"
)

func TestSubscriptionServiceCreation(t *testing.T) {
	subService := services.NewSubscriptionService(nil, nil)
	if subService == nil {
		t.Fatal("subscription service should not be nil")
	}
}

func TestSubscriptionPlans(t *testing.T) {
	subService := services.NewSubscriptionService(nil, nil)

	plans, err := subService.GetPlans()
	if err != nil {
		t.Fatalf("failed to get plans: %v", err)
	}

	if len(plans) != 3 {
		t.Errorf("expected 3 subscription plans, got %d", len(plans))
	}

	// Verify each plan has required fields
	for _, plan := range plans {
		if plan["id"] == nil || plan["id"] == "" {
			t.Error("plan missing id")
		}
		if plan["name"] == nil || plan["name"] == "" {
			t.Error("plan missing name")
		}
		if plan["tier"] == nil || plan["tier"] == "" {
			t.Error("plan missing tier")
		}
		if plan["price"] == nil {
			t.Error("plan missing price")
		}
		if plan["features"] == nil {
			t.Error("plan missing features")
		}
	}
}

func TestSubscriptionPlanTiers(t *testing.T) {
	subService := services.NewSubscriptionService(nil, nil)

	plans, err := subService.GetPlans()
	if err != nil {
		t.Fatalf("failed to get plans: %v", err)
	}

	expectedTiers := map[string]bool{
		"observer":  false,
		"supporter": false,
		"commander": false,
	}

	for _, plan := range plans {
		tier, ok := plan["tier"].(string)
		if ok {
			if _, exists := expectedTiers[tier]; exists {
				expectedTiers[tier] = true
			}
		}
	}

	for tier, found := range expectedTiers {
		if !found {
			t.Errorf("tier %s not found in plans", tier)
		}
	}
}

func TestSubscriptionPlanPrices(t *testing.T) {
	subService := services.NewSubscriptionService(nil, nil)

	plans, err := subService.GetPlans()
	if err != nil {
		t.Fatalf("failed to get plans: %v", err)
	}

	for _, plan := range plans {
		price, ok := plan["price"].(float64)
		if !ok {
			t.Errorf("plan %v has non-numeric price", plan["name"])
			continue
		}
		if price < 0 {
			t.Errorf("plan %v has negative price: %f", plan["name"], price)
		}
	}
}

func TestSubscriptionPlanFeatures(t *testing.T) {
	subService := services.NewSubscriptionService(nil, nil)

	plans, err := subService.GetPlans()
	if err != nil {
		t.Fatalf("failed to get plans: %v", err)
	}

	for _, plan := range plans {
		features, ok := plan["features"].([]string)
		if !ok {
			t.Errorf("plan %v features is not a string slice", plan["name"])
			continue
		}
		if len(features) == 0 {
			t.Errorf("plan %v has no features", plan["name"])
		}
	}
}
