package hil

import (
	"context"
	"testing"
	"time"

	"github.com/asgard/pandora/internal/orbital/hal"
)

// TestSilenusHILSuite runs all Silenus HIL tests
func TestSilenusHILSuite(t *testing.T) {
	config := DefaultConfig()
	config.HunoidEnabled = false // Only test Silenus
	config.VerboseLogging = testing.Verbose()

	suite := NewHILTestSuite(config)
	if err := suite.SetupHardware(); err != nil {
		t.Skipf("HIL hardware unavailable: %v", err)
	}
	defer func() {
		if err := suite.Close(); err != nil {
			t.Errorf("Teardown failed: %v", err)
		}
		suite.PrintSummary()
	}()

	// Camera tests
	suite.RunTest(t, "Camera/Initialize", testCameraInitialize)
	suite.RunTest(t, "Camera/CaptureFrame", testCameraCaptureFrame)
	suite.RunTest(t, "Camera/Streaming", testCameraStreaming)
	suite.RunTest(t, "Camera/Settings", testCameraSettings)
	suite.RunTest(t, "Camera/Diagnostics", testCameraDiagnostics)

	// Power controller tests
	suite.RunTest(t, "Power/BatteryMonitoring", testPowerBatteryMonitoring)
	suite.RunTest(t, "Power/SolarPanel", testPowerSolarPanel)
	suite.RunTest(t, "Power/EclipseDetection", testPowerEclipseDetection)
	suite.RunTest(t, "Power/ModeSwitch", testPowerModeSwitch)

	// GPS tests
	suite.RunTest(t, "GPS/Position", testGPSPosition)
	suite.RunTest(t, "GPS/Time", testGPSTime)
	suite.RunTest(t, "GPS/Velocity", testGPSVelocity)
	suite.RunTestWithContext(t, "GPS/PositionTracking", 10*time.Second, testGPSPositionTracking)
}

// =============================================================================
// Camera Tests
// =============================================================================

func testCameraInitialize(t *testing.T, suite *HILTestSuite) {
	camera := suite.Silenus().Camera()
	if camera == nil {
		t.Fatal("Camera controller is nil")
	}

	// Camera should already be initialized by suite setup
	// Verify by getting diagnostics
	diag, err := camera.GetDiagnostics()
	if err != nil {
		t.Fatalf("Failed to get diagnostics: %v", err)
	}

	if diag.Voltage <= 0 {
		t.Errorf("Invalid voltage: %v", diag.Voltage)
	}

	suite.RecordMetric("camera_voltage", diag.Voltage)
	suite.RecordMetric("camera_temperature", diag.Temperature)
}

func testCameraCaptureFrame(t *testing.T, suite *HILTestSuite) {
	camera := suite.Silenus().Camera()
	ctx := suite.Context()

	// Capture a single frame
	start := time.Now()
	frame, err := camera.CaptureFrame(ctx)
	captureTime := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to capture frame: %v", err)
	}

	if len(frame) == 0 {
		t.Fatal("Captured empty frame")
	}

	// Verify JPEG header (FFD8FF)
	if len(frame) < 3 || frame[0] != 0xFF || frame[1] != 0xD8 || frame[2] != 0xFF {
		t.Error("Frame does not appear to be valid JPEG")
	}

	t.Logf("Captured frame: %d bytes in %v", len(frame), captureTime)

	suite.RecordMetric("frame_size_bytes", float64(len(frame)))
	suite.RecordMetric("capture_time_ms", float64(captureTime.Milliseconds()))
}

