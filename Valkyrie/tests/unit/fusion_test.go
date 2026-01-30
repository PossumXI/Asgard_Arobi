package unit

import (
	"context"
	"testing"
	"time"

	"github.com/PossumXI/Asgard/Valkyrie/internal/fusion"
	"gonum.org/v1/gonum/mat"
)

func TestEKF_Creation(t *testing.T) {
	config := fusion.FusionConfig{
		UpdateRate:       100.0,
		SensorWeights:    make(map[fusion.SensorType]float64),
		OutlierThreshold: 3.0,
		MinSensors:       2,
		EnableAdaptive:   true,
	}
	config.SensorWeights[fusion.SensorGPS] = 1.0

	ekf := fusion.NewEKF(config)
	if ekf == nil {
		t.Fatal("EKF creation failed")
	}

	state := ekf.GetState()
	if state == nil {
		t.Fatal("Initial state is nil")
	}
}

func TestEKF_Prediction(t *testing.T) {
	config := fusion.FusionConfig{
		UpdateRate:       10.0,
		SensorWeights:    make(map[fusion.SensorType]float64),
		OutlierThreshold: 3.0,
		MinSensors:       1,
	}

	ekf := fusion.NewEKF(config)
	ctx := context.Background()

	// Perform prediction
	err := ekf.Predict(ctx)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	state := ekf.GetState()
	if state == nil {
		t.Fatal("State is nil after prediction")
	}
}

func TestEKF_GPSUpdate(t *testing.T) {
	config := fusion.FusionConfig{
		UpdateRate:       10.0,
		SensorWeights:    make(map[fusion.SensorType]float64),
		OutlierThreshold: 3.0,
		MinSensors:       1,
	}
	config.SensorWeights[fusion.SensorGPS] = 1.0

	ekf := fusion.NewEKF(config)

	// Simulate GPS reading
	gpsData := mat.NewVecDense(3, []float64{100.0, 200.0, 50.0})
	gpsCov := mat.NewSymDense(3, []float64{
		10.0, 0, 0,
		0, 10.0, 0,
		0, 0, 5.0,
	})

	reading := fusion.SensorReading{
		Type:       fusion.SensorGPS,
		Timestamp:  time.Now(),
		Data:       gpsData,
		Covariance: gpsCov,
		Quality:    0.95,
	}

	err := ekf.Update(reading)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	state := ekf.GetState()
	t.Logf("State after GPS update: Position=%.2f, %.2f, %.2f",
		state.Position[0], state.Position[1], state.Position[2])

	// After update, position should move toward GPS measurement
	if state.Position[0] == 0 && state.Position[1] == 0 && state.Position[2] == 0 {
		t.Error("Position did not update after GPS measurement")
	}
}

func TestEKF_INSUpdate(t *testing.T) {
	config := fusion.FusionConfig{
		UpdateRate:       10.0,
		SensorWeights:    make(map[fusion.SensorType]float64),
		OutlierThreshold: 3.0,
		MinSensors:       1,
	}
	config.SensorWeights[fusion.SensorINS] = 0.9

	ekf := fusion.NewEKF(config)

	// Simulate INS reading (acceleration and angular rates)
	insData := mat.NewVecDense(6, []float64{
		0.1, 0.2, 9.81, // Acceleration (with gravity)
		0.01, 0.02, 0.0, // Angular rates
	})
	insCov := mat.NewSymDense(6, nil)
	for i := 0; i < 6; i++ {
		insCov.SetSym(i, i, 0.01)
	}

	reading := fusion.SensorReading{
		Type:       fusion.SensorINS,
		Timestamp:  time.Now(),
		Data:       insData,
		Covariance: insCov,
		Quality:    0.9,
	}

	err := ekf.Update(reading)
	if err != nil {
		t.Fatalf("INS Update failed: %v", err)
	}

	state := ekf.GetState()
	t.Logf("State after INS update: Acceleration=%.2f, %.2f, %.2f",
		state.Acceleration[0], state.Acceleration[1], state.Acceleration[2])
}

func TestEKF_MultiSensorFusion(t *testing.T) {
	config := fusion.FusionConfig{
		UpdateRate:       10.0,
		SensorWeights:    make(map[fusion.SensorType]float64),
		OutlierThreshold: 3.0,
		MinSensors:       2,
		EnableAdaptive:   true,
	}
	config.SensorWeights[fusion.SensorGPS] = 1.0
	config.SensorWeights[fusion.SensorINS] = 0.9

	ekf := fusion.NewEKF(config)
	ctx := context.Background()

	// GPS measurement
	gpsData := mat.NewVecDense(3, []float64{100.0, 100.0, 100.0})
	gpsCov := mat.NewSymDense(3, []float64{5, 0, 0, 0, 5, 0, 0, 0, 3})

	// Predict
	ekf.Predict(ctx)

	// Update with GPS
	ekf.Update(fusion.SensorReading{
		Type:       fusion.SensorGPS,
		Timestamp:  time.Now(),
		Data:       gpsData,
		Covariance: gpsCov,
		Quality:    0.95,
	})

	// INS measurement
	insData := mat.NewVecDense(6, []float64{0.5, 0.5, 10.0, 0.0, 0.0, 0.1})
	insCov := mat.NewSymDense(6, nil)
	for i := 0; i < 6; i++ {
		insCov.SetSym(i, i, 0.01)
	}

	// Predict again
	ekf.Predict(ctx)

	// Update with INS
	ekf.Update(fusion.SensorReading{
		Type:       fusion.SensorINS,
		Timestamp:  time.Now(),
		Data:       insData,
		Covariance: insCov,
		Quality:    0.9,
	})

	state := ekf.GetState()
	t.Logf("Fused state: Position=%.2f,%.2f,%.2f Confidence=%.3f",
		state.Position[0], state.Position[1], state.Position[2], state.Confidence)

	if state.Confidence < 0 || state.Confidence > 1 {
		t.Error("Confidence out of range")
	}
}

func TestEKF_GetUpdateCount(t *testing.T) {
	config := fusion.FusionConfig{
		UpdateRate:       10.0,
		SensorWeights:    make(map[fusion.SensorType]float64),
		OutlierThreshold: 3.0,
		MinSensors:       1,
	}

	ekf := fusion.NewEKF(config)

	initialCount := ekf.GetUpdateCount()
	if initialCount != 0 {
		t.Errorf("Initial update count should be 0, got %d", initialCount)
	}

	// Perform an update
	gpsData := mat.NewVecDense(3, []float64{10, 20, 30})
	gpsCov := mat.NewSymDense(3, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1})
	ekf.Update(fusion.SensorReading{
		Type:       fusion.SensorGPS,
		Timestamp:  time.Now(),
		Data:       gpsData,
		Covariance: gpsCov,
		Quality:    0.9,
	})

	count := ekf.GetUpdateCount()
	if count != 1 {
		t.Errorf("Update count should be 1, got %d", count)
	}
}
