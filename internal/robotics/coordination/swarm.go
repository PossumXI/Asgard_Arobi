// Package coordination implements multi-robot coordination for Hunoid swarms.
package coordination

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SwarmState represents the collective state of a robot swarm
type SwarmState string

const (
	SwarmStateIdle       SwarmState = "idle"
	SwarmStateForming    SwarmState = "forming"
	SwarmStateOperating  SwarmState = "operating"
	SwarmStateDisbanding SwarmState = "disbanding"
	SwarmStateEmergency  SwarmState = "emergency"
)

// FormationType defines swarm formation patterns
type FormationType string

const (
	FormationLine     FormationType = "line"
	FormationColumn   FormationType = "column"
	FormationWedge    FormationType = "wedge"
	FormationCircle   FormationType = "circle"
	FormationGrid     FormationType = "grid"
	FormationScatter  FormationType = "scatter"
	FormationCustom   FormationType = "custom"
)

// Vector3 represents a 3D position
type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// RobotStatus represents individual robot status
type RobotStatus struct {
	ID             string
	Position       Vector3
	Velocity       Vector3
	Battery        float64
	Status         string
	CurrentTask    string
	LastHeartbeat  time.Time
	IsLeader       bool
	FormationSlot  int
}

// SwarmMission defines a coordinated mission
type SwarmMission struct {
	ID           string
	Name         string
	Type         string
	Priority     int
	TargetArea   Area
	Formation    FormationType
	Objectives   []Objective
	AssignedBots []string
	Status       string
	StartTime    time.Time
	EndTime      *time.Time
	Progress     float64
}

// Area defines a geographic region
type Area struct {
	Center  Vector3 `json:"center"`
	Radius  float64 `json:"radius"`
	Polygon []Vector3 `json:"polygon,omitempty"`
}

// Objective represents a mission objective
type Objective struct {
	ID          string
	Description string
	Location    Vector3
	AssignedTo  string
	Completed   bool
	Priority    int
}

// SwarmCommand represents a command to the swarm
type SwarmCommand struct {
	ID        string
	Type      string
	Targets   []string
	Payload   map[string]interface{}
	Timestamp time.Time
	Sender    string
}

// Coordinator manages multi-robot coordination
type Coordinator struct {
	mu            sync.RWMutex
	robots        map[string]*RobotStatus
	missions      map[string]*SwarmMission
	swarmState    SwarmState
	leaderID      string
	formation     FormationType
	formationPos  map[string]Vector3
	commandChan   chan SwarmCommand
	telemetryChan chan RobotStatus
	config        CoordinatorConfig
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// CoordinatorConfig configures the swarm coordinator
type CoordinatorConfig struct {
	HeartbeatInterval   time.Duration
	HeartbeatTimeout    time.Duration
	FormationSpacing    float64
	MaxSwarmSize        int
	ConsensusThreshold  float64
	EnableAutoFormation bool
}

// DefaultCoordinatorConfig returns default configuration
func DefaultCoordinatorConfig() CoordinatorConfig {
	return CoordinatorConfig{
		HeartbeatInterval:   1 * time.Second,
		HeartbeatTimeout:    5 * time.Second,
		FormationSpacing:    2.0,
		MaxSwarmSize:        20,
		ConsensusThreshold:  0.6,
		EnableAutoFormation: true,
	}
}

// NewCoordinator creates a new swarm coordinator
func NewCoordinator(cfg CoordinatorConfig) *Coordinator {
	return &Coordinator{
		robots:        make(map[string]*RobotStatus),
		missions:      make(map[string]*SwarmMission),
		swarmState:    SwarmStateIdle,
		formationPos:  make(map[string]Vector3),
		commandChan:   make(chan SwarmCommand, 100),
		telemetryChan: make(chan RobotStatus, 100),
		config:        cfg,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the coordinator
func (c *Coordinator) Start(ctx context.Context) error {
	// Start heartbeat monitor
	c.wg.Add(1)
	go c.heartbeatMonitor(ctx)

	// Start command processor
	c.wg.Add(1)
	go c.processCommands(ctx)

	// Start telemetry processor
	c.wg.Add(1)
	go c.processTelemetry(ctx)

	// Start formation controller
	c.wg.Add(1)
	go c.formationController(ctx)

	log.Printf("[Swarm] Coordinator started")
	return nil
}

// Stop shuts down the coordinator
func (c *Coordinator) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	log.Printf("[Swarm] Coordinator stopped")
}

// RegisterRobot adds a robot to the swarm
func (c *Coordinator) RegisterRobot(id string, position Vector3) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.robots) >= c.config.MaxSwarmSize {
		return fmt.Errorf("swarm at maximum capacity")
	}

	c.robots[id] = &RobotStatus{
		ID:            id,
		Position:      position,
		Battery:       100.0,
		Status:        "active",
		LastHeartbeat: time.Now(),
		FormationSlot: len(c.robots),
	}

	// First robot becomes leader
	if len(c.robots) == 1 {
		c.robots[id].IsLeader = true
		c.leaderID = id
	}

	log.Printf("[Swarm] Robot registered: %s", id)
	return nil
}

