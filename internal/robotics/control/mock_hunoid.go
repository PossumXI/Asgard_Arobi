package control

import (
	"context"
	"math"
	"sync"
	"time"
)

// MockHunoid simulates a humanoid robot
type MockHunoid struct {
	mu            sync.RWMutex
	id            string
	currentPose   Pose
	targetPose    Pose
	isMoving      bool
	joints        map[string]*Joint
	batteryPercent float64
}

func NewMockHunoid(id string) *MockHunoid {
	return &MockHunoid{
		id: id,
		currentPose: Pose{
			Position:    Vector3{X: 0, Y: 0, Z: 0},
			Orientation: Quaternion{W: 1, X: 0, Y: 0, Z: 0},
			Timestamp:   time.Now(),
		},
		joints: map[string]*Joint{
			"head_pan":       {ID: "head_pan", Position: 0},
			"head_tilt":      {ID: "head_tilt", Position: 0},
			"left_shoulder":  {ID: "left_shoulder", Position: 0},
			"left_elbow":     {ID: "left_elbow", Position: 0},
			"left_wrist":     {ID: "left_wrist", Position: 0},
			"right_shoulder": {ID: "right_shoulder", Position: 0},
			"right_elbow":    {ID: "right_elbow", Position: 0},
			"right_wrist":    {ID: "right_wrist", Position: 0},
			"left_hip":       {ID: "left_hip", Position: 0},
			"left_knee":      {ID: "left_knee", Position: 0},
			"left_ankle":     {ID: "left_ankle", Position: 0},
			"right_hip":      {ID: "right_hip", Position: 0},
			"right_knee":     {ID: "right_knee", Position: 0},
			"right_ankle":    {ID: "right_ankle", Position: 0},
		},
		batteryPercent: 100.0,
	}
}

func (h *MockHunoid) Initialize(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Simulate initialization delay
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (h *MockHunoid) GetCurrentPose() (Pose, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.currentPose, nil
}

func (h *MockHunoid) MoveTo(ctx context.Context, target Pose) error {
	h.mu.Lock()
	h.targetPose = target
	h.isMoving = true
	h.mu.Unlock()

	// Simulate movement in background
	go h.simulateMovement(ctx)

	return nil
}

func (h *MockHunoid) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.isMoving = false
	return nil
}

func (h *MockHunoid) GetJointStates() ([]Joint, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	joints := make([]Joint, 0, len(h.joints))
	for _, joint := range h.joints {
		joints = append(joints, *joint)
	}

	return joints, nil
}

func (h *MockHunoid) SetJointPositions(positions map[string]float64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	for jointID, position := range positions {
		if joint, exists := h.joints[jointID]; exists {
			joint.Position = position
			joint.Timestamp = time.Now()
		}
	}

	return nil
}

func (h *MockHunoid) IsMoving() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.isMoving
}

func (h *MockHunoid) simulateMovement(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.mu.Lock()

			if !h.isMoving {
				h.mu.Unlock()
				return
			}

			// Calculate distance to target
			dx := h.targetPose.Position.X - h.currentPose.Position.X
			dy := h.targetPose.Position.Y - h.currentPose.Position.Y
			dz := h.targetPose.Position.Z - h.currentPose.Position.Z
			distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

			if distance < 0.01 {
				// Reached target
				h.currentPose = h.targetPose
				h.isMoving = false
				h.mu.Unlock()
				return
			}

			// Move towards target (0.1 m/s)
			step := 0.01
			h.currentPose.Position.X += (dx / distance) * step
			h.currentPose.Position.Y += (dy / distance) * step
			h.currentPose.Position.Z += (dz / distance) * step
			h.currentPose.Timestamp = time.Now()

			// Drain battery slightly
			h.batteryPercent -= 0.01

			h.mu.Unlock()

		case <-ctx.Done():
			return
		}
	}
}

func (h *MockHunoid) GetBatteryPercent() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.batteryPercent
}
