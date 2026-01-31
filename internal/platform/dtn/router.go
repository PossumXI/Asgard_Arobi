package dtn

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/asgard/pandora/pkg/bundle"
)

// ContactGraphRouter implements a Contact Graph Routing algorithm for DTN.
// This is suitable for scenarios with predictable contact schedules (e.g., satellite orbits).
type ContactGraphRouter struct {
	mu           sync.RWMutex
	contactGraph map[string]map[string]*ContactWindow
	nodeEID      string
}

// ContactWindow represents a scheduled communication opportunity.
type ContactWindow struct {
	FromNode    string
	ToNode      string
	StartTime   time.Time
	EndTime     time.Time
	Bandwidth   int64 // bytes/second
	Latency     time.Duration
	Reliability float64 // 0.0 to 1.0
}

// NewContactGraphRouter creates a new CGR router.
func NewContactGraphRouter(nodeEID string) *ContactGraphRouter {
	return &ContactGraphRouter{
		contactGraph: make(map[string]map[string]*ContactWindow),
		nodeEID:      nodeEID,
	}
}

// SelectNextHop implements the Router interface using Contact Graph Routing.
func (r *ContactGraphRouter) SelectNextHop(ctx context.Context, b *bundle.Bundle, neighbors map[string]*Neighbor) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find destination in graph
	destEID := b.DestinationEID

	// If direct neighbor has matching EID, use it
	for id, neighbor := range neighbors {
		if neighbor.IsActive && neighbor.EID == destEID {
			return id, nil
		}
	}

	// Build list of candidate next hops
	type candidate struct {
		neighborID string
		score      float64
	}
	var candidates []candidate

	for id, neighbor := range neighbors {
		if !neighbor.IsActive {
			continue
		}

		// Calculate routing score based on:
		// 1. Link quality
		// 2. Estimated path to destination
		// 3. Bundle priority vs available bandwidth
		score := r.calculateRouteScore(neighbor, destEID, b.Priority)
		if score > 0 {
			candidates = append(candidates, candidate{id, score})
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no route to destination: %s", destEID)
	}

	// Sort by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	return candidates[0].neighborID, nil
}

// UpdateContactGraph adds or updates contact information.
func (r *ContactGraphRouter) UpdateContactGraph(nodeID string, neighbor *Neighbor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.contactGraph[nodeID] == nil {
		r.contactGraph[nodeID] = make(map[string]*ContactWindow)
	}

	r.contactGraph[nodeID][neighbor.ID] = &ContactWindow{
		FromNode:    nodeID,
		ToNode:      neighbor.ID,
		StartTime:   neighbor.ContactStart,
		EndTime:     neighbor.ContactEnd,
		Bandwidth:   neighbor.Bandwidth,
		Latency:     neighbor.Latency,
		Reliability: neighbor.LinkQuality,
	}
}

// calculateRouteScore computes a routing score for a neighbor.
func (r *ContactGraphRouter) calculateRouteScore(neighbor *Neighbor, destEID string, priority uint8) float64 {
	score := 0.0

	// Link quality factor (0-1)
	score += neighbor.LinkQuality * 0.4

	// Latency factor (lower is better)
	latencyScore := 1.0 - math.Min(float64(neighbor.Latency.Milliseconds())/10000, 1.0)
	score += latencyScore * 0.3

	// Bandwidth factor
	bandwidthScore := math.Min(float64(neighbor.Bandwidth)/1000000, 1.0) // Normalize to 1MB/s
	score += bandwidthScore * 0.2

	// Priority boost
	priorityBoost := float64(priority) / 2.0 * 0.1
	score += priorityBoost

	// Check if neighbor is on path to destination
	if r.isOnPathToDestination(neighbor.EID, destEID) {
		score += 0.5
	}

	return score
}

// isOnPathToDestination checks if a node EID is likely on the path.
func (r *ContactGraphRouter) isOnPathToDestination(nodeEID, destEID string) bool {
	// Simple heuristic: check EID prefix matching
	// e.g., dtn://mars/sat001 is on path to dtn://mars/base
	if nodeEID == destEID {
		return true
	}

	nodeParts := strings.Split(nodeEID, "/")
	destParts := strings.Split(destEID, "/")

	if len(nodeParts) >= 3 && len(destParts) >= 3 {
		return nodeParts[2] == destParts[2] // Same domain (e.g., "mars")
	}

	return false
}

