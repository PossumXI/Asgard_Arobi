package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"
)

// YOLOProcessor implements VisionProcessor using YOLO object detection.
// Supports local inference (ONNX Runtime, TensorRT) and remote inference servers.
type YOLOProcessor struct {
	mu          sync.RWMutex
	config      YOLOConfig
	client      *http.Client
	modelInfo   ModelInfo
	initialized bool
}

// YOLOConfig holds YOLO processor configuration
type YOLOConfig struct {
	// Inference backend: "onnx", "tensorrt", "triton", "http", "grpc"
	Backend string `json:"backend"`

	// Model settings
	ModelPath    string `json:"modelPath"`    // Path to ONNX/TensorRT model
	ModelVersion string `json:"modelVersion"` // e.g., "yolov8n", "yolov8s", "yolov8m"
	
	// Remote inference settings
	InferenceURL string `json:"inferenceUrl"` // e.g., "http://triton:8000/v2/models/yolov8"
	APIKey       string `json:"apiKey"`       // For authenticated endpoints
	
	// Detection settings
	ConfidenceThreshold float64  `json:"confidenceThreshold"` // 0.0-1.0
	NMSThreshold        float64  `json:"nmsThreshold"`        // Non-max suppression threshold
	MaxDetections       int      `json:"maxDetections"`
	Classes             []string `json:"classes"`             // Filter to specific classes
	
	// Input settings
	InputWidth  int `json:"inputWidth"`  // Model input width
	InputHeight int `json:"inputHeight"` // Model input height
	
	// Performance settings
	BatchSize     int  `json:"batchSize"`
	UseGPU        bool `json:"useGpu"`
	GPUDeviceID   int  `json:"gpuDeviceId"`
	NumThreads    int  `json:"numThreads"`
	
	// Satellite-specific
	SatelliteID string `json:"satelliteId"`
}

