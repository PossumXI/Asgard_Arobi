package tracking

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/orbital/vision"
	"github.com/google/uuid"
)

// Alert represents a triggered alert
type Alert struct {
	ID         uuid.UUID
	Type       string
	Confidence float64
	Location   string // Could be lat/lon in production
	VideoClip  []byte // Short video segment
	Timestamp  time.Time
	Status     AlertStatus
}

// AlertStatus represents alert state
type AlertStatus string

const (
	AlertStatusNew          AlertStatus = "new"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusDispatched   AlertStatus = "dispatched"
	AlertStatusResolved     AlertStatus = "resolved"
)

// Tracker processes detections and generates alerts
type Tracker struct {
	mu               sync.Mutex
	criteria         vision.AlertCriteria
	alertChan        chan<- Alert
	recentAlerts     map[string]time.Time // Deduplication
	locationProvider func(context.Context) (string, error)
	clipProvider     func(context.Context) ([]byte, error)
}

func NewTracker(criteria vision.AlertCriteria, alertChan chan<- Alert, locationProvider func(context.Context) (string, error), clipProvider func(context.Context) ([]byte, error)) *Tracker {
	return &Tracker{
		criteria:         criteria,
		alertChan:        alertChan,
		recentAlerts:     make(map[string]time.Time),
		locationProvider: locationProvider,
		clipProvider:     clipProvider,
	}
}

// ProcessDetections examines detections and generates alerts
func (t *Tracker) ProcessDetections(ctx context.Context, detections []vision.Detection) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, det := range detections {
		if t.criteria.ShouldAlert(det) {
			if err := t.generateAlert(ctx, det); err != nil {
				log.Printf("Failed to generate alert: %v", err)
				continue
			}
		}
	}

	// Clean up old deduplication entries
	t.cleanupRecentAlerts()

	return nil
}

func (t *Tracker) generateAlert(ctx context.Context, det vision.Detection) error {
	// Deduplication: don't alert for same class within 5 minutes
	if lastTime, exists := t.recentAlerts[det.Class]; exists {
		if time.Since(lastTime) < 5*time.Minute {
			return nil // Skip duplicate
		}
	}

	if t.locationProvider == nil || t.clipProvider == nil {
		return fmt.Errorf("alert providers not configured")
	}

	location, err := t.locationProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve alert location: %w", err)
	}

	clip, err := t.clipProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to build alert clip: %w", err)
	}

	alert := Alert{
		ID:         uuid.New(),
		Type:       det.Class,
		Confidence: det.Confidence,
		Location:   location,
		VideoClip:  clip,
		Timestamp:  time.Now().UTC(),
		Status:     AlertStatusNew,
	}

	log.Printf("ALERT GENERATED: %s (confidence: %.2f)", alert.Type, alert.Confidence)

	// Send to alert channel (non-blocking)
	select {
	case t.alertChan <- alert:
		t.recentAlerts[det.Class] = time.Now()
	default:
		return fmt.Errorf("alert channel full")
	}

	return nil
}

func (t *Tracker) cleanupRecentAlerts() {
	cutoff := time.Now().Add(-5 * time.Minute)
	for class, timestamp := range t.recentAlerts {
		if timestamp.Before(cutoff) {
			delete(t.recentAlerts, class)
		}
	}
}