// EnergyAwareRouter extends CGR with energy awareness for satellite nodes.
// This is critical for ASGARD satellites operating on limited solar power.
type EnergyAwareRouter struct {
	*ContactGraphRouter
	energyLevels map[string]float64 // Node ID -> battery percentage (0-100)
	energyMu     sync.RWMutex
}

// NewEnergyAwareRouter creates an energy-aware DTN router.
func NewEnergyAwareRouter(nodeEID string) *EnergyAwareRouter {
	return &EnergyAwareRouter{
		ContactGraphRouter: NewContactGraphRouter(nodeEID),
		energyLevels:       make(map[string]float64),
	}
}

// SelectNextHop implements energy-aware routing.
func (r *EnergyAwareRouter) SelectNextHop(ctx context.Context, b *bundle.Bundle, neighbors map[string]*Neighbor) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter neighbors by energy threshold
	activeNeighbors := make(map[string]*Neighbor)
	for id, neighbor := range neighbors {
		if neighbor.IsActive && r.hasEnoughEnergy(id, b.Priority) {
			activeNeighbors[id] = neighbor
		}
	}

	if len(activeNeighbors) == 0 {
		return "", fmt.Errorf("no energy-available route to: %s", b.DestinationEID)
	}

	return r.ContactGraphRouter.SelectNextHop(ctx, b, activeNeighbors)
}

// UpdateEnergy updates the energy level for a node.
func (r *EnergyAwareRouter) UpdateEnergy(nodeID string, batteryPercent float64) {
	r.energyMu.Lock()
	defer r.energyMu.Unlock()
	r.energyLevels[nodeID] = batteryPercent
}

// hasEnoughEnergy checks if a node has sufficient energy for transmission.
func (r *EnergyAwareRouter) hasEnoughEnergy(nodeID string, priority uint8) bool {
	r.energyMu.RLock()
	defer r.energyMu.RUnlock()

	level, exists := r.energyLevels[nodeID]
	if !exists {
		// Unknown energy, assume OK
		return true
	}

	// Minimum thresholds based on priority
	// Higher priority bundles can use nodes with lower energy
	thresholds := map[uint8]float64{
		bundle.PriorityBulk:      30.0, // Need at least 30% for bulk
		bundle.PriorityNormal:    20.0, // 20% for normal
		bundle.PriorityExpedited: 10.0, // 10% for expedited (critical)
	}

	threshold, ok := thresholds[priority]
	if !ok {
		threshold = 20.0
	}

	return level >= threshold
}

// StaticRouter is a simple router for testing with predefined routes.
type StaticRouter struct {
	routes map[string]string // destination EID -> next hop node ID
	mu     sync.RWMutex
}

// NewStaticRouter creates a static router with predefined routes.
func NewStaticRouter() *StaticRouter {
	return &StaticRouter{
		routes: make(map[string]string),
	}
}

// AddRoute adds a static route.
func (r *StaticRouter) AddRoute(destEID, nextHopID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[destEID] = nextHopID
}

// SelectNextHop implements the Router interface.
func (r *StaticRouter) SelectNextHop(ctx context.Context, b *bundle.Bundle, neighbors map[string]*Neighbor) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check for direct route
	if nextHop, exists := r.routes[b.DestinationEID]; exists {
		if neighbor, ok := neighbors[nextHop]; ok && neighbor.IsActive {
			return nextHop, nil
		}
	}

	// Check for prefix match
	for dest, nextHop := range r.routes {
		if strings.HasPrefix(b.DestinationEID, dest) {
			if neighbor, ok := neighbors[nextHop]; ok && neighbor.IsActive {
				return nextHop, nil
			}
		}
	}

	// Fallback: try any active neighbor
	for id, neighbor := range neighbors {
		if neighbor.IsActive {
			return id, nil
		}
	}

	return "", fmt.Errorf("no route to destination: %s", b.DestinationEID)
}

// UpdateContactGraph implements Router interface (no-op for static router).
func (r *StaticRouter) UpdateContactGraph(nodeID string, neighbor *Neighbor) {
	// Static router doesn't use contact graph
}
