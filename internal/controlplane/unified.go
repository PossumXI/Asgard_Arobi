package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/dtn"
	"github.com/asgard/pandora/internal/robotics/ethics"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// UnifiedControlPlane coordinates DTN, security, and autonomy systems.
type UnifiedControlPlane struct {
	mu sync.RWMutex

	// Core components
	eventBus    *CrossDomainEventBus
	coordinator *Coordinator

	// Connected systems
	dtnNodes      map[string]*dtn.Node
	ethicsKernel  *ethics.EthicalKernel
	securityScanner SecurityScannerAdapter

	// NATS connection for external events
	natsConn *nats.Conn

	// Status tracking
	systemStatus map[string]*SystemStatus
	health       *HealthStatus

	// Event history (ring buffer)
	eventHistory    []CrossDomainEvent
	eventHistoryIdx int
	eventHistoryMax int

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	metrics *ControlPlaneMetrics
}

// SecurityScannerAdapter adapts the security scanner to the control plane.
type SecurityScannerAdapter interface {
	IsRunning() bool
	GetStatistics() map[string]interface{}
	PauseScanning()
	ResumeScanning()
}

// SystemStatus tracks the status of a connected system.
type SystemStatus struct {
	SystemID      string            `json:"system_id"`
	SystemType    AutonomySystemType `json:"system_type"`
	Domain        EventDomain       `json:"domain"`
	State         AutonomyState     `json:"state"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	LastEvent     *CrossDomainEvent `json:"last_event,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// HealthStatus represents the overall health of the control plane.
type HealthStatus struct {
	Status           string                 `json:"status"` // healthy, degraded, critical
	LastCheck        time.Time              `json:"last_check"`
	ActiveSystems    int                    `json:"active_systems"`
	TotalSystems     int                    `json:"total_systems"`
	ActiveAlerts     int                    `json:"active_alerts"`
	DomainHealth     map[EventDomain]string `json:"domain_health"`
	Uptime           time.Duration          `json:"uptime_ns"`
	startTime        time.Time
}

// ControlPlaneMetrics tracks control plane performance.
type ControlPlaneMetrics struct {
	mu                sync.RWMutex
	EventsProcessed   int64             `json:"events_processed"`
	EventsByDomain    map[EventDomain]int64 `json:"events_by_domain"`
	CommandsIssued    int64             `json:"commands_issued"`
	CommandsSucceeded int64             `json:"commands_succeeded"`
	CommandsFailed    int64             `json:"commands_failed"`
	CoordinationRuns  int64             `json:"coordination_runs"`
	AverageLatency    time.Duration     `json:"average_latency_ns"`
	LastUpdated       time.Time         `json:"last_updated"`
}

// Config holds configuration for the UnifiedControlPlane.
type Config struct {
	NATSUrl           string
	EventHistorySize  int
	HealthCheckInterval time.Duration
	HeartbeatTimeout  time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		NATSUrl:           "nats://localhost:4222",
		EventHistorySize:  10000,
		HealthCheckInterval: 30 * time.Second,
		HeartbeatTimeout:  2 * time.Minute,
	}
}

// NewUnifiedControlPlane creates a new unified control plane.
func NewUnifiedControlPlane(cfg Config) (*UnifiedControlPlane, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ucp := &UnifiedControlPlane{
		eventBus:        NewCrossDomainEventBus(),
		dtnNodes:        make(map[string]*dtn.Node),
		systemStatus:    make(map[string]*SystemStatus),
		eventHistory:    make([]CrossDomainEvent, cfg.EventHistorySize),
		eventHistoryMax: cfg.EventHistorySize,
		ctx:             ctx,
		cancel:          cancel,
		health: &HealthStatus{
			Status:       "initializing",
			DomainHealth: make(map[EventDomain]string),
			startTime:    time.Now(),
		},
		metrics: &ControlPlaneMetrics{
			EventsByDomain: make(map[EventDomain]int64),
			LastUpdated:    time.Now(),
		},
	}

	// Create coordinator with reference to control plane
	ucp.coordinator = NewCoordinator(ucp)

	// Connect to NATS if configured
	if cfg.NATSUrl != "" {
		nc, err := nats.Connect(cfg.NATSUrl,
			nats.ReconnectWait(2*time.Second),
			nats.MaxReconnects(60),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				log.Printf("[ControlPlane] Reconnected to NATS: %s", nc.ConnectedUrl())
			}),
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				if err != nil {
					log.Printf("[ControlPlane] Disconnected from NATS: %v", err)
				}
			}),
		)
		if err != nil {
			log.Printf("[ControlPlane] NATS connection failed (continuing without): %v", err)
		} else {
			ucp.natsConn = nc
		}
	}

	// Subscribe to internal events
	ucp.eventBus.SubscribeAll(ucp.handleEvent)

	return ucp, nil
}

