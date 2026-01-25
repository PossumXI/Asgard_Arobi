package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/asgard/pandora/internal/platform/observability"
	secevents "github.com/asgard/pandora/internal/security/events"
	"github.com/asgard/pandora/internal/security/mitigation"
	"github.com/asgard/pandora/internal/security/scanner"
	"github.com/asgard/pandora/internal/security/threat"
	"github.com/google/gopacket/pcap"
	"github.com/google/uuid"
)

// NetworkDevice represents a network interface
type NetworkDevice struct {
	Name        string
	Description string
	Addresses   []NetworkAddress
}

// NetworkAddress represents an IP address on an interface
type NetworkAddress struct {
	IP      net.IP
	Netmask net.IPMask
}

func findNetworkDevices() ([]NetworkDevice, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}
	
	var result []NetworkDevice
	for _, dev := range devices {
		nd := NetworkDevice{
			Name:        dev.Name,
			Description: dev.Description,
		}
		for _, addr := range dev.Addresses {
			nd.Addresses = append(nd.Addresses, NetworkAddress{
				IP:      addr.IP,
				Netmask: addr.Netmask,
			})
		}
		result = append(result, nd)
	}
	return result, nil
}

// parseUUID parses a string to UUID
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func main() {
	// Command-line flags
	interfaceName := flag.String("interface", "", "Network interface to monitor (use -list-interfaces to see available)")
	listInterfaces := flag.Bool("list-interfaces", false, "List available network interfaces and exit")
	natsURL := flag.String("nats", "nats://localhost:4222", "NATS server URL")
	metricsAddr := flag.String("metrics-addr", ":9091", "Metrics server address")
	apiAddr := flag.String("api-addr", ":9090", "API server address for Pricilla integration")
	flag.Parse()

	// List interfaces if requested
	if *listInterfaces {
		listNetworkInterfaces()
		return
	}

	log.Println("=== ASGARD Giru - Security System ===")
	log.Printf("Monitoring interface: %s", *interfaceName)

	shutdownTracing, err := observability.InitTracing(context.Background(), "giru")
	if err != nil {
		log.Printf("Tracing disabled: %v", err)
	} else {
		defer func() {
			if err := shutdownTracing(context.Background()); err != nil {
				log.Printf("Tracing shutdown error: %v", err)
			}
		}()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize NATS publisher (optional - continues without if unavailable)
	var publisher *secevents.Publisher
	pubCfg := secevents.DefaultPublisherConfig()
	pubCfg.NATSURL = *natsURL
	pub, err := secevents.NewPublisher(pubCfg)
	if err != nil {
		log.Printf("Warning: NATS publisher unavailable: %v (continuing without real-time events)", err)
	} else {
		publisher = pub
		defer publisher.Close()
		log.Println("NATS publisher initialized - security events will be broadcast")
	}

	// Initialize scanner - prefer real-time capture or log ingestion
	var netScanner scanner.Scanner
	scannerMode := strings.ToLower(os.Getenv("SECURITY_SCANNER_MODE"))
	logSources := parseLogSources(os.Getenv("SECURITY_LOG_SOURCES"))

	switch scannerMode {
	case "log":
		netScanner = initLogIngestionScanner(logSources)
	case "pcap":
		netScanner = initRealtimeScanner(*interfaceName)
	default:
		if rs, err := scanner.NewRealtimeScanner(*interfaceName); err == nil {
			netScanner = rs
			log.Println("Real-time packet capture initialized on interface:", *interfaceName)
		} else if len(logSources) > 0 {
			log.Printf("Warning: Real-time packet capture unavailable: %v (switching to log ingestion)", err)
			netScanner = initLogIngestionScanner(logSources)
		} else {
			log.Printf("ERROR: Real-time packet capture failed: %v", err)
			log.Println("")
			log.Println("=== SETUP REQUIRED ===")
			log.Println("To use Giru's real-time scanner on Windows:")
			log.Println("1. Install Npcap from: https://npcap.com/dist/npcap-1.79.exe")
			log.Println("   - During install, check 'WinPcap API-compatible Mode'")
			log.Println("2. Run Giru as Administrator")
			log.Println("3. Use correct interface name. Find yours with:")
			log.Println("   getmac /v /fo list")
			log.Println("   or in PowerShell: Get-NetAdapter | Select Name, InterfaceDescription")
			log.Println("4. Run: giru.exe -interface \"\\Device\\NPF_{YOUR-GUID}\"")
			log.Println("")
			log.Println("Alternative: Use log ingestion mode:")
			log.Println("   set SECURITY_SCANNER_MODE=log")
			log.Println("   set SECURITY_LOG_SOURCES=C:\\path\\to\\logs:syslog")
			log.Println("======================")
			log.Fatal("Cannot start without packet capture or log sources")
		}
	}

	if err := netScanner.Start(ctx); err != nil {
		log.Fatalf("Failed to start scanner: %v", err)
	}
	defer netScanner.Stop()

	log.Println("Network scanner initialized")

	// Create threat channel
	threatChan := make(chan threat.Threat, 100)

	// Create threat detector
	detector := threat.NewDetector(netScanner, threatChan)
	if cb, ok := netScanner.(interface{ SetAnomalyCallback(func(*scanner.Anomaly)) }); ok {
		cb.SetAnomalyCallback(func(anomaly *scanner.Anomaly) {
			if err := detector.ProcessAnomaly(ctx, anomaly); err != nil {
				log.Printf("Anomaly processing error: %v", err)
			}
		})
	}

	// Create action channel
	actionChan := make(chan mitigation.MitigationAction, 100)

	// Create mitigation responder
	responder := mitigation.NewResponder(actionChan)

	// Start threat processor
	go processThreats(ctx, threatChan, responder, publisher)

	// Start action processor
	go processActions(ctx, actionChan, publisher)

	// Start statistics reporter
	go reportStatistics(ctx, netScanner, publisher)

	metricsServer := startMetricsServer(*metricsAddr)
	apiServer := startAPIServer(*apiAddr)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Giru...")
	cancel()
	shutdownMetricsServer(metricsServer)
	shutdownAPIServer(apiServer)
	time.Sleep(2 * time.Second)
	log.Println("Giru stopped")
}

func processThreats(ctx context.Context, threatChan <-chan threat.Threat, responder *mitigation.Responder, publisher *secevents.Publisher) {
	for {
		select {
		case t := <-threatChan:
			log.Printf("=== THREAT RECEIVED ===")
			log.Printf("ID: %s", t.ID)
			log.Printf("Type: %s", t.Type)
			log.Printf("Severity: %s", t.Severity)
			log.Printf("Source: %s", t.SourceIP)
			log.Printf("Description: %s", t.Description)
			log.Printf("======================")

			// Publish to NATS if available
			if publisher != nil {
				alert := secevents.NewAlertEvent(
					"giru",
					t.Type,
					t.SourceIP,
					secevents.ThreatSeverityToEventSeverity(string(t.Severity)),
					0.85, // Default confidence
					t.Description,
				)
				if err := publisher.PublishAlert(alert); err != nil {
					log.Printf("Failed to publish alert: %v", err)
				}
			}

			// Mitigate threat
			if err := responder.MitigateThreat(ctx, t); err != nil {
				log.Printf("Mitigation error: %v", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func processActions(ctx context.Context, actionChan <-chan mitigation.MitigationAction, publisher *secevents.Publisher) {
	for {
		select {
		case action := <-actionChan:
			log.Printf("=== MITIGATION ACTION ===")
			log.Printf("Threat ID: %s", action.ThreatID)
			log.Printf("Action: %s", action.ActionType)
			log.Printf("Target: %s", action.Target)
			log.Printf("Success: %t", action.Success)
			log.Printf("========================")

			// Publish response event to NATS if available
			if publisher != nil {
				// Parse threat ID from string
				threatID, err := parseUUID(action.ThreatID)
				if err != nil {
					log.Printf("Invalid threat ID format: %v", err)
					continue
				}
				response := secevents.NewResponseEvent(
					"giru",
					threatID,
					action.ActionType,
					action.Target,
					action.Success,
					time.Millisecond*100, // Approximate duration
				)
				if err := publisher.PublishResponse(response); err != nil {
					log.Printf("Failed to publish response: %v", err)
				}
			}

			executeMitigationAction(action)

		case <-ctx.Done():
			return
		}
	}
}

func executeMitigationAction(action mitigation.MitigationAction) {
	switch action.ActionType {
	case "block_ip":
		log.Printf("Blocking IP %s for %v hours", action.Target, action.Parameters["duration_hours"])
	case "monitor":
		log.Printf("Monitoring %s for %v minutes", action.Target, action.Parameters["duration_minutes"])
	case "log":
		log.Printf("Logging threat %s", action.ThreatID)
	default:
		log.Printf("Unknown mitigation action: %s", action.ActionType)
	}
}

func startMetricsServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", observability.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	log.Printf("Metrics server listening on %s", addr)
	return server
}

func shutdownMetricsServer(server *http.Server) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Metrics server shutdown error: %v", err)
	}
}

func reportStatistics(ctx context.Context, netScanner scanner.Scanner, publisher *secevents.Publisher) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := netScanner.GetStatistics()
			log.Printf("Scanner Stats: Packets=%d, Anomalies=%d, Blocked=%d",
				stats.PacketsScanned, stats.AnomaliesDetected, stats.ThreatsBlocked)

			if publisher != nil && publisher.IsConnected() {
				pubStats := publisher.Stats()
				log.Printf("NATS Stats: Alerts=%d, Findings=%d, Responses=%d, Errors=%d",
					pubStats.AlertsPublished, pubStats.FindingsPublished, pubStats.ResponsesPublished, pubStats.Errors)
			}

		case <-ctx.Done():
			return
		}
	}
}

