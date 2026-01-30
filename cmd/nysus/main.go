// Package main implements the Nysus central orchestration service.
// Nysus is the "central nervous system" of ASGARD, coordinating
// satellites, robots, security, and providing the API for web interfaces.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/asgard/pandora/internal/controlplane"
	"github.com/asgard/pandora/internal/nysus/agents"
	"github.com/asgard/pandora/internal/nysus/api"
	"github.com/asgard/pandora/internal/nysus/events"
	"github.com/asgard/pandora/internal/nysus/mcp"
	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/platform/observability"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Parse command line flags
	addr := flag.String("addr", ":8080", "HTTP server address")
	dbHost := flag.String("db-host", "localhost", "PostgreSQL host")
	dbPort := flag.String("db-port", "55432", "PostgreSQL port")
	mongoHost := flag.String("mongo-host", "localhost", "MongoDB host")
	mongoPort := flag.String("mongo-port", "27017", "MongoDB port")
	flag.Parse()

	log.Println("=== ASGARD Nysus - Central Nervous System ===")
	log.Printf("HTTP Server: %s", *addr)

	shutdownTracing, err := observability.InitTracing(context.Background(), "nysus")
	if err != nil {
		log.Printf("Tracing disabled: %v", err)
	} else {
		defer func() {
			if err := shutdownTracing(context.Background()); err != nil {
				log.Printf("Tracing shutdown error: %v", err)
			}
		}()
	}

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

	allowNoDB := strings.EqualFold(os.Getenv("ASGARD_ALLOW_NO_DB"), "true") || os.Getenv("ASGARD_ALLOW_NO_DB") == "1"

	// Connect to PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	pgDB, err := db.NewPostgresDB(dbCfg)
	if err != nil {
		if allowNoDB {
			log.Printf("Warning: PostgreSQL connection failed: %v (continuing without database)", err)
			pgDB = nil
		} else {
			log.Fatalf("PostgreSQL connection failed: %v", err)
		}
	}
	if pgDB != nil {
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

	// Start unified control plane
	cpCfg := controlplane.DefaultConfig()
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		cpCfg.NATSUrl = natsURL
	}
	controlPlane, err := controlplane.NewUnifiedControlPlane(cpCfg)
	if err != nil {
		log.Printf("Control plane init failed: %v", err)
	} else {
		if err := controlPlane.Start(); err != nil {
			log.Printf("Control plane start failed: %v", err)
		} else {
			defer controlPlane.Stop()
		}
	}

	// Subscribe to events for logging and control plane bridging
	eventBus.Subscribe(events.EventTypeAlert, func(ctx context.Context, event events.Event) error {
		log.Printf("[Event] Alert: %v", event.Payload)
		publishToControlPlane(controlPlane, event)
		return nil
	})

	eventBus.Subscribe(events.EventTypeThreat, func(ctx context.Context, event events.Event) error {
		log.Printf("[Event] Threat: %v", event.Payload)
		publishToControlPlane(controlPlane, event)
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

	// Handle nil DB connections gracefully - API server handles nil DBs
	server := api.NewServer(serverCfg, pgDB, mongoDB, eventBus)

	// Start MCP Server for LLM integration
	mcpCfg := mcp.DefaultConfig()
	if mcpAddr := os.Getenv("MCP_ADDR"); mcpAddr != "" {
		mcpCfg.Addr = mcpAddr
	}
	mcpServer := mcp.NewServer(mcpCfg)
	mcpServer.SetPostgresDB(pgDB)
	mcpServer.RegisterDefaultTools()
	if err := mcpServer.Start(); err != nil {
		log.Printf("Warning: MCP server failed to start: %v", err)
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			mcpServer.Stop(ctx)
		}()
		log.Println("MCP Server started - LLM tools available")
	}

	// Start AI Agent Coordinator
	agentCoordinator := agents.NewCoordinator()
	if err := agentCoordinator.Start(context.Background()); err != nil {
		log.Printf("Warning: Agent coordinator failed to start: %v", err)
	} else {
		defer agentCoordinator.Stop()
		log.Println("AI Agent Coordinator started with specialized agents")
	}

	// Start database-driven event publishing if DB is available
	if pgDB != nil {
		go startEventPublisher(context.Background(), eventBus, pgDB)
	}

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
	log.Println("  - WebSocket:  WS   /ws, /ws/events, /ws/realtime")
	log.Println("  - Signaling:  WS   /ws/signaling (WebRTC SFU)")
	log.Println("  - MCP:        HTTP :8085/mcp/* (LLM tools)")

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

// publishToControlPlane bridges nysus events to the unified control plane.
// It converts internal events to cross-domain events for system-wide coordination.
func publishToControlPlane(cp *controlplane.UnifiedControlPlane, event events.Event) {
	if cp == nil {
		return
	}

	// Map nysus event type to control plane event type and domain
	var eventType controlplane.CrossDomainEventType
	var domain controlplane.EventDomain
	var severity controlplane.Severity

	switch event.Type {
	case events.EventTypeAlert, events.EventTypeAlertUpdated:
		eventType = controlplane.EventControlAlert
		domain = controlplane.DomainAutonomy
	case events.EventTypeThreat:
		eventType = controlplane.EventSecurityThreat
		domain = controlplane.DomainSecurity
	case events.EventTypeThreatMitigated:
		eventType = controlplane.EventSecurityMitigated
		domain = controlplane.DomainSecurity
	default:
		eventType = controlplane.EventAutonomyStatus
		domain = controlplane.DomainAutonomy
	}

	// Map priority to severity (priority 0-10, higher is more urgent)
	switch {
	case event.Priority >= 9:
		severity = controlplane.SeverityCritical
	case event.Priority >= 7:
		severity = controlplane.SeverityHigh
	case event.Priority >= 5:
		severity = controlplane.SeverityMedium
	case event.Priority >= 3:
		severity = controlplane.SeverityLow
	default:
		severity = controlplane.SeverityInfo
	}

	// Create cross-domain event
	cpEvent := controlplane.NewCrossDomainEvent(
		eventType,
		domain,
		event.Source,
		severity,
		string(event.Type),
	)
	cpEvent.ID = event.ID
	cpEvent.Timestamp = event.Timestamp

	// Convert payload to map
	if event.Payload != nil {
		payloadBytes, err := json.Marshal(event.Payload)
		if err == nil {
			var payloadMap map[string]interface{}
			if json.Unmarshal(payloadBytes, &payloadMap) == nil {
				cpEvent.Payload = payloadMap
			}
		}
	}

	// Publish to control plane (fire and forget, log errors)
	if err := cp.PublishEvent(cpEvent); err != nil {
		log.Printf("[Nysus] Failed to publish event to control plane: %v", err)
	}
}

// startEventPublisher subscribes to database changes and publishes events to the event bus.
// This replaces the simulated events with real database-driven events.
func startEventPublisher(ctx context.Context, eventBus *events.EventBus, pgDB *db.PostgresDB) {
	if pgDB == nil {
		log.Println("Warning: No database connection, event publishing disabled")
		return
	}

	// Subscribe to PostgreSQL notifications for real-time events
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Check for new alerts in the last 5 seconds
				publishNewAlerts(eventBus, pgDB)
				
				// Check for telemetry updates
				publishTelemetryUpdates(eventBus, pgDB)
			}
		}
	}()
}

