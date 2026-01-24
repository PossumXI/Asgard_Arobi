package vla

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// OpenVLAModel implements VLAModel using OpenVLA or similar vision-language-action models.
// Supports local inference (ONNX, TensorRT) and remote inference servers.
type OpenVLAModel struct {
	mu          sync.RWMutex
	config      VLAConfig
	httpClient  *http.Client
	modelInfo   ModelInfo
	initialized bool
}

// VLAConfig holds VLA model configuration
type VLAConfig struct {
	// Inference backend: "onnx", "tensorrt", "http", "grpc", "transformers"
	Backend string `json:"backend"`

	// Model settings
	ModelPath    string `json:"modelPath"`    // Path to model weights
	ModelName    string `json:"modelName"`    // e.g., "openvla-7b", "rt-2-x"
	ModelVersion string `json:"modelVersion"`

	// Remote inference
	InferenceURL string `json:"inferenceUrl"` // API endpoint
	APIKey       string `json:"apiKey"`

	// Model parameters
	MaxTokens     int     `json:"maxTokens"`
	Temperature   float64 `json:"temperature"`
	TopP          float64 `json:"topP"`

	// Input settings
	ImageWidth    int `json:"imageWidth"`
	ImageHeight   int `json:"imageHeight"`
	MaxPromptLen  int `json:"maxPromptLen"`

	// Execution settings
	ActionSpace    []string `json:"actionSpace"`    // Available actions
	ConfidenceMin  float64  `json:"confidenceMin"`  // Minimum confidence threshold
	SafetyChecks   bool     `json:"safetyChecks"`   // Enable safety validation
}

// InferenceRequest represents a VLA inference request
type InferenceRequest struct {
	Image     string `json:"image"`      // Base64 encoded image
	Prompt    string `json:"prompt"`     // Natural language command
	History   []Turn `json:"history"`    // Conversation history
	MaxTokens int    `json:"max_tokens"`
}

// Turn represents a conversation turn
type Turn struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// InferenceResponse represents a VLA inference response
type InferenceResponse struct {
	Action     ActionOutput `json:"action"`
	Reasoning  string       `json:"reasoning"`
	Confidence float64      `json:"confidence"`
}

// ActionOutput represents the model's action output
type ActionOutput struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Trajectory []TrajectoryPoint      `json:"trajectory,omitempty"`
}

// TrajectoryPoint represents a point in a motion trajectory
type TrajectoryPoint struct {
	Position    [3]float64 `json:"position"`    // x, y, z
	Orientation [4]float64 `json:"orientation"` // quaternion w, x, y, z
	Gripper     float64    `json:"gripper"`     // 0.0 to 1.0
}

// NewOpenVLAModel creates a new VLA model
func NewOpenVLAModel(config VLAConfig) *OpenVLAModel {
	if config.MaxTokens == 0 {
		config.MaxTokens = 256
	}
	if config.Temperature == 0 {
		config.Temperature = 0.1
	}
	if config.TopP == 0 {
		config.TopP = 0.9
	}
	if config.ImageWidth == 0 {
		config.ImageWidth = 224
	}
	if config.ImageHeight == 0 {
		config.ImageHeight = 224
	}
	if config.ConfidenceMin == 0 {
		config.ConfidenceMin = 0.6
	}

	// Default action space
	if len(config.ActionSpace) == 0 {
		config.ActionSpace = []string{
			"navigate", "pick_up", "put_down", "open", "close",
			"push", "pull", "rotate", "inspect", "wait",
		}
	}

	return &OpenVLAModel{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // VLA inference can be slow
		},
		modelInfo: ModelInfo{
			Name:    config.ModelName,
			Version: config.ModelVersion,
			SupportedActions: []ActionType{
				ActionNavigate, ActionPickUp, ActionPutDown,
				ActionOpen, ActionClose, ActionInspect, ActionWait,
			},
		},
	}
}

// Initialize loads the model and prepares for inference
func (v *OpenVLAModel) Initialize(ctx context.Context, modelPath string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if modelPath != "" {
		v.config.ModelPath = modelPath
	}

	switch v.config.Backend {
	case "http":
		return v.initHTTPBackend(ctx)
	case "onnx":
		return v.initONNXBackend(ctx)
	case "transformers":
		return v.initTransformersBackend(ctx)
	default:
		// Default to HTTP if URL is provided
		if v.config.InferenceURL != "" {
			v.config.Backend = "http"
			return v.initHTTPBackend(ctx)
		}
		return fmt.Errorf("unsupported backend: %s", v.config.Backend)
	}
}

func (v *OpenVLAModel) initHTTPBackend(ctx context.Context) error {
	if v.config.InferenceURL == "" {
		return fmt.Errorf("inference URL required for HTTP backend")
	}

	// Test connection
	req, err := http.NewRequestWithContext(ctx, "GET", v.config.InferenceURL+"/health", nil)
	if err != nil {
		return err
	}

	if v.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+v.config.APIKey)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to VLA server: %w", err)
	}
	resp.Body.Close()

	v.initialized = true
	v.modelInfo.Name = "OpenVLA-" + v.config.ModelVersion + "-Remote"
	return nil
}

