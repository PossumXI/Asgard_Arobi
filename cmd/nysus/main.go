// Package main implements the Nysus central orchestration service.
// Nysus is the "central nervous system" of ASGARD, coordinating
// satellites, robots, security, and providing the API for web interfaces.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asgard/pandora/Nysus/internal/api"
	"github.com/asgard/pandora/Nysus/internal/events"
	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

func main() {
	// Parse command line flags
	addr := flag.String("addr", ":8080", "HTTP server address")
	dbHost := flag.String("db-host", "localhost", "PostgreSQL host")
	dbPort := flag.String("db-port", "5432", "PostgreSQL port")
	mongoHost := flag.String("mongo-host", "localhost", "MongoDB host")
	mongoPort := flag.String("mongo-port", "27017", "MongoDB port")
	flag.Parse()

	log.Println("=== ASGARD Nysus - Central Nervous System ===")
	log.Printf("HTTP Server: %s", *addr)

	// Override config from flags
	os.Setenv("POSTGRES_HOST", *dbHost)
	os.Setenv("POSTGRES_PORT", *dbPort)
	os.Setenv("MONGO_HOST", *mongoHost)
	os.Setenv("MONGO_PORT", *mongoPort)

	// Load database configuration
	dbCfg, err := db.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load database config: %v", err)
	}

	// Connect to PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	pgDB, err := db.NewPostgresDB(dbCfg)
	if err != nil {
		log.Printf("Warning: PostgreSQL connection failed: %v", err)
		log.Println("Continuing without database - API will return sample data")
		pgDB = nil
	} else {
		log.Println("PostgreSQL connected successfully")
		defer pgDB.Close()
	}

	// Connect to MongoDB
	log.Println("Connecting to MongoDB...")
	mongoDB, err := db.NewMongoDB(dbCfg)
	if err != nil {
		log.Printf("Warning: MongoDB connection failed: %v", err)
		mongoDB = nil
	} else {
		log.Println("MongoDB connected successfully")
		defer mongoDB.Close(context.Background())
	}

	// Create event bus
	eventBus := events.NewEventBus()
	eventBus.Start()
	defer eventBus.Stop()

	// Subscribe to events for logging
	eventBus.Subscribe(events.EventTypeAlert, func(ctx context.Context, event events.Event) error {
		log.Printf("[Event] Alert: %v", event.Payload)
		return nil
	})

	eventBus.Subscribe(events.EventTypeThreat, func(ctx context.Context, event events.Event) error {
		log.Printf("[Event] Threat: %v", event.Payload)
		return nil
	})

	log.Println("Event bus started")

	// Create API server
	serverCfg := api.Config{
		Addr:         *addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle nil DB connections gracefully
	var pgDBPtr *db.PostgresDB = pgDB
	var mongoDBPtr *db.MongoDB = mongoDB

	// Create a mock DB wrapper if needed
	if pgDB == nil {
		pgDBPtr = createMockPostgres()
	}
	if mongoDB == nil {
		mongoDBPtr = createMockMongo()
	}

	server := api.NewServer(serverCfg, pgDBPtr, mongoDBPtr, eventBus)

	// Start event simulation for demo purposes
	go simulateEvents(eventBus)

	// Start server in goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", *addr)
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Println("Nysus is ready and accepting connections")
	log.Println("API Endpoints:")
	log.Println("  - Health:     GET  /health")
	log.Println("  - Auth:       POST /api/auth/signin, /api/auth/signup")
	log.Println("  - Dashboard:  GET  /api/dashboard/stats")
	log.Println("  - Entities:   GET  /api/alerts, /api/missions, /api/satellites, /api/hunoids")
	log.Println("  - Streams:    GET  /api/streams, /api/streams/stats")
	log.Println("  - WebSocket:  WS   /ws/realtime")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Nysus...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Nysus stopped")
}

// simulateEvents generates periodic events for demonstration.
func simulateEvents(eventBus *events.EventBus) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	alertTypes := []string{"fire", "tsunami", "troop_movement", "maritime_distress"}

	for range ticker.C {
		// Simulate random alerts
		alertEvent := events.Event{
			ID:        uuid.New(),
			Type:      events.EventTypeAlert,
			Source:    "sat-" + uuid.New().String()[:8],
			Timestamp: time.Now().UTC(),
			Payload: events.AlertEvent{
				SatelliteID: "sat-" + uuid.New().String()[:8],
				AlertType:   alertTypes[time.Now().UnixNano()%int64(len(alertTypes))],
				Confidence:  0.75 + float64(time.Now().UnixNano()%25)/100,
				Location: events.GeoLocation{
					Latitude:  35.0 + float64(time.Now().UnixNano()%1000)/100,
					Longitude: -120.0 + float64(time.Now().UnixNano()%500)/100,
				},
			},
			Priority: 7,
		}
		eventBus.Publish(alertEvent)

		// Simulate telemetry
		telemetryEvent := events.Event{
			ID:        uuid.New(),
			Type:      events.EventTypeSatelliteTelemetry,
			Source:    "sat-001",
			Timestamp: time.Now().UTC(),
			Payload: events.TelemetryEvent{
				ComponentID:   "sat-001",
				ComponentType: "satellite",
				Metrics: map[string]float64{
					"battery":     85.5 + float64(time.Now().UnixNano()%100)/10,
					"temperature": 25.0 + float64(time.Now().UnixNano()%50)/10,
					"altitude":    400.0 + float64(time.Now().UnixNano()%100)/10,
				},
				Status: "operational",
			},
		}
		eventBus.Publish(telemetryEvent)
	}
}

// createMockPostgres creates a mock PostgresDB for testing without a database.
func createMockPostgres() *db.PostgresDB {
	// Return nil - the API handlers will use sample data
	return nil
}

// createMockMongo creates a mock MongoDB for testing without a database.
func createMockMongo() *db.MongoDB {
	// Return nil - the API handlers will use sample data
	return nil
}
