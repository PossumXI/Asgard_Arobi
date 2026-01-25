package sensors

import (
	"math"
	"time"
)

// WiFiRouter represents a WiFi access point used for CSI imaging.
type WiFiRouter struct {
	ID           string  `json:"id"`
	Position     Vector3D `json:"position"`
	FrequencyGHz float64 `json:"frequencyGhz"`
	TxPowerDbm   float64 `json:"txPowerDbm"`
}

// WiFiImagingFrame represents CSI-based propagation data.
type WiFiImagingFrame struct {
	RouterID        string     `json:"routerId"`
	ReceiverID      string     `json:"receiverId"`
	PathLossDb      float64    `json:"pathLossDb"`
	MultipathSpread float64    `json:"multipathSpread"`
	Timestamp       time.Time  `json:"timestamp"`
	Confidence      float64    `json:"confidence"`
}

// ThroughWallObservation describes estimated obstructions and target cues.
type ThroughWallObservation struct {
	EstimatedPosition Vector3D `json:"estimatedPosition"`
	Material          string   `json:"material"`
	EstimatedDepthM   float64  `json:"estimatedDepthM"`
	Confidence        float64  `json:"confidence"`
}

// WiFiImagingModel provides coarse through-wall estimation from WiFi CSI.
type WiFiImagingModel struct {
	materialLossDb map[string]float64
}

// NewWiFiImagingModel creates a model with default material loss estimates.
func NewWiFiImagingModel() *WiFiImagingModel {
	return &WiFiImagingModel{
		materialLossDb: map[string]float64{
			"drywall":   3.0,
			"brick":     8.0,
			"concrete":  12.0,
			"glass":     2.0,
			"wood":      4.0,
			"composite": 6.0,
		},
	}
}

// EstimateThroughWall uses CSI-derived path loss to estimate concealed targets.
func (m *WiFiImagingModel) EstimateThroughWall(frame WiFiImagingFrame, routers []WiFiRouter) ([]ThroughWallObservation, float64) {
	route, ok := m.findRouter(frame.RouterID, routers)
	if !ok {
		return nil, 0.0
	}

	// Normalize loss to a 0-1 confidence band.
	attenuationScore := math.Max(0.0, math.Min(1.0, 1.0-(frame.PathLossDb/120.0)))
	phaseSpread := math.Max(0.0, math.Min(1.0, 1.0-(frame.MultipathSpread/20.0)))
	confidence := 0.5*attenuationScore + 0.5*phaseSpread

	// Approximate depth based on additional loss beyond free-space.
	freeSpaceLoss := 32.4 + 20*math.Log10(route.FrequencyGHz) // simplified FSPL at 1m
	excessLoss := math.Max(0.0, frame.PathLossDb-freeSpaceLoss)
	depthMeters := math.Max(0.5, excessLoss/6.0)

	observation := ThroughWallObservation{
		EstimatedPosition: Vector3D{
			X: route.Position.X + depthMeters,
			Y: route.Position.Y,
			Z: route.Position.Z,
		},
		Material:        m.bestMaterial(excessLoss),
		EstimatedDepthM: depthMeters,
		Confidence:      confidence,
	}

	return []ThroughWallObservation{observation}, confidence
}

func (m *WiFiImagingModel) findRouter(id string, routers []WiFiRouter) (WiFiRouter, bool) {
	for _, router := range routers {
		if router.ID == id {
			return router, true
		}
	}
	return WiFiRouter{}, false
}

func (m *WiFiImagingModel) bestMaterial(excessLoss float64) string {
	best := "drywall"
	bestDelta := math.MaxFloat64
	for material, loss := range m.materialLossDb {
		delta := math.Abs(excessLoss - loss)
		if delta < bestDelta {
			bestDelta = delta
			best = material
		}
	}
	return best
}
