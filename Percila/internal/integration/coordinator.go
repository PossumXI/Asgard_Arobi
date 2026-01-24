package integration

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/asgard/pandora/Percila/internal/guidance"
	"github.com/asgard/pandora/internal/nysus/events"
	"github.com/asgard/pandora/internal/platform/dtn"
	"github.com/asgard/pandora/pkg/bundle"
	"github.com/google/uuid"
)

// SystemCoordinator integrates Percila with all ASGARD systems.
type SystemCoordinator struct {
	guidanceEngine *guidance.AIGuidanceEngine
	dtnNode        *dtn.Node
	eventBus       *events.EventBus
	activeMissions map[string]*GuidedMission
	mu             sync.RWMutex
}

// GuidedMission tracks an active guidance mission.
type GuidedMission struct {
	ID              string
	PayloadID       string
	PayloadType     guidance.PayloadType
	CurrentTraj     *guidance.Trajectory
	TargetReached   bool
	TelemetryStream chan guidance.State
}

func NewSystemCoordinator(engine *guidance.AIGuidanceEngine, dtnNode *dtn.Node, eventBus *events.EventBus) *SystemCoordinator {
	return &SystemCoordinator{
		guidanceEngine: engine,
		dtnNode:        dtnNode,
		eventBus:       eventBus,
		activeMissions: make(map[string]*GuidedMission),
	}
}

// StartGuidedMission initiates guidance for a payload.
func (c *SystemCoordinator) StartGuidedMission(ctx context.Context, payloadID string, payloadType guidance.PayloadType, target guidance.Vector3D) error {
	log.Printf("Percila: Starting guided mission for %s (type: %s)", payloadID, payloadType)

	// Get current position from Nysus via event bus
	currentPos, err := c.getCurrentPosition(payloadID)
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}

	// Get threat data from Giru
	threats := c.getActiveThreatLocations()

	// Plan trajectory
	req := guidance.TrajectoryRequest{
		PayloadType:    payloadType,
		StartPosition:  currentPos,
		TargetPosition: target,
		Priority:       guidance.PriorityHigh,
		Constraints: guidance.MissionConstraints{
			StealthRequired:  true,
			MaxDetectionRisk: 0.3,
			MustAvoidThreats: threats,
		},
	}

	traj, err := c.guidanceEngine.PlanTrajectory(ctx, req)
	if err != nil {
		return fmt.Errorf("trajectory planning failed: %w", err)
	}

	log.Printf("Percila: Trajectory planned with %d waypoints, stealth score: %.2f",
		len(traj.Waypoints), traj.StealthScore)

	// Create mission
	mission := &GuidedMission{
		ID:              uuid.New().String(),
		PayloadID:       payloadID,
		PayloadType:     payloadType,
		CurrentTraj:     traj,
		TelemetryStream: make(chan guidance.State, 100),
	}

	c.mu.Lock()
	c.activeMissions[mission.ID] = mission
	c.mu.Unlock()

	// Send trajectory to payload via Sat_Net
	if err := c.transmitTrajectory(traj, payloadID); err != nil {
		return fmt.Errorf("failed to transmit trajectory: %w", err)
	}

	// Start monitoring
	go c.monitorMission(ctx, mission)

	return nil
}

// UpdateMission recalculates trajectory based on real-time data.
func (c *SystemCoordinator) UpdateMission(missionID string, currentState guidance.State) error {
	c.mu.RLock()
	mission, exists := c.activeMissions[missionID]
	c.mu.RUnlock()
	if !exists {
		return fmt.Errorf("mission not found: %s", missionID)
	}

	// Update trajectory
	newTraj, err := c.guidanceEngine.UpdateTrajectory(context.Background(), currentState, mission.CurrentTraj)
	if err != nil {
		return fmt.Errorf("trajectory update failed: %w", err)
	}

	// If trajectory changed significantly, retransmit
	if newTraj.ID != mission.CurrentTraj.ID {
		log.Printf("Percila: Trajectory updated for mission %s", missionID)
		mission.CurrentTraj = newTraj
		if err := c.transmitTrajectory(newTraj, mission.PayloadID); err != nil {
			log.Printf("Percila: Failed to retransmit trajectory: %v", err)
		}
	}

	return nil
}