func publishNewAlerts(eventBus *events.EventBus, pgDB *db.PostgresDB) {
	if pgDB == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := pgDB.QueryContext(ctx, `
		SELECT id, satellite_id, alert_type, confidence_score, latitude, longitude, created_at
		FROM alerts
		WHERE created_at > NOW() - INTERVAL '10 seconds'
		  AND status = 'new'
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, satID, alertType string
		var confidence, lat, lon float64
		var createdAt time.Time

		if err := rows.Scan(&id, &satID, &alertType, &confidence, &lat, &lon, &createdAt); err != nil {
			continue
		}

		alertEvent := events.Event{
			ID:        uuid.MustParse(id),
			Type:      events.EventTypeAlert,
			Source:    satID,
			Timestamp: createdAt,
			Payload: events.AlertEvent{
				SatelliteID: satID,
				AlertType:   alertType,
				Confidence:  confidence,
				Location: events.GeoLocation{
					Latitude:  lat,
					Longitude: lon,
				},
			},
			Priority: 7,
		}
		eventBus.Publish(alertEvent)
	}
}

func publishTelemetryUpdates(eventBus *events.EventBus, pgDB *db.PostgresDB) {
	if pgDB == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish satellite telemetry
	satRows, err := pgDB.QueryContext(ctx, `
		SELECT id, name, current_battery_percent, status, last_telemetry
		FROM satellites
		WHERE last_telemetry > NOW() - INTERVAL '30 seconds'
		  AND status = 'operational'
	`)
	if err == nil {
		defer satRows.Close()
		for satRows.Next() {
			var id, name, status string
			var battery float64
			var lastTelemetry time.Time

			if err := satRows.Scan(&id, &name, &battery, &status, &lastTelemetry); err != nil {
				continue
			}

			telemetryEvent := events.Event{
				ID:        uuid.New(),
				Type:      events.EventTypeSatelliteTelemetry,
				Source:    id,
				Timestamp: lastTelemetry,
				Payload: events.TelemetryEvent{
					ComponentID:   id,
					ComponentType: "satellite",
					Metrics: map[string]float64{
						"battery": battery,
					},
					Status: status,
				},
			}
			eventBus.Publish(telemetryEvent)
		}
	}

	// Publish hunoid telemetry
	hunRows, err := pgDB.QueryContext(ctx, `
		SELECT id, serial_number, battery_percent, status, last_telemetry
		FROM hunoids
		WHERE last_telemetry > NOW() - INTERVAL '30 seconds'
		  AND status IN ('active', 'idle')
	`)
	if err == nil {
		defer hunRows.Close()
		for hunRows.Next() {
			var id, serial, status string
			var battery float64
			var lastTelemetry time.Time

			if err := hunRows.Scan(&id, &serial, &battery, &status, &lastTelemetry); err != nil {
				continue
			}

			telemetryEvent := events.Event{
				ID:        uuid.New(),
				Type:      events.EventTypeHunoidTelemetry,
				Source:    id,
				Timestamp: lastTelemetry,
				Payload: events.TelemetryEvent{
					ComponentID:   id,
					ComponentType: "hunoid",
					Metrics: map[string]float64{
						"battery": battery,
					},
					Status: status,
				},
			}
			eventBus.Publish(telemetryEvent)
		}
	}
}
