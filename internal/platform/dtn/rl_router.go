package dtn

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/asgard/pandora/pkg/bundle"
)

type RLRoutingModel struct {
	Version             int                  `json:"version"`
	TrainedAt           string               `json:"trained_at"`
	FeatureOrder        []string             `json:"feature_order"`
	PriorityWeights     map[string][]float64 `json:"priority_weights"`
	MinEnergyByPriority map[string]float64   `json:"min_energy_by_priority"`
	Notes               string               `json:"notes"`
}

func LoadRLRoutingModel(path string) (*RLRoutingModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rl model: %w", err)
	}

	var model RLRoutingModel
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("parse rl model: %w", err)
	}

	if len(model.FeatureOrder) == 0 {
		return nil, fmt.Errorf("rl model missing feature order")
	}
	if len(model.PriorityWeights) == 0 {
		return nil, fmt.Errorf("rl model missing weights")
	}

	for priority, weights := range model.PriorityWeights {
		if len(weights) != len(model.FeatureOrder) {
			return nil, fmt.Errorf("weight length mismatch for priority %s", priority)
		}
	}

	return &model, nil
}

type RLRoutingAgent struct {
	model       *RLRoutingModel
	nodeEID     string
	energyMu    sync.RWMutex
	energyLevel map[string]float64
}

func NewRLRoutingAgent(nodeEID, modelPath string) (*RLRoutingAgent, error) {
	model, err := LoadRLRoutingModel(modelPath)
	if err != nil {
		return nil, err
	}

	return &RLRoutingAgent{
		model:       model,
		nodeEID:     nodeEID,
		energyLevel: make(map[string]float64),
	}, nil
}

func (r *RLRoutingAgent) UpdateEnergy(nodeID string, batteryPercent float64) {
	r.energyMu.Lock()
	defer r.energyMu.Unlock()
	r.energyLevel[nodeID] = batteryPercent
}

func (r *RLRoutingAgent) SelectNextHop(ctx context.Context, b *bundle.Bundle, neighbors map[string]*Neighbor) (string, error) {
	priorityKey := fmt.Sprintf("%d", b.Priority)
	weights, ok := r.model.PriorityWeights[priorityKey]
	if !ok {
		return "", fmt.Errorf("missing weights for priority %d", b.Priority)
	}

	minEnergy := r.model.MinEnergyByPriority[priorityKey]

	type scored struct {
		id      string
		score   float64
		latency time.Duration
	}

	candidates := make([]scored, 0, len(neighbors))
	for id, neighbor := range neighbors {
		if !neighbor.IsActive {
			continue
		}

		energyScore := r.energyScore(id)
		if energyScore < minEnergy {
			continue
		}

		features := r.buildFeatures(neighbor, b, energyScore)
		score := dot(weights, features)
		candidates = append(candidates, scored{
			id:      id,
			score:   score,
			latency: neighbor.Latency,
		})
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no rl route to destination: %s", b.DestinationEID)
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].latency < candidates[j].latency
		}
		return candidates[i].score > candidates[j].score
	})

	return candidates[0].id, nil
}

func (r *RLRoutingAgent) UpdateContactGraph(nodeID string, neighbor *Neighbor) {
	// No-op for RL router, but kept for Router interface compatibility.
}

func (r *RLRoutingAgent) buildFeatures(neighbor *Neighbor, b *bundle.Bundle, energyScore float64) []float64 {
	latencyScore := 1.0 - minFloat(float64(neighbor.Latency.Milliseconds())/10000.0, 1.0)
	bandwidthScore := minFloat(float64(neighbor.Bandwidth)/1000000.0, 1.0)
	contactActive := 0.0
	now := time.Now().UTC()
	if neighbor.IsActive && (neighbor.ContactEnd.IsZero() || neighbor.ContactEnd.After(now)) {
		contactActive = 1.0
	}

	pathMatch := 0.0
	if isOnPath(neighbor.EID, b.DestinationEID) {
		pathMatch = 1.0
	}

	featureMap := map[string]float64{
		"link_quality":   neighbor.LinkQuality,
		"latency_score":  latencyScore,
		"bandwidth":      bandwidthScore,
		"contact_active": contactActive,
		"path_match":     pathMatch,
		"energy_score":   energyScore,
	}

	features := make([]float64, 0, len(r.model.FeatureOrder))
	for _, key := range r.model.FeatureOrder {
		features = append(features, featureMap[key])
	}

	return features
}

func (r *RLRoutingAgent) energyScore(nodeID string) float64 {
	r.energyMu.RLock()
	defer r.energyMu.RUnlock()
	if value, ok := r.energyLevel[nodeID]; ok {
		return clamp(value/100.0, 0.0, 1.0)
	}
	return 1.0
}

func dot(weights, features []float64) float64 {
	sum := 0.0
	for i, w := range weights {
		sum += w * features[i]
	}
	return sum
}

func isOnPath(nodeEID, destEID string) bool {
	if nodeEID == destEID {
		return true
	}

	nodeParts := strings.Split(nodeEID, "/")
	destParts := strings.Split(destEID, "/")
	if len(nodeParts) >= 3 && len(destParts) >= 3 {
		return nodeParts[2] == destParts[2]
	}
	return false
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
