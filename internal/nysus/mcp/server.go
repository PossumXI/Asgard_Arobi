// Package mcp implements the Model Context Protocol server for Nysus.
// MCP allows LLMs to interact with ASGARD systems through a standardized interface.
package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/google/uuid"
)

// Tool represents an MCP tool that can be called by LLMs
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Handler     ToolHandler     `json:"-"`
}

// ToolHandler is a function that executes a tool
type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// Resource represents an MCP resource that provides context
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// Server is the MCP server implementation
type Server struct {
	mu        sync.RWMutex
	tools     map[string]*Tool
	resources map[string]*Resource
	sessions  map[string]*Session
	addr      string
	server    *http.Server
	pgDB      *db.PostgresDB
}

// Session tracks an MCP client session
type Session struct {
	ID        string
	CreatedAt time.Time
	LastSeen  time.Time
	Context   map[string]interface{}
}

// Config holds MCP server configuration
type Config struct {
	Addr string
}

// DefaultConfig returns default MCP configuration
func DefaultConfig() Config {
	return Config{
		Addr: ":8085",
	}
}

// NewServer creates a new MCP server
func NewServer(cfg Config) *Server {
	return &Server{
		tools:     make(map[string]*Tool),
		resources: make(map[string]*Resource),
		sessions:  make(map[string]*Session),
		addr:      cfg.Addr,
	}
}

// SetPostgresDB configures the MCP server database handle.
func (s *Server) SetPostgresDB(pgDB *db.PostgresDB) {
	s.pgDB = pgDB
}

// RegisterTool adds a tool to the MCP server
func (s *Server) RegisterTool(tool *Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = tool
	log.Printf("[MCP] Registered tool: %s", tool.Name)
}

// RegisterResource adds a resource to the MCP server
func (s *Server) RegisterResource(resource *Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[resource.URI] = resource
	log.Printf("[MCP] Registered resource: %s", resource.URI)
}

// Start begins the MCP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// MCP protocol endpoints
	mux.HandleFunc("/mcp/initialize", s.handleInitialize)
	mux.HandleFunc("/mcp/tools/list", s.handleListTools)
	mux.HandleFunc("/mcp/tools/call", s.handleCallTool)
	mux.HandleFunc("/mcp/resources/list", s.handleListResources)
	mux.HandleFunc("/mcp/resources/read", s.handleReadResource)
	mux.HandleFunc("/mcp/prompts/list", s.handleListPrompts)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	go func() {
		log.Printf("[MCP] Server listening on %s", s.addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[MCP] Server error: %v", err)
		}
	}()

	return nil
}

