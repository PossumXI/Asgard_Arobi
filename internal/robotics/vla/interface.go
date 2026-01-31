package vla

import (
	"context"
)

// Action represents a robot action command
type Action struct {
	Type       ActionType
	Parameters map[string]interface{}
	Confidence float64
}

// ActionType defines types of robot actions
type ActionType string

const (
	ActionNavigate ActionType = "navigate"
	ActionPickUp   ActionType = "pick_up"
	ActionPutDown  ActionType = "put_down"
	ActionOpen     ActionType = "open"
	ActionClose    ActionType = "close"
	ActionInspect  ActionType = "inspect"
	ActionWait     ActionType = "wait"
)

// VLAModel defines the vision-language-action interface
type VLAModel interface {
	Initialize(ctx context.Context, modelPath string) error
	InferAction(ctx context.Context, visualObs []byte, textCommand string) (*Action, error)
	GetModelInfo() ModelInfo
	Shutdown() error
}

// ModelInfo contains VLA model metadata
type ModelInfo struct {
	Name             string
	Version          string
	SupportedActions []ActionType
}
