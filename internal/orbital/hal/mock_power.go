package hal

import (
	"math"
	"sync"
	"time"
)

// MockPowerController simulates satellite power system
type MockPowerController struct {
	mu            sync.Mutex
	batteryPercent float64
	solarPower    float64
	inEclipse     bool
	mode          PowerMode
	startTime     time.Time
}

func NewMockPowerController() *MockPowerController {
	return &MockPowerController{
		batteryPercent: 85.0,
		solarPower:    50.0,
		inEclipse:     false,
		mode:          PowerModeNormal,
		startTime:     time.Now(),
	}
}

func (p *MockPowerController) GetBatteryPercent() (float64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Simulate battery drain/charge based on eclipse
	p.simulatePowerDynamics()

	return p.batteryPercent, nil
}

func (p *MockPowerController) GetBatteryVoltage() (float64, error) {
	batteryPercent, _ := p.GetBatteryPercent()
	// Typical Li-ion voltage curve
	voltage := 3.0 + (batteryPercent / 100.0 * 1.2)
	return voltage, nil
}

func (p *MockPowerController) GetSolarPanelPower() (float64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.inEclipse {
		return 0.0, nil
	}

	return p.solarPower, nil
}

func (p *MockPowerController) IsInEclipse() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Simulate orbital eclipse every 90 minutes (LEO orbit)
	elapsed := time.Since(p.startTime).Minutes()
	orbitalPhase := math.Mod(elapsed, 90.0)

	// Eclipse for ~30 minutes per orbit
	p.inEclipse = orbitalPhase > 60.0

	return p.inEclipse, nil
}

func (p *MockPowerController) SetPowerMode(mode PowerMode) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mode = mode
	return nil
}

func (p *MockPowerController) simulatePowerDynamics() {
	if p.inEclipse {
		// Drain battery
		p.batteryPercent -= 0.1
		if p.batteryPercent < 0 {
			p.batteryPercent = 0
		}
	} else {
		// Charge battery
		p.batteryPercent += 0.2
		if p.batteryPercent > 100 {
			p.batteryPercent = 100
		}
	}
}
