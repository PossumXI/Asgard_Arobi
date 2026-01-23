// Package main implements the Sat_Net router node for ASGARD's
// Delay Tolerant Networking layer. This executable runs on satellites
// and ground stations to enable interplanetary communication.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asgard/pandora/internal/platform/dtn"
)

func main() {
	// Parse command line flags
	nodeID := flag.String("id", "", "Unique node identifier (required)")
	nodeEID := flag.String("eid", "", "DTN Endpoint Identifier (e.g., dtn://earth/ground001)")
	listenAddr := flag.String("listen", ":4556", "Address to listen for DTN connections")
	bufferSize := flag.Int("buffer", 1000, "Bundle buffer size")
	energyAware := flag.Bool("energy-aware", false, "Enable energy-aware routing for satellites")
	initialBattery := flag.Float64("battery", 100.0, "Initial battery percentage (0-100)")
	flag.Parse()

	if *nodeID == "" || *nodeEID == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("=== ASGARD Sat_Net Router ===")
	log.Printf("Node ID: %s", *nodeID)
	log.Printf("Node EID: %s", *nodeEID)
	log.Printf("Listen Address: %s", *listenAddr)
	log.Printf("Energy Aware: %v", *energyAware)

	// Create storage
	storage := dtn.NewInMemoryStorage(10000)

	// Create router
	var router dtn.Router
	if *energyAware {
		energyRouter := dtn.NewEnergyAwareRouter(*nodeEID)
		energyRouter.UpdateEnergy(*nodeID, *initialBattery)
		router = energyRouter
	} else {
		router = dtn.NewContactGraphRouter(*nodeEID)
	}

	// Create node configuration
	config := dtn.NodeConfig{
		BufferSize:     *bufferSize,
		ProcessTimeout: 30 * time.Second,
		MaxRetries:     3,
		PurgeInterval:  5 * time.Minute,
	}

	// Create and start node
	node := dtn.NewNode(*nodeID, *nodeEID, storage, router, config)

	if err := node.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	log.Printf("Sat_Net router started successfully")

	// Register some test neighbors for demonstration
	registerTestNeighbors(node, *nodeEID)

	// Start telemetry reporter
	go reportTelemetry(node)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	if err := node.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	log.Println("Sat_Net router stopped")
}

// registerTestNeighbors adds simulated neighbors for demonstration.
func registerTestNeighbors(node *dtn.Node, selfEID string) {
	// Simulated ground stations and satellites
	neighbors := []struct {
		id          string
		eid         string
		linkQuality float64
		latency     time.Duration
		bandwidth   int64
	}{
		{"ground_earth_1", "dtn://earth/ground001", 0.95, 50 * time.Millisecond, 10000000},
		{"ground_earth_2", "dtn://earth/ground002", 0.90, 75 * time.Millisecond, 8000000},
		{"sat_leo_001", "dtn://leo/sat001", 0.85, 20 * time.Millisecond, 5000000},
		{"sat_leo_002", "dtn://leo/sat002", 0.80, 25 * time.Millisecond, 5000000},
		{"relay_mars_1", "dtn://mars/relay001", 0.60, 14 * time.Minute, 100000}, // Mars relay ~14 min light delay
		{"relay_lunar_1", "dtn://lunar/relay001", 0.75, 1300 * time.Millisecond, 2000000}, // Moon ~1.3 sec delay
	}

	// Filter out self
	for _, n := range neighbors {
		if n.eid == selfEID {
			continue
		}

		neighbor := &dtn.Neighbor{
			ID:           n.id,
			EID:          n.eid,
			LinkQuality:  n.linkQuality,
			LastContact:  time.Now().UTC(),
			IsActive:     true,
			Latency:      n.latency,
			Bandwidth:    n.bandwidth,
			ContactStart: time.Now().UTC(),
			ContactEnd:   time.Now().Add(24 * time.Hour), // Simulated 24hr contact window
		}

		node.RegisterNeighbor(neighbor)
		log.Printf("Registered neighbor: %s (%s) - quality=%.2f, latency=%v",
			neighbor.ID, neighbor.EID, neighbor.LinkQuality, neighbor.Latency)
	}
}

// reportTelemetry periodically logs node metrics.
func reportTelemetry(node *dtn.Node) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics := node.GetMetrics()
		log.Printf("[Telemetry] Received=%d, Sent=%d, Dropped=%d, Expired=%d, Connections=%d",
			metrics.BundlesReceived,
			metrics.BundlesSent,
			metrics.BundlesDropped,
			metrics.BundlesExpired,
			metrics.ActiveConnections)
	}
}
