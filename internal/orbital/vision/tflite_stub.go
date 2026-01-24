//go:build !tflite

package vision

import (
	"context"
)

// NewTFLiteVisionProcessor falls back to the simple processor when TFLite is unavailable.
func NewTFLiteVisionProcessor() *TFLiteVisionProcessor {
	return &TFLiteVisionProcessor{fallback: NewSimpleVisionProcessor()}
}

type TFLiteVisionProcessor struct {
	fallback *SimpleVisionProcessor
}

func (p *TFLiteVisionProcessor) Initialize(ctx context.Context, modelPath string) error {
	return p.fallback.Initialize(ctx, modelPath)
}

func (p *TFLiteVisionProcessor) Detect(ctx context.Context, frame []byte) ([]Detection, error) {
	return p.fallback.Detect(ctx, frame)
}

func (p *TFLiteVisionProcessor) GetModelInfo() ModelInfo {
	info := p.fallback.GetModelInfo()
	info.Name = "TFLite-Processor (fallback)"
	return info
}

func (p *TFLiteVisionProcessor) Shutdown() error {
	return p.fallback.Shutdown()
}