type logSourceConfig struct {
	path    string
	logType string
}

func parseLogSources(value string) []logSourceConfig {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	results := make([]logSourceConfig, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		chunks := strings.SplitN(part, ":", 2)
		cfg := logSourceConfig{path: strings.TrimSpace(chunks[0]), logType: "syslog"}
		if len(chunks) == 2 && strings.TrimSpace(chunks[1]) != "" {
			cfg.logType = strings.TrimSpace(chunks[1])
		}
		results = append(results, cfg)
	}
	return results
}

func initRealtimeScanner(interfaceName string) scanner.Scanner {
	realtimeScanner, err := scanner.NewRealtimeScanner(interfaceName)
	if err != nil {
		log.Fatalf("Failed to initialize real-time packet capture: %v", err)
	}
	log.Println("Real-time packet capture initialized")
	return realtimeScanner
}

func initLogIngestionScanner(sources []logSourceConfig) scanner.Scanner {
	if len(sources) == 0 {
		log.Fatal("SECURITY_LOG_SOURCES is required for log ingestion mode")
	}
	logScanner := scanner.NewLogIngestionScanner()
	for _, src := range sources {
		if err := logScanner.AddLogSource(src.path, src.logType); err != nil {
			log.Fatalf("Failed to add log source %s: %v", src.path, err)
		}
	}
	log.Printf("Log ingestion scanner initialized with %d sources", len(sources))
	return logScanner
}

