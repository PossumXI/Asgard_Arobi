//go:build tflite

package vision

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"time"

	"github.com/mattn/go-tflite"
)

// TFLiteVisionProcessor provides a real TFLite backend when built with -tags=tflite.
type TFLiteVisionProcessor struct {
	model       *tflite.Model
	interpreter *tflite.Interpreter
	info        ModelInfo
	inputWidth  int
	inputHeight int
}

func NewTFLiteVisionProcessor() *TFLiteVisionProcessor {
	return &TFLiteVisionProcessor{
		info: ModelInfo{
			Name:      "TFLite-Processor",
			Version:   "1.0.0",
			InputSize: [2]int{640, 480},
			Classes:   []string{"fire", "smoke", "aircraft", "ship", "vehicle", "person"},
		},
	}
}

func (p *TFLiteVisionProcessor) Initialize(ctx context.Context, modelPath string) error {
	model := tflite.NewModelFromFile(modelPath)
	if model == nil {
		return fmt.Errorf("failed to load tflite model: %s", modelPath)
	}
	interpreter := tflite.NewInterpreter(model, nil)
	if interpreter == nil {
		model.Delete()
		return fmt.Errorf("failed to create tflite interpreter")
	}
	if status := interpreter.AllocateTensors(); status != tflite.OK {
		interpreter.Delete()
		model.Delete()
		return fmt.Errorf("failed to allocate tensors")
	}

	inputTensor := interpreter.GetInputTensor(0)
	if inputTensor == nil {
		interpreter.Delete()
		model.Delete()
		return fmt.Errorf("missing input tensor")
	}
	if inputTensor.NumDims() < 4 {
		interpreter.Delete()
		model.Delete()
		return fmt.Errorf("unexpected input tensor dims: %v", inputTensor.Shape())
	}
	p.inputHeight = inputTensor.Dim(1)
	p.inputWidth = inputTensor.Dim(2)
	p.info.InputSize = [2]int{p.inputWidth, p.inputHeight}

	p.model = model
	p.interpreter = interpreter
	return nil
}

func (p *TFLiteVisionProcessor) Detect(ctx context.Context, frame []byte) ([]Detection, error) {
	if p.interpreter == nil {
		return nil, fmt.Errorf("tflite interpreter not initialized")
	}

	img, err := decodeJPEGTFLite(frame)
	if err != nil {
		return nil, err
	}

	resized := resizeNearest(img, p.inputWidth, p.inputHeight)
	inputTensor := p.interpreter.GetInputTensor(0)
	if inputTensor == nil {
		return nil, fmt.Errorf("input tensor unavailable")
	}

	switch inputTensor.Type() {
	case tflite.UInt8:
		input := make([]uint8, p.inputWidth*p.inputHeight*3)
		fillUint8Input(resized, input)
		if status := inputTensor.CopyFromBuffer(&input[0]); status != tflite.OK {
			return nil, fmt.Errorf("failed to copy uint8 input")
		}
	case tflite.Float32:
		input := make([]float32, p.inputWidth*p.inputHeight*3)
		fillFloatInput(resized, input)
		if status := inputTensor.CopyFromBuffer(&input[0]); status != tflite.OK {
			return nil, fmt.Errorf("failed to copy float input")
		}
	default:
		return nil, fmt.Errorf("unsupported input tensor type: %v", inputTensor.Type())
	}

	if status := p.interpreter.Invoke(); status != tflite.OK {
		return nil, fmt.Errorf("tflite invoke failed")
	}

	return p.parseSSDOutputs()
}

func (p *TFLiteVisionProcessor) GetModelInfo() ModelInfo {
	return p.info
}

func (p *TFLiteVisionProcessor) Shutdown() error {
	if p.interpreter != nil {
		p.interpreter.Delete()
	}
	if p.model != nil {
		p.model.Delete()
	}
	return nil
}

