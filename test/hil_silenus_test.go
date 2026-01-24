package test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
	"time"

	"github.com/asgard/pandora/internal/orbital/hal"
	"github.com/asgard/pandora/internal/orbital/vision"
)

type mockCamera struct {
	streaming bool
}

func (m *mockCamera) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockCamera) CaptureFrame(ctx context.Context) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 60, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *mockCamera) StartStream(ctx context.Context, frameChan chan<- []byte) error {
	m.streaming = true
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				frame, err := m.CaptureFrame(ctx)
				if err != nil {
					continue
				}
				select {
				case frameChan <- frame:
				default:
				}
			}
		}
	}()
	return nil
}

func (m *mockCamera) StopStream() error {
	m.streaming = false
	return nil
}

func (m *mockCamera) SetExposure(microseconds int) error { return nil }
func (m *mockCamera) SetGain(gain float64) error         { return nil }
func (m *mockCamera) GetDiagnostics() (hal.CameraDiagnostics, error) {
	return hal.CameraDiagnostics{Temperature: 25.0, Voltage: 12.0}, nil
}
func (m *mockCamera) Shutdown() error { return nil }

func TestSilenusHILPipeline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	camera := &mockCamera{}
	if err := camera.Initialize(ctx); err != nil {
		t.Fatalf("camera init failed: %v", err)
	}
	defer camera.Shutdown()

	frameChan := make(chan []byte, 2)
	if err := camera.StartStream(ctx, frameChan); err != nil {
		t.Fatalf("camera stream failed: %v", err)
	}
	defer camera.StopStream()

	var frame []byte
	select {
	case frame = <-frameChan:
	case <-ctx.Done():
		t.Fatal("timed out waiting for frame")
	}
	if len(frame) == 0 {
		t.Fatal("received empty frame")
	}

	visionProc := vision.NewSimpleVisionProcessor()
	if err := visionProc.Initialize(ctx, ""); err != nil {
		t.Fatalf("vision init failed: %v", err)
	}
	defer visionProc.Shutdown()

	if _, err := visionProc.Detect(ctx, frame); err != nil {
		t.Fatalf("vision detect failed: %v", err)
	}
}
