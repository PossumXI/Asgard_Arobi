package vision

import (
	"context"
	"math/rand"
	"time"
)

// MockVisionProcessor simulates AI object detection
type MockVisionProcessor struct {
	model        ModelInfo
	detectionRate float64 // Probability of detection per frame
}

func NewMockVisionProcessor() *MockVisionProcessor {
	return &MockVisionProcessor{
		model: ModelInfo{
			Name:      "YOLOv8-Nano-Mock",
			Version:   "1.0.0",
			InputSize: [2]int{640, 480},
			Classes: []string{
				"person",
				"vehicle",
				"aircraft",
				"ship",
				"fire",
				"smoke",
				"building",
			},
		},
		detectionRate: 0.15, // 15% chance of detection per frame
	}
}

func (p *MockVisionProcessor) Initialize(ctx context.Context, modelPath string) error {
	// Simulate model loading
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (p *MockVisionProcessor) Detect(ctx context.Context, frame []byte) ([]Detection, error) {
	var detections []Detection

	// Randomly generate detections
	if rand.Float64() < p.detectionRate {
		numDetections := rand.Intn(3) + 1

		for i := 0; i < numDetections; i++ {
			det := Detection{
				Class:      p.model.Classes[rand.Intn(len(p.model.Classes))],
				Confidence: 0.7 + (rand.Float64() * 0.3), // 0.7-1.0
				BoundingBox: BoundingBox{
					X:      rand.Intn(p.model.InputSize[0] - 100),
					Y:      rand.Intn(p.model.InputSize[1] - 100),
					Width:  50 + rand.Intn(150),
					Height: 50 + rand.Intn(150),
				},
				Timestamp: time.Now().Unix(),
			}

			detections = append(detections, det)
		}
	}

	return detections, nil
}

func (p *MockVisionProcessor) GetModelInfo() ModelInfo {
	return p.model
}

func (p *MockVisionProcessor) Shutdown() error {
	return nil
}