// Start begins control plane operations.
func (ucp *UnifiedControlPlane) Start() error {
	log.Println("[ControlPlane] Starting unified control plane")

	// Start event bus
	ucp.eventBus.Start()

	// Start coordinator
	ucp.coordinator.Start()

	// Start health check goroutine
	ucp.wg.Add(1)
	go ucp.runHealthCheck()

	// Subscribe to external NATS events if connected
	if ucp.natsConn != nil {
		ucp.subscribeToNATSEvents()
	}

	ucp.mu.Lock()
	ucp.health.Status = "healthy"
	ucp.health.LastCheck = time.Now()
	ucp.mu.Unlock()

	log.Println("[ControlPlane] Started successfully")
	return nil
}

// Stop gracefully shuts down the control plane.
func (ucp *UnifiedControlPlane) Stop() error {
	log.Println("[ControlPlane] Shutting down")

	ucp.cancel()
	ucp.wg.Wait()

	ucp.coordinator.Stop()
	ucp.eventBus.Stop()

	if ucp.natsConn != nil {
		ucp.natsConn.Close()
	}

	log.Println("[ControlPlane] Shutdown complete")
	return nil
}

// RegisterDTNNode registers a DTN node with the control plane.
func (ucp *UnifiedControlPlane) RegisterDTNNode(node *dtn.Node) {
	ucp.mu.Lock()
	defer ucp.mu.Unlock()

	ucp.dtnNodes[node.ID] = node
	ucp.systemStatus[node.ID] = &SystemStatus{
		SystemID:      node.ID,
		SystemType:    SystemTypeSatellite,
		Domain:        DomainDTN,
		State:         StateActive,
		LastHeartbeat: time.Now(),
		Metadata:      map[string]interface{}{"eid": node.EID},
	}

	log.Printf("[ControlPlane] Registered DTN node: %s", node.ID)
}

// UnregisterDTNNode removes a DTN node from the control plane.
func (ucp *UnifiedControlPlane) UnregisterDTNNode(nodeID string) {
	ucp.mu.Lock()
	defer ucp.mu.Unlock()

	delete(ucp.dtnNodes, nodeID)
	delete(ucp.systemStatus, nodeID)

	log.Printf("[ControlPlane] Unregistered DTN node: %s", nodeID)
}