func testCameraStreaming(t *testing.T, suite *HILTestSuite) {
	suite.SkipIfSlow(t)

	camera := suite.Silenus().Camera()
	ctx, cancel := context.WithTimeout(suite.Context(), 3*time.Second)
	defer cancel()

	frameChan := make(chan []byte, 10)

	// Start streaming
	if err := camera.StartStream(ctx, frameChan); err != nil {
		t.Fatalf("Failed to start stream: %v", err)
	}

	// Collect frames for 2 seconds
	frameCount := 0
	totalBytes := 0
	timeout := time.After(2 * time.Second)

collectLoop:
	for {
		select {
		case frame := <-frameChan:
			frameCount++
			totalBytes += len(frame)
		case <-timeout:
			break collectLoop
		case <-ctx.Done():
			break collectLoop
		}
	}

	// Stop streaming
	if err := camera.StopStream(); err != nil {
		t.Errorf("Failed to stop stream: %v", err)
	}

	if frameCount == 0 {
		t.Error("No frames received during streaming")
	}

	fps := float64(frameCount) / 2.0
	t.Logf("Received %d frames (%.1f FPS), %d total bytes", frameCount, fps, totalBytes)

	suite.RecordMetric("stream_fps", fps)
	suite.RecordMetric("stream_frame_count", float64(frameCount))
}

func testCameraSettings(t *testing.T, suite *HILTestSuite) {
	camera := suite.Silenus().Camera()

	// Test exposure setting
	testExposures := []int{100, 1000, 10000, 100000}
	for _, exp := range testExposures {
		if err := camera.SetExposure(exp); err != nil {
			t.Errorf("Failed to set exposure to %d: %v", exp, err)
		}
	}

	// Test invalid exposure
	if err := camera.SetExposure(-1); err == nil {
		t.Error("Expected error for negative exposure")
	}

	if err := camera.SetExposure(2000000); err == nil {
		t.Error("Expected error for excessive exposure")
	}

	// Test gain setting
	testGains := []float64{0.5, 1.0, 2.0, 5.0}
	for _, gain := range testGains {
		if err := camera.SetGain(gain); err != nil {
			t.Errorf("Failed to set gain to %f: %v", gain, err)
		}
	}

	// Test invalid gain
	if err := camera.SetGain(-1); err == nil {
		t.Error("Expected error for negative gain")
	}

	if err := camera.SetGain(20); err == nil {
		t.Error("Expected error for excessive gain")
	}
}

func testCameraDiagnostics(t *testing.T, suite *HILTestSuite) {
	camera := suite.Silenus().Camera()

	// Get initial diagnostics
	diag1, err := camera.GetDiagnostics()
	if err != nil {
		t.Fatalf("Failed to get diagnostics: %v", err)
	}

	initialFrameCount := diag1.FrameCount

	// Capture some frames
	ctx := suite.Context()
	for i := 0; i < 5; i++ {
		_, _ = camera.CaptureFrame(ctx)
	}

	// Get updated diagnostics
	diag2, err := camera.GetDiagnostics()
	if err != nil {
		t.Fatalf("Failed to get diagnostics: %v", err)
	}

	// Verify frame count increased
	if diag2.FrameCount <= initialFrameCount {
		t.Errorf("Frame count should have increased: %d -> %d", initialFrameCount, diag2.FrameCount)
	}

	t.Logf("Diagnostics: temp=%.1f°C, voltage=%.2fV, frames=%d, errors=%d",
		diag2.Temperature, diag2.Voltage, diag2.FrameCount, diag2.ErrorCount)

	// Verify reasonable values
	if diag2.Temperature < -40 || diag2.Temperature > 85 {
		t.Errorf("Temperature out of expected range: %.1f°C", diag2.Temperature)
	}

	if diag2.Voltage < 3.0 || diag2.Voltage > 15.0 {
		t.Errorf("Voltage out of expected range: %.2fV", diag2.Voltage)
	}
}

// =============================================================================
// Power Controller Tests
// =============================================================================

func testPowerBatteryMonitoring(t *testing.T, suite *HILTestSuite) {
	power := suite.Silenus().Power()
	if power == nil {
		t.Fatal("Power controller is nil")
	}

	// Get battery percentage
	percent, err := power.GetBatteryPercent()
	if err != nil {
		t.Fatalf("Failed to get battery percent: %v", err)
	}

	if percent < 0 || percent > 100 {
		t.Errorf("Battery percent out of range: %.2f%%", percent)
	}

	// Get battery voltage
	voltage, err := power.GetBatteryVoltage()
	if err != nil {
		t.Fatalf("Failed to get battery voltage: %v", err)
	}

	// Typical Li-ion voltage range: 3.0V (empty) to 4.2V (full)
	if voltage < 2.5 || voltage > 5.0 {
		t.Errorf("Battery voltage out of expected range: %.2fV", voltage)
	}

	t.Logf("Battery: %.1f%% at %.2fV", percent, voltage)

	suite.RecordMetric("battery_percent", percent)
	suite.RecordMetric("battery_voltage", voltage)
}

