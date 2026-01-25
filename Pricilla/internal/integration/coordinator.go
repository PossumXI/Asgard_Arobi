package integration

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/asgard/pandora/Pricilla/internal/guidance"
	"github.com/asgard/pandora/internal/nysus/events"
	"github.com/asgard/pandora/internal/platform/dtn"
	"github.com/asgard/pandora/pkg/bundle"
	"github.com/google/uuid"
)

// SystemCoordinator integrates Pricilla with all ASGARD systems.
type SystemCoordinator struct {
	guidanceEngine *guidance.AIGuidanceEngine
	dtnNode        *dtn.Node
	eventBus       *events.EventBus
	silenusClient  SilenusClient
	giruClient     GiruClient
	nysusClient    NysusClient
	satnetClient   SatNetClient
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

func NewSystemCoordinator(
	engine *guidance.AIGuidanceEngine,
	dtnNode *dtn.Node,
	eventBus *events.EventBus,
	silenusClient SilenusClient,
	giruClient GiruClient,
	nysusClient NysusClient,
	satnetClient SatNetClient,
) *SystemCoordinator {
	return &SystemCoordinator{
		guidanceEngine: engine,
		dtnNode:        dtnNode,
		eventBus:       eventBus,
		silenusClient:  silenusClient,
		giruClient:     giruClient,
		nysusClient:    nysusClient,
		satnetClient:   satnetClient,
		activeMissions: make(map[string]*GuidedMission),
	}
}

// StartGuidedMission initiates guidance for a payload.
func (c *SystemCoordinator) StartGuidedMission(ctx context.Context, payloadID string, payloadType guidance.PayloadType, target guidance.Vector3D) error {
	log.Printf("Pricilla: Starting guided mission for %s (type: %s)", payloadID, payloadType)

	// Get current position from Nysus via event bus
	currentPos, err := c.getCurrentPosition(ctx, payloadID)
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}

	// Get threat data from Giru
	threats, err := c.getActiveThreatLocations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get threat locations: %w", err)
	}

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

	log.Printf("Pricilla: Trajectory planned with %d waypoints, stealth score: %.2f",
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
		log.Printf("Pricilla: Trajectory updated for mission %s", missionID)
		mission.CurrentTraj = newTraj
		if err := c.transmitTrajectory(newTraj, mission.PayloadID); err != nil {
			log.Printf("Pricilla: Failed to retransmit trajectory: %v", err)
		}
	}

	return nil
}

// IntegrateWithSilenus uses satellite imagery for terrain mapping.
func (c *SystemCoordinator) IntegrateWithSilenus(satelliteID string) ([][]float64, error) {
	log.Printf("Pricilla: Requesting terrain data from Silenus satellite %s", satelliteID)

	if c.silenusClient == nil {
		return nil, fmt.Errorf("silenus client not configured")
	}
	if satelliteID == "" {
		return nil, fmt.Errorf("satellite ID required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pos, err := c.silenusClient.GetSatellitePosition(ctx, satelliteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get satellite position: %w", err)
	}

	terrain, err := c.silenusClient.RequestTerrainMap(ctx, GeoCoord{
		Latitude:  pos.Latitude,
		Longitude: pos.Longitude,
		Altitude:  0,
	}, 25.0)
	if err != nil {
		return nil, fmt.Errorf("failed to request terrain map: %w", err)
	}

	return terrain.Elevation, nil
}

// IntegrateWithGiru gets real-time threat intelligence.
func (c *SystemCoordinator) getActiveThreatLocations(ctx context.Context) ([]guidance.ThreatLocation, error) {
	if c.giruClient == nil {
		return nil, nil
	}

	zones, err := c.giruClient.GetThreatZones(ctx)
	if err != nil {
		return nil, err
	}

	threats := make([]guidance.ThreatLocation, 0, len(zones))
	for _, zone := range zones {
		if !zone.Active {
			continue
		}
		threats = append(threats, guidance.ThreatLocation{
			Position: guidance.Vector3D{
				X: zone.Center.Longitude * 111000,
				Y: zone.Center.Latitude * 111000,
				Z: zone.Center.Altitude,
			},
			ThreatType:   zone.ThreatType,
			EffectRadius: zone.RadiusKm * 1000,
			Confidence:   zone.ThreatLevel,
			LastUpdated:  time.Now().UTC(),
		})
	}

	return threats, nil
}

// getCurrentPosition queries Nysus for payload location.
func (c *SystemCoordinator) getCurrentPosition(ctx context.Context, payloadID string) (guidance.Vector3D, error) {
	if c.satnetClient == nil {
		return guidance.Vector3D{}, fmt.Errorf("satnet client not configured")
	}

	telemetry, err := c.satnetClient.GetTelemetry(ctx, payloadID)
	if err != nil {
		return guidance.Vector3D{}, err
	}

	return guidance.Vector3D{
		X: telemetry.Position.X,
		Y: telemetry.Position.Y,
		Z: telemetry.Position.Z,
	}, nil
}

// transmitTrajectory sends trajectory to payload via Sat_Net DTN.
func (c *SystemCoordinator) transmitTrajectory(traj *guidance.Trajectory, payloadID string) error {
	// Serialize trajectory (simplified)
	trajData := fmt.Sprintf("TRAJ:%s:%d_waypoints", traj.ID, len(traj.Waypoints))

	// Create DTN bundle
	b, err := bundle.NewPriorityBundle(
		"dtn://asgard/pricilla",
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

	log.Printf("Pricilla: Trajectory transmitted to %s", payloadID)
	return nil
}

// monitorMission tracks mission progress.
func (c *SystemCoordinator) monitorMission(ctx context.Context, mission *GuidedMission) {
	log.Printf("Pricilla: Monitoring mission %s", mission.ID)

	for {
		select {
		case state := <-mission.TelemetryStream:
			// Update trajectory based on current state
			if err := c.UpdateMission(mission.ID, state); err != nil {
				log.Printf("Pricilla: Mission update failed: %v", err)
			}

			// Check if target reached
			targetPos := mission.CurrentTraj.Waypoints[len(mission.CurrentTraj.Waypoints)-1].Position
			distance := guidance.CalculateDistance(state.Position, targetPos)

			if distance < 10.0 { // Within 10 meters
				mission.TargetReached = true
				log.Printf("Pricilla: Mission %s completed - target reached", mission.ID)

				// Publish mission complete event
				c.publishMissionComplete(mission)
				return
			}

		case <-ctx.Done():
			log.Printf("Pricilla: Mission %s monitoring stopped", mission.ID)
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
		Source:    "pricilla",
		Timestamp: time.Now().UTC(),
		Priority:  2,
		Payload: events.MissionEvent{
			MissionID: mission.ID,
			Status:    "completed",
		},
	}

	if err := c.eventBus.Publish(event); err != nil {
		log.Printf("Pricilla: Failed to publish mission completion: %v", err)
	}
}
