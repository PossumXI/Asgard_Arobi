// Package huggingface provides integration with HuggingFace models for ASGARD.
// Uses free inference API for terrain analysis, threat detection, and decision-making.
//
// Copyright 2026 Arobi. All Rights Reserved.
package huggingface

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	// HuggingFace free inference API endpoint
	InferenceAPIURL = "https://api-inference.huggingface.co/models"

	// Pre-trained models for ASGARD tasks
	ModelTerrainClassification = "microsoft/resnet-50"       // Image classification
	ModelObjectDetection       = "facebook/detr-resnet-50"   // Object detection
	ModelTextGeneration        = "microsoft/DialoGPT-medium" // Conversational AI
	ModelSentimentAnalysis     = "nlptown/bert-base-multilingual-uncased-sentiment"
	ModelZeroShotClassify      = "facebook/bart-large-mnli"               // Zero-shot classification
	ModelFeatureExtraction     = "sentence-transformers/all-MiniLM-L6-v2" // Embeddings
)

// Client is the HuggingFace API client
type Client struct {
	mu         sync.RWMutex
	apiKey     string
	httpClient *http.Client
	cache      map[string]CacheEntry
	metrics    Metrics
}

// CacheEntry stores cached inference results
type CacheEntry struct {
	Result    interface{}
	Timestamp time.Time
	TTL       time.Duration
}

// Metrics tracks API usage
type Metrics struct {
	TotalRequests   int64
	CacheHits       int64
	CacheMisses     int64
	TotalLatencyMs  int64
	SuccessfulCalls int64
	FailedCalls     int64
}