// UnregisterRobot removes a robot from the swarm
func (c *Coordinator) UnregisterRobot(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.robots[id]; !exists {
		return fmt.Errorf("robot not found: %s", id)
	}

	wasLeader := c.robots[id].IsLeader
	delete(c.robots, id)

	// Elect new leader if needed
	if wasLeader {
		c.electLeader()
	}

	// Recalculate formation slots
	c.reassignFormationSlots()

	log.Printf("[Swarm] Robot unregistered: %s", id)
	return nil
}

// UpdateTelemetry updates robot status
func (c *Coordinator) UpdateTelemetry(status RobotStatus) {
	select {
	case c.telemetryChan <- status:
	default:
		log.Printf("[Swarm] Warning: telemetry channel full")
	}
}

// SendCommand sends a command to the swarm
func (c *Coordinator) SendCommand(cmdType string, targets []string, payload map[string]interface{}) error {
	cmd := SwarmCommand{
		ID:        uuid.New().String(),
		Type:      cmdType,
		Targets:   targets,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	select {
	case c.commandChan <- cmd:
		log.Printf("[Swarm] Command sent: %s to %v", cmdType, targets)
		return nil
	default:
		return fmt.Errorf("command channel full")
	}
}

// CreateMission creates a new swarm mission
func (c *Coordinator) CreateMission(name, missionType string, targetArea Area, formation FormationType) (*SwarmMission, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	mission := &SwarmMission{
		ID:           uuid.New().String(),
		Name:         name,
		Type:         missionType,
		Priority:     5,
		TargetArea:   targetArea,
		Formation:    formation,
		AssignedBots: make([]string, 0),
		Status:       "created",
		Objectives:   make([]Objective, 0),
	}

	c.missions[mission.ID] = mission
	log.Printf("[Swarm] Mission created: %s (%s)", name, mission.ID)
	return mission, nil
}

// AssignMission assigns robots to a mission
func (c *Coordinator) AssignMission(missionID string, robotIDs []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	mission, exists := c.missions[missionID]
	if !exists {
		return fmt.Errorf("mission not found: %s", missionID)
	}

	for _, id := range robotIDs {
		if _, exists := c.robots[id]; !exists {
			return fmt.Errorf("robot not found: %s", id)
		}
	}

	mission.AssignedBots = robotIDs
	mission.Status = "assigned"

	log.Printf("[Swarm] Mission %s assigned to %d robots", missionID, len(robotIDs))
	return nil
}

// StartMission begins mission execution
func (c *Coordinator) StartMission(missionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	mission, exists := c.missions[missionID]
	if !exists {
		return fmt.Errorf("mission not found: %s", missionID)
	}

	if len(mission.AssignedBots) == 0 {
		return fmt.Errorf("no robots assigned to mission")
	}

	mission.Status = "active"
	mission.StartTime = time.Now()
	c.formation = mission.Formation
	c.swarmState = SwarmStateOperating

	// Calculate formation positions
	c.calculateFormation(mission.Formation, mission.TargetArea.Center)

	log.Printf("[Swarm] Mission %s started", missionID)
	return nil
}

// SetFormation changes the swarm formation
func (c *Coordinator) SetFormation(formation FormationType) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.formation = formation
	c.swarmState = SwarmStateForming

	// Get center of current swarm
	center := c.getSwarmCenter()
	c.calculateFormation(formation, center)

	log.Printf("[Swarm] Formation changed to: %s", formation)
	return nil
}

