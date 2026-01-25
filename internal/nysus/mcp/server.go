// Package mcp implements the Model Context Protocol server for Nysus.
// MCP allows LLMs to interact with ASGARD systems through a standardized interface.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

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

	// Return resource content (placeholder - would fetch real data)
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      uri,
					"mimeType": "application/json",
					"text":     `{"status": "available"}`,
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
			satID, _ := params["satellite_id"].(string)
			return map[string]interface{}{
				"satellite_id": satID,
				"status":       "operational",
				"battery":      85.5,
				"altitude_km":  550.0,
				"last_contact": time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	})

	s.RegisterTool(&Tool{
		Name:        "command_satellite",
		Description: "Send a command to a satellite",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"satellite_id":{"type":"string"},"command":{"type":"string"},"parameters":{"type":"object"}},"required":["satellite_id","command"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			satID, _ := params["satellite_id"].(string)
			cmd, _ := params["command"].(string)
			return map[string]interface{}{
				"satellite_id": satID,
				"command":      cmd,
				"status":       "queued",
				"estimated_execution": time.Now().Add(30 * time.Second).UTC().Format(time.RFC3339),
			}, nil
		},
	})

	// Hunoid control tools
	s.RegisterTool(&Tool{
		Name:        "get_hunoid_status",
		Description: "Get the current status of a Hunoid unit",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"hunoid_id":{"type":"string"}},"required":["hunoid_id"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			hunoidID, _ := params["hunoid_id"].(string)
			return map[string]interface{}{
				"hunoid_id":    hunoidID,
				"status":       "idle",
				"battery":      92.0,
				"location":     map[string]float64{"lat": 34.05, "lon": -118.25},
				"current_mission": nil,
			}, nil
		},
	})

	s.RegisterTool(&Tool{
		Name:        "dispatch_mission",
		Description: "Dispatch a Hunoid unit to execute a mission",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"hunoid_id":{"type":"string"},"mission_type":{"type":"string"},"target_location":{"type":"object"}},"required":["hunoid_id","mission_type"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			hunoidID, _ := params["hunoid_id"].(string)
			missionType, _ := params["mission_type"].(string)
			return map[string]interface{}{
				"hunoid_id":    hunoidID,
				"mission_type": missionType,
				"mission_id":   uuid.New().String(),
				"status":       "dispatched",
				"estimated_start": time.Now().Add(5 * time.Second).UTC().Format(time.RFC3339),
			}, nil
		},
	})

	// Security tools
	s.RegisterTool(&Tool{
		Name:        "get_threat_status",
		Description: "Get current threat landscape from Giru",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"active_threats": 0,
				"threat_level":   "low",
				"last_scan":      time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
				"zones_monitored": 3,
			}, nil
		},
	})

	s.RegisterTool(&Tool{
		Name:        "initiate_scan",
		Description: "Start a security scan",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"target":{"type":"string"},"scan_type":{"type":"string"}},"required":["target"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			target, _ := params["target"].(string)
			scanType, _ := params["scan_type"].(string)
			if scanType == "" {
				scanType = "quick"
			}
			return map[string]interface{}{
				"scan_id":   uuid.New().String(),
				"target":    target,
				"scan_type": scanType,
				"status":    "in_progress",
				"started":   time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	})

	// Guidance tools (Pricilla)
	s.RegisterTool(&Tool{
		Name:        "calculate_trajectory",
		Description: "Calculate optimal trajectory using Pricilla",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"start":{"type":"object"},"destination":{"type":"object"},"constraints":{"type":"object"}},"required":["start","destination"]}`),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"trajectory_id": uuid.New().String(),
				"waypoints":     5,
				"distance_km":   125.5,
				"eta_seconds":   3600,
				"fuel_required": 45.2,
				"status":        "calculated",
			}, nil
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