func (p *TFLiteVisionProcessor) parseSSDOutputs() ([]Detection, error) {
	boxesTensor := p.interpreter.GetOutputTensor(0)
	classesTensor := p.interpreter.GetOutputTensor(1)
	scoresTensor := p.interpreter.GetOutputTensor(2)
	countTensor := p.interpreter.GetOutputTensor(3)

	if boxesTensor == nil || classesTensor == nil || scoresTensor == nil || countTensor == nil {
		return nil, fmt.Errorf("expected SSD output tensors not available")
	}

	boxes, err := readFloatTensor(boxesTensor)
	if err != nil {
		return nil, err
	}
	classes, err := readFloatTensor(classesTensor)
	if err != nil {
		return nil, err
	}
	scores, err := readFloatTensor(scoresTensor)
	if err != nil {
		return nil, err
	}
	counts, err := readFloatTensor(countTensor)
	if err != nil {
		return nil, err
	}
	if len(counts) == 0 {
		return nil, nil
	}

	num := int(math.Round(float64(counts[0])))
	detections := make([]Detection, 0)

	for i := 0; i < num; i++ {
		score := scores[i]
		if score < 0.5 {
			continue
		}

		boxOffset := i * 4
		ymin := boxes[boxOffset]
		xmin := boxes[boxOffset+1]
		ymax := boxes[boxOffset+2]
		xmax := boxes[boxOffset+3]

		classID := int(classes[i])
		class := "unknown"
		if classID >= 0 && classID < len(p.info.Classes) {
			class = p.info.Classes[classID]
		}

		detections = append(detections, Detection{
			Class:      class,
			Confidence: float64(score),
			BoundingBox: BoundingBox{
				X:      int(xmin * float32(p.inputWidth)),
				Y:      int(ymin * float32(p.inputHeight)),
				Width:  int((xmax - xmin) * float32(p.inputWidth)),
				Height: int((ymax - ymin) * float32(p.inputHeight)),
			},
			Timestamp: time.Now().Unix(),
		})
	}

	return detections, nil
}

func readFloatTensor(tensor *tflite.Tensor) ([]float32, error) {
	switch tensor.Type() {
	case tflite.Float32:
		buf := make([]float32, tensor.ByteSize()/4)
		if status := tensor.CopyToBuffer(&buf[0]); status != tflite.OK {
			return nil, fmt.Errorf("failed to read float tensor")
		}
		return buf, nil
	case tflite.UInt8:
		buf := make([]uint8, tensor.ByteSize())
		if status := tensor.CopyToBuffer(&buf[0]); status != tflite.OK {
			return nil, fmt.Errorf("failed to read uint8 tensor")
		}
		q := tensor.QuantizationParams()
		out := make([]float32, len(buf))
		for i, v := range buf {
			out[i] = float32(q.Scale) * float32(int(v)-q.ZeroPoint)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported tensor type: %v", tensor.Type())
	}
}

func decodeJPEGTFLite(frame []byte) (image.Image, error) {
	img, err := jpeg.Decode(bytes.NewReader(frame))
	if err != nil {
		return nil, fmt.Errorf("jpeg decode failed: %w", err)
	}
	return img, nil
}

func resizeNearest(img image.Image, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	for y := 0; y < height; y++ {
		srcY := srcBounds.Min.Y + y*srcH/height
		for x := 0; x < width; x++ {
			srcX := srcBounds.Min.X + x*srcW/width
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	return dst
}

func fillUint8Input(img *image.RGBA, buffer []uint8) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	idx := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			buffer[idx] = uint8(r >> 8)
			buffer[idx+1] = uint8(g >> 8)
			buffer[idx+2] = uint8(b >> 8)
			idx += 3
		}
	}
}

func fillFloatInput(img *image.RGBA, buffer []float32) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	idx := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			buffer[idx] = float32(r>>8) / 255.0
			buffer[idx+1] = float32(g>>8) / 255.0
			buffer[idx+2] = float32(b>>8) / 255.0
			idx += 3
		}
	}
}