// Stop shuts down the MCP server
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// MCPRequest represents an incoming MCP request
type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (s *Server) handleInitialize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := uuid.New().String()
	s.mu.Lock()
	s.sessions[sessionID] = &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Context:   make(map[string]interface{}),
	}
	s.mu.Unlock()

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools":     map[string]bool{"listChanged": true},
				"resources": map[string]bool{"subscribe": true, "listChanged": true},
				"prompts":   map[string]bool{"listChanged": true},
			},
			"serverInfo": map[string]string{
				"name":    "ASGARD-Nysus-MCP",
				"version": "1.0.0",
			},
			"sessionId": sessionID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]map[string]interface{}, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleCallTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, -32700, "Parse error", nil)
		return
	}

	toolName, _ := req.Params["name"].(string)
	args, _ := req.Params["arguments"].(map[string]interface{})

	s.mu.RLock()
	tool, exists := s.tools[toolName]
	s.mu.RUnlock()

	if !exists {
		s.writeError(w, -32601, fmt.Sprintf("Tool not found: %s", toolName), nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := tool.Handler(ctx, args)
	if err != nil {
		s.writeError(w, -32000, err.Error(), nil)
		return
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("%v", result),
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleListResources(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]map[string]interface{}, 0, len(s.resources))
	for _, res := range s.resources {
		resources = append(resources, map[string]interface{}{
			"uri":         res.URI,
			"name":        res.Name,
			"description": res.Description,
			"mimeType":    res.MimeType,
		})
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]interface{}{
			"resources": resources,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleReadResource(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		s.writeError(w, -32602, "Missing uri parameter", nil)
		return
	}

	s.mu.RLock()
	_, exists := s.resources[uri]
	s.mu.RUnlock()

	if !exists {
		s.writeError(w, -32601, fmt.Sprintf("Resource not found: %s", uri), nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	mimeType, text, err := s.readResource(ctx, uri)
	if err != nil {
		s.writeError(w, -32000, err.Error(), nil)
		return
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      uri,
					"mimeType": mimeType,
					"text":     text,
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleListPrompts(w http.ResponseWriter, r *http.Request) {
	prompts := []map[string]interface{}{
		{
			"name":        "analyze_satellite",
			"description": "Analyze satellite telemetry and imagery",
			"arguments": []map[string]interface{}{
				{"name": "satellite_id", "description": "Satellite identifier", "required": true},
			},
		},
		{
			"name":        "dispatch_hunoid",
			"description": "Dispatch a Hunoid unit to a mission",
			"arguments": []map[string]interface{}{
				{"name": "mission_type", "description": "Type of mission", "required": true},
				{"name": "location", "description": "Target location", "required": true},
			},
		},
		{
			"name":        "security_scan",
			"description": "Initiate a security scan with Giru",
			"arguments": []map[string]interface{}{
				{"name": "target", "description": "Scan target", "required": true},
				{"name": "depth", "description": "Scan depth (quick/full)", "required": false},
			},
		},
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]interface{}{
			"prompts": prompts,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	toolCount := len(s.tools)
	resourceCount := len(s.resources)
	sessionCount := len(s.sessions)
	s.mu.RUnlock()

	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "ASGARD-Nysus-MCP",
		"tools":     toolCount,
		"resources": resourceCount,
		"sessions":  sessionCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) writeError(w http.ResponseWriter, code int, message string, data interface{}) {
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      nil,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RegisterDefaultTools registers the default ASGARD tools
func (s *Server) RegisterDefaultTools() {
	// Satellite control tools
	s.RegisterTool(&Tool{
		Name:        "get_satellite_status",
		Description: "Get the current status of a satellite",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"satellite_id":{"type":"string"}},"required":["satellite_id"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return s.handleSatelliteStatus(ctx, params)
		},
	})

	s.RegisterTool(&Tool{
		Name:        "command_satellite",
		Description: "Send a command to a satellite",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"satellite_id":{"type":"string"},"command":{"type":"string"},"parameters":{"type":"object"}},"required":["satellite_id","command"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return s.handleSatelliteCommand(ctx, params)
		},
	})

	// Hunoid control tools
	s.RegisterTool(&Tool{
		Name:        "get_hunoid_status",
		Description: "Get the current status of a Hunoid unit",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"hunoid_id":{"type":"string"}},"required":["hunoid_id"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return s.handleHunoidStatus(ctx, params)
		},
	})

	s.RegisterTool(&Tool{
		Name:        "dispatch_mission",
		Description: "Dispatch a Hunoid unit to execute a mission",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"hunoid_id":{"type":"string"},"mission_type":{"type":"string"},"target_location":{"type":"object"}},"required":["hunoid_id","mission_type"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return s.handleDispatchMission(ctx, params)
		},
	})

	// Security tools
	s.RegisterTool(&Tool{
		Name:        "get_threat_status",
		Description: "Get current threat landscape from Giru",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return s.handleThreatStatus(ctx)
		},
	})

	s.RegisterTool(&Tool{
		Name:        "initiate_scan",
		Description: "Start a security scan",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"target":{"type":"string"},"scan_type":{"type":"string"}},"required":["target"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return s.handleInitiateScan(ctx, params)
		},
	})

	// Guidance tools (Pricilla)
	s.RegisterTool(&Tool{
		Name:        "calculate_trajectory",
		Description: "Calculate optimal trajectory using Pricilla",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"start":{"type":"object"},"destination":{"type":"object"},"constraints":{"type":"object"}},"required":["start","destination"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return s.handleCalculateTrajectory(ctx, params)
		},
	})

	// Register default resources
	s.RegisterResource(&Resource{
		URI:         "asgard://satellites/list",
		Name:        "Satellite List",
		Description: "List of all tracked satellites",
		MimeType:    "application/json",
	})

	s.RegisterResource(&Resource{
		URI:         "asgard://hunoids/list",
		Name:        "Hunoid List",
		Description: "List of all Hunoid units",
		MimeType:    "application/json",
	})

	s.RegisterResource(&Resource{
		URI:         "asgard://alerts/recent",
		Name:        "Recent Alerts",
		Description: "Recent system alerts",
		MimeType:    "application/json",
	})

	s.RegisterResource(&Resource{
		URI:         "asgard://threats/active",
		Name:        "Active Threats",
		Description: "Currently active security threats",
		MimeType:    "application/json",
	})
}

func (s *Server) readResource(ctx context.Context, uri string) (string, string, error) {
	if s.pgDB == nil {
		return "", "", fmt.Errorf("postgres database not configured")
	}

	switch uri {
	case "asgard://satellites/list":
		repo := repositories.NewSatelliteRepository(s.pgDB)
		satellites, err := repo.GetAll()
		if err != nil {
			return "", "", err
		}
		items := make([]map[string]interface{}, 0, len(satellites))
		for _, sat := range satellites {
			item := map[string]interface{}{
				"id":     sat.ID.String(),
				"name":   sat.Name,
				"status": sat.Status,
			}
			if sat.NoradID.Valid {
				item["norad_id"] = sat.NoradID.Int32
			}
			if sat.CurrentBatteryPercent.Valid {
				item["battery_percent"] = sat.CurrentBatteryPercent.Float64
			}
			if sat.LastTelemetry.Valid {
				item["last_telemetry"] = sat.LastTelemetry.Time.UTC().Format(time.RFC3339)
			}
			if sat.FirmwareVersion.Valid {
				item["firmware_version"] = sat.FirmwareVersion.String
			}
			items = append(items, item)
		}
		payload, err := marshalJSON(map[string]interface{}{
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"count":        len(items),
			"satellites":   items,
		})
		if err != nil {
			return "", "", err
		}
		return "application/json", payload, nil
	case "asgard://hunoids/list":
		repo := repositories.NewHunoidRepository(s.pgDB)
		hunoids, err := repo.GetAll()
		if err != nil {
			return "", "", err
		}
		items := make([]map[string]interface{}, 0, len(hunoids))
		for _, hunoid := range hunoids {
			item := map[string]interface{}{
				"id":     hunoid.ID.String(),
				"serial": hunoid.SerialNumber,
				"status": hunoid.Status,
			}
			if hunoid.CurrentMissionID.Valid {
				item["current_mission_id"] = hunoid.CurrentMissionID.String
			}
			if hunoid.BatteryPercent.Valid {
				item["battery_percent"] = hunoid.BatteryPercent.Float64
			}
			if hunoid.LastTelemetry.Valid {
				item["last_telemetry"] = hunoid.LastTelemetry.Time.UTC().Format(time.RFC3339)
			}
			items = append(items, item)
		}
		payload, err := marshalJSON(map[string]interface{}{
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"count":        len(items),
			"hunoids":      items,
		})
		if err != nil {
			return "", "", err
		}
		return "application/json", payload, nil
	case "asgard://alerts/recent":
		repo := repositories.NewAlertRepository(s.pgDB)
		alerts, err := repo.GetAll()
		if err != nil {
			return "", "", err
		}
		items := make([]map[string]interface{}, 0, len(alerts))
		for _, alert := range alerts {
			item := map[string]interface{}{
				"id":         alert.ID.String(),
				"type":       alert.AlertType,
				"confidence": alert.ConfidenceScore,
				"status":     alert.Status,
				"created_at": alert.CreatedAt.UTC().Format(time.RFC3339),
			}
			if alert.SatelliteID.Valid {
				item["satellite_id"] = alert.SatelliteID.String
			}
			items = append(items, item)
		}
		payload, err := marshalJSON(map[string]interface{}{
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"count":        len(items),
			"alerts":       items,
		})
		if err != nil {
			return "", "", err
		}
		return "application/json", payload, nil
	case "asgard://threats/active":
		rows, err := s.pgDB.QueryContext(ctx, `
			SELECT id, threat_type, severity, source_ip, target_component, status, detected_at
			FROM threats
			WHERE status IS NULL OR status <> 'resolved'
			ORDER BY detected_at DESC
			LIMIT 100
		`)
		if err != nil {
			return "", "", err
		}
		defer rows.Close()

		items := make([]map[string]interface{}, 0)
		for rows.Next() {
			var id uuid.UUID
			var threatType, severity, status string
			var sourceIP, targetComponent sql.NullString
			var detectedAt time.Time
			if err := rows.Scan(&id, &threatType, &severity, &sourceIP, &targetComponent, &status, &detectedAt); err != nil {
				return "", "", err
			}
			item := map[string]interface{}{
				"id":          id.String(),
				"type":        threatType,
				"severity":    severity,
				"status":      status,
				"detected_at": detectedAt.UTC().Format(time.RFC3339),
			}
			if sourceIP.Valid {
				item["source_ip"] = sourceIP.String
			}
			if targetComponent.Valid {
				item["target_component"] = targetComponent.String
			}
			items = append(items, item)
		}
		payload, err := marshalJSON(map[string]interface{}{
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"count":        len(items),
			"threats":      items,
		})
		if err != nil {
			return "", "", err
		}
		return "application/json", payload, nil
	default:
		return "", "", fmt.Errorf("resource not found: %s", uri)
	}
}

func (s *Server) handleSatelliteStatus(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.pgDB == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	satelliteID, _ := params["satellite_id"].(string)
	if satelliteID == "" {
		return nil, fmt.Errorf("satellite_id is required")
	}

	repo := repositories.NewSatelliteRepository(s.pgDB)
	sat, err := repo.GetByID(satelliteID)
	if err != nil {
		return nil, err
	}

	response := map[string]interface{}{
		"satellite_id": sat.ID.String(),
		"name":         sat.Name,
		"status":       sat.Status,
	}
	if sat.NoradID.Valid {
		response["norad_id"] = sat.NoradID.Int32
	}
	if sat.CurrentBatteryPercent.Valid {
		response["battery_percent"] = sat.CurrentBatteryPercent.Float64
	}
	if sat.LastTelemetry.Valid {
		response["last_telemetry"] = sat.LastTelemetry.Time.UTC().Format(time.RFC3339)
	}
	if sat.FirmwareVersion.Valid {
		response["firmware_version"] = sat.FirmwareVersion.String
	}

	return response, nil
}

func (s *Server) handleSatelliteCommand(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return s.createControlCommand(ctx, params, "command_satellite", "satellite", "satellite_id")
}

func (s *Server) handleHunoidStatus(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if s.pgDB == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	hunoidID, _ := params["hunoid_id"].(string)
	if hunoidID == "" {
		return nil, fmt.Errorf("hunoid_id is required")
	}

	repo := repositories.NewHunoidRepository(s.pgDB)
	hunoid, err := repo.GetByID(hunoidID)
	if err != nil {
		return nil, err
	}

	response := map[string]interface{}{
		"hunoid_id": hunoid.ID.String(),
		"serial":    hunoid.SerialNumber,
		"status":    hunoid.Status,
	}
	if hunoid.CurrentMissionID.Valid {
		response["current_mission_id"] = hunoid.CurrentMissionID.String
	}
	if hunoid.BatteryPercent.Valid {
		response["battery_percent"] = hunoid.BatteryPercent.Float64
	}
	if hunoid.LastTelemetry.Valid {
		response["last_telemetry"] = hunoid.LastTelemetry.Time.UTC().Format(time.RFC3339)
	}

	location, err := repo.GetLocation(hunoidID)
	if err == nil && location != nil {
		response["location"] = map[string]interface{}{
			"lat": location.Latitude,
			"lon": location.Longitude,
			"alt": location.Altitude,
		}
	}

	return response, nil
}

func (s *Server) handleDispatchMission(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return s.createControlCommand(ctx, params, "dispatch_mission", "hunoid", "hunoid_id")
}

func (s *Server) handleThreatStatus(ctx context.Context) (interface{}, error) {
	if s.pgDB == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	var activeCount int
	if err := s.pgDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM threats WHERE status IS NULL OR status <> 'resolved'
	`).Scan(&activeCount); err != nil {
		return nil, err
	}

	var lastDetected sql.NullTime
	if err := s.pgDB.QueryRowContext(ctx, `
		SELECT MAX(detected_at) FROM threats
	`).Scan(&lastDetected); err != nil {
		return nil, err
	}

	var severityRank int
	if err := s.pgDB.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(CASE severity
			WHEN 'critical' THEN 4
			WHEN 'high' THEN 3
			WHEN 'medium' THEN 2
			WHEN 'low' THEN 1
			ELSE 0 END), 0)
		FROM threats
		WHERE status IS NULL OR status <> 'resolved'
	`).Scan(&severityRank); err != nil {
		return nil, err
	}

	highestSeverity := "none"
	switch severityRank {
	case 4:
		highestSeverity = "critical"
	case 3:
		highestSeverity = "high"
	case 2:
		highestSeverity = "medium"
	case 1:
		highestSeverity = "low"
	}

	response := map[string]interface{}{
		"active_threats":   activeCount,
		"highest_severity": highestSeverity,
	}
	if lastDetected.Valid {
		response["last_detected_at"] = lastDetected.Time.UTC().Format(time.RFC3339)
	}

	return response, nil
}

