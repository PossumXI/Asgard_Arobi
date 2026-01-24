package vision

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"time"
)

// SimpleVisionProcessor implements a deterministic, non-ML vision backend.
// It detects fire/smoke-like patterns via color heuristics.
type SimpleVisionProcessor struct {
	model        ModelInfo
	alertClasses map[string]struct{}
}

func NewSimpleVisionProcessor() *SimpleVisionProcessor {
	return &SimpleVisionProcessor{
		model: ModelInfo{
			Name:      "Silenus-Simple-Vision",
			Version:   "1.0.0",
			InputSize: [2]int{640, 480},
			Classes: []string{
				"fire",
				"smoke",
			},
		},
		alertClasses: map[string]struct{}{
			"fire":  {},
			"smoke": {},
		},
	}
}

func (p *SimpleVisionProcessor) Initialize(ctx context.Context, modelPath string) error {
	// No external model needed; validate we can process JPEG data.
	return nil
}

func (p *SimpleVisionProcessor) Detect(ctx context.Context, frame []byte) ([]Detection, error) {
	img, err := decodeJPEG(frame)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return nil, fmt.Errorf("empty frame")
	}

	step := 8
	totalSamples := 0
	fireSamples := 0
	smokeSamples := 0
	minX, minY := width, height
	maxX, maxY := 0, 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			totalSamples++
			r, g, b, _ := img.At(x, y).RGBA()
			rr := uint8(r >> 8)
			gg := uint8(g >> 8)
			bb := uint8(b >> 8)

			if isFirePixel(rr, gg, bb) {
				fireSamples++
				minX = minInt(minX, x)
				minY = minInt(minY, y)
				maxX = maxInt(maxX, x)
				maxY = maxInt(maxY, y)
				continue
			}

			if isSmokePixel(rr, gg, bb) {
				smokeSamples++
			}
		}
	}

	if totalSamples == 0 {
		return nil, nil
	}

	fireRatio := float64(fireSamples) / float64(totalSamples)
	smokeRatio := float64(smokeSamples) / float64(totalSamples)

	detections := []Detection{}
	if fireRatio >= 0.012 {
		detections = append(detections, Detection{
			Class:      "fire",
			Confidence: minFloat(0.99, fireRatio*6),
			BoundingBox: BoundingBox{
				X:      clampInt(minX, 0, width),
				Y:      clampInt(minY, 0, height),
				Width:  clampInt(maxX-minX+step, 0, width),
				Height: clampInt(maxY-minY+step, 0, height),
			},
			Timestamp: time.Now().Unix(),
		})
	}

	if smokeRatio >= 0.02 {
		detections = append(detections, Detection{
			Class:      "smoke",
			Confidence: minFloat(0.95, smokeRatio*4),
			BoundingBox: BoundingBox{
				X:      0,
				Y:      0,
				Width:  width,
				Height: height,
			},
			Timestamp: time.Now().Unix(),
		})
	}

	return detections, nil
}

func (p *SimpleVisionProcessor) GetModelInfo() ModelInfo {
	return p.model
}

func (p *SimpleVisionProcessor) Shutdown() error {
	return nil
}

func decodeJPEG(frame []byte) (image.Image, error) {
	img, err := jpeg.Decode(bytes.NewReader(frame))
	if err != nil {
		return nil, fmt.Errorf("jpeg decode failed: %w", err)
	}
	return img, nil
}

func isFirePixel(r, g, b uint8) bool {
	return r > 170 && g > 60 && g < 200 && b < 80 && r > g && g > b
}

func isSmokePixel(r, g, b uint8) bool {
	c := color.RGBA{R: r, G: g, B: b, A: 255}
	avg := (int(c.R) + int(c.G) + int(c.B)) / 3
	lowVariance := absInt(int(c.R)-avg) < 15 && absInt(int(c.G)-avg) < 15 && absInt(int(c.B)-avg) < 15
	return avg > 170 && lowVariance
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
