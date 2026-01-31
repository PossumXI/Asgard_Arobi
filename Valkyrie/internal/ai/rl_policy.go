package ai

import (
	"math"
	"math/rand"

	"github.com/PossumXI/Asgard/Valkyrie/internal/fusion"
)

// extractFeatures extracts state features for function approximation
func (rl *ReinforcementLearningPolicy) extractFeatures(
	state *fusion.FusionState,
	threats []*Threat,
	weather *WeatherConditions,
	posError [3]float64,
) []float64 {
	features := make([]float64, 20) // 20-dimensional feature vector

	if state == nil {
		return features
	}

	// Position error features (normalized)
	features[0] = posError[0] / 1000.0 // X error (km)
	features[1] = posError[1] / 1000.0 // Y error (km)
	features[2] = posError[2] / 100.0  // Z error (100m)

	// Velocity features (normalized)
	features[3] = state.Velocity[0] / 100.0 // Vx (m/s -> normalized)
	features[4] = state.Velocity[1] / 100.0 // Vy
	features[5] = state.Velocity[2] / 50.0  // Vz

	// Attitude features
	features[6] = state.Attitude[0] / math.Pi       // Roll (normalized to [-1, 1])
	features[7] = state.Attitude[1] / math.Pi       // Pitch
	features[8] = state.Attitude[2] / (2 * math.Pi) // Yaw (normalized)

	// Threat features
	if len(threats) > 0 {
		minDist := math.MaxFloat64
		for _, t := range threats {
			if t.Distance < minDist {
				minDist = t.Distance
			}
		}
		features[9] = minDist / 10000.0             // Nearest threat distance (normalized)
		features[10] = float64(len(threats)) / 10.0 // Threat count (normalized)
	} else {
		features[9] = 1.0 // No threats
		features[10] = 0.0
	}

	// Weather features
	if weather != nil {
		features[11] = weather.WindSpeed / 50.0     // Wind speed (normalized)
		features[12] = weather.Visibility / 10000.0 // Visibility
		features[13] = weather.Turbulence           // Already normalized
	} else {
		features[11] = 0.1
		features[12] = 1.0
		features[13] = 0.0
	}

	// Energy state features
	features[14] = state.Position[2] / 10000.0 // Altitude (normalized)

	// Remaining features for future expansion
	for i := 15; i < 20; i++ {
		features[i] = 0.0
	}

	return features
}

// computeQValues computes Q-values for action space using linear function approximation
func (rl *ReinforcementLearningPolicy) computeQValues(features []float64) map[string]float64 {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	qValues := make(map[string]float64)

	// Action space: roll_left, roll_right, pitch_up, pitch_down, yaw_left, yaw_right, throttle_up, throttle_down
	actions := []string{"roll_left", "roll_right", "pitch_up", "pitch_down",
		"yaw_left", "yaw_right", "throttle_up", "throttle_down", "maintain"}

	for i, action := range actions {
		// Linear Q-value: Q(s,a) = sum(w_i * phi_i(s))
		qValue := 0.0
		weightOffset := i * len(features)

		for j, feature := range features {
			if weightOffset+j < len(rl.weights) {
				qValue += rl.weights[weightOffset+j] * feature
			}
		}

		qValues[action] = qValue
	}

	return qValues
}

// exploreAction generates a random action within safety bounds
func (rl *ReinforcementLearningPolicy) exploreAction(state *fusion.FusionState) *RLAction {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	maxRoll := 0.785    // 45 degrees
	maxPitch := 0.524   // 30 degrees
	maxYawRate := 0.349 // 20 deg/s

	return &RLAction{
		RollAngle:    (rand.Float64() - 0.5) * 2 * maxRoll,
		PitchAngle:   (rand.Float64() - 0.5) * 2 * maxPitch,
		YawRate:      (rand.Float64() - 0.5) * 2 * maxYawRate,
		Throttle:     0.5 + (rand.Float64()-0.5)*0.4, // 0.3 to 0.7
		AutoThrottle: true,
	}
}

// exploitAction selects the best action based on Q-values
func (rl *ReinforcementLearningPolicy) exploitAction(
	qValues map[string]float64,
	state *fusion.FusionState,
	threats []*Threat,
	weather *WeatherConditions,
) *RLAction {
	// Find action with highest Q-value
	bestAction := "maintain"
	bestQ := qValues["maintain"]

	for action, qValue := range qValues {
		if qValue > bestQ {
			bestQ = qValue
			bestAction = action
		}
	}

	// Convert action string to RLAction
	action := &RLAction{
		Throttle:     0.7,
		AutoThrottle: true,
	}

	switch bestAction {
	case "roll_left":
		action.RollAngle = -0.3
	case "roll_right":
		action.RollAngle = 0.3
	case "pitch_up":
		action.PitchAngle = 0.2
	case "pitch_down":
		action.PitchAngle = -0.2
	case "yaw_left":
		action.YawRate = -0.1
	case "yaw_right":
		action.YawRate = 0.1
	case "throttle_up":
		action.Throttle = 0.9
	case "throttle_down":
		action.Throttle = 0.5
	case "maintain":
		// Maintain current attitude
		action.RollAngle = 0.0
		action.PitchAngle = 0.0
		action.YawRate = 0.0
	}

	return action
}

// applySafetyConstraints applies safety constraints to action
func (rl *ReinforcementLearningPolicy) applySafetyConstraints(
	action *RLAction,
	state *fusion.FusionState,
	threats []*Threat,
	weather *WeatherConditions,
) *RLAction {
	maxRoll := 0.785    // 45 degrees
	maxPitch := 0.524   // 30 degrees
	maxYawRate := 0.349 // 20 deg/s

	// Clamp roll
	if action.RollAngle > maxRoll {
		action.RollAngle = maxRoll
	} else if action.RollAngle < -maxRoll {
		action.RollAngle = -maxRoll
	}

	// Clamp pitch
	if action.PitchAngle > maxPitch {
		action.PitchAngle = maxPitch
	} else if action.PitchAngle < -maxPitch {
		action.PitchAngle = -maxPitch
	}

	// Clamp yaw rate
	if action.YawRate > maxYawRate {
		action.YawRate = maxYawRate
	} else if action.YawRate < -maxYawRate {
		action.YawRate = -maxYawRate
	}

	// Clamp throttle
	if action.Throttle < 0.0 {
		action.Throttle = 0.0
	} else if action.Throttle > 1.0 {
		action.Throttle = 1.0
	}

	// Weather-based adjustments
	if weather != nil {
		if weather.Turbulence > 0.5 {
			// Reduce control authority in turbulence
			action.RollAngle *= 0.7
			action.PitchAngle *= 0.7
		}
		if weather.WindSpeed > 20.0 {
			// Increase throttle to maintain airspeed in high wind
			action.Throttle = math.Min(1.0, action.Throttle*1.2)
		}
	}

	// Threat avoidance override
	if len(threats) > 0 {
		minDist := math.MaxFloat64
		for _, t := range threats {
			if t.Distance < minDist {
				minDist = t.Distance
			}
		}
		if minDist < 1000 {
			// Emergency evasion: max control authority
			action.RollAngle = math.Copysign(maxRoll*0.9, action.RollAngle)
			action.Throttle = 1.0
		}
	}

	return action
}