func (s *Server) handleInitiateScan(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return s.createControlCommand(ctx, params, "security_scan", "system", "")
}

func (s *Server) handleCalculateTrajectory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return s.createControlCommand(ctx, params, "calculate_trajectory", "system", "")
}

func (s *Server) createControlCommand(
	ctx context.Context,
	params map[string]interface{},
	commandType string,
	targetType string,
	targetIDKey string,
) (interface{}, error) {
	if s.pgDB == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	var targetID *uuid.UUID
	if targetIDKey != "" {
		rawID, _ := params[targetIDKey].(string)
		if rawID == "" {
			return nil, fmt.Errorf("%s is required", targetIDKey)
		}
		parsed, err := uuid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", targetIDKey, err)
		}
		targetID = &parsed
	}

	payloadBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	priority := 5
	if rawPriority, ok := params["priority"]; ok {
		if priorityFloat, ok := rawPriority.(float64); ok {
			priority = int(priorityFloat)
		}
	}

	var commandID uuid.UUID
	err = s.pgDB.QueryRowContext(ctx, `
		INSERT INTO control_commands (command_type, target_type, target_id, payload, status, priority)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id
	`, commandType, targetType, targetID, payloadBytes, priority).Scan(&commandID)
	if err != nil {
		return nil, fmt.Errorf("failed to create control command: %w", err)
	}

	response := map[string]interface{}{
		"command_id": commandID.String(),
		"command":    commandType,
		"status":     "pending",
		"target_type": targetType,
	}
	if targetID != nil {
		response["target_id"] = targetID.String()
	}
	return response, nil
}

func marshalJSON(value interface{}) (string, error) {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