// =============================================================================
// API Server for Pricilla Integration
// =============================================================================

// ThreatZone represents a geographic threat zone for Pricilla guidance
type ThreatZone struct {
	ID          string  `json:"id"`
	ThreatType  string  `json:"threatType"`
	ThreatLevel float64 `json:"threatLevel"`
	Center      struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Altitude  float64 `json:"altitude"`
	} `json:"center"`
	RadiusKm    float64 `json:"radiusKm"`
	Active      bool    `json:"active"`
	Description string  `json:"description"`
}

var activeThreatZones = []ThreatZone{
	{
		ID:          "tz-001",
		ThreatType:  "SAM_SITE",
		ThreatLevel: 0.85,
		Center: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Altitude  float64 `json:"altitude"`
		}{Latitude: 34.0522, Longitude: -118.2437, Altitude: 500},
		RadiusKm:    15.0,
		Active:      true,
		Description: "Surface-to-Air Missile battery - avoid",
	},
	{
		ID:          "tz-002",
		ThreatType:  "RADAR",
		ThreatLevel: 0.6,
		Center: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Altitude  float64 `json:"altitude"`
		}{Latitude: 34.1522, Longitude: -118.3437, Altitude: 200},
		RadiusKm:    25.0,
		Active:      true,
		Description: "Early warning radar station",
	},
	{
		ID:          "tz-003",
		ThreatType:  "AAA",
		ThreatLevel: 0.7,
		Center: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Altitude  float64 `json:"altitude"`
		}{Latitude: 33.9522, Longitude: -118.1437, Altitude: 100},
		RadiusKm:    8.0,
		Active:      true,
		Description: "Anti-aircraft artillery emplacement",
	},
}

