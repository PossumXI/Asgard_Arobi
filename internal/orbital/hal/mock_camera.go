package hal

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math/rand"
	"sync"
	"time"
)

// MockCamera simulates a camera for testing
type MockCamera struct {
	mu          sync.Mutex
	isStreaming bool
	frameCount  uint64
	exposure    int
	gain        float64
	temperature float64
	voltage     float64
}

func NewMockCamera() *MockCamera {
	return &MockCamera{
		exposure:    1000,
		gain:        1.0,
		temperature: 25.0,
		voltage:     12.0,
	}
}

func (c *MockCamera) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frameCount = 0
	return nil
}

func (c *MockCamera) CaptureFrame(ctx context.Context) ([]byte, error) {
	c.mu.Lock()
	c.frameCount++
	c.mu.Unlock()

	// Generate a test image
	img := c.generateTestImage()

	// Encode as JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("failed to encode frame: %w", err)
	}

	return buf.Bytes(), nil
}

func (c *MockCamera) StartStream(ctx context.Context, frameChan chan<- []byte) error {
	c.mu.Lock()
	if c.isStreaming {
		c.mu.Unlock()
		return fmt.Errorf("stream already active")
	}
	c.isStreaming = true
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond) // 10 FPS
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				frame, err := c.CaptureFrame(ctx)
				if err != nil {
					continue
				}

				select {
				case frameChan <- frame:
				case <-ctx.Done():
					return
				default:
					// Drop frame if channel is full
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (c *MockCamera) StopStream() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isStreaming = false
	return nil
}

func (c *MockCamera) SetExposure(microseconds int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if microseconds < 0 || microseconds > 1000000 {
		return fmt.Errorf("exposure out of range: %d", microseconds)
	}

	c.exposure = microseconds
	return nil
}

func (c *MockCamera) SetGain(gain float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if gain < 0 || gain > 10 {
		return fmt.Errorf("gain out of range: %f", gain)
	}

	c.gain = gain
	return nil
}

func (c *MockCamera) GetDiagnostics() (CameraDiagnostics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Simulate temperature drift
	c.temperature = 25.0 + (rand.Float64() * 10.0)

	return CameraDiagnostics{
		Temperature: c.temperature,
		Voltage:     c.voltage,
		FrameCount:  c.frameCount,
		ErrorCount:  0,
	}, nil
}

func (c *MockCamera) Shutdown() error {
	return c.StopStream()
}

func (c *MockCamera) generateTestImage() image.Image {
	// Generate a 640x480 test pattern
	width, height := 640, 480
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a gradient with some noise
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(rand.Intn(256))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return img
}