// IntegrateWithSilenus uses satellite imagery for terrain mapping.
func (c *SystemCoordinator) IntegrateWithSilenus(satelliteID string) ([][]float64, error) {
	log.Printf("Percila: Requesting terrain data from Silenus satellite %s", satelliteID)

	// TODO: In production, fetch actual satellite imagery
	// For now, generate mock terrain
	terrainMap := make([][]float64, 100)
	for i := range terrainMap {
		terrainMap[i] = make([]float64, 100)
		for j := range terrainMap[i] {
			// Simulate hills and valleys
			terrainMap[i][j] = 500 + (100 * float64(i%10)) - (50 * float64(j%5))
		}
	}

	return terrainMap, nil
}

// IntegrateWithGiru gets real-time threat intelligence.
func (c *SystemCoordinator) getActiveThreatLocations() []guidance.ThreatLocation {
	// TODO: Query Giru for active threats
	// For now, return mock data
	return []guidance.ThreatLocation{
		{
			Position:     guidance.Vector3D{X: 5000, Y: 5000, Z: 0},
			ThreatType:   "radar_station",
			EffectRadius: 10000,
			Confidence:   0.9,
			LastUpdated:  time.Now().UTC(),
		},
		{
			Position:     guidance.Vector3D{X: 8000, Y: 3000, Z: 0},
			ThreatType:   "sam_site",
			EffectRadius: 15000,
			Confidence:   0.85,
			LastUpdated:  time.Now().UTC(),
		},
	}
}

// getCurrentPosition queries Nysus for payload location.
func (c *SystemCoordinator) getCurrentPosition(payloadID string) (guidance.Vector3D, error) {
	// TODO: Query Nysus database for current position
	// For now, return mock position
	return guidance.Vector3D{X: 0, Y: 0, Z: 1000}, nil
}

// transmitTrajectory sends trajectory to payload via Sat_Net DTN.
func (c *SystemCoordinator) transmitTrajectory(traj *guidance.Trajectory, payloadID string) error {
	// Serialize trajectory (simplified)
	trajData := fmt.Sprintf("TRAJ:%s:%d_waypoints", traj.ID, len(traj.Waypoints))

	// Create DTN bundle
	b, err := bundle.NewPriorityBundle(
		"dtn://asgard/percila",
		fmt.Sprintf("dtn://asgard/%s", payloadID),
		[]byte(trajData),
		bundle.PriorityExpedited,
	)
	if err != nil {
		return err
	}

	// Send via DTN node
	if c.dtnNode != nil {
		return c.dtnNode.Send(context.Background(), b)
	}

	log.Printf("Percila: Trajectory transmitted to %s", payloadID)
	return nil
}

// monitorMission tracks mission progress.
func (c *SystemCoordinator) monitorMission(ctx context.Context, mission *GuidedMission) {
	log.Printf("Percila: Monitoring mission %s", mission.ID)

	for {
		select {
		case state := <-mission.TelemetryStream:
			// Update trajectory based on current state
			if err := c.UpdateMission(mission.ID, state); err != nil {
				log.Printf("Percila: Mission update failed: %v", err)
			}

			// Check if target reached
			targetPos := mission.CurrentTraj.Waypoints[len(mission.CurrentTraj.Waypoints)-1].Position
			distance := math.Sqrt(guidance.Distance(state.Position, targetPos))

			if distance < 10.0 { // Within 10 meters
				mission.TargetReached = true
				log.Printf("Percila: Mission %s completed - target reached", mission.ID)

				// Publish mission complete event
				c.publishMissionComplete(mission)
				return
			}

		case <-ctx.Done():
			log.Printf("Percila: Mission %s monitoring stopped", mission.ID)
			return
		}
	}
}

// publishMissionComplete sends event to Nysus.
func (c *SystemCoordinator) publishMissionComplete(mission *GuidedMission) {
	if c.eventBus == nil {
		return
	}

	event := events.Event{
		ID:        uuid.New(),
		Type:      events.EventTypeMissionCompleted,
		Source:    "percila",
		Timestamp: time.Now().UTC(),
		Priority:  2,
		Payload: events.MissionEvent{
			MissionID: mission.ID,
			Status:    "completed",
		},
	}

	if err := c.eventBus.Publish(event); err != nil {
		log.Printf("Percila: Failed to publish mission completion: %v", err)
	}
}
