package vision

// Note: Detection, BoundingBox, ModelInfo, and VisionProcessor are defined in yolo_processor.go

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