// RegisterEthicsKernel registers the ethics kernel.
func (ucp *UnifiedControlPlane) RegisterEthicsKernel(kernel *ethics.EthicalKernel) {
	ucp.mu.Lock()
	defer ucp.mu.Unlock()

	ucp.ethicsKernel = kernel
	ucp.systemStatus["ethics-kernel"] = &SystemStatus{
		SystemID:      "ethics-kernel",
		Domain:        DomainEthics,
		State:         StateActive,
		LastHeartbeat: time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	log.Println("[ControlPlane] Registered ethics kernel")
}

// RegisterSecurityScanner registers a security scanner adapter.
func (ucp *UnifiedControlPlane) RegisterSecurityScanner(scanner SecurityScannerAdapter) {
	ucp.mu.Lock()
	defer ucp.mu.Unlock()

	ucp.securityScanner = scanner
	ucp.systemStatus["security-scanner"] = &SystemStatus{
		SystemID:      "security-scanner",
		Domain:        DomainSecurity,
		State:         StateActive,
		LastHeartbeat: time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	log.Println("[ControlPlane] Registered security scanner")
}

// RegisterAutonomySystem registers a generic autonomous system.
func (ucp *UnifiedControlPlane) RegisterAutonomySystem(systemID string, systemType AutonomySystemType, metadata map[string]interface{}) {
	ucp.mu.Lock()
	defer ucp.mu.Unlock()

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	ucp.systemStatus[systemID] = &SystemStatus{
		SystemID:      systemID,
		SystemType:    systemType,
		Domain:        DomainAutonomy,
		State:         StateActive,
		LastHeartbeat: time.Now(),
		Metadata:      metadata,
	}

	log.Printf("[ControlPlane] Registered autonomy system: %s (%s)", systemID, systemType)
}

// PublishEvent publishes a cross-domain event.
func (ucp *UnifiedControlPlane) PublishEvent(event CrossDomainEvent) error {
	// Record in history
	ucp.recordEvent(event)

	// Update metrics
	ucp.metrics.mu.Lock()
	ucp.metrics.EventsProcessed++
	ucp.metrics.EventsByDomain[event.Domain]++
	ucp.metrics.LastUpdated = time.Now()
	ucp.metrics.mu.Unlock()

	// Publish to internal event bus
	if err := ucp.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	// Publish to NATS if connected
	if ucp.natsConn != nil {
		subject := fmt.Sprintf("asgard.controlplane.%s", event.Type)
		data, err := json.Marshal(event.ToMap())
		if err == nil {
			ucp.natsConn.Publish(subject, data)
		}
	}

	return nil
}

// IssueCommand issues a cross-domain command.
func (ucp *UnifiedControlPlane) IssueCommand(cmd ControlCommand) (*ControlResponse, error) {
	startTime := time.Now()

	// Update metrics
	ucp.metrics.mu.Lock()
	ucp.metrics.CommandsIssued++
	ucp.metrics.mu.Unlock()

	// Execute command based on target domain
	response := &ControlResponse{
		CommandID:  cmd.ID,
		ExecutedAt: time.Now(),
	}

	var err error
	switch cmd.TargetDomain {
	case DomainDTN:
		err = ucp.executeDTNCommand(cmd)
	case DomainSecurity:
		err = ucp.executeSecurityCommand(cmd)
	case DomainAutonomy:
		err = ucp.executeAutonomyCommand(cmd)
	case DomainEthics:
		err = ucp.executeEthicsCommand(cmd)
	default:
		err = fmt.Errorf("unknown target domain: %s", cmd.TargetDomain)
	}

	response.Duration = time.Since(startTime)
	response.Success = err == nil
	if err != nil {
		response.Error = err.Error()
		ucp.metrics.mu.Lock()
		ucp.metrics.CommandsFailed++
		ucp.metrics.mu.Unlock()
	} else {
		ucp.metrics.mu.Lock()
		ucp.metrics.CommandsSucceeded++
		ucp.metrics.mu.Unlock()
	}

	// Publish response event
	responseEvent := NewCrossDomainEvent(
		EventControlResponse,
		DomainControl,
		"controlplane",
		SeverityInfo,
		fmt.Sprintf("Command %s executed", cmd.CommandType),
	)
	responseEvent.Payload["command_id"] = cmd.ID.String()
	responseEvent.Payload["success"] = response.Success
	ucp.PublishEvent(responseEvent)

	return response, err
}

// GetStatus returns the current status of all systems.
func (ucp *UnifiedControlPlane) GetStatus() map[string]*SystemStatus {
	ucp.mu.RLock()
	defer ucp.mu.RUnlock()

	result := make(map[string]*SystemStatus, len(ucp.systemStatus))
	for k, v := range ucp.systemStatus {
		status := *v
		result[k] = &status
	}
	return result
}

// GetHealth returns the overall health status.
func (ucp *UnifiedControlPlane) GetHealth() *HealthStatus {
	ucp.mu.RLock()
	defer ucp.mu.RUnlock()

	health := *ucp.health
	health.Uptime = time.Since(ucp.health.startTime)
	return &health
}

// GetMetrics returns control plane metrics.
func (ucp *UnifiedControlPlane) GetMetrics() *ControlPlaneMetrics {
	ucp.metrics.mu.RLock()
	defer ucp.metrics.mu.RUnlock()

	// Copy metrics fields individually to avoid copying the mutex
	eventsByDomain := make(map[EventDomain]int64, len(ucp.metrics.EventsByDomain))
	for k, v := range ucp.metrics.EventsByDomain {
		eventsByDomain[k] = v
	}

	return &ControlPlaneMetrics{
		EventsProcessed:   ucp.metrics.EventsProcessed,
		EventsByDomain:    eventsByDomain,
		CommandsIssued:    ucp.metrics.CommandsIssued,
		CommandsSucceeded: ucp.metrics.CommandsSucceeded,
		CommandsFailed:    ucp.metrics.CommandsFailed,
		CoordinationRuns:  ucp.metrics.CoordinationRuns,
		AverageLatency:    ucp.metrics.AverageLatency,
		LastUpdated:       ucp.metrics.LastUpdated,
	}
}

// GetRecentEvents returns recent cross-domain events.
func (ucp *UnifiedControlPlane) GetRecentEvents(limit int) []CrossDomainEvent {
	ucp.mu.RLock()
	defer ucp.mu.RUnlock()

	if limit <= 0 || limit > ucp.eventHistoryMax {
		limit = 100
	}

	events := make([]CrossDomainEvent, 0, limit)
	idx := (ucp.eventHistoryIdx - 1 + ucp.eventHistoryMax) % ucp.eventHistoryMax

	for i := 0; i < limit; i++ {
		event := ucp.eventHistory[idx]
		if event.ID == uuid.Nil {
			break
		}
		events = append(events, event)
		idx = (idx - 1 + ucp.eventHistoryMax) % ucp.eventHistoryMax
	}

	return events
}

// GetCoordinator returns the coordinator instance.
func (ucp *UnifiedControlPlane) GetCoordinator() *Coordinator {
	return ucp.coordinator
}

// GetEventsByFilter returns events matching the given filter.
func (ucp *UnifiedControlPlane) GetEventsByFilter(filter EventFilter, limit int) []CrossDomainEvent {
	ucp.mu.RLock()
	defer ucp.mu.RUnlock()

	if limit <= 0 || limit > ucp.eventHistoryMax {
		limit = 100
	}

	events := make([]CrossDomainEvent, 0, limit)
	idx := (ucp.eventHistoryIdx - 1 + ucp.eventHistoryMax) % ucp.eventHistoryMax

	for i := 0; i < ucp.eventHistoryMax && len(events) < limit; i++ {
		event := ucp.eventHistory[idx]
		if event.ID == uuid.Nil {
			break
		}
		if filter(event) {
			events = append(events, event)
		}
		idx = (idx - 1 + ucp.eventHistoryMax) % ucp.eventHistoryMax
	}

	return events
}

// handleEvent processes incoming events.
func (ucp *UnifiedControlPlane) handleEvent(event CrossDomainEvent) error {
	// Update system status if applicable
	ucp.updateSystemStatus(event)

	// Forward to coordinator for policy evaluation
	ucp.coordinator.HandleEvent(event)

	return nil
}

// updateSystemStatus updates system status based on events.
func (ucp *UnifiedControlPlane) updateSystemStatus(event CrossDomainEvent) {
	ucp.mu.Lock()
	defer ucp.mu.Unlock()

	// Check if this is a status event
	if event.Type == EventAutonomyStatus {
		if status, ok := ucp.systemStatus[event.Source]; ok {
			status.LastHeartbeat = event.Timestamp
			status.LastEvent = &event
		}
	}

	// Update domain health based on event severity
	if event.Severity == SeverityCritical {
		ucp.health.DomainHealth[event.Domain] = "critical"
	} else if event.Severity == SeverityHigh {
		if ucp.health.DomainHealth[event.Domain] != "critical" {
			ucp.health.DomainHealth[event.Domain] = "degraded"
		}
	}
}

// recordEvent adds an event to the history ring buffer.
func (ucp *UnifiedControlPlane) recordEvent(event CrossDomainEvent) {
	ucp.mu.Lock()
	defer ucp.mu.Unlock()

	ucp.eventHistory[ucp.eventHistoryIdx] = event
	ucp.eventHistoryIdx = (ucp.eventHistoryIdx + 1) % ucp.eventHistoryMax
}

// runHealthCheck periodically checks system health.
func (ucp *UnifiedControlPlane) runHealthCheck() {
	defer ucp.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ucp.ctx.Done():
			return
		case <-ticker.C:
			ucp.checkHealth()
		}
	}
}

