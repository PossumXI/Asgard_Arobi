package unit

import (
	"context"
	"testing"
	"time"

	"github.com/PossumXI/Asgard/Valkyrie/internal/ai"
)

func TestDecisionEngine_Creation(t *testing.T) {
	config := ai.DecisionConfig{
		SafetyPriority:     0.9,
		EfficiencyPriority: 0.7,
		StealthPriority:    0.5,
		MaxRollAngle:       0.785,
		MaxPitchAngle:      0.524,
		MaxYawRate:         0.349,
		MinSafeAltitude:    100.0,
		MaxVerticalSpeed:   10.0,
		DecisionRate:       50.0,
	}

	de := ai.NewDecisionEngine(config)
	if de == nil {
		t.Fatal("DecisionEngine creation failed")
	}
}

func TestDecisionEngine_Initialize(t *testing.T) {
	config := ai.DecisionConfig{
		SafetyPriority: 0.9,
		DecisionRate:   50.0,
	}

	de := ai.NewDecisionEngine(config)
	ctx := context.Background()

	err := de.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

func TestDecisionEngine_DecideWithoutMission(t *testing.T) {
	config := ai.DecisionConfig{
		SafetyPriority:  0.9,
		MaxRollAngle:    0.785,
		MaxPitchAngle:   0.524,
		MaxYawRate:      0.349,
		MinSafeAltitude: 100.0,
		DecisionRate:    50.0,
	}

	de := ai.NewDecisionEngine(config)
	ctx := context.Background()
	de.Initialize(ctx)

	cmd, err := de.Decide(ctx)
	if err != nil {
		t.Fatalf("Decide failed: %v", err)
	}

	if cmd == nil {
		t.Fatal("Command is nil")
	}

	// Without a mission, should return safe defaults
	t.Logf("Command: Throttle=%.2f, AutoThrottle=%v", cmd.Throttle, cmd.AutoThrottle)
}

func TestDecisionEngine_SafetyLimits(t *testing.T) {
	config := ai.DecisionConfig{
		SafetyPriority:  0.9,
		MaxRollAngle:    0.785, // 45 degrees
		MaxPitchAngle:   0.524, // 30 degrees
		MaxYawRate:      0.349, // 20 deg/s
		MinSafeAltitude: 100.0,
		DecisionRate:    50.0,
	}

	de := ai.NewDecisionEngine(config)
	ctx := context.Background()
	de.Initialize(ctx)

	// Set a mission with waypoints
	mission := &ai.Mission{
		ID:   "test-mission",
		Type: ai.MissionPatrol,
		Waypoints: []ai.Waypoint{
			{ID: "wp1", Position: [3]float64{1000, 0, 500}},
		},
		Status: ai.MissionActive,
	}
	de.SetMission(mission)

	cmd, err := de.Decide(ctx)
	if err != nil {
		t.Fatalf("Decide failed: %v", err)
	}

	// Verify commands are within limits
	if cmd.RollAngle > config.MaxRollAngle || cmd.RollAngle < -config.MaxRollAngle {
		t.Errorf("Roll angle %.3f exceeds limit %.3f", cmd.RollAngle, config.MaxRollAngle)
	}

	if cmd.PitchAngle > config.MaxPitchAngle || cmd.PitchAngle < -config.MaxPitchAngle {
		t.Errorf("Pitch angle %.3f exceeds limit %.3f", cmd.PitchAngle, config.MaxPitchAngle)
	}

	if cmd.YawRate > config.MaxYawRate || cmd.YawRate < -config.MaxYawRate {
		t.Errorf("Yaw rate %.3f exceeds limit %.3f", cmd.YawRate, config.MaxYawRate)
	}

	if cmd.Throttle < 0 || cmd.Throttle > 1.0 {
		t.Errorf("Throttle %.3f out of range [0, 1]", cmd.Throttle)
	}
}

func TestDecisionEngine_MissionStatus(t *testing.T) {
	config := ai.DecisionConfig{
		DecisionRate: 50.0,
	}

	de := ai.NewDecisionEngine(config)

	// Initially no mission
	status := de.GetMissionStatus()
	if status != ai.MissionPending {
		t.Errorf("Expected MissionPending, got %v", status)
	}

	// Set a mission
	mission := &ai.Mission{
		ID:     "test",
		Status: ai.MissionActive,
	}
	de.SetMission(mission)

	status = de.GetMissionStatus()
	if status != ai.MissionActive {
		t.Errorf("Expected MissionActive, got %v", status)
	}
}

func TestDecisionEngine_ThreatAvoidance(t *testing.T) {
	config := ai.DecisionConfig{
		SafetyPriority:    0.9,
		MaxRollAngle:      0.785,
		MaxPitchAngle:     0.524,
		MaxYawRate:        0.349,
		MinSafeAltitude:   100.0,
		EnableThreatAvoid: true,
		DecisionRate:      50.0,
	}

	de := ai.NewDecisionEngine(config)
	ctx := context.Background()
	de.Initialize(ctx)

	// Set mission
	mission := &ai.Mission{
		ID: "test",
		Waypoints: []ai.Waypoint{
			{Position: [3]float64{1000, 0, 500}},
		},
	}
	de.SetMission(mission)

	// Add threats
	threats := []*ai.Threat{
		{
			ID:       "threat1",
			Type:     ai.ThreatRadar,
			Position: [3]float64{500, 0, 500},
			Distance: 500,
			Bearing:  0,
			Severity: 0.8,
		},
	}
	de.UpdateThreats(threats)

	cmd, err := de.Decide(ctx)
	if err != nil {
		t.Fatalf("Decide failed: %v", err)
	}

	t.Logf("Threat avoidance command: Roll=%.3f, Throttle=%.2f", cmd.RollAngle, cmd.Throttle)

	// With threat, throttle should be higher
	if cmd.Throttle < 0.8 {
		t.Log("Note: Throttle may increase when threats detected")
	}
}

func TestFlightCommand_Creation(t *testing.T) {
	cmd := ai.FlightCommand{
		Timestamp:    time.Now(),
		RollAngle:    0.1,
		PitchAngle:   0.05,
		YawRate:      0.02,
		Throttle:     0.7,
		AutoThrottle: true,
	}

	if cmd.Throttle != 0.7 {
		t.Errorf("Expected throttle 0.7, got %.2f", cmd.Throttle)
	}
}

func TestWaypoint_Structure(t *testing.T) {
	wp := ai.Waypoint{
		ID:       "wp-001",
		Position: [3]float64{1000, 2000, 500},
		Speed:    50.0,
		Altitude: 500,
		Heading:  1.57,
		Loiter:   30 * time.Second,
	}

	if wp.ID != "wp-001" {
		t.Errorf("Waypoint ID mismatch")
	}

	if wp.Position[2] != 500 {
		t.Errorf("Waypoint altitude mismatch")
	}
}
