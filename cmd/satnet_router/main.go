// Package main implements the Sat_Net router node for ASGARD's
// Delay Tolerant Networking layer. This executable runs on satellites
// and ground stations to enable interplanetary communication.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
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
	rlModel := flag.String("rl-model", "models/rl_router.json", "Path to RL routing model")
	useRL := flag.Bool("rl", false, "Enable RL-based routing policy")
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
	log.Printf("RL Routing: %v", *useRL)

	// Create storage
	storage, cleanup := buildStorage()
	defer cleanup()

	// Create router
	var router dtn.Router
	if *useRL {
		rlRouter, err := dtn.NewRLRoutingAgent(*nodeEID, *rlModel)
		if err != nil {
			log.Fatalf("Failed to load RL model: %v", err)
		}
		rlRouter.UpdateEnergy(*nodeID, *initialBattery)
		router = rlRouter
	} else if *energyAware {
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

	// Initialize transport
	transportConfig := dtn.DefaultTCPTransportConfig()
	transportConfig.ListenAddress = *listenAddr
	transport := dtn.NewTCPTransport(*nodeID, transportConfig)

	// Create and start node with transport
	node := dtn.NewNodeWithTransport(*nodeID, *nodeEID, storage, router, transport, config)

	if err := node.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	log.Printf("Sat_Net router started successfully")

	// Register neighbors from configuration
	neighbors := parseNeighborConfig(os.Getenv("DTN_NEIGHBORS"))
	if len(neighbors) == 0 {
		log.Println("No neighbors configured (DTN_NEIGHBORS is empty)")
	} else {
		registerNeighbors(context.Background(), node, transport, neighbors)
	}

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

type neighborConfig struct {
	id        string
	eid       string
	address   string
	quality   float64
	latency   time.Duration
	bandwidth int64
}

func parseNeighborConfig(raw string) []neighborConfig {
	entries := strings.Split(strings.TrimSpace(raw), ";")
	var neighbors []neighborConfig

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.Split(entry, "@")
		if len(parts) < 3 {
			continue
		}

		neighbors = append(neighbors, neighborConfig{
			id:        parts[0],
			eid:       parts[1],
			address:   parts[2],
			quality:   0.9,
			latency:   50 * time.Millisecond,
			bandwidth: 5_000_000,
		})
	}

	return neighbors
}

func buildStorage() (dtn.BundleStorage, func()) {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("DTN_STORAGE_BACKEND")))
	if backend == "" {
		backend = "memory"
	}
	switch backend {
	case "postgres":
		cfg, err := db.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load DB config: %v", err)
		}
		pgDB, err := db.NewPostgresDB(cfg)
		if err != nil {
			log.Fatalf("Failed to connect to Postgres: %v", err)
		}
		storage, err := dtn.NewPostgresBundleStorage(pgDB)
		if err != nil {
			log.Fatalf("Failed to initialize Postgres storage: %v", err)
		}
		log.Printf("Using Postgres-backed DTN storage")
		return storage, func() { _ = pgDB.Close() }
	default:
		log.Printf("Using in-memory DTN storage")
		return dtn.NewInMemoryStorage(10000), func() {}
	}
}

func registerNeighbors(ctx context.Context, node *dtn.Node, transport *dtn.TCPTransport, neighbors []neighborConfig) {
	for _, n := range neighbors {
		neighbor := &dtn.Neighbor{
			ID:           n.id,
			EID:          n.eid,
			LinkQuality:  n.quality,
			LastContact:  time.Now().UTC(),
			IsActive:     true,
			Latency:      n.latency,
			Bandwidth:    n.bandwidth,
			ContactStart: time.Now().UTC(),
			ContactEnd:   time.Now().Add(24 * time.Hour),
		}

		node.RegisterNeighbor(neighbor)
		if err := transport.Connect(ctx, neighbor.ID, n.address); err != nil {
			log.Printf("Failed to connect to neighbor %s at %s: %v", neighbor.ID, n.address, err)
			continue
		}

		log.Printf("Registered neighbor: %s (%s) at %s", neighbor.ID, neighbor.EID, n.address)
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
