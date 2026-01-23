package vision

import (
	"context"
)

// Detection represents a detected object
type Detection struct {
	Class       string
	Confidence  float64
	BoundingBox BoundingBox
	Timestamp   int64
}

// BoundingBox defines object location in image
type BoundingBox struct {
	X      int
	Y      int
	Width  int
	Height int
}

// VisionProcessor defines the interface for AI vision
type VisionProcessor interface {
	Initialize(ctx context.Context, modelPath string) error
	Detect(ctx context.Context, frame []byte) ([]Detection, error)
	GetModelInfo() ModelInfo
	Shutdown() error
}

// ModelInfo contains model metadata
type ModelInfo struct {
	Name      string
	Version   string
	InputSize [2]int // width, height
	Classes   []string
}

// AlertCriteria defines when to generate alerts
type AlertCriteria struct {
	MinConfidence float64
	AlertClasses  []string
}

// ShouldAlert checks if detection meets alert criteria
func (c *AlertCriteria) ShouldAlert(det Detection) bool {
	if det.Confidence < c.MinConfidence {
		return false
	}

	for _, class := range c.AlertClasses {
		if det.Class == class {
			return true
		}
	}

	return false
}
