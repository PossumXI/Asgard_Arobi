// Package decision provides benchmarks to validate sub-100ms latency requirement.
//
// Copyright 2026 Arobi. All Rights Reserved.
package decision

import (
	"testing"
	"time"

	"github.com/asgard/pandora/internal/robotics/perception"
)

// BenchmarkFullRescuePrioritization benchmarks the complete rescue prioritization pipeline
// Target: < 100ms for full 360-degree scan with 50 objects
func BenchmarkFullRescuePrioritization(b *testing.B) {
	prioritizer := NewRescuePrioritizer()

	// Create test scenario with 50 objects (realistic crowded scene)
	scan := createTestScan(50, 10) // 50 objects, 10 humans in danger

	hunoidState := HunoidState{
		Position:         perception.Vector3{X: 0, Y: 0, Z: 0},
		Velocity:         perception.Vector3{X: 0, Y: 0, Z: 0},
		BatteryLevel:     0.8,
		MaxSpeed:         5.0,
		CarryingCapacity: 2,
		CurrentLoad:      0,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = prioritizer.CalculatePriorities(hunoidState, scan)
	}
}

// BenchmarkMonteCarloSimulation benchmarks just the Monte Carlo component
func BenchmarkMonteCarloSimulation(b *testing.B) {
	prioritizer := NewRescuePrioritizer()

	target := perception.TrackedObject{
		ID:          "target-1",
		ClassType:   perception.ClassHuman,
		Position:    perception.Vector3{X: 20, Y: 10, Z: 0},
		Velocity:    perception.Vector3{X: 1, Y: 0, Z: 0},
		ThreatLevel: 0.7,
	}

	hunoidState := HunoidState{
		Position:     perception.Vector3{X: 0, Y: 0, Z: 0},
		BatteryLevel: 0.8,
		MaxSpeed:     5.0,
	}

	scan := createTestScan(20, 5)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = prioritizer.runMonteCarloSimulation(hunoidState, target, scan)
	}
}

// BenchmarkEthicsKernel benchmarks the ethics evaluation
func BenchmarkEthicsKernel(b *testing.B) {
	kernel := NewEthicsKernel()

	target := perception.TrackedObject{
		ID:          "target-1",
		ClassType:   perception.ClassHuman,
		Position:    perception.Vector3{X: 20, Y: 10, Z: 0},
		Velocity:    perception.Vector3{X: 1, Y: 0, Z: 0},
		ThreatLevel: 0.7,
		Metadata:    make(map[string]interface{}),
	}

	hunoidState := HunoidState{
		Position:     perception.Vector3{X: 0, Y: 0, Z: 0},
		BatteryLevel: 0.8,
		MaxSpeed:     5.0,
	}

	path := []perception.Vector3{
		{X: 5, Y: 2.5, Z: 0},
		{X: 10, Y: 5, Z: 0},
		{X: 15, Y: 7.5, Z: 0},
		{X: 20, Y: 10, Z: 0},
	}

	allObjects := createTestObjects(30)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = kernel.EvaluateRescueAction(hunoidState, target, allObjects, path)
	}
}

// BenchmarkOctreeQuery benchmarks spatial queries
func BenchmarkOctreeQuery(b *testing.B) {
	octree := perception.NewOctree(perception.Vector3{X: 0, Y: 0, Z: 0}, 100.0)

	// Insert 1000 objects
	for i := 0; i < 1000; i++ {
		obj := &perception.TrackedObject{
			ID:        string(rune(i)),
			ClassType: perception.ClassHuman,
			Position: perception.Vector3{
				X: float64(i%100) - 50,
				Y: float64((i/100)%100) - 50,
				Z: float64(i/10000) - 5,
			},
		}
		octree.Insert(obj)
	}

	center := perception.Vector3{X: 0, Y: 0, Z: 0}
	radius := 25.0

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = octree.QueryRadius(center, radius)
	}
}

// TestLatencyRequirement validates the 100ms latency requirement
func TestLatencyRequirement(t *testing.T) {
	prioritizer := NewRescuePrioritizer()
	kernel := NewEthicsKernel()

	// Worst-case scenario: 100 objects, 20 humans in danger
	scan := createTestScan(100, 20)

	hunoidState := HunoidState{
		Position:         perception.Vector3{X: 0, Y: 0, Z: 0},
		Velocity:         perception.Vector3{X: 0, Y: 0, Z: 0},
		BatteryLevel:     0.8,
		MaxSpeed:         5.0,
		CarryingCapacity: 2,
		CurrentLoad:      0,
	}

	// Run multiple iterations to get stable timing
	iterations := 100
	totalDuration := time.Duration(0)

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Full pipeline: scan -> prioritize -> ethics check
		priorities := prioritizer.CalculatePriorities(hunoidState, scan)

		if len(priorities) > 0 {
			path := []perception.Vector3{perception.Vector3{X: 1, Y: 0, Z: 0}.Scale(priorities[0].Components.AccessibilityScore)}
			_ = kernel.EvaluateRescueAction(hunoidState, scan.Objects[0], scan.Objects, path)
		}

		totalDuration += time.Since(start)
	}

	averageLatency := totalDuration / time.Duration(iterations)

	t.Logf("Average latency: %v", averageLatency)
	t.Logf("Max allowed: 100ms")

	if averageLatency > 100*time.Millisecond {
		t.Errorf("Latency requirement FAILED: %v > 100ms", averageLatency)
	} else {
		t.Logf("Latency requirement PASSED: %v < 100ms", averageLatency)
	}
}

// Helper functions

func createTestScan(totalObjects, humansInDanger int) *perception.ScanResult360 {
	objects := make([]perception.TrackedObject, totalObjects)

	for i := 0; i < totalObjects; i++ {
		objClass := perception.ClassObstacle
		threatLevel := 0.0

		if i < humansInDanger {
			objClass = perception.ClassHuman
			threatLevel = 0.5 + float64(i)*0.05 // Varying threat levels
		} else if i < humansInDanger+10 {
			objClass = perception.ClassHuman
			threatLevel = 0.1 // Safe humans
		} else if i < humansInDanger+20 {
			objClass = perception.ClassVehicle
		} else if i < humansInDanger+30 {
			objClass = perception.ClassDebris
		}

		objects[i] = perception.TrackedObject{
			ID:        string(rune('A' + i)),
			ClassType: objClass,
			Position: perception.Vector3{
				X: float64((i*7)%100) - 50,
				Y: float64((i*11)%100) - 50,
				Z: float64((i*3)%20) - 10,
			},
			Velocity: perception.Vector3{
				X: float64(i%10) - 5,
				Y: float64((i+1)%10) - 5,
				Z: 0,
			},
			ThreatLevel: threatLevel,
			Confidence:  0.9,
			Metadata:    make(map[string]interface{}),
		}
	}

	return &perception.ScanResult360{
		Timestamp:  time.Now(),
		Objects:    objects,
		HumanCount: humansInDanger + 10,
	}
}

func createTestObjects(count int) []perception.TrackedObject {
	objects := make([]perception.TrackedObject, count)
	for i := 0; i < count; i++ {
		objects[i] = perception.TrackedObject{
			ID:        string(rune('A' + i)),
			ClassType: perception.ClassHuman,
			Position: perception.Vector3{
				X: float64((i*7)%100) - 50,
				Y: float64((i*11)%100) - 50,
				Z: 0,
			},
			Metadata: make(map[string]interface{}),
		}
	}
	return objects
}