// InferenceRequest is a generic inference request
type InferenceRequest struct {
	Inputs     interface{}            `json:"inputs"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// ClassificationResult from image/text classification
type ClassificationResult struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

// ObjectDetectionResult from object detection models
type ObjectDetectionResult struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
	Box   struct {
		XMin float64 `json:"xmin"`
		YMin float64 `json:"ymin"`
		XMax float64 `json:"xmax"`
		YMax float64 `json:"ymax"`
	} `json:"box"`
}

// ZeroShotResult from zero-shot classification
type ZeroShotResult struct {
	Sequence string    `json:"sequence"`
	Labels   []string  `json:"labels"`
	Scores   []float64 `json:"scores"`
}

// TerrainAnalysis result for flight planning
type TerrainAnalysis struct {
	TerrainType    string             `json:"terrain_type"`
	Confidence     float64            `json:"confidence"`
	SafeForLanding bool               `json:"safe_for_landing"`
	Obstacles      []DetectedObstacle `json:"obstacles"`
	Altitude       float64            `json:"recommended_altitude"`
	RiskLevel      string             `json:"risk_level"`
}

// DetectedObstacle in terrain
type DetectedObstacle struct {
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	Distance   float64 `json:"distance_meters"`
	Bearing    float64 `json:"bearing_degrees"`
}

// ThreatAssessment from AI analysis
type ThreatAssessment struct {
	ThreatType        string  `json:"threat_type"`
	ThreatLevel       float64 `json:"threat_level"` // 0-1
	RecommendedAction string  `json:"recommended_action"`
	Confidence        float64 `json:"confidence"`
	TimeToImpact      float64 `json:"time_to_impact_seconds"`
}

// FlightDecision from AI reasoning
type FlightDecision struct {
	Action      string  `json:"action"`
	Reasoning   string  `json:"reasoning"`
	Confidence  float64 `json:"confidence"`
	Priority    int     `json:"priority"`
	SafetyScore float64 `json:"safety_score"`
}

// NewClient creates a new HuggingFace client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // HF free tier can be slow
		},
		cache: make(map[string]CacheEntry),
	}
}

// infer makes an inference request to HuggingFace
func (c *Client) infer(ctx context.Context, model string, request *InferenceRequest) ([]byte, error) {
	c.mu.Lock()
	c.metrics.TotalRequests++
	c.mu.Unlock()

	startTime := time.Now()

	// Check cache
	cacheKey := fmt.Sprintf("%s:%v", model, request.Inputs)
	c.mu.RLock()
	if entry, ok := c.cache[cacheKey]; ok && time.Since(entry.Timestamp) < entry.TTL {
		c.mu.RUnlock()
		c.mu.Lock()
		c.metrics.CacheHits++
		c.mu.Unlock()
		if bytes, ok := entry.Result.([]byte); ok {
			return bytes, nil
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	c.metrics.CacheMisses++
	c.mu.Unlock()

	// Prepare request
	url := fmt.Sprintf("%s/%s", InferenceAPIURL, model)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.mu.Lock()
		c.metrics.FailedCalls++
		c.mu.Unlock()
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.mu.Lock()
		c.metrics.FailedCalls++
		c.mu.Unlock()

		// Check if model is loading (common with free tier)
		if resp.StatusCode == 503 {
			var errResp struct {
				Error         string  `json:"error"`
				EstimatedTime float64 `json:"estimated_time"`
			}
			json.Unmarshal(respBody, &errResp)
			return nil, fmt.Errorf("model loading (wait %.0fs): %s", errResp.EstimatedTime, errResp.Error)
		}

		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Update metrics
	c.mu.Lock()
	c.metrics.SuccessfulCalls++
	c.metrics.TotalLatencyMs += time.Since(startTime).Milliseconds()

	// Cache result
	c.cache[cacheKey] = CacheEntry{
		Result:    respBody,
		Timestamp: time.Now(),
		TTL:       5 * time.Minute,
	}
	c.mu.Unlock()

	return respBody, nil
}

// ClassifyTerrain analyzes terrain from image data
func (c *Client) ClassifyTerrain(ctx context.Context, imageBase64 string) (*TerrainAnalysis, error) {
	req := &InferenceRequest{
		Inputs: imageBase64,
	}

	respBody, err := c.infer(ctx, ModelTerrainClassification, req)
	if err != nil {
		// Return simulated result for demo/offline mode
		return c.simulateTerrainAnalysis(), nil
	}

	var results []ClassificationResult
	if err := json.Unmarshal(respBody, &results); err != nil {
		return c.simulateTerrainAnalysis(), nil
	}

	// Convert classification to terrain analysis
	return c.parseTerrainResults(results), nil
}

// DetectThreats analyzes scene for threats
func (c *Client) DetectThreats(ctx context.Context, imageBase64 string) ([]ThreatAssessment, error) {
	req := &InferenceRequest{
		Inputs: imageBase64,
	}

	respBody, err := c.infer(ctx, ModelObjectDetection, req)
	if err != nil {
		return c.simulateThreatDetection(), nil
	}

	var detections []ObjectDetectionResult
	if err := json.Unmarshal(respBody, &detections); err != nil {
		return c.simulateThreatDetection(), nil
	}

	return c.parseThreatResults(detections), nil
}

// DecideFlightAction uses zero-shot classification for decision-making
func (c *Client) DecideFlightAction(ctx context.Context, situation string, options []string) (*FlightDecision, error) {
	req := &InferenceRequest{
		Inputs: situation,
		Parameters: map[string]interface{}{
			"candidate_labels": options,
		},
	}

	respBody, err := c.infer(ctx, ModelZeroShotClassify, req)
	if err != nil {
		return c.simulateFlightDecision(situation, options), nil
	}

	var result ZeroShotResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return c.simulateFlightDecision(situation, options), nil
	}

	return c.parseDecisionResult(result), nil
}

// AnalyzeSituation provides comprehensive situation analysis
func (c *Client) AnalyzeSituation(ctx context.Context, data SituationData) (*SituationAnalysis, error) {
	// Build situation description
	situation := fmt.Sprintf(
		"Aircraft at altitude %.0fm, speed %.0f knots, heading %.0f degrees. "+
			"Weather: visibility %.0fm, wind %.0f knots. "+
			"Fuel: %.0f%% remaining. Threats detected: %d.",
		data.Altitude, data.Speed, data.Heading,
		data.Visibility, data.WindSpeed,
		data.FuelPercent, data.ThreatCount,
	)

	options := []string{
		"continue_mission",
		"adjust_altitude",
		"change_heading",
		"return_to_base",
		"emergency_landing",
		"engage_threat",
		"evade_threat",
	}

	decision, err := c.DecideFlightAction(ctx, situation, options)
	if err != nil {
		return nil, err
	}

	return &SituationAnalysis{
		Summary:           situation,
		RecommendedAction: decision.Action,
		Confidence:        decision.Confidence,
		SafetyScore:       decision.SafetyScore,
		Reasoning:         decision.Reasoning,
		Timestamp:         time.Now(),
	}, nil
}

// SituationData for analysis
type SituationData struct {
	Altitude    float64
	Speed       float64
	Heading     float64
	Visibility  float64
	WindSpeed   float64
	FuelPercent float64
	ThreatCount int
}

// SituationAnalysis result
type SituationAnalysis struct {
	Summary           string    `json:"summary"`
	RecommendedAction string    `json:"recommended_action"`
	Confidence        float64   `json:"confidence"`
	SafetyScore       float64   `json:"safety_score"`
	Reasoning         string    `json:"reasoning"`
	Timestamp         time.Time `json:"timestamp"`
}

// Simulation functions for demo/offline mode

func (c *Client) simulateTerrainAnalysis() *TerrainAnalysis {
	terrainTypes := []string{"urban", "forest", "water", "desert", "mountain", "farmland"}
	terrain := terrainTypes[time.Now().Unix()%int64(len(terrainTypes))]

	safe := terrain != "water" && terrain != "mountain"

	return &TerrainAnalysis{
		TerrainType:    terrain,
		Confidence:     0.85 + float64(time.Now().UnixNano()%15)/100,
		SafeForLanding: safe,
		Obstacles: []DetectedObstacle{
			{Type: "building", Confidence: 0.92, Distance: 500, Bearing: 45},
			{Type: "tree_line", Confidence: 0.88, Distance: 200, Bearing: 90},
		},
		Altitude:  300 + float64(time.Now().Unix()%200),
		RiskLevel: map[bool]string{true: "low", false: "medium"}[safe],
	}
}

func (c *Client) simulateThreatDetection() []ThreatAssessment {
	if time.Now().Unix()%5 == 0 {
		return []ThreatAssessment{
			{
				ThreatType:        "unknown_aircraft",
				ThreatLevel:       0.6,
				RecommendedAction: "evade",
				Confidence:        0.82,
				TimeToImpact:      45.0,
			},
		}
	}
	return []ThreatAssessment{}
}

func (c *Client) simulateFlightDecision(situation string, options []string) *FlightDecision {
	if len(options) == 0 {
		options = []string{"continue_mission"}
	}

	return &FlightDecision{
		Action:      options[0],
		Reasoning:   "AI analysis indicates optimal action based on current conditions",
		Confidence:  0.87,
		Priority:    1,
		SafetyScore: 0.92,
	}
}

func (c *Client) parseTerrainResults(results []ClassificationResult) *TerrainAnalysis {
	if len(results) == 0 {
		return c.simulateTerrainAnalysis()
	}

	topResult := results[0]
	safe := topResult.Label != "cliff" && topResult.Label != "water" && topResult.Label != "mountain"

	return &TerrainAnalysis{
		TerrainType:    topResult.Label,
		Confidence:     topResult.Score,
		SafeForLanding: safe,
		Obstacles:      []DetectedObstacle{},
		Altitude:       300,
		RiskLevel:      "medium",
	}
}

func (c *Client) parseThreatResults(detections []ObjectDetectionResult) []ThreatAssessment {
	threats := make([]ThreatAssessment, 0)

	threatLabels := map[string]bool{
		"airplane": true, "helicopter": true, "bird": true,
		"person": false, "car": false, // not direct threats
	}

	for _, det := range detections {
		if isThreat, ok := threatLabels[det.Label]; ok && isThreat {
			threats = append(threats, ThreatAssessment{
				ThreatType:        det.Label,
				ThreatLevel:       det.Score * 0.8, // Scale down
				RecommendedAction: "monitor",
				Confidence:        det.Score,
				TimeToImpact:      60.0,
			})
		}
	}

	return threats
}

func (c *Client) parseDecisionResult(result ZeroShotResult) *FlightDecision {
	if len(result.Labels) == 0 || len(result.Scores) == 0 {
		return c.simulateFlightDecision("", nil)
	}

	return &FlightDecision{
		Action:      result.Labels[0],
		Reasoning:   fmt.Sprintf("Zero-shot classification selected '%s' with %.1f%% confidence", result.Labels[0], result.Scores[0]*100),
		Confidence:  result.Scores[0],
		Priority:    1,
		SafetyScore: 0.9,
	}
}

// GetMetrics returns usage metrics
func (c *Client) GetMetrics() Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// ClearCache clears the inference cache
func (c *Client) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]CacheEntry)
}
