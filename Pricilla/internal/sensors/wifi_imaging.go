package sensors

import (
	"fmt"
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
	RSSIDbm         float64    `json:"rssiDbm,omitempty"`
	NoiseFloorDbm   float64    `json:"noiseFloorDbm,omitempty"`
	CsiMagnitudes   []float64  `json:"csiMagnitudes,omitempty"`
	CsiPhases       []float64  `json:"csiPhases,omitempty"`
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

type wifiSample struct {
	router     WiFiRouter
	distance   float64
	confidence float64
	excessLoss float64
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
	observations, confidence, _ := m.EstimateThroughWallMulti([]WiFiImagingFrame{frame}, routers)
	return observations, confidence
}

// EstimateThroughWallMulti uses multiple routers for triangulation.
func (m *WiFiImagingModel) EstimateThroughWallMulti(frames []WiFiImagingFrame, routers []WiFiRouter) ([]ThroughWallObservation, float64, error) {
	if len(frames) == 0 {
		return nil, 0.0, fmt.Errorf("no wifi imaging frames provided")
	}

	samples := make([]wifiSample, 0, len(frames))
	for _, frame := range frames {
		router, ok := m.findRouter(frame.RouterID, routers)
		if !ok {
			continue
		}

		distance, excessLoss, confidence := m.estimateRangeMeters(frame, router)
		if distance <= 0 || confidence <= 0 {
			continue
		}

		samples = append(samples, wifiSample{
			router:     router,
			distance:   distance,
			confidence: confidence,
			excessLoss: excessLoss,
		})
	}

	if len(samples) < 2 {
		return nil, 0.0, fmt.Errorf("insufficient router coverage for triangulation")
	}

	estimated := triangulate2D(samples)
	avgDistance := 0.0
	avgExcessLoss := 0.0
	avgConfidence := 0.0
	weightSum := 0.0
	fitError := 0.0

	for _, s := range samples {
		avgDistance += s.distance * s.confidence
		avgExcessLoss += s.excessLoss * s.confidence
		avgConfidence += s.confidence
		weightSum += s.confidence

		dx := estimated.X - s.router.Position.X
		dy := estimated.Y - s.router.Position.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		fitError += math.Abs(dist-s.distance) * s.confidence
	}

	if weightSum > 0 {
		avgDistance /= weightSum
		avgExcessLoss /= weightSum
		avgConfidence /= float64(len(samples))
		fitError /= weightSum
	}

	routerFactor := math.Min(1.0, float64(len(samples))/3.0)
	fitScore := math.Exp(-fitError / 5.0)
	confidence := clamp(0.6*avgConfidence+0.2*routerFactor+0.2*fitScore, 0.0, 1.0)

	observation := ThroughWallObservation{
		EstimatedPosition: estimated,
		Material:          m.bestMaterial(avgExcessLoss),
		EstimatedDepthM:   avgDistance,
		Confidence:        confidence,
	}

	return []ThroughWallObservation{observation}, confidence, nil
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

func (m *WiFiImagingModel) estimateRangeMeters(frame WiFiImagingFrame, router WiFiRouter) (float64, float64, float64) {
	if frame.PathLossDb <= 0 || router.FrequencyGHz <= 0 {
		return 0, 0, 0
	}

	freeSpaceLoss := 32.4 + 20*math.Log10(router.FrequencyGHz)
	excessLoss := math.Max(0.0, frame.PathLossDb-freeSpaceLoss)

	pathLossExponent := 2.0 + math.Min(2.0, frame.MultipathSpread/10.0)
	distance := math.Pow(10, (frame.PathLossDb-freeSpaceLoss)/(10*pathLossExponent))
	if distance < 0.5 {
		distance = 0.5
	}

	attenuationScore := clamp(1.0-(frame.PathLossDb/120.0), 0.0, 1.0)
	phaseSpread := clamp(1.0-(frame.MultipathSpread/20.0), 0.0, 1.0)
	csiQuality := m.estimateCSIQuality(frame)
	confidence := clamp(0.4*attenuationScore+0.3*phaseSpread+0.3*csiQuality, 0.0, 1.0)

	if frame.Confidence > 0 {
		confidence = clamp(0.7*confidence+0.3*clamp(frame.Confidence, 0.0, 1.0), 0.0, 1.0)
	}

	return distance, excessLoss, confidence
}

func (m *WiFiImagingModel) estimateCSIQuality(frame WiFiImagingFrame) float64 {
	qualitySum := 0.0
	parts := 0

	if len(frame.CsiPhases) > 1 {
		mean := 0.0
		for _, v := range frame.CsiPhases {
			mean += v
		}
		mean /= float64(len(frame.CsiPhases))

		variance := 0.0
		for _, v := range frame.CsiPhases {
			diff := v - mean
			variance += diff * diff
		}
		variance /= float64(len(frame.CsiPhases))
		qualitySum += math.Exp(-variance)
		parts++
	}

	if len(frame.CsiMagnitudes) > 1 {
		mean := 0.0
		for _, v := range frame.CsiMagnitudes {
			mean += v
		}
		mean /= float64(len(frame.CsiMagnitudes))

		variance := 0.0
		for _, v := range frame.CsiMagnitudes {
			diff := v - mean
			variance += diff * diff
		}
		variance /= float64(len(frame.CsiMagnitudes))
		if mean > 0 {
			variance /= mean * mean
		}
		qualitySum += 1.0 / (1.0 + variance)
		parts++
	}

	if parts == 0 {
		return 0.5
	}

	return clamp(qualitySum/float64(parts), 0.0, 1.0)
}

func triangulate2D(samples []wifiSample) Vector3D {
	x := 0.0
	y := 0.0
	z := 0.0
	weightSum := 0.0
	for _, s := range samples {
		weightSum += s.confidence
		x += s.router.Position.X * s.confidence
		y += s.router.Position.Y * s.confidence
		z += s.router.Position.Z * s.confidence
	}
	if weightSum > 0 {
		x /= weightSum
		y /= weightSum
		z /= weightSum
	}

	for iter := 0; iter < 6; iter++ {
		var j00, j01, j11, b0, b1 float64

		for _, s := range samples {
			dx := x - s.router.Position.X
			dy := y - s.router.Position.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1e-3 {
				continue
			}

			r := dist - s.distance
			jx := dx / dist
			jy := dy / dist
			w := s.confidence

			j00 += w * jx * jx
			j01 += w * jx * jy
			j11 += w * jy * jy
			b0 += w * jx * r
			b1 += w * jy * r
		}

		det := j00*j11 - j01*j01
		if math.Abs(det) < 1e-6 {
			break
		}

		dx := (-b0*j11 + b1*j01) / det
		dy := (-b1*j00 + b0*j01) / det
		x += dx
		y += dy
		if math.Sqrt(dx*dx+dy*dy) < 0.01 {
			break
		}
	}

	return Vector3D{X: x, Y: y, Z: z}
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
