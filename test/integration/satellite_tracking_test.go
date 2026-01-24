package integration_test

import (
	"testing"
	"time"

	"github.com/asgard/pandora/internal/platform/satellite"
)

func TestSatelliteClientConfigCreation(t *testing.T) {
	config := satellite.DefaultConfig()

	if config.CacheTTL.Minutes() < 30 {
		t.Error("cache TTL should be at least 30 minutes")
	}
}

func TestSatelliteClientConfigWithN2YOKey(t *testing.T) {
	config := satellite.DefaultConfig()
	config.N2YOAPIKey = "test-api-key-12345"

	if config.N2YOAPIKey != "test-api-key-12345" {
		t.Error("N2YO API key not set correctly")
	}
}

func TestCommonNORADIDs(t *testing.T) {
	// Verify common satellite IDs are defined and positive
	ids := map[string]int{
		"ISS":      satellite.NoradISS,
		"Hubble":   satellite.NoradHubble,
		"Landsat8": satellite.NoradLandsat8,
		"Terra":    satellite.NoradTerra,
		"Aqua":     satellite.NoradAqua,
		"NOAA19":   satellite.NoradNOAA19,
	}

	for name, id := range ids {
		if id <= 0 {
			t.Errorf("%s NORAD ID should be positive, got %d", name, id)
		}
		t.Logf("%s: NORAD %d", name, id)
	}
}

func TestTLEParsing(t *testing.T) {
	// Sample ISS TLE lines
	line1 := "1 25544U 98067A   26024.50000000  .00023329  00000+0  42269-3 0  9992"
	line2 := "2 25544  51.6331 308.6863 0007748  41.1873 318.9699 15.49488068548921"

	elements, err := satellite.ParseTLE(line1, line2)
	if err != nil {
		t.Fatalf("failed to parse TLE: %v", err)
	}

	if elements == nil {
		t.Fatal("elements should not be nil")
	}

	// Verify ISS inclination is approximately 51.6 degrees
	if elements.Inclination < 51 || elements.Inclination > 52 {
		t.Errorf("ISS inclination should be ~51.6°, got %.2f°", elements.Inclination)
	}

	// Verify mean motion is approximately 15.5 revs/day (ISS)
	if elements.MeanMotion < 15 || elements.MeanMotion > 16 {
		t.Errorf("ISS mean motion should be ~15.5 rev/day, got %.2f", elements.MeanMotion)
	}

	t.Logf("Parsed TLE: inclination=%.2f°, mean_motion=%.2f rev/day", elements.Inclination, elements.MeanMotion)
}

func TestPropagatorCreation(t *testing.T) {
	// Create TLE
	tle := &satellite.TLE{
		SatelliteID: 25544,
		Name:        "ISS (ZARYA)",
		Line1:       "1 25544U 98067A   26024.50000000  .00023329  00000+0  42269-3 0  9992",
		Line2:       "2 25544  51.6331 308.6863 0007748  41.1873 318.9699 15.49488068548921",
		Epoch:       time.Now(),
		RetrievedAt: time.Now(),
		Source:      "test",
	}

	propagator, err := satellite.NewPropagator(tle)
	if err != nil {
		t.Fatalf("failed to create propagator: %v", err)
	}

	if propagator == nil {
		t.Fatal("propagator should not be nil")
	}
}

func TestPropagatorPosition(t *testing.T) {
	tle := &satellite.TLE{
		SatelliteID: 25544,
		Name:        "ISS (ZARYA)",
		Line1:       "1 25544U 98067A   26024.50000000  .00023329  00000+0  42269-3 0  9992",
		Line2:       "2 25544  51.6331 308.6863 0007748  41.1873 318.9699 15.49488068548921",
		Epoch:       time.Now(),
		RetrievedAt: time.Now(),
		Source:      "test",
	}

	propagator, err := satellite.NewPropagator(tle)
	if err != nil {
		t.Fatalf("failed to create propagator: %v", err)
	}

	// Propagate to current time
	lat, lon, alt := propagator.Propagate(time.Now())

	// ISS should be at approximately 400-420 km altitude
	if alt < 350 || alt > 500 {
		t.Errorf("ISS altitude %f km seems out of range (expected 350-500 km)", alt)
	}

	// ISS latitude should be within inclination (51.6°)
	if lat < -52 || lat > 52 {
		t.Errorf("ISS latitude %f exceeds inclination limit", lat)
	}

	// Longitude should be valid
	if lon < -180 || lon > 180 {
		t.Errorf("longitude %f out of valid range", lon)
	}

	t.Logf("ISS position: lat=%.4f°, lon=%.4f°, alt=%.2f km", lat, lon, alt)
}

func TestPropagatorRange(t *testing.T) {
	tle := &satellite.TLE{
		SatelliteID: 25544,
		Name:        "ISS (ZARYA)",
		Line1:       "1 25544U 98067A   26024.50000000  .00023329  00000+0  42269-3 0  9992",
		Line2:       "2 25544  51.6331 308.6863 0007748  41.1873 318.9699 15.49488068548921",
		Epoch:       time.Now(),
		RetrievedAt: time.Now(),
		Source:      "test",
	}

	propagator, err := satellite.NewPropagator(tle)
	if err != nil {
		t.Fatalf("failed to create propagator: %v", err)
	}

	// Generate 10-minute track
	positions := propagator.PropagateRange(time.Now(), 10*time.Minute, time.Minute)

	// Verify we got a reasonable number of positions
	if len(positions) < 10 || len(positions) > 12 {
		t.Errorf("expected ~10-11 positions, got %d", len(positions))
	}

	// Verify all positions are valid
	for i, pos := range positions {
		if pos.Latitude < -90 || pos.Latitude > 90 {
			t.Errorf("point %d: invalid latitude %f", i, pos.Latitude)
		}
		if pos.Longitude < -180 || pos.Longitude > 180 {
			t.Errorf("point %d: invalid longitude %f", i, pos.Longitude)
		}
		if pos.Altitude < 0 {
			t.Errorf("point %d: negative altitude %f", i, pos.Altitude)
		}
	}
}

func TestSatelliteClientCreation(t *testing.T) {
	config := satellite.DefaultConfig()
	client := satellite.NewClient(config)

	if client == nil {
		t.Fatal("client should not be nil")
	}
}
