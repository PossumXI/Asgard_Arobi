package integration_test

import (
	"testing"

	"github.com/asgard/pandora/internal/orbital/vision"
)

func TestAlertCriteriaShouldAlert(t *testing.T) {
	criteria := vision.AlertCriteria{
		MinConfidence: 0.7,
		AlertClasses:  []string{"fire", "smoke", "vehicle"},
	}

	testCases := []struct {
		name        string
		detection   vision.Detection
		shouldAlert bool
	}{
		{
			name: "fire above threshold",
			detection: vision.Detection{
				Class:      "fire",
				Confidence: 0.85,
			},
			shouldAlert: true,
		},
		{
			name: "fire below threshold",
			detection: vision.Detection{
				Class:      "fire",
				Confidence: 0.5,
			},
			shouldAlert: false,
		},
		{
			name: "non-alert class",
			detection: vision.Detection{
				Class:      "person",
				Confidence: 0.95,
			},
			shouldAlert: false,
		},
		{
			name: "smoke at threshold",
			detection: vision.Detection{
				Class:      "smoke",
				Confidence: 0.7,
			},
			shouldAlert: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := criteria.ShouldAlert(tc.detection)
			if result != tc.shouldAlert {
				t.Errorf("expected shouldAlert=%v, got %v", tc.shouldAlert, result)
			}
		})
	}
}

func TestAlertCriteriaEmptyClasses(t *testing.T) {
	criteria := vision.AlertCriteria{
		MinConfidence: 0.5,
		AlertClasses:  []string{}, // Empty - should match nothing
	}

	detection := vision.Detection{
		Class:      "fire",
		Confidence: 0.9,
	}

	if criteria.ShouldAlert(detection) {
		t.Error("empty alert classes should not match any detection")
	}
}

func TestAlertCriteriaHighThreshold(t *testing.T) {
	criteria := vision.AlertCriteria{
		MinConfidence: 0.99,
		AlertClasses:  []string{"fire"},
	}

	detection := vision.Detection{
		Class:      "fire",
		Confidence: 0.95,
	}

	if criteria.ShouldAlert(detection) {
		t.Error("detection below high threshold should not alert")
	}
}

func TestDetectionBoundingBox(t *testing.T) {
	detection := vision.Detection{
		Class:      "vehicle",
		Confidence: 0.88,
		BoundingBox: vision.BoundingBox{
			X:      100,
			Y:      200,
			Width:  50,
			Height: 30,
		},
	}

	// Verify bounding box values
	if detection.BoundingBox.X != 100 {
		t.Errorf("expected X=100, got %d", detection.BoundingBox.X)
	}
	if detection.BoundingBox.Width != 50 {
		t.Errorf("expected Width=50, got %d", detection.BoundingBox.Width)
	}
}

func TestMultipleDetections(t *testing.T) {
	criteria := vision.AlertCriteria{
		MinConfidence: 0.6,
		AlertClasses:  []string{"fire", "smoke"},
	}

	detections := []vision.Detection{
		{Class: "fire", Confidence: 0.9},   // Should alert
		{Class: "smoke", Confidence: 0.7},  // Should alert
		{Class: "person", Confidence: 0.95}, // Should not alert (wrong class)
		{Class: "fire", Confidence: 0.5},   // Should not alert (low confidence)
	}

	alertCount := 0
	for _, det := range detections {
		if criteria.ShouldAlert(det) {
			alertCount++
		}
	}

	if alertCount != 2 {
		t.Errorf("expected 2 alerts, got %d", alertCount)
	}
}
