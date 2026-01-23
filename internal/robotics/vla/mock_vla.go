package vla

import (
	"context"
	"strings"
	"time"
)

// MockVLA simulates a vision-language-action model
type MockVLA struct {
	modelInfo ModelInfo
}

func NewMockVLA() *MockVLA {
	return &MockVLA{
		modelInfo: ModelInfo{
			Name:    "OpenVLA-Mock",
			Version: "1.0.0",
			SupportedActions: []ActionType{
				ActionNavigate,
				ActionPickUp,
				ActionPutDown,
				ActionOpen,
				ActionClose,
				ActionInspect,
				ActionWait,
			},
		},
	}
}

func (v *MockVLA) Initialize(ctx context.Context, modelPath string) error {
	// Simulate model loading
	time.Sleep(1 * time.Second)
	return nil
}

func (v *MockVLA) InferAction(ctx context.Context, visualObs []byte, textCommand string) (*Action, error) {
	// Simple keyword-based action inference (mock)
	command := strings.ToLower(textCommand)

	var action *Action

	switch {
	case strings.Contains(command, "pick up") || strings.Contains(command, "grab") || strings.Contains(command, "lift"):
		action = &Action{
			Type: ActionPickUp,
			Parameters: map[string]interface{}{
				"object": "detected_object",
				"force":  "gentle",
			},
			Confidence: 0.89,
		}

	case strings.Contains(command, "put down") || strings.Contains(command, "place") || strings.Contains(command, "drop"):
		action = &Action{
			Type: ActionPutDown,
			Parameters: map[string]interface{}{
				"location": "detected_surface",
			},
			Confidence: 0.87,
		}

	case strings.Contains(command, "open"):
		action = &Action{
			Type: ActionOpen,
			Parameters: map[string]interface{}{
				"target": "gripper",
			},
			Confidence: 0.95,
		}

	case strings.Contains(command, "close"):
		action = &Action{
			Type: ActionClose,
			Parameters: map[string]interface{}{
				"target": "gripper",
			},
			Confidence: 0.95,
		}

	case strings.Contains(command, "go to") || strings.Contains(command, "move to") || strings.Contains(command, "navigate"):
		action = &Action{
			Type: ActionNavigate,
			Parameters: map[string]interface{}{
				"x": 1.0,
				"y": 0.5,
				"z": 0.0,
			},
			Confidence: 0.82,
		}

	case strings.Contains(command, "inspect") || strings.Contains(command, "look at") || strings.Contains(command, "examine"):
		action = &Action{
			Type: ActionInspect,
			Parameters: map[string]interface{}{
				"duration_seconds": 5,
			},
			Confidence: 0.78,
		}

	default:
		action = &Action{
			Type: ActionWait,
			Parameters: map[string]interface{}{
				"reason": "unclear_command",
			},
			Confidence: 0.50,
		}
	}

	// Simulate inference time
	time.Sleep(100 * time.Millisecond)

	return action, nil
}

func (v *MockVLA) GetModelInfo() ModelInfo {
	return v.modelInfo
}

func (v *MockVLA) Shutdown() error {
	return nil
}