// checkHealth evaluates system health.
func (ucp *UnifiedControlPlane) checkHealth() {
	ucp.mu.Lock()
	defer ucp.mu.Unlock()

	now := time.Now()
	heartbeatTimeout := 2 * time.Minute

	activeCount := 0
	criticalCount := 0
	degradedCount := 0

	// Reset domain health
	for domain := range ucp.health.DomainHealth {
		ucp.health.DomainHealth[domain] = "healthy"
	}

	// Check each system
	for _, status := range ucp.systemStatus {
		if now.Sub(status.LastHeartbeat) > heartbeatTimeout {
			status.State = StateOffline
			ucp.health.DomainHealth[status.Domain] = "degraded"
			degradedCount++
		} else if status.State == StateEmergency {
			criticalCount++
			ucp.health.DomainHealth[status.Domain] = "critical"
		} else if status.State == StateActive || status.State == StateIdle {
			activeCount++
		}
	}

	// Update overall health
	if criticalCount > 0 {
		ucp.health.Status = "critical"
	} else if degradedCount > 0 {
		ucp.health.Status = "degraded"
	} else {
		ucp.health.Status = "healthy"
	}

	ucp.health.ActiveSystems = activeCount
	ucp.health.TotalSystems = len(ucp.systemStatus)
	ucp.health.LastCheck = now
}