func startAPIServer(addr string) *http.Server {
	mux := http.NewServeMux()
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"service":"GIRU","status":"healthy","version":"1.0.0"}`))
	})
	
	// Threat zones endpoint for Pricilla
	mux.HandleFunc("/api/threat-zones", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		
		response := struct {
			Zones []ThreatZone `json:"zones"`
			Count int          `json:"count"`
		}{
			Zones: activeThreatZones,
			Count: len(activeThreatZones),
		}
		
		json.NewEncoder(w).Encode(response)
		log.Printf("[GIRU] Served %d threat zones to Pricilla", len(activeThreatZones))
	})
	
	// Active threats endpoint
	mux.HandleFunc("/api/threats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		threats := []map[string]interface{}{
			{
				"id":          "threat-001",
				"type":        "network_intrusion",
				"severity":    "high",
				"sourceIP":    "192.168.1.100",
				"description": "Suspicious network activity detected",
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
			},
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"threats": threats,
			"count":   len(threats),
		})
	})
	
	// Security scan endpoint
	mux.HandleFunc("/api/scans", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		if r.Method == "POST" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"scanId":    uuid.New().String(),
				"status":    "initiated",
				"startTime": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
		
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("[GIRU] API server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[GIRU] API server error: %v", err)
		}
	}()

	return server
}

func shutdownAPIServer(server *http.Server) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[GIRU] API server shutdown error: %v", err)
	}
}

func listNetworkInterfaces() {
	log.Println("=== Available Network Interfaces ===")
	log.Println("")
	
	// Try to use pcap to find devices
	devices, err := findNetworkDevices()
	if err != nil {
		log.Printf("Error finding devices: %v", err)
		log.Println("")
		log.Println("Make sure Npcap is installed: https://npcap.com/")
		return
	}
	
	if len(devices) == 0 {
		log.Println("No network interfaces found.")
		log.Println("Make sure Npcap is installed and you're running as Administrator.")
		return
	}
	
	for i, dev := range devices {
		log.Printf("%d. %s", i+1, dev.Name)
		if dev.Description != "" {
			log.Printf("   Description: %s", dev.Description)
		}
		for _, addr := range dev.Addresses {
			log.Printf("   Address: %s", addr.IP)
		}
		log.Println("")
	}
	
	log.Println("To use an interface, run:")
	log.Println("  giru.exe -interface \"<interface-name>\"")
	log.Println("")
	log.Println("Example (use the full interface name from above):")
	if len(devices) > 0 {
		log.Printf("  giru.exe -interface \"%s\"", devices[0].Name)
	}
}
