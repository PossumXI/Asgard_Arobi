package main

import (
	"context"
	"log"
	"time"

	"github.com/asgard/pandora/internal/platform/dtn"
	"github.com/asgard/pandora/pkg/bundle"
)

func main() {
	log.Println("ASGARD Sat_Net Verification")

	router, err := dtn.NewRLRoutingAgent("dtn://earth/ground001", "models/rl_router.json")
	if err != nil {
		log.Fatalf("Failed to load RL router: %v", err)
	}

	router.UpdateEnergy("ground_earth_1", 80)
	router.UpdateEnergy("ground_earth_2", 75)
	router.UpdateEnergy("sat_leo_001", 55)

	neighbors := map[string]*dtn.Neighbor{
		"ground_earth_1": {
			ID:           "ground_earth_1",
			EID:          "dtn://earth/ground001",
			LinkQuality:  0.95,
			IsActive:     true,
			Latency:      50 * time.Millisecond,
			Bandwidth:    10_000_000,
			ContactStart: time.Now().UTC(),
			ContactEnd:   time.Now().UTC().Add(2 * time.Hour),
		},
		"ground_earth_2": {
			ID:           "ground_earth_2",
			EID:          "dtn://earth/ground002",
			LinkQuality:  0.9,
			IsActive:     true,
			Latency:      75 * time.Millisecond,
			Bandwidth:    8_000_000,
			ContactStart: time.Now().UTC(),
			ContactEnd:   time.Now().UTC().Add(2 * time.Hour),
		},
		"sat_leo_001": {
			ID:           "sat_leo_001",
			EID:          "dtn://leo/sat001",
			LinkQuality:  0.85,
			IsActive:     true,
			Latency:      25 * time.Millisecond,
			Bandwidth:    5_000_000,
			ContactStart: time.Now().UTC(),
			ContactEnd:   time.Now().UTC().Add(30 * time.Minute),
		},
	}

	testBundle := bundle.NewBundle(
		"dtn://earth/nysus",
		"dtn://earth/ground001",
		[]byte("verification payload"),
	)

	nextHop, err := router.SelectNextHop(context.Background(), testBundle, neighbors)
	if err != nil {
		log.Fatalf("Initial routing failed: %v", err)
	}
	log.Printf("Initial next hop: %s", nextHop)

	neighbors["ground_earth_1"].IsActive = false
	nextHopAfterFailure, err := router.SelectNextHop(context.Background(), testBundle, neighbors)
	if err != nil {
		log.Fatalf("Reroute failed after node outage: %v", err)
	}

	if nextHopAfterFailure == "ground_earth_1" {
		log.Fatalf("Reroute failed: selected inactive neighbor")
	}

	log.Printf("Reroute next hop after outage: %s", nextHopAfterFailure)
	log.Println("Sat_Net verification passed")
}