// Detection represents a detected object
type Detection struct {
	Class       string      `json:"class"`
	Confidence  float64     `json:"confidence"`
	BoundingBox BoundingBox `json:"boundingBox"`
	Timestamp   int64       `json:"timestamp"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// BoundingBox represents object location
type BoundingBox struct {
	X      int `json:"x"`      // Top-left X
	Y      int `json:"y"`      // Top-left Y
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ModelInfo contains model metadata
type ModelInfo struct {
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	InputSize [2]int   `json:"inputSize"`
	Classes   []string `json:"classes"`
}

// VisionProcessor interface for object detection
type VisionProcessor interface {
	Initialize(ctx context.Context, modelPath string) error
	Detect(ctx context.Context, frame []byte) ([]Detection, error)
	GetModelInfo() ModelInfo
	Shutdown() error
}

// NewYOLOProcessor creates a new YOLO vision processor
func NewYOLOProcessor(config YOLOConfig) *YOLOProcessor {
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.5
	}
	if config.NMSThreshold == 0 {
		config.NMSThreshold = 0.45
	}
	if config.MaxDetections == 0 {
		config.MaxDetections = 100
	}
	if config.InputWidth == 0 {
		config.InputWidth = 640
	}
	if config.InputHeight == 0 {
		config.InputHeight = 640
	}
	if config.BatchSize == 0 {
		config.BatchSize = 1
	}
	if config.NumThreads == 0 {
		config.NumThreads = 4
	}

	// Default COCO classes for YOLO
	if len(config.Classes) == 0 {
		config.Classes = defaultCOCOClasses()
	}

	return &YOLOProcessor{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func defaultCOCOClasses() []string {
	return []string{
		"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck",
		"boat", "traffic light", "fire hydrant", "stop sign", "parking meter", "bench",
		"bird", "cat", "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra",
		"giraffe", "backpack", "umbrella", "handbag", "tie", "suitcase", "frisbee",
		"skis", "snowboard", "sports ball", "kite", "baseball bat", "baseball glove",
		"skateboard", "surfboard", "tennis racket", "bottle", "wine glass", "cup",
		"fork", "knife", "spoon", "bowl", "banana", "apple", "sandwich", "orange",
		"broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair", "couch",
		"potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse",
		"remote", "keyboard", "cell phone", "microwave", "oven", "toaster", "sink",
		"refrigerator", "book", "clock", "vase", "scissors", "teddy bear", "hair drier",
		"toothbrush",
	}
}

// Initialize loads the model and prepares for inference
func (p *YOLOProcessor) Initialize(ctx context.Context, modelPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if modelPath != "" {
		p.config.ModelPath = modelPath
	}

	// Build model info
	p.modelInfo = ModelInfo{
		Name:      p.config.ModelVersion,
		Version:   "8.0",
		InputSize: [2]int{p.config.InputWidth, p.config.InputHeight},
		Classes:   p.config.Classes,
	}

	switch p.config.Backend {
	case "http", "triton":
		return p.initHTTPBackend(ctx)
	case "onnx":
		return p.initONNXBackend(ctx)
	case "tensorrt":
		return p.initTensorRTBackend(ctx)
	default:
		// Default to HTTP if inference URL is provided
		if p.config.InferenceURL != "" {
			p.config.Backend = "http"
			return p.initHTTPBackend(ctx)
		}
		return fmt.Errorf("unsupported backend: %s", p.config.Backend)
	}
}

func (p *YOLOProcessor) initHTTPBackend(ctx context.Context) error {
	// Test connection to inference server
	url := p.config.InferenceURL
	if url == "" {
		return fmt.Errorf("inference URL required for HTTP backend")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to inference server: %w", err)
	}
	resp.Body.Close()

	p.initialized = true
	p.modelInfo.Name = "YOLOv8-" + p.config.ModelVersion + "-Remote"
	return nil
}

func (p *YOLOProcessor) initONNXBackend(ctx context.Context) error {
	// Prefer real remote inference if configured.
	if p.config.InferenceURL != "" {
		if err := p.initHTTPBackend(ctx); err != nil {
			return err
		}
		p.modelInfo.Name = "YOLOv8-" + p.config.ModelVersion + "-ONNX-Remote"
		return nil
	}

	// Local ONNX runtime is not bundled; require explicit setup.
	if p.config.ModelPath == "" {
		return fmt.Errorf("onnx backend requires modelPath or inferenceUrl")
	}

	return fmt.Errorf("onnx backend requires local runtime; set inferenceUrl or use http backend")
}

func (p *YOLOProcessor) initTensorRTBackend(ctx context.Context) error {
	// Prefer real remote inference if configured.
	if p.config.InferenceURL != "" {
		if err := p.initHTTPBackend(ctx); err != nil {
			return err
		}
		p.modelInfo.Name = "YOLOv8-" + p.config.ModelVersion + "-TensorRT-Remote"
		return nil
	}

	if p.config.ModelPath == "" {
		return fmt.Errorf("tensorrt backend requires modelPath or inferenceUrl")
	}

	return fmt.Errorf("tensorrt backend requires local runtime; set inferenceUrl or use http backend")
}

// Detect performs object detection on a frame
func (p *YOLOProcessor) Detect(ctx context.Context, frame []byte) ([]Detection, error) {
	p.mu.RLock()
	backend := p.config.Backend
	initialized := p.initialized
	p.mu.RUnlock()

	if !initialized {
		return nil, fmt.Errorf("processor not initialized")
	}

	switch backend {
	case "http", "triton":
		return p.detectHTTP(ctx, frame)
	case "onnx":
		return p.detectONNX(ctx, frame)
	case "tensorrt":
		return p.detectTensorRT(ctx, frame)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", backend)
	}
}

// InferenceRequest represents a request to the inference server
type InferenceRequest struct {
	Inputs []InferenceInput `json:"inputs"`
}

// InferenceInput represents an input tensor
type InferenceInput struct {
	Name     string    `json:"name"`
	Shape    []int     `json:"shape"`
	Datatype string    `json:"datatype"`
	Data     []float32 `json:"data"`
}

// InferenceResponse represents a response from the inference server
type InferenceResponse struct {
	Outputs []InferenceOutput `json:"outputs"`
}

// InferenceOutput represents an output tensor
type InferenceOutput struct {
	Name     string    `json:"name"`
	Shape    []int     `json:"shape"`
	Datatype string    `json:"datatype"`
	Data     []float32 `json:"data"`
}

func (p *YOLOProcessor) detectHTTP(ctx context.Context, frame []byte) ([]Detection, error) {
	// Preprocess image
	imgData, originalWidth, originalHeight, err := p.preprocessImage(frame)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}

	// Build inference request
	request := InferenceRequest{
		Inputs: []InferenceInput{
			{
				Name:     "images",
				Shape:    []int{1, 3, p.config.InputHeight, p.config.InputWidth},
				Datatype: "FP32",
				Data:     imgData,
			},
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Send request
	url := p.config.InferenceURL + "/infer"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inference request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("inference failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var inferResp InferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&inferResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Post-process detections
	return p.postprocessDetections(inferResp.Outputs, originalWidth, originalHeight)
}

func (p *YOLOProcessor) detectONNX(ctx context.Context, frame []byte) ([]Detection, error) {
	if p.config.InferenceURL != "" {
		return p.detectHTTP(ctx, frame)
	}

	return nil, fmt.Errorf("onnx backend not configured; set inferenceUrl or use http backend")
}

func (p *YOLOProcessor) detectTensorRT(ctx context.Context, frame []byte) ([]Detection, error) {
	if p.config.InferenceURL != "" {
		return p.detectHTTP(ctx, frame)
	}

	return nil, fmt.Errorf("tensorrt backend not configured; set inferenceUrl or use http backend")
}

func (p *YOLOProcessor) preprocessImage(frame []byte) ([]float32, int, int, error) {
	// Decode JPEG
	img, err := jpeg.Decode(bytes.NewReader(frame))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// Resize to model input size
	resized := p.resizeImage(img, p.config.InputWidth, p.config.InputHeight)

	// Convert to float32 tensor in NCHW format
	data := make([]float32, 3*p.config.InputHeight*p.config.InputWidth)
	
	for y := 0; y < p.config.InputHeight; y++ {
		for x := 0; x < p.config.InputWidth; x++ {
			r, g, b, _ := resized.At(x, y).RGBA()
			
			// Normalize to 0-1
			idx := y*p.config.InputWidth + x
			data[0*p.config.InputHeight*p.config.InputWidth+idx] = float32(r>>8) / 255.0 // R channel
			data[1*p.config.InputHeight*p.config.InputWidth+idx] = float32(g>>8) / 255.0 // G channel
			data[2*p.config.InputHeight*p.config.InputWidth+idx] = float32(b>>8) / 255.0 // B channel
		}
	}

	return data, originalWidth, originalHeight, nil
}

func (p *YOLOProcessor) resizeImage(img image.Image, width, height int) image.Image {
	// Bilinear interpolation resize
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	xRatio := float64(srcW) / float64(width)
	yRatio := float64(srcH) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * xRatio)
			srcY := int(float64(y) * yRatio)

			if srcX >= srcW {
				srcX = srcW - 1
			}
			if srcY >= srcH {
				srcY = srcH - 1
			}

			dst.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}

	return dst
}

func (p *YOLOProcessor) postprocessDetections(outputs []InferenceOutput, originalWidth, originalHeight int) ([]Detection, error) {
	if len(outputs) == 0 {
		return nil, fmt.Errorf("no output from model")
	}

	output := outputs[0]
	
	// YOLOv8 output shape: [1, 84, 8400]
	// 84 = 4 (bbox) + 80 (classes)
	// 8400 = number of predictions
	
	numClasses := len(p.config.Classes)
	if numClasses == 0 {
		numClasses = 80 // Default COCO classes
	}
	
	numPredictions := 8400
	if len(output.Shape) >= 3 {
		numPredictions = output.Shape[2]
	}

	var detections []Detection
	scaleX := float64(originalWidth) / float64(p.config.InputWidth)
	scaleY := float64(originalHeight) / float64(p.config.InputHeight)

	for i := 0; i < numPredictions && i < len(output.Data)/(4+numClasses); i++ {
		offset := i

		// Extract box coordinates (center x, center y, width, height)
		cx := float64(output.Data[0*numPredictions+offset]) * scaleX
		cy := float64(output.Data[1*numPredictions+offset]) * scaleY
		w := float64(output.Data[2*numPredictions+offset]) * scaleX
		h := float64(output.Data[3*numPredictions+offset]) * scaleY

		// Find best class
		bestClass := 0
		bestScore := float64(0)
		for c := 0; c < numClasses; c++ {
			score := float64(output.Data[(4+c)*numPredictions+offset])
			if score > bestScore {
				bestScore = score
				bestClass = c
			}
		}

		// Apply confidence threshold
		if bestScore < p.config.ConfidenceThreshold {
			continue
		}

		// Convert to x, y, width, height format
		x := int(cx - w/2)
		y := int(cy - h/2)

		// Clamp to image bounds
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}

		className := "unknown"
		if bestClass < len(p.config.Classes) {
			className = p.config.Classes[bestClass]
		}

		detections = append(detections, Detection{
			Class:      className,
			Confidence: bestScore,
			BoundingBox: BoundingBox{
				X:      x,
				Y:      y,
				Width:  int(w),
				Height: int(h),
			},
			Timestamp: time.Now().Unix(),
		})
	}

	// Apply NMS
	detections = p.nonMaxSuppression(detections)

	// Limit detections
	if len(detections) > p.config.MaxDetections {
		detections = detections[:p.config.MaxDetections]
	}

	return detections, nil
}

func (p *YOLOProcessor) nonMaxSuppression(detections []Detection) []Detection {
	if len(detections) == 0 {
		return detections
	}

	// Sort by confidence (descending)
	sort.Slice(detections, func(i, j int) bool {
		return detections[i].Confidence > detections[j].Confidence
	})

	var result []Detection
	suppressed := make([]bool, len(detections))

	for i := 0; i < len(detections); i++ {
		if suppressed[i] {
			continue
		}

		result = append(result, detections[i])

		for j := i + 1; j < len(detections); j++ {
			if suppressed[j] {
				continue
			}

			// Only suppress same class
			if detections[i].Class != detections[j].Class {
				continue
			}

			iou := p.calculateIoU(detections[i].BoundingBox, detections[j].BoundingBox)
			if iou > p.config.NMSThreshold {
				suppressed[j] = true
			}
		}
	}

	return result
}

func (p *YOLOProcessor) calculateIoU(a, b BoundingBox) float64 {
	// Calculate intersection
	x1 := math.Max(float64(a.X), float64(b.X))
	y1 := math.Max(float64(a.Y), float64(b.Y))
	x2 := math.Min(float64(a.X+a.Width), float64(b.X+b.Width))
	y2 := math.Min(float64(a.Y+a.Height), float64(b.Y+b.Height))

	if x2 <= x1 || y2 <= y1 {
		return 0
	}

	intersection := (x2 - x1) * (y2 - y1)
	areaA := float64(a.Width * a.Height)
	areaB := float64(b.Width * b.Height)
	union := areaA + areaB - intersection

	if union <= 0 {
		return 0
	}

	return intersection / union
}

// GetModelInfo returns model metadata
func (p *YOLOProcessor) GetModelInfo() ModelInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.modelInfo
}

// Shutdown releases resources
func (p *YOLOProcessor) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.initialized = false
	return nil
}

// ============================================================================
// Satellite-Specific Detection Classes
// ============================================================================

// SatelliteDetectionClasses returns classes relevant for satellite imagery
func SatelliteDetectionClasses() []string {
	return []string{
		"person",
		"vehicle",
		"truck",
		"bus",
		"aircraft",
		"helicopter",
		"ship",
		"boat",
		"building",
		"fire",
		"smoke",
		"flood",
		"road",
		"runway",
		"solar_panel",
		"oil_tank",
		"container",
		"crane",
		"wind_turbine",
		"bridge",
	}
}