func (v *OpenVLAModel) initONNXBackend(ctx context.Context) error {
	if v.config.InferenceURL != "" {
		if err := v.initHTTPBackend(ctx); err != nil {
			return err
		}
		v.modelInfo.Name = "OpenVLA-" + v.config.ModelVersion + "-ONNX-Remote"
		return nil
	}

	if v.config.ModelPath == "" {
		return fmt.Errorf("onnx backend requires modelPath or inferenceUrl")
	}

	return fmt.Errorf("onnx backend requires local runtime; set inferenceUrl or use http backend")
}

func (v *OpenVLAModel) initTransformersBackend(ctx context.Context) error {
	if v.config.InferenceURL != "" {
		if err := v.initHTTPBackend(ctx); err != nil {
			return err
		}
		v.modelInfo.Name = "OpenVLA-" + v.config.ModelVersion + "-Transformers-Remote"
		return nil
	}

	return fmt.Errorf("transformers backend requires inferenceUrl or http backend")
}

// InferAction performs VLA inference to determine the appropriate action
func (v *OpenVLAModel) InferAction(ctx context.Context, visualObs []byte, textCommand string) (*Action, error) {
	v.mu.RLock()
	backend := v.config.Backend
	initialized := v.initialized
	v.mu.RUnlock()

	if !initialized {
		return nil, fmt.Errorf("model not initialized")
	}

	switch backend {
	case "http":
		return v.inferHTTP(ctx, visualObs, textCommand)
	case "onnx":
		return v.inferONNX(ctx, visualObs, textCommand)
	case "transformers":
		return v.inferTransformers(ctx, visualObs, textCommand)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", backend)
	}
}

func (v *OpenVLAModel) inferHTTP(ctx context.Context, visualObs []byte, textCommand string) (*Action, error) {
	// Encode image to base64
	imageB64 := base64.StdEncoding.EncodeToString(visualObs)

	// Build request
	request := InferenceRequest{
		Image:     imageB64,
		Prompt:    v.buildPrompt(textCommand),
		MaxTokens: v.config.MaxTokens,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Send request
	url := v.config.InferenceURL + "/v1/infer"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if v.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+v.config.APIKey)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inference request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("inference failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var inferResp InferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&inferResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to Action
	return v.parseActionOutput(inferResp)
}

func (v *OpenVLAModel) inferONNX(ctx context.Context, visualObs []byte, textCommand string) (*Action, error) {
	if v.config.InferenceURL != "" {
		return v.inferHTTP(ctx, visualObs, textCommand)
	}

	return nil, fmt.Errorf("onnx backend not configured; set inferenceUrl or use http backend")
}

func (v *OpenVLAModel) inferTransformers(ctx context.Context, visualObs []byte, textCommand string) (*Action, error) {
	if v.config.InferenceURL != "" {
		return v.inferHTTP(ctx, visualObs, textCommand)
	}

	return nil, fmt.Errorf("transformers backend not configured; set inferenceUrl or use http backend")
}

func (v *OpenVLAModel) buildPrompt(command string) string {
	// Build a structured prompt for the VLA model
	return fmt.Sprintf(`You are a robot assistant that helps with physical tasks.
Given the current camera observation, determine the appropriate action to execute.

User command: %s

Available actions:
- navigate: Move to a location (parameters: x, y, z coordinates)
- pick_up: Grasp an object (parameters: object description)
- put_down: Release held object (parameters: target location)
- open: Open gripper or door (parameters: target)
- close: Close gripper or door (parameters: target)
- push: Push an object (parameters: direction, force)
- pull: Pull an object (parameters: direction, force)
- rotate: Rotate end-effector (parameters: angle)
- inspect: Look closely at something (parameters: target, duration)
- wait: Wait for condition (parameters: duration or condition)

Respond with a JSON object containing:
{
  "action": {
    "type": "<action_type>",
    "parameters": {<action_parameters>},
    "trajectory": [<optional trajectory points>]
  },
  "reasoning": "<brief explanation>",
  "confidence": <0.0-1.0>
}`, command)
}

func (v *OpenVLAModel) parseActionOutput(resp InferenceResponse) (*Action, error) {
	// Apply confidence threshold
	if resp.Confidence < v.config.ConfidenceMin {
		return &Action{
			Type: ActionWait,
			Parameters: map[string]interface{}{
				"reason":     "low_confidence",
				"confidence": resp.Confidence,
			},
			Confidence: resp.Confidence,
		}, nil
	}

	// Map action type
	actionType := v.mapActionType(resp.Action.Type)

	// Validate action if safety checks enabled
	if v.config.SafetyChecks {
		if err := v.validateAction(actionType, resp.Action.Parameters); err != nil {
			return &Action{
				Type: ActionWait,
				Parameters: map[string]interface{}{
					"reason": "safety_check_failed",
					"error":  err.Error(),
				},
				Confidence: 0.0,
			}, nil
		}
	}

	return &Action{
		Type:       actionType,
		Parameters: resp.Action.Parameters,
		Confidence: resp.Confidence,
	}, nil
}

func (v *OpenVLAModel) mapActionType(typeStr string) ActionType {
	switch typeStr {
	case "navigate", "move", "go_to":
		return ActionNavigate
	case "pick_up", "grasp", "grab", "lift":
		return ActionPickUp
	case "put_down", "place", "release", "drop":
		return ActionPutDown
	case "open":
		return ActionOpen
	case "close":
		return ActionClose
	case "inspect", "look", "examine":
		return ActionInspect
	default:
		return ActionWait
	}
}

func (v *OpenVLAModel) validateAction(actionType ActionType, params map[string]interface{}) error {
	// Safety validation rules
	switch actionType {
	case ActionNavigate:
		// Check if navigation coordinates are within safe bounds
		if x, ok := params["x"].(float64); ok {
			if x < -10 || x > 10 {
				return fmt.Errorf("x coordinate out of safe bounds: %f", x)
			}
		}
		if y, ok := params["y"].(float64); ok {
			if y < -10 || y > 10 {
				return fmt.Errorf("y coordinate out of safe bounds: %f", y)
			}
		}

	case ActionPickUp:
		// Validate gripper force
		if force, ok := params["force"].(float64); ok {
			if force > 100 {
				return fmt.Errorf("gripper force exceeds safe limit: %f N", force)
			}
		}
	}

	return nil
}

// GetModelInfo returns model metadata
func (v *OpenVLAModel) GetModelInfo() ModelInfo {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.modelInfo
}

// Shutdown releases resources
func (v *OpenVLAModel) Shutdown() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.initialized = false
	return nil
}

