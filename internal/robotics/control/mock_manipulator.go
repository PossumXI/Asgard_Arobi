package control

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// MockManipulator simulates robot arm/gripper
type MockManipulator struct {
	mu           sync.RWMutex
	gripperState float64 // 0.0 = closed, 1.0 = open
	armPosition  Vector3
}

func NewMockManipulator() *MockManipulator {
	return &MockManipulator{
		gripperState: 1.0, // Start open
		armPosition:  Vector3{X: 0.3, Y: 0, Z: 0.5}, // Default position
	}
}

func (m *MockManipulator) OpenGripper() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gripperState = 1.0
	time.Sleep(200 * time.Millisecond) // Simulate actuation time
	return nil
}

func (m *MockManipulator) CloseGripper() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gripperState = 0.0
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (m *MockManipulator) GetGripperState() (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.gripperState, nil
}

func (m *MockManipulator) ReachTo(ctx context.Context, position Vector3) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate reachability (simple sphere check)
	distance := math.Sqrt(position.X*position.X + position.Y*position.Y + position.Z*position.Z)
	if distance > 1.0 { // Max reach 1 meter
		return fmt.Errorf("position out of reach: %.2fm", distance)
	}

	// Simulate movement
	time.Sleep(500 * time.Millisecond)
	m.armPosition = position

	return nil
}