// subscribeToNATSEvents subscribes to external event streams.
func (ucp *UnifiedControlPlane) subscribeToNATSEvents() {
	// Subscribe to security alerts
	ucp.natsConn.Subscribe("asgard.security.>", func(m *nats.Msg) {
		var payload map[string]interface{}
		if err := json.Unmarshal(m.Data, &payload); err != nil {
			return
		}

		event := NewCrossDomainEvent(
			EventSecurityThreat,
			DomainSecurity,
			"giru-scanner",
			Severity(getStringOr(payload, "severity", "medium")),
			getStringOr(payload, "description", "Security event"),
		)
		event.Payload = payload
		ucp.PublishEvent(event)
	})

	// Subscribe to DTN events
	ucp.natsConn.Subscribe("asgard.dtn.>", func(m *nats.Msg) {
		var payload map[string]interface{}
		if err := json.Unmarshal(m.Data, &payload); err != nil {
			return
		}

		event := NewCrossDomainEvent(
			EventDTNBundleReceived,
			DomainDTN,
			getStringOr(payload, "node_id", "unknown"),
			SeverityInfo,
			"DTN event received",
		)
		event.Payload = payload
		ucp.PublishEvent(event)
	})

	log.Println("[ControlPlane] Subscribed to NATS event streams")
}

// executeDTNCommand executes a command targeting the DTN domain.
func (ucp *UnifiedControlPlane) executeDTNCommand(cmd ControlCommand) error {
	ucp.mu.RLock()
	defer ucp.mu.RUnlock()

	switch cmd.CommandType {
	case "adjust_priority":
		// Adjust bundle priorities across DTN nodes
		log.Printf("[ControlPlane] Adjusting DTN priorities: %v", cmd.Parameters)
		return nil
	case "pause_node":
		nodeID, ok := cmd.Parameters["node_id"].(string)
		if !ok {
			return fmt.Errorf("missing node_id parameter")
		}
		if _, exists := ucp.dtnNodes[nodeID]; !exists {
			return fmt.Errorf("unknown DTN node: %s", nodeID)
		}
		log.Printf("[ControlPlane] Pausing DTN node: %s", nodeID)
		return nil
	case "resume_node":
		nodeID, ok := cmd.Parameters["node_id"].(string)
		if !ok {
			return fmt.Errorf("missing node_id parameter")
		}
		log.Printf("[ControlPlane] Resuming DTN node: %s", nodeID)
		return nil
	default:
		return fmt.Errorf("unknown DTN command: %s", cmd.CommandType)
	}
}

// executeSecurityCommand executes a command targeting the security domain.
func (ucp *UnifiedControlPlane) executeSecurityCommand(cmd ControlCommand) error {
	ucp.mu.RLock()
	defer ucp.mu.RUnlock()

	if ucp.securityScanner == nil {
		return fmt.Errorf("no security scanner registered")
	}

	switch cmd.CommandType {
	case "pause_scanning":
		ucp.securityScanner.PauseScanning()
		log.Println("[ControlPlane] Paused security scanning")
		return nil
	case "resume_scanning":
		ucp.securityScanner.ResumeScanning()
		log.Println("[ControlPlane] Resumed security scanning")
		return nil
	case "escalate_threat":
		log.Printf("[ControlPlane] Escalating threat: %v", cmd.Parameters)
		return nil
	default:
		return fmt.Errorf("unknown security command: %s", cmd.CommandType)
	}
}