// GetSwarmStatus returns current swarm status
func (c *Coordinator) GetSwarmStatus() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	robots := make([]map[string]interface{}, 0, len(c.robots))
	for _, r := range c.robots {
		robots = append(robots, map[string]interface{}{
			"id":             r.ID,
			"position":       r.Position,
			"battery":        r.Battery,
			"status":         r.Status,
			"isLeader":       r.IsLeader,
			"formationSlot":  r.FormationSlot,
			"currentTask":    r.CurrentTask,
		})
	}

	activeMissions := make([]string, 0)
	for id, m := range c.missions {
		if m.Status == "active" {
			activeMissions = append(activeMissions, id)
		}
	}

	return map[string]interface{}{
		"swarmState":      c.swarmState,
		"formation":       c.formation,
		"leaderID":        c.leaderID,
		"robotCount":      len(c.robots),
		"robots":          robots,
		"activeMissions":  activeMissions,
		"swarmCenter":     c.getSwarmCenter(),
	}
}

// GetRobotStatus returns status of a specific robot
func (c *Coordinator) GetRobotStatus(robotID string) (*RobotStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	robot, exists := c.robots[robotID]
	if !exists {
		return nil, fmt.Errorf("robot not found: %s", robotID)
	}
	return robot, nil
}

// GetFormationPosition returns target position for a robot
func (c *Coordinator) GetFormationPosition(robotID string) (Vector3, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	pos, exists := c.formationPos[robotID]
	if !exists {
		return Vector3{}, fmt.Errorf("no formation position for robot: %s", robotID)
	}
	return pos, nil
}

// EmergencyStop halts all robots
func (c *Coordinator) EmergencyStop() {
	c.mu.Lock()
	c.swarmState = SwarmStateEmergency
	c.mu.Unlock()

	targets := make([]string, 0)
	c.mu.RLock()
	for id := range c.robots {
		targets = append(targets, id)
	}
	c.mu.RUnlock()

	c.SendCommand("emergency_stop", targets, nil)
	log.Printf("[Swarm] EMERGENCY STOP issued")
}

// Internal methods
func (c *Coordinator) heartbeatMonitor(ctx context.Context) {
	defer c.wg.Done()
	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkHeartbeats()
		}
	}
}

func (c *Coordinator) checkHeartbeats() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for id, robot := range c.robots {
		if now.Sub(robot.LastHeartbeat) > c.config.HeartbeatTimeout {
			robot.Status = "offline"
			log.Printf("[Swarm] Robot %s heartbeat timeout", id)

			// Handle leader failure
			if robot.IsLeader {
				robot.IsLeader = false
				c.electLeader()
			}
		}
	}
}

func (c *Coordinator) processCommands(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		case cmd := <-c.commandChan:
			c.executeCommand(cmd)
		}
	}
}

func (c *Coordinator) executeCommand(cmd SwarmCommand) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, targetID := range cmd.Targets {
		robot, exists := c.robots[targetID]
		if !exists {
			continue
		}

		switch cmd.Type {
		case "move_to":
			if pos, ok := cmd.Payload["position"].(Vector3); ok {
				robot.CurrentTask = fmt.Sprintf("moving to (%.2f, %.2f, %.2f)", pos.X, pos.Y, pos.Z)
			}
		case "emergency_stop":
			robot.CurrentTask = "stopped"
			robot.Status = "emergency_stop"
		case "resume":
			robot.Status = "active"
		case "formation_position":
			if pos, ok := c.formationPos[targetID]; ok {
				robot.CurrentTask = fmt.Sprintf("moving to formation (%.2f, %.2f)", pos.X, pos.Y)
			}
		}
	}
}

func (c *Coordinator) processTelemetry(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		case status := <-c.telemetryChan:
			c.mu.Lock()
			if robot, exists := c.robots[status.ID]; exists {
				robot.Position = status.Position
				robot.Velocity = status.Velocity
				robot.Battery = status.Battery
				robot.Status = status.Status
				robot.LastHeartbeat = time.Now()
			}
			c.mu.Unlock()
		}
	}
}

func (c *Coordinator) formationController(ctx context.Context) {
	defer c.wg.Done()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.config.EnableAutoFormation && c.swarmState == SwarmStateOperating {
				c.adjustFormation()
			}
		}
	}
}