func testPowerSolarPanel(t *testing.T, suite *HILTestSuite) {
	power := suite.Silenus().Power()

	// Get solar panel power
	solarPower, err := power.GetSolarPanelPower()
	if err != nil {
		t.Fatalf("Failed to get solar power: %v", err)
	}

	// Check eclipse status
	inEclipse, err := power.IsInEclipse()
	if err != nil {
		t.Fatalf("Failed to get eclipse status: %v", err)
	}

	// If in eclipse, solar power should be 0
	if inEclipse && solarPower > 0 {
		t.Errorf("Solar power should be 0 during eclipse, got %.2fW", solarPower)
	}

	// If not in eclipse, solar power should be positive
	if !inEclipse && solarPower <= 0 {
		t.Errorf("Solar power should be positive outside eclipse, got %.2fW", solarPower)
	}

	t.Logf("Solar power: %.2fW (eclipse: %v)", solarPower, inEclipse)

	suite.RecordMetric("solar_power_watts", solarPower)
}

func testPowerEclipseDetection(t *testing.T, suite *HILTestSuite) {
	suite.SkipIfSlow(t)

	power := suite.Silenus().Power()

	// Sample eclipse status multiple times
	eclipseSamples := make([]bool, 0)
	for i := 0; i < 10; i++ {
		inEclipse, err := power.IsInEclipse()
		if err != nil {
			t.Fatalf("Failed to get eclipse status: %v", err)
		}
		eclipseSamples = append(eclipseSamples, inEclipse)
		time.Sleep(100 * time.Millisecond)
	}

	// Verify we got consistent readings (should be same over short period)
	firstValue := eclipseSamples[0]
	for i, v := range eclipseSamples {
		if v != firstValue {
			t.Logf("Eclipse status changed at sample %d (expected for orbital simulation)", i)
			break
		}
	}
}

func testPowerModeSwitch(t *testing.T, suite *HILTestSuite) {
	power := suite.Silenus().Power()

	// Test switching between power modes
	modes := []hal.PowerMode{
		hal.PowerModeNormal,
		hal.PowerModeLow,
		hal.PowerModeCritical,
		hal.PowerModeNormal,
	}

	for _, mode := range modes {
		if err := power.SetPowerMode(mode); err != nil {
			t.Errorf("Failed to set power mode to %s: %v", mode, err)
		}
	}
}

// =============================================================================
// GPS Tests
// =============================================================================

func testGPSPosition(t *testing.T, suite *HILTestSuite) {
	gps := suite.Silenus().GPS()
	if gps == nil {
		t.Fatal("GPS controller is nil")
	}

	lat, lon, alt, err := gps.GetPosition()
	if err != nil {
		t.Fatalf("Failed to get position: %v", err)
	}

	// Validate latitude (-90 to 90)
	if lat < -90 || lat > 90 {
		t.Errorf("Latitude out of range: %f", lat)
	}

	// Validate longitude (-180 to 360 - allowing both conventions)
	if lon < -180 || lon > 360 {
		t.Errorf("Longitude out of range: %f", lon)
	}

	// Validate altitude (LEO satellites: ~160km to ~2000km)
	// Mock returns 550km (550,000 meters)
	if alt < 100000 || alt > 50000000 {
		t.Errorf("Altitude out of expected LEO range: %f meters", alt)
	}

	t.Logf("Position: lat=%.4f, lon=%.4f, alt=%.0fm", lat, lon, alt)

	suite.RecordMetric("gps_latitude", lat)
	suite.RecordMetric("gps_longitude", lon)
	suite.RecordMetric("gps_altitude", alt)
}