// ============================================================================
// Batch Inference Support
// ============================================================================

// BatchInferenceRequest represents multiple inference requests
type BatchInferenceRequest struct {
	Requests []InferenceRequest `json:"requests"`
}

// BatchInferActions performs batch inference for multiple observations
func (v *OpenVLAModel) BatchInferActions(ctx context.Context, observations []struct {
	Image   []byte
	Command string
}) ([]*Action, error) {
	v.mu.RLock()
	backend := v.config.Backend
	initialized := v.initialized
	v.mu.RUnlock()

	if !initialized {
		return nil, fmt.Errorf("model not initialized")
	}

	if backend != "http" {
		// Non-HTTP backends process sequentially
		actions := make([]*Action, len(observations))
		for i, obs := range observations {
			action, err := v.InferAction(ctx, obs.Image, obs.Command)
			if err != nil {
				return nil, err
			}
			actions[i] = action
		}
		return actions, nil
	}

	// Build batch request
	requests := make([]InferenceRequest, len(observations))
	for i, obs := range observations {
		requests[i] = InferenceRequest{
			Image:     base64.StdEncoding.EncodeToString(obs.Image),
			Prompt:    v.buildPrompt(obs.Command),
			MaxTokens: v.config.MaxTokens,
		}
	}

	batchReq := BatchInferenceRequest{Requests: requests}
	body, err := json.Marshal(batchReq)
	if err != nil {
		return nil, err
	}

	// Send batch request
	url := v.config.InferenceURL + "/v1/batch_infer"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if v.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+v.config.APIKey)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("batch inference failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("batch inference failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse batch response
	var responses []InferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
		return nil, err
	}

	// Convert to actions
	actions := make([]*Action, len(responses))
	for i, inferResp := range responses {
		action, err := v.parseActionOutput(inferResp)
		if err != nil {
			return nil, err
		}
		actions[i] = action
	}

	return actions, nil
}

// ============================================================================
// Streaming Inference Support
// ============================================================================

// StreamInferAction performs streaming inference with incremental results
func (v *OpenVLAModel) StreamInferAction(ctx context.Context, visualObs []byte, textCommand string, callback func(*Action) error) error {
	v.mu.RLock()
	backend := v.config.Backend
	initialized := v.initialized
	v.mu.RUnlock()

	if !initialized {
		return fmt.Errorf("model not initialized")
	}

	if backend != "http" {
		// Non-streaming fallback
		action, err := v.InferAction(ctx, visualObs, textCommand)
		if err != nil {
			return err
		}
		return callback(action)
	}

	// Build streaming request
	imageB64 := base64.StdEncoding.EncodeToString(visualObs)
	request := map[string]interface{}{
		"image":      imageB64,
		"prompt":     v.buildPrompt(textCommand),
		"max_tokens": v.config.MaxTokens,
		"stream":     true,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return err
	}

	url := v.config.InferenceURL + "/v1/infer/stream"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if v.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+v.config.APIKey)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("streaming inference failed: %w", err)
	}
	defer resp.Body.Close()

	// Process SSE stream
	reader := resp.Body
	buf := make([]byte, 4096)
	
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// Parse SSE event
		data := string(buf[:n])
		if len(data) > 6 && data[:6] == "data: " {
			var partialResp InferenceResponse
			if err := json.Unmarshal([]byte(data[6:]), &partialResp); err == nil {
				action, _ := v.parseActionOutput(partialResp)
				if err := callback(action); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