func (c *Coordinator) adjustFormation() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if robots need position updates
	for id, robot := range c.robots {
		targetPos, exists := c.formationPos[id]
		if !exists {
			continue
		}

		dist := distance(robot.Position, targetPos)
		if dist > c.config.FormationSpacing/2 {
			// Send move command
			go c.SendCommand("formation_position", []string{id}, map[string]interface{}{
				"position": targetPos,
			})
		}
	}
}

func (c *Coordinator) calculateFormation(formation FormationType, center Vector3) {
	robots := make([]*RobotStatus, 0, len(c.robots))
	for _, r := range c.robots {
		robots = append(robots, r)
	}

	n := len(robots)
	if n == 0 {
		return
	}

	spacing := c.config.FormationSpacing

	switch formation {
	case FormationLine:
		for i, robot := range robots {
			offset := float64(i) - float64(n-1)/2
			c.formationPos[robot.ID] = Vector3{
				X: center.X + offset*spacing,
				Y: center.Y,
				Z: center.Z,
			}
		}

	case FormationColumn:
		for i, robot := range robots {
			offset := float64(i) - float64(n-1)/2
			c.formationPos[robot.ID] = Vector3{
				X: center.X,
				Y: center.Y + offset*spacing,
				Z: center.Z,
			}
		}

	case FormationWedge:
		for i, robot := range robots {
			row := i / 2
			side := i % 2
			xOffset := float64(row) * spacing
			yOffset := float64(row) * spacing * 0.5
			if side == 1 {
				yOffset = -yOffset
			}
			c.formationPos[robot.ID] = Vector3{
				X: center.X - xOffset,
				Y: center.Y + yOffset,
				Z: center.Z,
			}
		}

	case FormationCircle:
		radius := spacing * float64(n) / (2 * math.Pi)
		if radius < spacing {
			radius = spacing
		}
		for i, robot := range robots {
			angle := 2 * math.Pi * float64(i) / float64(n)
			c.formationPos[robot.ID] = Vector3{
				X: center.X + radius*math.Cos(angle),
				Y: center.Y + radius*math.Sin(angle),
				Z: center.Z,
			}
		}

	case FormationGrid:
		cols := int(math.Ceil(math.Sqrt(float64(n))))
		for i, robot := range robots {
			row := i / cols
			col := i % cols
			c.formationPos[robot.ID] = Vector3{
				X: center.X + float64(col-cols/2)*spacing,
				Y: center.Y + float64(row)*spacing,
				Z: center.Z,
			}
		}

	case FormationScatter:
		// Random positions within radius
		for _, robot := range robots {
			angle := 2 * math.Pi * float64(robot.FormationSlot) / float64(n)
			radius := spacing * (1 + float64(robot.FormationSlot%3))
			c.formationPos[robot.ID] = Vector3{
				X: center.X + radius*math.Cos(angle),
				Y: center.Y + radius*math.Sin(angle),
				Z: center.Z,
			}
		}
	}
}

func (c *Coordinator) electLeader() {
	// Simple leader election: highest battery among active robots
	var newLeader *RobotStatus
	maxBattery := 0.0

	for _, robot := range c.robots {
		if robot.Status == "active" && robot.Battery > maxBattery {
			maxBattery = robot.Battery
			newLeader = robot
		}
	}

	if newLeader != nil {
		newLeader.IsLeader = true
		c.leaderID = newLeader.ID
		log.Printf("[Swarm] New leader elected: %s", newLeader.ID)
	}
}

func (c *Coordinator) reassignFormationSlots() {
	slot := 0
	for _, robot := range c.robots {
		robot.FormationSlot = slot
		slot++
	}
}

func (c *Coordinator) getSwarmCenter() Vector3 {
	if len(c.robots) == 0 {
		return Vector3{}
	}

	var sum Vector3
	for _, robot := range c.robots {
		sum.X += robot.Position.X
		sum.Y += robot.Position.Y
		sum.Z += robot.Position.Z
	}

	n := float64(len(c.robots))
	return Vector3{
		X: sum.X / n,
		Y: sum.Y / n,
		Z: sum.Z / n,
	}
}

func distance(a, b Vector3) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// GenerateReport creates a JSON report of swarm status
func (c *Coordinator) GenerateReport() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	report := map[string]interface{}{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"swarm_status": c.GetSwarmStatus(),
		"missions":     c.missions,
	}

	return json.MarshalIndent(report, "", "  ")
}