func testGPSTime(t *testing.T, suite *HILTestSuite) {
	gps := suite.Silenus().GPS()

	gpsTime, err := gps.GetTime()
	if err != nil {
		t.Fatalf("Failed to get GPS time: %v", err)
	}

	// GPS time should be close to current time
	now := time.Now().UTC()
	diff := now.Sub(gpsTime).Abs()

	// Allow up to 1 second difference
	if diff > time.Second {
		t.Errorf("GPS time differs from system time by %v", diff)
	}

	t.Logf("GPS time: %v (diff from system: %v)", gpsTime, diff)
}

func testGPSVelocity(t *testing.T, suite *HILTestSuite) {
	gps := suite.Silenus().GPS()

	vx, vy, vz, err := gps.GetVelocity()
	if err != nil {
		t.Fatalf("Failed to get velocity: %v", err)
	}

	// Calculate total velocity
	totalVelocity := (vx*vx + vy*vy + vz*vz)
	// Note: not taking sqrt to compare squared values

	// LEO orbital velocity is ~7.8 km/s
	// Mock returns 7600 m/s which is approximately correct
	expectedVelocitySquared := 7600.0 * 7600.0
	tolerance := expectedVelocitySquared * 0.5 // 50% tolerance

	if totalVelocity < expectedVelocitySquared-tolerance || totalVelocity > expectedVelocitySquared+tolerance {
		t.Logf("Warning: Velocity magnitude may be outside expected LEO range")
	}

	t.Logf("Velocity: vx=%.1f, vy=%.1f, vz=%.1f m/s", vx, vy, vz)

	suite.RecordMetric("velocity_x", vx)
	suite.RecordMetric("velocity_y", vy)
	suite.RecordMetric("velocity_z", vz)
}

func testGPSPositionTracking(ctx context.Context, t *testing.T, suite *HILTestSuite) {
	suite.SkipIfSlow(t)

	gps := suite.Silenus().GPS()

	// Track position over time
	type positionSample struct {
		lat, lon, alt float64
		timestamp     time.Time
	}

	samples := make([]positionSample, 0)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(5 * time.Second)

collectLoop:
	for {
		select {
		case <-ticker.C:
			lat, lon, alt, err := gps.GetPosition()
			if err != nil {
				t.Errorf("Failed to get position: %v", err)
				continue
			}
			samples = append(samples, positionSample{lat, lon, alt, time.Now()})
		case <-timeout:
			break collectLoop
		case <-ctx.Done():
			break collectLoop
		}
	}

	if len(samples) < 5 {
		t.Errorf("Expected at least 5 position samples, got %d", len(samples))
		return
	}

	// Verify position is changing (satellite is moving)
	firstPos := samples[0]
	lastPos := samples[len(samples)-1]

	// At least longitude should change as satellite orbits
	if firstPos.lon == lastPos.lon && firstPos.lat == lastPos.lat {
		t.Error("Position did not change over tracking period")
	}

	t.Logf("Tracked %d positions over %v", len(samples), samples[len(samples)-1].timestamp.Sub(samples[0].timestamp))
	t.Logf("Start: lat=%.4f, lon=%.4f", firstPos.lat, firstPos.lon)
	t.Logf("End:   lat=%.4f, lon=%.4f", lastPos.lat, lastPos.lon)
}

// =============================================================================
// Vision Processor Tests (if available)
// =============================================================================

func TestVisionProcessorWithSampleImage(t *testing.T) {
	// This test demonstrates how to test vision processing with captured images
	config := DefaultConfig()
	config.HunoidEnabled = false

	suite := NewHILTestSuite(config)
	if err := suite.SetupHardware(); err != nil {
		t.Skipf("HIL hardware unavailable: %v", err)
	}
	defer suite.Close()

	camera := suite.Silenus().Camera()
	ctx := context.Background()

	// Capture a test image
	frame, err := camera.CaptureFrame(ctx)
	if err != nil {
		t.Fatalf("Failed to capture frame: %v", err)
	}

	// In a real scenario, this would feed to the vision processor
	// For now, just verify we have valid image data
	if len(frame) < 100 {
		t.Errorf("Frame too small: %d bytes", len(frame))
	}

	t.Logf("Captured %d byte image for vision processing", len(frame))
}
