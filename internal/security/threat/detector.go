package threat

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/observability"
	"github.com/asgard/pandora/internal/security/scanner"
	"github.com/google/uuid"
)

// Threat represents a security threat
type Threat struct {
	ID          uuid.UUID
	Type        string
	Severity    scanner.ThreatLevel
	SourceIP    string
	Target      string
	Description string
	DetectedAt  time.Time
	Status      ThreatStatus
	MitigatedAt *time.Time
}

// ThreatStatus represents threat state
type ThreatStatus string

const (
	ThreatStatusNew           ThreatStatus = "new"
	ThreatStatusAnalyzing     ThreatStatus = "analyzing"
	ThreatStatusMitigating    ThreatStatus = "mitigating"
	ThreatStatusMitigated     ThreatStatus = "mitigated"
	ThreatStatusFalsePositive ThreatStatus = "false_positive"
)

// Detector processes anomalies and generates threats
type Detector struct {
	mu            sync.RWMutex
	threatChan    chan<- Threat
	recentThreats map[string]time.Time // Deduplication
	scanner       scanner.Scanner
}

// NewDetector creates a new threat detector
func NewDetector(scanner scanner.Scanner, threatChan chan<- Threat) *Detector {
	return &Detector{
		threatChan:    threatChan,
		recentThreats: make(map[string]time.Time),
		scanner:       scanner,
	}
}

// ProcessAnomaly converts an anomaly to a threat
func (d *Detector) ProcessAnomaly(ctx context.Context, anomaly *scanner.Anomaly) error {
	if anomaly == nil {
		return nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Deduplication: don't create threat for same source within 1 minute
	key := anomaly.SourceIP.String() + ":" + anomaly.Type
	if lastTime, exists := d.recentThreats[key]; exists {
		if time.Since(lastTime) < 1*time.Minute {
			return nil // Skip duplicate
		}
	}

	threat := Threat{
		ID:          uuid.New(),
		Type:        anomaly.Type,
		Severity:    anomaly.Severity,
		SourceIP:    anomaly.SourceIP.String(),
		Target:      "asgard_system",
		Description: anomaly.Description,
		DetectedAt:  anomaly.Timestamp,
		Status:      ThreatStatusNew,
	}

	log.Printf("THREAT DETECTED: %s (severity: %s, confidence: %.2f)", threat.Type, threat.Severity, anomaly.Confidence)
	observability.RecordThreatDetected(threat.Type, string(threat.Severity))

	// Send to threat channel (non-blocking)
	select {
	case d.threatChan <- threat:
		d.recentThreats[key] = time.Now()
	default:
		log.Printf("Threat channel full, dropping threat %s", threat.ID)
	}

	// Clean up old deduplication entries
	d.cleanupRecentThreats()

	return nil
}

func (d *Detector) cleanupRecentThreats() {
	cutoff := time.Now().Add(-5 * time.Minute)
	for key, timestamp := range d.recentThreats {
		if timestamp.Before(cutoff) {
			delete(d.recentThreats, key)
		}
	}
}