// executeAutonomyCommand executes a command targeting the autonomy domain.
func (ucp *UnifiedControlPlane) executeAutonomyCommand(cmd ControlCommand) error {
	ucp.mu.RLock()
	defer ucp.mu.RUnlock()

	systemID, _ := cmd.Parameters["system_id"].(string)
	if systemID == "" {
		systemID = cmd.TargetSystem
	}

	status, exists := ucp.systemStatus[systemID]
	if !exists && systemID != "" {
		return fmt.Errorf("unknown autonomy system: %s", systemID)
	}

	switch cmd.CommandType {
	case "halt":
		if status != nil {
			status.State = StatePaused
		}
		log.Printf("[ControlPlane] Halting autonomy system: %s", systemID)
		return nil
	case "resume":
		if status != nil {
			status.State = StateActive
		}
		log.Printf("[ControlPlane] Resuming autonomy system: %s", systemID)
		return nil
	case "emergency_stop":
		if status != nil {
			status.State = StateEmergency
		}
		log.Printf("[ControlPlane] Emergency stop for: %s", systemID)
		return nil
	case "halt_all":
		for id, s := range ucp.systemStatus {
			if s.Domain == DomainAutonomy {
				s.State = StatePaused
				log.Printf("[ControlPlane] Halting: %s", id)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown autonomy command: %s", cmd.CommandType)
	}
}

// executeEthicsCommand executes a command targeting the ethics domain.
func (ucp *UnifiedControlPlane) executeEthicsCommand(cmd ControlCommand) error {
	switch cmd.CommandType {
	case "override_decision":
		log.Printf("[ControlPlane] Overriding ethics decision: %v", cmd.Parameters)
		return nil
	case "escalate_to_human":
		log.Printf("[ControlPlane] Escalating to human review: %v", cmd.Parameters)
		return nil
	default:
		return fmt.Errorf("unknown ethics command: %s", cmd.CommandType)
	}
}

// Helper function to safely get string from map
func getStringOr(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}

// CrossDomainEventBus manages cross-domain event distribution.
type CrossDomainEventBus struct {
	mu        sync.RWMutex
	handlers  map[CrossDomainEventType][]EventHandler
	wildcard  []EventHandler
	eventChan chan CrossDomainEvent
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewCrossDomainEventBus creates a new event bus.
func NewCrossDomainEventBus() *CrossDomainEventBus {
	ctx, cancel := context.WithCancel(context.Background())

	return &CrossDomainEventBus{
		handlers:  make(map[CrossDomainEventType][]EventHandler),
		eventChan: make(chan CrossDomainEvent, 10000),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Subscribe registers a handler for a specific event type.
func (eb *CrossDomainEventBus) Subscribe(eventType CrossDomainEventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// SubscribeAll registers a handler for all events.
func (eb *CrossDomainEventBus) SubscribeAll(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.wildcard = append(eb.wildcard, handler)
}

// Publish sends an event to all subscribers.
func (eb *CrossDomainEventBus) Publish(event CrossDomainEvent) error {
	select {
	case eb.eventChan <- event:
		return nil
	case <-eb.ctx.Done():
		return eb.ctx.Err()
	}
}

// Start begins processing events.
func (eb *CrossDomainEventBus) Start() {
	eb.wg.Add(1)
	go eb.processEvents()
	log.Println("[EventBus] Cross-domain event bus started")
}

// Stop gracefully shuts down the event bus.
func (eb *CrossDomainEventBus) Stop() {
	eb.cancel()
	eb.wg.Wait()
	close(eb.eventChan)
	log.Println("[EventBus] Cross-domain event bus stopped")
}

func (eb *CrossDomainEventBus) processEvents() {
	defer eb.wg.Done()

	for {
		select {
		case event := <-eb.eventChan:
			eb.dispatch(event)
		case <-eb.ctx.Done():
			return
		}
	}
}

func (eb *CrossDomainEventBus) dispatch(event CrossDomainEvent) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	wildcardHandlers := eb.wildcard
	eb.mu.RUnlock()

	// Dispatch to specific handlers
	for _, handler := range handlers {
		if err := handler(event); err != nil {
			log.Printf("[EventBus] Handler error for event %s: %v", event.ID, err)
		}
	}

	// Dispatch to wildcard handlers
	for _, handler := range wildcardHandlers {
		if err := handler(event); err != nil {
			log.Printf("[EventBus] Wildcard handler error: %v", err)
		}
	}
}
