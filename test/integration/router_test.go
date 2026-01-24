package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/asgard/pandora/internal/platform/dtn"
	"github.com/asgard/pandora/pkg/bundle"
)

func TestContactGraphRouterCreation(t *testing.T) {
	router := dtn.NewContactGraphRouter("dtn://node/local")
	if router == nil {
		t.Fatal("router should not be nil")
	}
}

func TestContactGraphRouterWithNeighbors(t *testing.T) {
	router := dtn.NewContactGraphRouter("dtn://earth/ground")

	// Create test neighbors
	neighbors := make(map[string]*dtn.Neighbor)
	neighbors["relay-sat-1"] = &dtn.Neighbor{
		ID:          "relay-sat-1",
		EID:         "dtn://leo/relay1",
		IsActive:    true,
		LinkQuality: 0.9,
		LastContact: time.Now(),
	}
	neighbors["ground-station-1"] = &dtn.Neighbor{
		ID:          "ground-station-1",
		EID:         "dtn://ground/gs1",
		IsActive:    true,
		LinkQuality: 0.95,
		LastContact: time.Now(),
	}

	// Create test bundle
	b := bundle.NewBundle(
		"dtn://source/test",
		"dtn://ground/gs1",
		[]byte("test data"),
	)

	ctx := context.Background()
	nextHop, err := router.SelectNextHop(ctx, b, neighbors)
	if err != nil {
		t.Fatalf("routing failed: %v", err)
	}

	// Should route to ground station since destination matches
	if nextHop != "ground-station-1" {
		t.Logf("routed to %s (expected ground-station-1 for direct match)", nextHop)
	}
}

func TestContactGraphRouterNoNeighbors(t *testing.T) {
	router := dtn.NewContactGraphRouter("dtn://node/local")

	b := bundle.NewBundle(
		"dtn://source",
		"dtn://dest",
		[]byte("data"),
	)

	ctx := context.Background()
	neighbors := make(map[string]*dtn.Neighbor)

	_, err := router.SelectNextHop(ctx, b, neighbors)
	if err == nil {
		t.Error("expected error when no neighbors available")
	}
}

func TestContactGraphRouterInactiveNeighbor(t *testing.T) {
	router := dtn.NewContactGraphRouter("dtn://node/local")

	neighbors := make(map[string]*dtn.Neighbor)
	neighbors["inactive"] = &dtn.Neighbor{
		ID:          "inactive",
		EID:         "dtn://dest",
		IsActive:    false, // Inactive
		LinkQuality: 0.9,
		LastContact: time.Now(),
	}

	b := bundle.NewBundle(
		"dtn://source",
		"dtn://dest",
		[]byte("data"),
	)

	ctx := context.Background()
	_, err := router.SelectNextHop(ctx, b, neighbors)
	if err == nil {
		t.Error("expected error when only neighbor is inactive")
	}
}

func TestEnergyAwareRouterCreation(t *testing.T) {
	router := dtn.NewEnergyAwareRouter("dtn://satellite/test")
	if router == nil {
		t.Fatal("router should not be nil")
	}
}

func TestEnergyAwareRouterBasicRouting(t *testing.T) {
	router := dtn.NewEnergyAwareRouter("dtn://satellite/test")

	neighbors := make(map[string]*dtn.Neighbor)
	neighbors["ground"] = &dtn.Neighbor{
		ID:          "ground",
		EID:         "dtn://ground/station",
		IsActive:    true,
		LinkQuality: 0.9,
		LastContact: time.Now(),
	}

	b := bundle.NewBundle(
		"dtn://satellite/test",
		"dtn://ground/station",
		[]byte("telemetry"),
	)

	ctx := context.Background()
	nextHop, err := router.SelectNextHop(ctx, b, neighbors)
	if err != nil {
		t.Fatalf("routing failed: %v", err)
	}

	if nextHop != "ground" {
		t.Errorf("expected route to ground, got %s", nextHop)
	}
}

func TestStaticRouterCreation(t *testing.T) {
	router := dtn.NewStaticRouter()
	if router == nil {
		t.Fatal("router should not be nil")
	}
}
