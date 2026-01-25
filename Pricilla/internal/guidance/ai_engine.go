package guidance

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ================================================================================
// PRICILLA AI Guidance Engine v3.0
// Advanced Multi-Domain Trajectory Planning with MARL and PINN
// ================================================================================

// AIGuidanceEngine implements intelligent trajectory planning with MARL and PINN
type AIGuidanceEngine struct {
	modelVersion    string
	learningRate    float64
	trajectoryDB    map[string]*Trajectory
	mu              sync.RWMutex
	stealthModule   StealthOptimizer
	threatDB        map[string]ThreatLocation
	agentPool       *MARLAgentPool
	pinnOptimizer   *PINNTrajectoryOptimizer
	threatAdapter   *RealTimeThreatAdapter
	payloadProfiles map[PayloadType]*PayloadProfile
	scoringEngine   *MultiCriteriaScorer
	experienceReplay *ExperienceBuffer
}

// ================================================================================
// Multi-Agent Reinforcement Learning (MARL) Structures
// ================================================================================

// MARLAgentPool manages multiple RL agents for cooperative trajectory planning
type MARLAgentPool struct {
	agents           map[string]*RLAgent
	communicationBus chan AgentMessage
	consensusWeight  float64
	explorationRate  float64
	discountFactor   float64
	mu               sync.RWMutex
}

// RLAgent represents a single reinforcement learning agent
type RLAgent struct {
	ID              string
	PolicyNetwork   *NeuralPolicy
	ValueNetwork    *NeuralValue
	Specialization  AgentSpecialization
	TotalReward     float64
	EpisodeCount    int
	LearningRate    float64
	EntropyCoeff    float64
	ClipRange       float64
	lastAction      TrajectoryAction
	lastState       AgentState
}

// AgentSpecialization defines what the agent optimizes for
type AgentSpecialization string

const (
	SpecializationStealth     AgentSpecialization = "stealth"
	SpecializationSpeed       AgentSpecialization = "speed"
	SpecializationFuel        AgentSpecialization = "fuel"
	SpecializationThreat      AgentSpecialization = "threat_avoidance"
	SpecializationTerrain     AgentSpecialization = "terrain_following"
	SpecializationPhysics     AgentSpecialization = "physics_optimal"
	SpecializationMultiDomain AgentSpecialization = "multi_domain"
)

// AgentMessage for inter-agent communication
type AgentMessage struct {
	FromAgent       string
	ToAgent         string
	MessageType     string
	TrajectoryVote  *Trajectory
	ThreatInfo      *ThreatLocation
	ConsensusWeight float64
	Timestamp       time.Time
}

// AgentState represents the state observed by an RL agent
type AgentState struct {
	Position         Vector3D
	Velocity         Vector3D
	TargetDistance   float64
	ThreatProximity  float64
	FuelRemaining    float64
	TimeRemaining    float64
	StealthScore     float64
	TerrainFeatures  []float64
	WeatherConditions []float64
	PayloadStatus    []float64
}

// TrajectoryAction represents an action taken by an RL agent
type TrajectoryAction struct {
	DeltaHeading    float64 // Change in heading (radians)
	DeltaPitch      float64 // Change in pitch (radians)
	ThrustLevel     float64 // 0.0 - 1.0
	AltitudeChange  float64 // Target altitude change
	WaypointSkip    int     // Number of waypoints to skip (for speed)
	StealthActivate bool    // Activate stealth mode
}

// NeuralPolicy represents a policy network for action selection
type NeuralPolicy struct {
	InputSize    int
	HiddenLayers []int
	OutputSize   int
	Weights      [][][]float64
	Biases       [][]float64
	Activation   string
}

// NeuralValue represents a value network for state evaluation
type NeuralValue struct {
	InputSize    int
	HiddenLayers []int
	Weights      [][][]float64
	Biases       [][]float64
}

// ExperienceBuffer stores experience tuples for replay
type ExperienceBuffer struct {
	Experiences []Experience
	MaxSize     int
	mu          sync.RWMutex
}

// Experience represents a single experience tuple
type Experience struct {
	State      AgentState
	Action     TrajectoryAction
	Reward     float64
	NextState  AgentState
	Done       bool
	AgentID    string
	Timestamp  time.Time
}

// ================================================================================
// Physics-Informed Neural Network (PINN) Structures
// ================================================================================

// PINNTrajectoryOptimizer uses physics constraints in neural optimization
type PINNTrajectoryOptimizer struct {
	physicsLayers     []PhysicsLayer
	collocationPoints int
	boundaryConditions []BoundaryCondition
	pdeResidualWeight float64
	dataLossWeight    float64
	physicsModels     map[PayloadType]*PhysicsModel
	adaptiveWeights   *AdaptiveWeightScheduler
}

// PhysicsLayer represents a layer that enforces physics constraints
type PhysicsLayer struct {
	LayerType       string
	EquationType    string // "navier_stokes", "orbital_mechanics", "aerodynamics"
	Coefficients    []float64
	Residuals       []float64
}

// BoundaryCondition defines physics boundary conditions
type BoundaryCondition struct {
	Type       string // "dirichlet", "neumann", "periodic"
	Location   string // "start", "end", "boundary"
	Value      float64
	Variable   string // "position", "velocity", "acceleration"
}

// PhysicsModel contains physics equations for a payload type
type PhysicsModel struct {
	PayloadType        PayloadType
	Mass               float64
	DragCoefficient    float64
	LiftCoefficient    float64
	ThrustCapacity     float64
	FuelConsumptionRate float64
	GravityModel       string // "flat", "spherical", "j2_perturbation"
	AtmosphereModel    string // "none", "exponential", "us_standard"
	EquationsOfMotion  []MotionEquation
	Constraints        []PhysicsConstraint
}

// MotionEquation represents a differential equation of motion
type MotionEquation struct {
	Variable    string // "x", "y", "z", "vx", "vy", "vz"
	Expression  string // Symbolic representation
	Coefficients []float64
}

// PhysicsConstraint defines a physical constraint
type PhysicsConstraint struct {
	Type       string // "max_acceleration", "max_q", "thermal_limit"
	Value      float64
	Penalty    float64
}

// AdaptiveWeightScheduler adjusts PINN loss weights during training
type AdaptiveWeightScheduler struct {
	DataWeight         float64
	PhysicsWeight      float64
	BoundaryWeight     float64
	AdaptationRate     float64
	HistoricalLosses   []LossRecord
}

// LossRecord stores loss values for adaptive scheduling
type LossRecord struct {
	DataLoss     float64
	PhysicsLoss  float64
	BoundaryLoss float64
	TotalLoss    float64
	Iteration    int
	Timestamp    time.Time
}

// ================================================================================
// Multi-Criteria Scoring System
// ================================================================================

// MultiCriteriaScorer implements sophisticated trajectory scoring
type MultiCriteriaScorer struct {
	criteria      map[string]*ScoringCriterion
	weights       map[string]float64
	paretoFront   []*Trajectory
	mu            sync.RWMutex
}

// ScoringCriterion defines a single scoring dimension
type ScoringCriterion struct {
	Name            string
	Weight          float64
	MinimizeGoal    bool // true = lower is better, false = higher is better
	Normalize       bool
	ThresholdLow    float64
	ThresholdHigh   float64
	ScoreFunc       func(*Trajectory, TrajectoryRequest) float64
}

// TrajectoryScore contains detailed scoring breakdown
type TrajectoryScore struct {
	TotalScore      float64
	CriteriaScores  map[string]float64
	ParetoRank      int
	DominatedBy     int
	Dominates       int
	CrowdingDistance float64
}

// ================================================================================
// Real-Time Threat Adaptation
// ================================================================================

// RealTimeThreatAdapter handles dynamic threat response
type RealTimeThreatAdapter struct {
	threatHistory     []ThreatEvent
	predictionModel   *ThreatPredictor
	evasionStrategies map[string]*EvasionStrategy
	alertLevel        AlertLevel
	adaptationRate    float64
	mu                sync.RWMutex
}

// ThreatEvent represents a detected threat
type ThreatEvent struct {
	ID            string
	ThreatType    string
	Location      Vector3D
	Velocity      Vector3D
	DetectedAt    time.Time
	Confidence    float64
	ThreatLevel   float64
	Tracking      bool
	PredictedPath []Vector3D
}

// ThreatPredictor predicts threat movements
type ThreatPredictor struct {
	ModelType       string // "kalman", "particle_filter", "neural"
	StateEstimate   []float64
	Covariance      [][]float64
	ProcessNoise    [][]float64
	MeasurementNoise [][]float64
}

// EvasionStrategy defines how to evade a specific threat type
type EvasionStrategy struct {
	ThreatType      string
	PreferredAltitude float64
	PreferredSpeed  float64
	ManeuverType    string // "terrain_mask", "high_altitude", "speed_burst", "decoy"
	SuccessRate     float64
	FuelCost        float64
}

// AlertLevel defines system alert state
type AlertLevel string

const (
	AlertNormal   AlertLevel = "normal"
	AlertElevated AlertLevel = "elevated"
	AlertHigh     AlertLevel = "high"
	AlertCritical AlertLevel = "critical"
	AlertCombat   AlertLevel = "combat"
)

// ================================================================================
// Payload-Specific Profiles
// ================================================================================

// PayloadProfile contains payload-specific characteristics
type PayloadProfile struct {
	Type              PayloadType
	MaxSpeed          float64
	MinSpeed          float64
	MaxAcceleration   float64
	MaxAltitude       float64
	MinAltitude       float64
	MaxTurnRate       float64
	FuelCapacity      float64
	FuelEfficiency    float64
	StealthCapability float64
	SensorRange       float64
	CommunicationRange float64
	OperatingDomain   string // "air", "space", "ground", "underwater", "interstellar"
	PhysicsModel      *PhysicsModel
}

// StealthOptimizer interface for stealth optimization
type StealthOptimizer interface {
	OptimizeTrajectory(traj *Trajectory, mode StealthMode) (*Trajectory, error)
	CalculateRCS(wp Waypoint, heading float64) float64
	CalculateThermalSignature(wp Waypoint) float64
}

// ================================================================================
// Constructor and Initialization
// ================================================================================

// NewAIGuidanceEngine creates a new AI guidance engine with all advanced features
func NewAIGuidanceEngine(stealth StealthOptimizer) *AIGuidanceEngine {
	engine := &AIGuidanceEngine{
		modelVersion:    "PRICILLA-v3.0.0-MARL-PINN",
		learningRate:    0.0003,
		trajectoryDB:    make(map[string]*Trajectory),
		stealthModule:   stealth,
		threatDB:        make(map[string]ThreatLocation),
		payloadProfiles: initializePayloadProfiles(),
		experienceReplay: newExperienceBuffer(100000),
	}

	engine.agentPool = newMARLAgentPool()
	engine.pinnOptimizer = newPINNOptimizer()
	engine.threatAdapter = newThreatAdapter()
	engine.scoringEngine = newMultiCriteriaScorer()

	return engine
}

// initializePayloadProfiles creates profiles for all supported payload types
func initializePayloadProfiles() map[PayloadType]*PayloadProfile {
	profiles := make(map[PayloadType]*PayloadProfile)

	// Hunoid (humanoid robot)
	profiles[PayloadHunoid] = &PayloadProfile{
		Type:              PayloadHunoid,
		MaxSpeed:          15.0, // m/s (running speed)
		MinSpeed:          0.0,
		MaxAcceleration:   5.0,
		MaxAltitude:       100.0, // climbing/jumping
		MinAltitude:       -10.0, // underground
		MaxTurnRate:       math.Pi, // rad/s
		FuelCapacity:      100.0,
		FuelEfficiency:    0.95,
		StealthCapability: 0.7,
		SensorRange:       500.0,
		CommunicationRange: 10000.0,
		OperatingDomain:   "ground",
	}

	// UAV (Unmanned Aerial Vehicle)
	profiles[PayloadUAV] = &PayloadProfile{
		Type:              PayloadUAV,
		MaxSpeed:          150.0, // m/s
		MinSpeed:          20.0,
		MaxAcceleration:   15.0,
		MaxAltitude:       15000.0,
		MinAltitude:       50.0,
		MaxTurnRate:       math.Pi / 2,
		FuelCapacity:      500.0,
		FuelEfficiency:    0.85,
		StealthCapability: 0.6,
		SensorRange:       30000.0,
		CommunicationRange: 200000.0,
		OperatingDomain:   "air",
	}

	// Rocket
	profiles[PayloadRocket] = &PayloadProfile{
		Type:              PayloadRocket,
		MaxSpeed:          8000.0, // m/s
		MinSpeed:          100.0,
		MaxAcceleration:   100.0, // 10g
		MaxAltitude:       400000.0, // orbital
		MinAltitude:       0.0,
		MaxTurnRate:       0.1,
		FuelCapacity:      10000.0,
		FuelEfficiency:    0.3,
		StealthCapability: 0.1,
		SensorRange:       1000.0,
		CommunicationRange: 1000000.0,
		OperatingDomain:   "space",
	}

	// Missile
	profiles[PayloadMissile] = &PayloadProfile{
		Type:              PayloadMissile,
		MaxSpeed:          2000.0, // m/s (hypersonic)
		MinSpeed:          200.0,
		MaxAcceleration:   300.0, // 30g
		MaxAltitude:       50000.0,
		MinAltitude:       10.0,
		MaxTurnRate:       math.Pi,
		FuelCapacity:      200.0,
		FuelEfficiency:    0.4,
		StealthCapability: 0.5,
		SensorRange:       50000.0,
		CommunicationRange: 500000.0,
		OperatingDomain:   "air",
	}

	// Spacecraft
	profiles[PayloadSpacecraft] = &PayloadProfile{
		Type:              PayloadSpacecraft,
		MaxSpeed:          30000.0, // m/s (orbital velocity)
		MinSpeed:          0.0,
		MaxAcceleration:   50.0,
		MaxAltitude:       math.MaxFloat64, // deep space
		MinAltitude:       200000.0, // LEO
		MaxTurnRate:       0.05,
		FuelCapacity:       50000.0,
		FuelEfficiency:    0.95,
		StealthCapability: 0.2,
		SensorRange:       1000000.0,
		CommunicationRange: math.MaxFloat64,
		OperatingDomain:   "space",
	}

	// Drone (small quadcopter)
	profiles[PayloadDrone] = &PayloadProfile{
		Type:              PayloadDrone,
		MaxSpeed:          30.0, // m/s
		MinSpeed:          0.0,
		MaxAcceleration:   20.0,
		MaxAltitude:       500.0,
		MinAltitude:       1.0,
		MaxTurnRate:       math.Pi * 2,
		FuelCapacity:      50.0, // battery
		FuelEfficiency:    0.9,
		StealthCapability: 0.8,
		SensorRange:       1000.0,
		CommunicationRange: 5000.0,
		OperatingDomain:   "air",
	}

	// Ground Robot
	profiles[PayloadGroundRobot] = &PayloadProfile{
		Type:              PayloadGroundRobot,
		MaxSpeed:          10.0, // m/s
		MinSpeed:          0.0,
		MaxAcceleration:   3.0,
		MaxAltitude:       0.0,
		MinAltitude:       -5.0, // tunnels
		MaxTurnRate:       math.Pi,
		FuelCapacity:      200.0,
		FuelEfficiency:    0.92,
		StealthCapability: 0.75,
		SensorRange:       300.0,
		CommunicationRange: 8000.0,
		OperatingDomain:   "ground",
	}

	// Submarine
	profiles[PayloadSubmarine] = &PayloadProfile{
		Type:              PayloadSubmarine,
		MaxSpeed:          20.0, // m/s (~40 knots)
		MinSpeed:          0.0,
		MaxAcceleration:   2.0,
		MaxAltitude:       0.0,
		MinAltitude:       -1000.0, // depth
		MaxTurnRate:       0.1,
		FuelCapacity:      100000.0,
		FuelEfficiency:    0.98, // nuclear
		StealthCapability: 0.95,
		SensorRange:       50000.0, // sonar
		CommunicationRange: 100.0, // underwater limited
		OperatingDomain:   "underwater",
	}

	// Interstellar probe
	profiles[PayloadInterstellar] = &PayloadProfile{
		Type:              PayloadInterstellar,
		MaxSpeed:          150000.0, // m/s (0.05% c)
		MinSpeed:          0.0,
		MaxAcceleration:   0.1, // gentle for long duration
		MaxAltitude:       math.MaxFloat64,
		MinAltitude:       0.0,
		MaxTurnRate:       0.001,
		FuelCapacity:      1000000.0,
		FuelEfficiency:    0.99,
		StealthCapability: 0.0,
		SensorRange:       math.MaxFloat64,
		CommunicationRange: math.MaxFloat64,
		OperatingDomain:   "interstellar",
	}

	return profiles
}

// newMARLAgentPool creates the multi-agent RL system
func newMARLAgentPool() *MARLAgentPool {
	pool := &MARLAgentPool{
		agents:           make(map[string]*RLAgent),
		communicationBus: make(chan AgentMessage, 1000),
		consensusWeight:  0.7,
		explorationRate:  0.1,
		discountFactor:   0.99,
	}

	// Create specialized agents
	specializations := []AgentSpecialization{
		SpecializationStealth,
		SpecializationSpeed,
		SpecializationFuel,
		SpecializationThreat,
		SpecializationTerrain,
		SpecializationPhysics,
		SpecializationMultiDomain,
	}

	for _, spec := range specializations {
		agent := &RLAgent{
			ID:             uuid.New().String(),
			Specialization: spec,
			LearningRate:   0.0003,
			EntropyCoeff:   0.01,
			ClipRange:      0.2,
			PolicyNetwork:  newPolicyNetwork(64, []int{256, 256, 128}, 6),
			ValueNetwork:   newValueNetwork(64, []int{256, 256}),
		}
		pool.agents[string(spec)] = agent
	}

	return pool
}

// newPolicyNetwork creates a policy neural network
func newPolicyNetwork(inputSize int, hiddenLayers []int, outputSize int) *NeuralPolicy {
	policy := &NeuralPolicy{
		InputSize:    inputSize,
		HiddenLayers: hiddenLayers,
		OutputSize:   outputSize,
		Activation:   "tanh",
	}

	// Initialize weights with Xavier initialization
	layerSizes := append([]int{inputSize}, hiddenLayers...)
	layerSizes = append(layerSizes, outputSize)

	policy.Weights = make([][][]float64, len(layerSizes)-1)
	policy.Biases = make([][]float64, len(layerSizes)-1)

	for i := 0; i < len(layerSizes)-1; i++ {
		policy.Weights[i] = make([][]float64, layerSizes[i])
		policy.Biases[i] = make([]float64, layerSizes[i+1])

		scale := math.Sqrt(2.0 / float64(layerSizes[i]+layerSizes[i+1]))
		for j := 0; j < layerSizes[i]; j++ {
			policy.Weights[i][j] = make([]float64, layerSizes[i+1])
			for k := 0; k < layerSizes[i+1]; k++ {
				policy.Weights[i][j][k] = rand.NormFloat64() * scale
			}
		}
	}

	return policy
}

// newValueNetwork creates a value neural network
func newValueNetwork(inputSize int, hiddenLayers []int) *NeuralValue {
	value := &NeuralValue{
		InputSize:    inputSize,
		HiddenLayers: hiddenLayers,
	}

	layerSizes := append([]int{inputSize}, hiddenLayers...)
	layerSizes = append(layerSizes, 1)

	value.Weights = make([][][]float64, len(layerSizes)-1)
	value.Biases = make([][]float64, len(layerSizes)-1)

	for i := 0; i < len(layerSizes)-1; i++ {
		value.Weights[i] = make([][]float64, layerSizes[i])
		value.Biases[i] = make([]float64, layerSizes[i+1])

		scale := math.Sqrt(2.0 / float64(layerSizes[i]+layerSizes[i+1]))
		for j := 0; j < layerSizes[i]; j++ {
			value.Weights[i][j] = make([]float64, layerSizes[i+1])
			for k := 0; k < layerSizes[i+1]; k++ {
				value.Weights[i][j][k] = rand.NormFloat64() * scale
			}
		}
	}

	return value
}

// newPINNOptimizer creates the physics-informed optimizer
func newPINNOptimizer() *PINNTrajectoryOptimizer {
	return &PINNTrajectoryOptimizer{
		physicsLayers: []PhysicsLayer{
			{LayerType: "dynamics", EquationType: "navier_stokes"},
			{LayerType: "gravity", EquationType: "orbital_mechanics"},
			{LayerType: "aero", EquationType: "aerodynamics"},
		},
		collocationPoints: 100,
		boundaryConditions: []BoundaryCondition{
			{Type: "dirichlet", Location: "start", Variable: "position"},
			{Type: "dirichlet", Location: "end", Variable: "position"},
			{Type: "neumann", Location: "start", Variable: "velocity"},
		},
		pdeResidualWeight: 1.0,
		dataLossWeight:    1.0,
		physicsModels:     initializePhysicsModels(),
		adaptiveWeights: &AdaptiveWeightScheduler{
			DataWeight:     1.0,
			PhysicsWeight:  1.0,
			BoundaryWeight: 10.0,
			AdaptationRate: 0.01,
		},
	}
}

// initializePhysicsModels creates physics models for each payload type
func initializePhysicsModels() map[PayloadType]*PhysicsModel {
	models := make(map[PayloadType]*PhysicsModel)

	// Air domain physics
	airModel := &PhysicsModel{
		DragCoefficient:    0.3,
		LiftCoefficient:    0.5,
		GravityModel:       "spherical",
		AtmosphereModel:    "us_standard",
		FuelConsumptionRate: 0.1,
		EquationsOfMotion: []MotionEquation{
			{Variable: "x", Expression: "vx"},
			{Variable: "y", Expression: "vy"},
			{Variable: "z", Expression: "vz"},
			{Variable: "vx", Expression: "Fx/m - Cd*vx*|v|/(2m)"},
			{Variable: "vy", Expression: "Fy/m - Cd*vy*|v|/(2m)"},
			{Variable: "vz", Expression: "Fz/m - g - Cd*vz*|v|/(2m)"},
		},
		Constraints: []PhysicsConstraint{
			{Type: "max_q", Value: 50000, Penalty: 1000},
			{Type: "thermal_limit", Value: 2000, Penalty: 10000},
		},
	}

	models[PayloadUAV] = copyPhysicsModel(airModel, PayloadUAV, 500, 50000)
	models[PayloadMissile] = copyPhysicsModel(airModel, PayloadMissile, 1000, 500000)
	models[PayloadDrone] = copyPhysicsModel(airModel, PayloadDrone, 5, 500)

	// Space domain physics
	spaceModel := &PhysicsModel{
		DragCoefficient:    0.0,
		GravityModel:       "j2_perturbation",
		AtmosphereModel:    "none",
		FuelConsumptionRate: 0.01,
		EquationsOfMotion: []MotionEquation{
			{Variable: "x", Expression: "vx"},
			{Variable: "y", Expression: "vy"},
			{Variable: "z", Expression: "vz"},
			{Variable: "vx", Expression: "-mu*x/r^3 + Fx/m"},
			{Variable: "vy", Expression: "-mu*y/r^3 + Fy/m"},
			{Variable: "vz", Expression: "-mu*z/r^3 + Fz/m"},
		},
	}

	models[PayloadRocket] = copyPhysicsModel(spaceModel, PayloadRocket, 50000, 1000000)
	models[PayloadSpacecraft] = copyPhysicsModel(spaceModel, PayloadSpacecraft, 10000, 100000)
	models[PayloadInterstellar] = copyPhysicsModel(spaceModel, PayloadInterstellar, 1000, 10000000)

	// Ground domain physics
	groundModel := &PhysicsModel{
		DragCoefficient:    0.5,
		GravityModel:       "flat",
		AtmosphereModel:    "none",
		FuelConsumptionRate: 0.05,
		EquationsOfMotion: []MotionEquation{
			{Variable: "x", Expression: "vx"},
			{Variable: "y", Expression: "vy"},
			{Variable: "vx", Expression: "Fx/m - friction*sign(vx)"},
			{Variable: "vy", Expression: "Fy/m - friction*sign(vy)"},
		},
	}

	models[PayloadHunoid] = copyPhysicsModel(groundModel, PayloadHunoid, 80, 1000)
	models[PayloadGroundRobot] = copyPhysicsModel(groundModel, PayloadGroundRobot, 200, 5000)

	// Underwater physics
	underwaterModel := &PhysicsModel{
		DragCoefficient:    1.0,
		GravityModel:       "flat",
		AtmosphereModel:    "none",
		FuelConsumptionRate: 0.001,
		EquationsOfMotion: []MotionEquation{
			{Variable: "x", Expression: "vx"},
			{Variable: "y", Expression: "vy"},
			{Variable: "z", Expression: "vz"},
			{Variable: "vx", Expression: "Fx/m - Cd*vx*|v|*rho/(2m)"},
			{Variable: "vy", Expression: "Fy/m - Cd*vy*|v|*rho/(2m)"},
			{Variable: "vz", Expression: "Fz/m - buoyancy - Cd*vz*|v|*rho/(2m)"},
		},
	}

	models[PayloadSubmarine] = copyPhysicsModel(underwaterModel, PayloadSubmarine, 50000000, 100000000)

	return models
}

func copyPhysicsModel(base *PhysicsModel, payloadType PayloadType, mass, thrust float64) *PhysicsModel {
	model := *base
	model.PayloadType = payloadType
	model.Mass = mass
	model.ThrustCapacity = thrust
	return &model
}

// newThreatAdapter creates the threat adaptation system
func newThreatAdapter() *RealTimeThreatAdapter {
	adapter := &RealTimeThreatAdapter{
		threatHistory:     make([]ThreatEvent, 0),
		alertLevel:        AlertNormal,
		adaptationRate:    0.5,
		evasionStrategies: make(map[string]*EvasionStrategy),
	}

	// Initialize evasion strategies
	adapter.evasionStrategies["radar"] = &EvasionStrategy{
		ThreatType:      "radar",
		PreferredAltitude: 50.0, // terrain masking
		ManeuverType:    "terrain_mask",
		SuccessRate:     0.85,
		FuelCost:        1.2,
	}

	adapter.evasionStrategies["sam"] = &EvasionStrategy{
		ThreatType:      "sam",
		PreferredAltitude: 100.0,
		PreferredSpeed:  500.0, // high speed
		ManeuverType:    "speed_burst",
		SuccessRate:     0.7,
		FuelCost:        2.0,
	}

	adapter.evasionStrategies["interceptor"] = &EvasionStrategy{
		ThreatType:      "interceptor",
		PreferredAltitude: 15000.0, // high altitude
		ManeuverType:    "high_altitude",
		SuccessRate:     0.6,
		FuelCost:        1.5,
	}

	adapter.evasionStrategies["jamming"] = &EvasionStrategy{
		ThreatType:      "jamming",
		ManeuverType:    "decoy",
		SuccessRate:     0.9,
		FuelCost:        0.5,
	}

	adapter.predictionModel = &ThreatPredictor{
		ModelType:     "kalman",
		StateEstimate: make([]float64, 6),
		Covariance:    make([][]float64, 6),
	}

	return adapter
}

// newMultiCriteriaScorer creates the scoring engine
func newMultiCriteriaScorer() *MultiCriteriaScorer {
	scorer := &MultiCriteriaScorer{
		criteria:    make(map[string]*ScoringCriterion),
		weights:     make(map[string]float64),
		paretoFront: make([]*Trajectory, 0),
	}

	// Initialize scoring criteria
	scorer.criteria["distance"] = &ScoringCriterion{
		Name:         "distance",
		Weight:       0.15,
		MinimizeGoal: true,
		Normalize:    true,
	}

	scorer.criteria["time"] = &ScoringCriterion{
		Name:         "time",
		Weight:       0.15,
		MinimizeGoal: true,
		Normalize:    true,
	}

	scorer.criteria["fuel"] = &ScoringCriterion{
		Name:         "fuel",
		Weight:       0.15,
		MinimizeGoal: true,
		Normalize:    true,
	}

	scorer.criteria["stealth"] = &ScoringCriterion{
		Name:         "stealth",
		Weight:       0.20,
		MinimizeGoal: false,
		Normalize:    true,
	}

	scorer.criteria["threat_exposure"] = &ScoringCriterion{
		Name:         "threat_exposure",
		Weight:       0.20,
		MinimizeGoal: true,
		Normalize:    true,
	}

	scorer.criteria["physics_feasibility"] = &ScoringCriterion{
		Name:         "physics_feasibility",
		Weight:       0.10,
		MinimizeGoal: false,
		Normalize:    true,
	}

	scorer.criteria["smoothness"] = &ScoringCriterion{
		Name:         "smoothness",
		Weight:       0.05,
		MinimizeGoal: false,
		Normalize:    true,
	}

	return scorer
}

// newExperienceBuffer creates the experience replay buffer
func newExperienceBuffer(maxSize int) *ExperienceBuffer {
	return &ExperienceBuffer{
		Experiences: make([]Experience, 0, maxSize),
		MaxSize:     maxSize,
	}
}

// ================================================================================
// Main Trajectory Planning (MARL + PINN Enhanced)
// ================================================================================

// PlanTrajectory generates optimal path using MARL and PINN
func (e *AIGuidanceEngine) PlanTrajectory(ctx context.Context, req TrajectoryRequest) (*Trajectory, error) {
	// Get payload profile
	profile := e.payloadProfiles[req.PayloadType]
	if profile == nil {
		return nil, fmt.Errorf("unsupported payload type: %s", req.PayloadType)
	}

	// Check for real-time threats and adapt
	e.threatAdapter.updateAlertLevel(req.Constraints.MustAvoidThreats)

	// Phase 1: MARL-based candidate generation
	marlCandidates := e.generateMARLTrajectories(ctx, req, profile, 5)

	// Phase 2: PINN optimization of candidates
	pinnOptimized := make([]*Trajectory, 0)
	for _, candidate := range marlCandidates {
		optimized, err := e.pinnOptimizer.optimize(candidate, profile)
		if err == nil {
			pinnOptimized = append(pinnOptimized, optimized)
		}
	}

	// Phase 3: Generate traditional candidates as backup
	traditionalCandidates := e.generateCandidates(req, 5)
	allCandidates := append(pinnOptimized, traditionalCandidates...)

	// Phase 4: Multi-criteria scoring
	scoredCandidates := e.scoringEngine.scoreAll(allCandidates, req)

	// Phase 5: Pareto-optimal selection
	paretoFront := e.scoringEngine.findParetoFront(scoredCandidates)

	// Select best from Pareto front based on mission priority
	bestTraj := e.selectFromParetoFront(paretoFront, req)

	if bestTraj == nil {
		return nil, fmt.Errorf("failed to generate valid trajectory")
	}

	// Phase 6: Apply stealth optimization if required
	if req.StealthMode != StealthModeNone && e.stealthModule != nil {
		stealthTraj, err := e.stealthModule.OptimizeTrajectory(bestTraj, req.StealthMode)
		if err == nil && stealthTraj != nil {
			bestTraj = stealthTraj
		}
	}

	// Phase 7: Real-time threat adaptation
	if e.threatAdapter.alertLevel != AlertNormal {
		adaptedTraj := e.threatAdapter.adaptTrajectory(bestTraj, e.threatDB)
		if adaptedTraj != nil {
			bestTraj = adaptedTraj
		}
	}

	// Store for learning
	e.mu.Lock()
	e.trajectoryDB[bestTraj.ID] = bestTraj
	e.mu.Unlock()

	// Store experience for RL training
	e.storeExperience(req, bestTraj)

	return bestTraj, nil
}

// generateMARLTrajectories uses multi-agent RL for trajectory generation
func (e *AIGuidanceEngine) generateMARLTrajectories(ctx context.Context, req TrajectoryRequest, profile *PayloadProfile, count int) []*Trajectory {
	trajectories := make([]*Trajectory, 0, count)

	// Create initial state
	initialState := e.createAgentState(req, profile)

	// Get trajectory proposals from each agent
	agentProposals := make(map[string]*Trajectory)

	e.agentPool.mu.RLock()
	for name, agent := range e.agentPool.agents {
		proposal := e.agentGenerateTrajectory(agent, initialState, req, profile)
		if proposal != nil {
			agentProposals[name] = proposal
		}
	}
	e.agentPool.mu.RUnlock()

	// Consensus voting mechanism
	consensusTraj := e.agentPool.reachConsensus(agentProposals, req)
	if consensusTraj != nil {
		trajectories = append(trajectories, consensusTraj)
	}

	// Add individual agent trajectories
	for _, traj := range agentProposals {
		if len(trajectories) < count {
			trajectories = append(trajectories, traj)
		}
	}

	// Fill remaining with exploration trajectories
	for len(trajectories) < count {
		explorationTraj := e.generateExplorationTrajectory(req, profile)
		trajectories = append(trajectories, explorationTraj)
	}

	return trajectories
}

// createAgentState creates the state representation for RL agents
func (e *AIGuidanceEngine) createAgentState(req TrajectoryRequest, profile *PayloadProfile) AgentState {
	targetDist := math.Sqrt(Distance(req.StartPosition, req.TargetPosition))

	// Calculate threat proximity
	threatProximity := 0.0
	for _, threat := range req.Constraints.MustAvoidThreats {
		dist := math.Sqrt(Distance(req.StartPosition, threat.Position))
		if dist < threat.EffectRadius*2 {
			threatProximity += (1.0 - dist/(threat.EffectRadius*2)) * threat.Confidence
		}
	}

	return AgentState{
		Position:        req.StartPosition,
		Velocity:        Vector3D{X: 0, Y: 0, Z: 0},
		TargetDistance:  targetDist,
		ThreatProximity: threatProximity,
		FuelRemaining:   profile.FuelCapacity,
		TimeRemaining:   float64(req.MaxTime.Seconds()),
		StealthScore:    profile.StealthCapability,
		TerrainFeatures: make([]float64, 8),
		WeatherConditions: make([]float64, 4),
		PayloadStatus:   make([]float64, 6),
	}
}

// agentGenerateTrajectory generates a trajectory proposal from a single agent
func (e *AIGuidanceEngine) agentGenerateTrajectory(agent *RLAgent, state AgentState, req TrajectoryRequest, profile *PayloadProfile) *Trajectory {
	waypoints := make([]Waypoint, 0)
	currentPos := req.StartPosition
	currentVel := Vector3D{X: 0, Y: 0, Z: 0}
	currentTime := time.Now()

	// Start waypoint
	waypoints = append(waypoints, Waypoint{
		Position:  currentPos,
		Velocity:  currentVel,
		Timestamp: currentTime,
		Constraints: WaypointConstraints{
			MaxSpeed:        profile.MaxSpeed,
			MaxAcceleration: profile.MaxAcceleration,
			MinAltitude:     profile.MinAltitude,
			MaxAltitude:     profile.MaxAltitude,
		},
	})

	// Generate trajectory through policy rollout
	maxSteps := 50
	targetReached := false

	for step := 0; step < maxSteps && !targetReached; step++ {
		// Get action from policy network
		action := agent.selectAction(state, e.agentPool.explorationRate)

		// Apply action to get next state
		nextPos, nextVel := e.applyAction(currentPos, currentVel, action, profile)

		// Check if target reached
		distToTarget := math.Sqrt(Distance(nextPos, req.TargetPosition))
		if distToTarget < 100.0 { // Within 100m of target
			targetReached = true
			nextPos = req.TargetPosition
		}

		// Create waypoint
		stepTime := currentTime.Add(time.Duration(step+1) * time.Second)
		wp := Waypoint{
			Position:  nextPos,
			Velocity:  nextVel,
			Timestamp: stepTime,
			Constraints: WaypointConstraints{
				MaxSpeed:        profile.MaxSpeed,
				MaxAcceleration: profile.MaxAcceleration,
				MinAltitude:     profile.MinAltitude,
				MaxAltitude:     profile.MaxAltitude,
				StealthRequired: req.Constraints.StealthRequired,
			},
		}
		waypoints = append(waypoints, wp)

		// Update state
		currentPos = nextPos
		currentVel = nextVel
		state = e.updateAgentState(state, nextPos, nextVel, req, profile)
	}

	// Ensure we end at target
	if !targetReached {
		waypoints = append(waypoints, Waypoint{
			Position:  req.TargetPosition,
			Velocity:  Vector3D{X: 0, Y: 0, Z: 0},
			Timestamp: currentTime.Add(time.Duration(len(waypoints)) * time.Second),
			Constraints: WaypointConstraints{
				MaxSpeed:        profile.MaxSpeed,
				MaxAcceleration: profile.MaxAcceleration,
				MinAltitude:     profile.MinAltitude,
				MaxAltitude:     profile.MaxAltitude,
			},
		})
	}

	traj := &Trajectory{
		ID:          uuid.New().String(),
		PayloadType: req.PayloadType,
		Waypoints:   waypoints,
		CreatedAt:   time.Now(),
	}

	// Calculate trajectory metrics
	traj.TotalDistance = e.calculateTrajDistance(traj)
	traj.StealthScore = e.calculateStealthScore(traj)
	traj.ThreatExposure = e.calculateThreatExposure(traj, req.Constraints.MustAvoidThreats)
	traj.FuelRequired = e.estimateFuel(traj)
	traj.Confidence = 0.85 + agent.TotalReward*0.001 // Base confidence + learning bonus

	return traj
}

// selectAction selects an action using the policy network
func (agent *RLAgent) selectAction(state AgentState, explorationRate float64) TrajectoryAction {
	// Convert state to input vector
	input := stateToVector(state)

	// Forward pass through policy network
	output := agent.PolicyNetwork.forward(input)

	// Apply exploration (epsilon-greedy with Gaussian noise)
	if rand.Float64() < explorationRate {
		for i := range output {
			output[i] += rand.NormFloat64() * 0.1
		}
	}

	// Convert output to action
	return TrajectoryAction{
		DeltaHeading:    output[0] * math.Pi / 4,  // Max 45 degree turn
		DeltaPitch:      output[1] * math.Pi / 8,  // Max 22.5 degree pitch
		ThrustLevel:     sigmoid(output[2]),       // 0-1 thrust
		AltitudeChange:  output[3] * 100.0,        // Max 100m altitude change
		WaypointSkip:    int(math.Max(0, output[4])),
		StealthActivate: output[5] > 0,
	}
}

// forward performs a forward pass through the policy network
func (p *NeuralPolicy) forward(input []float64) []float64 {
	// Pad or truncate input to match expected size
	paddedInput := make([]float64, p.InputSize)
	for i := 0; i < len(paddedInput) && i < len(input); i++ {
		paddedInput[i] = input[i]
	}

	current := paddedInput
	for i := 0; i < len(p.Weights); i++ {
		next := make([]float64, len(p.Biases[i]))

		// Matrix multiplication
		for j := 0; j < len(current) && j < len(p.Weights[i]); j++ {
			for k := 0; k < len(next) && k < len(p.Weights[i][j]); k++ {
				next[k] += current[j] * p.Weights[i][j][k]
			}
		}

		// Add bias and activation
		for j := range next {
			next[j] += p.Biases[i][j]
			if i < len(p.Weights)-1 { // Hidden layers use tanh
				next[j] = math.Tanh(next[j])
			}
		}

		current = next
	}

	return current
}

// stateToVector converts agent state to input vector
func stateToVector(state AgentState) []float64 {
	vec := make([]float64, 0, 64)

	// Position (normalized)
	vec = append(vec, state.Position.X/100000.0)
	vec = append(vec, state.Position.Y/100000.0)
	vec = append(vec, state.Position.Z/10000.0)

	// Velocity (normalized)
	vec = append(vec, state.Velocity.X/1000.0)
	vec = append(vec, state.Velocity.Y/1000.0)
	vec = append(vec, state.Velocity.Z/1000.0)

	// Scalars
	vec = append(vec, state.TargetDistance/100000.0)
	vec = append(vec, state.ThreatProximity)
	vec = append(vec, state.FuelRemaining/1000.0)
	vec = append(vec, state.TimeRemaining/3600.0)
	vec = append(vec, state.StealthScore)

	// Terrain and weather features
	vec = append(vec, state.TerrainFeatures...)
	vec = append(vec, state.WeatherConditions...)
	vec = append(vec, state.PayloadStatus...)

	// Pad to 64 dimensions
	for len(vec) < 64 {
		vec = append(vec, 0.0)
	}

	return vec
}

// applyAction applies an action to get next position and velocity
func (e *AIGuidanceEngine) applyAction(pos, vel Vector3D, action TrajectoryAction, profile *PayloadProfile) (Vector3D, Vector3D) {
	// Calculate current heading
	currentHeading := math.Atan2(vel.Y, vel.X)
	currentSpeed := math.Sqrt(vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z)

	if currentSpeed < 0.1 {
		currentSpeed = profile.MinSpeed
		if currentSpeed == 0 {
			currentSpeed = 1.0
		}
	}

	// Apply heading change
	newHeading := currentHeading + action.DeltaHeading

	// Apply thrust
	targetSpeed := currentSpeed * (0.5 + action.ThrustLevel*0.5)
	targetSpeed = math.Min(targetSpeed, profile.MaxSpeed)
	targetSpeed = math.Max(targetSpeed, profile.MinSpeed)

	// Calculate new velocity
	newVel := Vector3D{
		X: targetSpeed * math.Cos(newHeading) * math.Cos(action.DeltaPitch),
		Y: targetSpeed * math.Sin(newHeading) * math.Cos(action.DeltaPitch),
		Z: targetSpeed * math.Sin(action.DeltaPitch),
	}

	// Calculate new position (simple Euler integration, 1 second timestep)
	newPos := Vector3D{
		X: pos.X + newVel.X,
		Y: pos.Y + newVel.Y,
		Z: pos.Z + newVel.Z + action.AltitudeChange,
	}

	// Enforce altitude constraints
	newPos.Z = math.Max(newPos.Z, profile.MinAltitude)
	newPos.Z = math.Min(newPos.Z, profile.MaxAltitude)

	return newPos, newVel
}

// updateAgentState updates the agent state after taking an action
func (e *AIGuidanceEngine) updateAgentState(state AgentState, pos, vel Vector3D, req TrajectoryRequest, profile *PayloadProfile) AgentState {
	state.Position = pos
	state.Velocity = vel
	state.TargetDistance = math.Sqrt(Distance(pos, req.TargetPosition))

	// Recalculate threat proximity
	state.ThreatProximity = 0.0
	for _, threat := range req.Constraints.MustAvoidThreats {
		dist := math.Sqrt(Distance(pos, threat.Position))
		if dist < threat.EffectRadius*2 {
			state.ThreatProximity += (1.0 - dist/(threat.EffectRadius*2)) * threat.Confidence
		}
	}

	return state
}

// reachConsensus implements multi-agent consensus voting
func (pool *MARLAgentPool) reachConsensus(proposals map[string]*Trajectory, req TrajectoryRequest) *Trajectory {
	if len(proposals) == 0 {
		return nil
	}

	// Weight each agent's proposal by their historical performance
	weightedWaypoints := make(map[int]Vector3D)
	totalWeight := 0.0

	// Find the trajectory with median number of waypoints
	waypointCounts := make([]int, 0)
	for _, traj := range proposals {
		waypointCounts = append(waypointCounts, len(traj.Waypoints))
	}
	sort.Ints(waypointCounts)
	targetWaypointCount := waypointCounts[len(waypointCounts)/2]

	// Interpolate/sample waypoints to match target count
	normalizedTrajs := make([]*Trajectory, 0)
	for _, traj := range proposals {
		normalized := interpolateWaypoints(traj, targetWaypointCount)
		normalizedTrajs = append(normalizedTrajs, normalized)
	}

	// Average positions weighted by confidence
	for _, traj := range normalizedTrajs {
		weight := traj.Confidence
		for i, wp := range traj.Waypoints {
			if _, exists := weightedWaypoints[i]; !exists {
				weightedWaypoints[i] = Vector3D{}
			}
			current := weightedWaypoints[i]
			weightedWaypoints[i] = Vector3D{
				X: current.X + wp.Position.X*weight,
				Y: current.Y + wp.Position.Y*weight,
				Z: current.Z + wp.Position.Z*weight,
			}
		}
		totalWeight += weight
	}

	// Create consensus trajectory
	consensusWaypoints := make([]Waypoint, targetWaypointCount)
	for i := 0; i < targetWaypointCount; i++ {
		wp := weightedWaypoints[i]
		consensusWaypoints[i] = Waypoint{
			Position: Vector3D{
				X: wp.X / totalWeight,
				Y: wp.Y / totalWeight,
				Z: wp.Z / totalWeight,
			},
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Constraints: WaypointConstraints{
				MaxSpeed:        100.0,
				MaxAcceleration: 10.0,
				MinAltitude:     0,
				MaxAltitude:     10000,
			},
		}
	}

	// Calculate velocities between waypoints
	for i := 1; i < len(consensusWaypoints); i++ {
		dt := consensusWaypoints[i].Timestamp.Sub(consensusWaypoints[i-1].Timestamp).Seconds()
		if dt > 0 {
			consensusWaypoints[i].Velocity = Vector3D{
				X: (consensusWaypoints[i].Position.X - consensusWaypoints[i-1].Position.X) / dt,
				Y: (consensusWaypoints[i].Position.Y - consensusWaypoints[i-1].Position.Y) / dt,
				Z: (consensusWaypoints[i].Position.Z - consensusWaypoints[i-1].Position.Z) / dt,
			}
		}
	}

	return &Trajectory{
		ID:          uuid.New().String(),
		PayloadType: req.PayloadType,
		Waypoints:   consensusWaypoints,
		Confidence:  pool.consensusWeight,
		CreatedAt:   time.Now(),
	}
}

// interpolateWaypoints resamples waypoints to target count
func interpolateWaypoints(traj *Trajectory, targetCount int) *Trajectory {
	if len(traj.Waypoints) == targetCount {
		return traj
	}

	newWaypoints := make([]Waypoint, targetCount)

	for i := 0; i < targetCount; i++ {
		t := float64(i) / float64(targetCount-1)
		srcIdx := t * float64(len(traj.Waypoints)-1)

		lowIdx := int(srcIdx)
		highIdx := lowIdx + 1
		if highIdx >= len(traj.Waypoints) {
			highIdx = len(traj.Waypoints) - 1
			lowIdx = highIdx - 1
			if lowIdx < 0 {
				lowIdx = 0
			}
		}

		frac := srcIdx - float64(lowIdx)

		low := traj.Waypoints[lowIdx]
		high := traj.Waypoints[highIdx]

		newWaypoints[i] = Waypoint{
			Position: Vector3D{
				X: low.Position.X + (high.Position.X-low.Position.X)*frac,
				Y: low.Position.Y + (high.Position.Y-low.Position.Y)*frac,
				Z: low.Position.Z + (high.Position.Z-low.Position.Z)*frac,
			},
			Velocity: Vector3D{
				X: low.Velocity.X + (high.Velocity.X-low.Velocity.X)*frac,
				Y: low.Velocity.Y + (high.Velocity.Y-low.Velocity.Y)*frac,
				Z: low.Velocity.Z + (high.Velocity.Z-low.Velocity.Z)*frac,
			},
			Timestamp:   low.Timestamp.Add(time.Duration(frac*float64(high.Timestamp.Sub(low.Timestamp))) * time.Nanosecond),
			Constraints: low.Constraints,
		}
	}

	return &Trajectory{
		ID:          traj.ID,
		PayloadType: traj.PayloadType,
		Waypoints:   newWaypoints,
		Confidence:  traj.Confidence,
	}
}

// generateExplorationTrajectory generates a random exploration trajectory
func (e *AIGuidanceEngine) generateExplorationTrajectory(req TrajectoryRequest, profile *PayloadProfile) *Trajectory {
	waypoints := make([]Waypoint, 0)
	currentPos := req.StartPosition
	currentTime := time.Now()

	waypoints = append(waypoints, Waypoint{
		Position:  currentPos,
		Timestamp: currentTime,
		Constraints: WaypointConstraints{
			MaxSpeed:        profile.MaxSpeed,
			MaxAcceleration: profile.MaxAcceleration,
			MinAltitude:     profile.MinAltitude,
			MaxAltitude:     profile.MaxAltitude,
		},
	})

	// Random walk towards target
	numSteps := 5 + rand.Intn(10)
	for i := 0; i < numSteps; i++ {
		progress := float64(i+1) / float64(numSteps)

		// Base position interpolated towards target
		basePos := Vector3D{
			X: req.StartPosition.X + (req.TargetPosition.X-req.StartPosition.X)*progress,
			Y: req.StartPosition.Y + (req.TargetPosition.Y-req.StartPosition.Y)*progress,
			Z: req.StartPosition.Z + (req.TargetPosition.Z-req.StartPosition.Z)*progress,
		}

		// Add random perturbation
		perturbation := 1000.0 * (1.0 - progress) // Decrease perturbation as we get closer
		newPos := Vector3D{
			X: basePos.X + (rand.Float64()-0.5)*perturbation,
			Y: basePos.Y + (rand.Float64()-0.5)*perturbation,
			Z: math.Max(profile.MinAltitude, math.Min(profile.MaxAltitude, basePos.Z+(rand.Float64()-0.5)*perturbation/10)),
		}

		currentPos = newPos
		waypoints = append(waypoints, Waypoint{
			Position:  currentPos,
			Timestamp: currentTime.Add(time.Duration(i+1) * 10 * time.Second),
			Constraints: WaypointConstraints{
				MaxSpeed:        profile.MaxSpeed,
				MaxAcceleration: profile.MaxAcceleration,
				MinAltitude:     profile.MinAltitude,
				MaxAltitude:     profile.MaxAltitude,
			},
		})
	}

	// Final waypoint at target
	waypoints = append(waypoints, Waypoint{
		Position:  req.TargetPosition,
		Timestamp: currentTime.Add(time.Duration(numSteps+1) * 10 * time.Second),
		Constraints: WaypointConstraints{
			MaxSpeed:        profile.MaxSpeed,
			MaxAcceleration: profile.MaxAcceleration,
			MinAltitude:     profile.MinAltitude,
			MaxAltitude:     profile.MaxAltitude,
		},
	})

	// Calculate velocities
	for i := 1; i < len(waypoints); i++ {
		dt := waypoints[i].Timestamp.Sub(waypoints[i-1].Timestamp).Seconds()
		if dt > 0 {
			waypoints[i].Velocity = Vector3D{
				X: (waypoints[i].Position.X - waypoints[i-1].Position.X) / dt,
				Y: (waypoints[i].Position.Y - waypoints[i-1].Position.Y) / dt,
				Z: (waypoints[i].Position.Z - waypoints[i-1].Position.Z) / dt,
			}
		}
	}

	traj := &Trajectory{
		ID:          uuid.New().String(),
		PayloadType: req.PayloadType,
		Waypoints:   waypoints,
		Confidence:  0.5, // Lower confidence for exploration
		CreatedAt:   time.Now(),
	}

	traj.TotalDistance = e.calculateTrajDistance(traj)
	traj.StealthScore = e.calculateStealthScore(traj)
	traj.ThreatExposure = e.calculateThreatExposure(traj, req.Constraints.MustAvoidThreats)
	traj.FuelRequired = e.estimateFuel(traj)

	return traj
}

// ================================================================================
// Physics-Informed Neural Network Optimization
// ================================================================================

// optimize applies PINN optimization to a trajectory
func (p *PINNTrajectoryOptimizer) optimize(traj *Trajectory, profile *PayloadProfile) (*Trajectory, error) {
	if traj == nil || len(traj.Waypoints) < 2 {
		return nil, fmt.Errorf("invalid trajectory for PINN optimization")
	}

	physicsModel := p.physicsModels[profile.Type]
	if physicsModel == nil {
		// Use default air model
		physicsModel = p.physicsModels[PayloadUAV]
	}

	// Create optimized trajectory
	optimized := &Trajectory{
		ID:          uuid.New().String(),
		PayloadType: traj.PayloadType,
		Waypoints:   make([]Waypoint, len(traj.Waypoints)),
		CreatedAt:   time.Now(),
	}

	copy(optimized.Waypoints, traj.Waypoints)

	// Apply physics-informed corrections
	for iteration := 0; iteration < 10; iteration++ {
		// Calculate PDE residuals
		residuals := p.calculatePDEResiduals(optimized, physicsModel)

		// Calculate boundary condition errors
		boundaryErrors := p.calculateBoundaryErrors(optimized, traj)

		// Update waypoints based on residuals and errors
		for i := 1; i < len(optimized.Waypoints)-1; i++ {
			// Position correction based on physics residuals
			correction := p.calculatePhysicsCorrection(i, residuals, physicsModel)

			optimized.Waypoints[i].Position.X += correction.X * p.adaptiveWeights.PhysicsWeight
			optimized.Waypoints[i].Position.Y += correction.Y * p.adaptiveWeights.PhysicsWeight
			optimized.Waypoints[i].Position.Z += correction.Z * p.adaptiveWeights.PhysicsWeight

			// Enforce altitude constraints
			optimized.Waypoints[i].Position.Z = math.Max(profile.MinAltitude,
				math.Min(profile.MaxAltitude, optimized.Waypoints[i].Position.Z))
		}

		// Recalculate velocities for consistency
		p.recalculateVelocities(optimized)

		// Update adaptive weights
		totalResidual := 0.0
		for _, r := range residuals {
			totalResidual += math.Abs(r)
		}
		totalBoundary := boundaryErrors[0] + boundaryErrors[1]

		p.adaptiveWeights.HistoricalLosses = append(p.adaptiveWeights.HistoricalLosses, LossRecord{
			PhysicsLoss:  totalResidual,
			BoundaryLoss: totalBoundary,
			TotalLoss:    totalResidual + totalBoundary,
			Iteration:    iteration,
			Timestamp:    time.Now(),
		})

		// Early stopping if converged
		if totalResidual < 0.01 && totalBoundary < 0.01 {
			break
		}
	}

	// Copy metrics from original and update
	optimized.TotalDistance = calculateTrajDistanceStatic(optimized)
	optimized.Confidence = traj.Confidence * 1.1 // Boost confidence after physics optimization
	if optimized.Confidence > 1.0 {
		optimized.Confidence = 1.0
	}

	return optimized, nil
}

// calculatePDEResiduals calculates physics equation residuals
func (p *PINNTrajectoryOptimizer) calculatePDEResiduals(traj *Trajectory, model *PhysicsModel) []float64 {
	residuals := make([]float64, len(traj.Waypoints)*6) // 6 DoF per waypoint

	for i := 1; i < len(traj.Waypoints)-1; i++ {
		prev := traj.Waypoints[i-1]
		curr := traj.Waypoints[i]
		next := traj.Waypoints[i+1]

		dt1 := curr.Timestamp.Sub(prev.Timestamp).Seconds()
		dt2 := next.Timestamp.Sub(curr.Timestamp).Seconds()

		if dt1 <= 0 || dt2 <= 0 {
			dt1 = 1.0
			dt2 = 1.0
		}

		// Calculate numerical derivatives
		vel := Vector3D{
			X: (next.Position.X - prev.Position.X) / (dt1 + dt2),
			Y: (next.Position.Y - prev.Position.Y) / (dt1 + dt2),
			Z: (next.Position.Z - prev.Position.Z) / (dt1 + dt2),
		}

		acc := Vector3D{
			X: (next.Position.X - 2*curr.Position.X + prev.Position.X) / (dt1 * dt2),
			Y: (next.Position.Y - 2*curr.Position.Y + prev.Position.Y) / (dt1 * dt2),
			Z: (next.Position.Z - 2*curr.Position.Z + prev.Position.Z) / (dt1 * dt2),
		}

		// Physics residuals based on model
		speed := math.Sqrt(vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z)
		drag := model.DragCoefficient * speed * speed / (2 * model.Mass)

		// Position residual (velocity should match position derivative)
		residuals[i*6+0] = vel.X - curr.Velocity.X
		residuals[i*6+1] = vel.Y - curr.Velocity.Y
		residuals[i*6+2] = vel.Z - curr.Velocity.Z

		// Velocity residual (acceleration should follow physics)
		gravity := 9.81
		if model.GravityModel == "none" {
			gravity = 0.0
		}

		residuals[i*6+3] = acc.X + drag*vel.X/speed
		residuals[i*6+4] = acc.Y + drag*vel.Y/speed
		residuals[i*6+5] = acc.Z + gravity + drag*vel.Z/speed
	}

	return residuals
}

// calculateBoundaryErrors calculates boundary condition errors
func (p *PINNTrajectoryOptimizer) calculateBoundaryErrors(optimized, original *Trajectory) []float64 {
	errors := make([]float64, 2)

	if len(optimized.Waypoints) > 0 && len(original.Waypoints) > 0 {
		// Start boundary error
		errors[0] = math.Sqrt(Distance(optimized.Waypoints[0].Position, original.Waypoints[0].Position))

		// End boundary error
		lastOpt := len(optimized.Waypoints) - 1
		lastOrig := len(original.Waypoints) - 1
		errors[1] = math.Sqrt(Distance(optimized.Waypoints[lastOpt].Position, original.Waypoints[lastOrig].Position))
	}

	return errors
}

// calculatePhysicsCorrection calculates position correction based on physics
func (p *PINNTrajectoryOptimizer) calculatePhysicsCorrection(idx int, residuals []float64, model *PhysicsModel) Vector3D {
	if idx*6+5 >= len(residuals) {
		return Vector3D{}
	}

	// Use residuals to calculate correction
	learningRate := 0.01
	return Vector3D{
		X: -residuals[idx*6+0] * learningRate,
		Y: -residuals[idx*6+1] * learningRate,
		Z: -residuals[idx*6+2] * learningRate,
	}
}

// recalculateVelocities updates velocities based on positions
func (p *PINNTrajectoryOptimizer) recalculateVelocities(traj *Trajectory) {
	for i := 1; i < len(traj.Waypoints); i++ {
		dt := traj.Waypoints[i].Timestamp.Sub(traj.Waypoints[i-1].Timestamp).Seconds()
		if dt <= 0 {
			dt = 1.0
		}

		traj.Waypoints[i].Velocity = Vector3D{
			X: (traj.Waypoints[i].Position.X - traj.Waypoints[i-1].Position.X) / dt,
			Y: (traj.Waypoints[i].Position.Y - traj.Waypoints[i-1].Position.Y) / dt,
			Z: (traj.Waypoints[i].Position.Z - traj.Waypoints[i-1].Position.Z) / dt,
		}
	}
}

// ================================================================================
// Multi-Criteria Scoring
// ================================================================================

// scoreAll scores all trajectory candidates
func (s *MultiCriteriaScorer) scoreAll(candidates []*Trajectory, req TrajectoryRequest) []*ScoredTrajectory {
	scored := make([]*ScoredTrajectory, 0, len(candidates))

	for _, traj := range candidates {
		score := s.calculateScore(traj, req)
		scored = append(scored, &ScoredTrajectory{
			Trajectory: traj,
			Score:      score,
		})
	}

	return scored
}

// ScoredTrajectory pairs a trajectory with its score
type ScoredTrajectory struct {
	Trajectory *Trajectory
	Score      *TrajectoryScore
}

// calculateScore calculates comprehensive multi-criteria score
func (s *MultiCriteriaScorer) calculateScore(traj *Trajectory, req TrajectoryRequest) *TrajectoryScore {
	score := &TrajectoryScore{
		CriteriaScores: make(map[string]float64),
	}

	// Distance score (minimize)
	distanceScore := 1.0 / (1.0 + traj.TotalDistance/10000.0)
	score.CriteriaScores["distance"] = distanceScore

	// Time score (minimize)
	timeScore := 1.0 / (1.0 + float64(traj.EstimatedTime.Seconds())/3600.0)
	score.CriteriaScores["time"] = timeScore

	// Fuel score (minimize)
	fuelScore := 1.0 / (1.0 + traj.FuelRequired/100.0)
	score.CriteriaScores["fuel"] = fuelScore

	// Stealth score (maximize)
	score.CriteriaScores["stealth"] = traj.StealthScore

	// Threat exposure score (minimize)
	threatScore := 1.0 - traj.ThreatExposure
	score.CriteriaScores["threat_exposure"] = threatScore

	// Physics feasibility score
	physicsScore := calculatePhysicsFeasibility(traj)
	score.CriteriaScores["physics_feasibility"] = physicsScore

	// Smoothness score
	smoothnessScore := calculateSmoothness(traj)
	score.CriteriaScores["smoothness"] = smoothnessScore

	// Calculate weighted total
	totalScore := 0.0
	for name, criterion := range s.criteria {
		criterionScore := score.CriteriaScores[name]
		totalScore += criterionScore * criterion.Weight
	}

	// Apply priority-based adjustments
	switch req.Priority {
	case PriorityCritical:
		totalScore *= 1.0 + threatScore*0.5 // Extra weight on threat avoidance
	case PriorityHigh:
		totalScore *= 1.0 + (timeScore+threatScore)*0.25
	}

	score.TotalScore = totalScore

	return score
}

// calculatePhysicsFeasibility checks if trajectory is physically realistic
func calculatePhysicsFeasibility(traj *Trajectory) float64 {
	if len(traj.Waypoints) < 2 {
		return 0.0
	}

	violations := 0.0
	checks := 0.0

	for i := 1; i < len(traj.Waypoints); i++ {
		prev := traj.Waypoints[i-1]
		curr := traj.Waypoints[i]

		dt := curr.Timestamp.Sub(prev.Timestamp).Seconds()
		if dt <= 0 {
			dt = 1.0
		}

		// Check velocity consistency
		expectedVel := Vector3D{
			X: (curr.Position.X - prev.Position.X) / dt,
			Y: (curr.Position.Y - prev.Position.Y) / dt,
			Z: (curr.Position.Z - prev.Position.Z) / dt,
		}

		velError := math.Sqrt(Distance(expectedVel, curr.Velocity))
		if velError > 10.0 { // More than 10 m/s error
			violations++
		}
		checks++

		// Check acceleration limits
		if i > 1 {
			prevPrev := traj.Waypoints[i-2]
			dt1 := prev.Timestamp.Sub(prevPrev.Timestamp).Seconds()
			if dt1 <= 0 {
				dt1 = 1.0
			}

			acc := Vector3D{
				X: (curr.Velocity.X - prev.Velocity.X) / dt,
				Y: (curr.Velocity.Y - prev.Velocity.Y) / dt,
				Z: (curr.Velocity.Z - prev.Velocity.Z) / dt,
			}

			accMag := math.Sqrt(acc.X*acc.X + acc.Y*acc.Y + acc.Z*acc.Z)
			if accMag > curr.Constraints.MaxAcceleration*2 {
				violations++
			}
			checks++
		}
	}

	if checks == 0 {
		return 1.0
	}

	return 1.0 - (violations / checks)
}

// calculateSmoothness evaluates trajectory smoothness
func calculateSmoothness(traj *Trajectory) float64 {
	if len(traj.Waypoints) < 3 {
		return 1.0
	}

	totalJerk := 0.0
	for i := 2; i < len(traj.Waypoints); i++ {
		prev := traj.Waypoints[i-2]
		mid := traj.Waypoints[i-1]
		curr := traj.Waypoints[i]

		dt1 := mid.Timestamp.Sub(prev.Timestamp).Seconds()
		dt2 := curr.Timestamp.Sub(mid.Timestamp).Seconds()

		if dt1 <= 0 || dt2 <= 0 {
			continue
		}

		acc1 := Vector3D{
			X: (mid.Velocity.X - prev.Velocity.X) / dt1,
			Y: (mid.Velocity.Y - prev.Velocity.Y) / dt1,
			Z: (mid.Velocity.Z - prev.Velocity.Z) / dt1,
		}

		acc2 := Vector3D{
			X: (curr.Velocity.X - mid.Velocity.X) / dt2,
			Y: (curr.Velocity.Y - mid.Velocity.Y) / dt2,
			Z: (curr.Velocity.Z - mid.Velocity.Z) / dt2,
		}

		jerk := Vector3D{
			X: (acc2.X - acc1.X) / ((dt1 + dt2) / 2),
			Y: (acc2.Y - acc1.Y) / ((dt1 + dt2) / 2),
			Z: (acc2.Z - acc1.Z) / ((dt1 + dt2) / 2),
		}

		totalJerk += math.Sqrt(jerk.X*jerk.X + jerk.Y*jerk.Y + jerk.Z*jerk.Z)
	}

	avgJerk := totalJerk / float64(len(traj.Waypoints)-2)
	return 1.0 / (1.0 + avgJerk/10.0)
}

// findParetoFront identifies non-dominated solutions
func (s *MultiCriteriaScorer) findParetoFront(scored []*ScoredTrajectory) []*ScoredTrajectory {
	if len(scored) == 0 {
		return scored
	}

	paretoFront := make([]*ScoredTrajectory, 0)

	for i, candidate := range scored {
		dominated := false

		for j, other := range scored {
			if i == j {
				continue
			}

			if s.dominates(other.Score, candidate.Score) {
				dominated = true
				candidate.Score.DominatedBy++
			}
		}

		if !dominated {
			paretoFront = append(paretoFront, candidate)
		}
	}

	// Calculate crowding distance for diversity
	s.calculateCrowdingDistance(paretoFront)

	return paretoFront
}

// dominates checks if score1 Pareto-dominates score2
func (s *MultiCriteriaScorer) dominates(score1, score2 *TrajectoryScore) bool {
	atLeastOneStrictlyBetter := false

	for name := range s.criteria {
		val1 := score1.CriteriaScores[name]
		val2 := score2.CriteriaScores[name]

		if val1 < val2 {
			return false // score1 is worse in this criterion
		}
		if val1 > val2 {
			atLeastOneStrictlyBetter = true
		}
	}

	return atLeastOneStrictlyBetter
}

// calculateCrowdingDistance computes NSGA-II crowding distance
func (s *MultiCriteriaScorer) calculateCrowdingDistance(front []*ScoredTrajectory) {
	n := len(front)
	if n <= 2 {
		for _, st := range front {
			st.Score.CrowdingDistance = math.MaxFloat64
		}
		return
	}

	// Initialize distances
	for _, st := range front {
		st.Score.CrowdingDistance = 0.0
	}

	// Calculate for each criterion
	for name := range s.criteria {
		// Sort by this criterion
		sort.Slice(front, func(i, j int) bool {
			return front[i].Score.CriteriaScores[name] < front[j].Score.CriteriaScores[name]
		})

		// Boundary points get infinite distance
		front[0].Score.CrowdingDistance = math.MaxFloat64
		front[n-1].Score.CrowdingDistance = math.MaxFloat64

		// Calculate range
		minVal := front[0].Score.CriteriaScores[name]
		maxVal := front[n-1].Score.CriteriaScores[name]
		rangeVal := maxVal - minVal

		if rangeVal == 0 {
			continue
		}

		// Update intermediate distances
		for i := 1; i < n-1; i++ {
			front[i].Score.CrowdingDistance +=
				(front[i+1].Score.CriteriaScores[name] - front[i-1].Score.CriteriaScores[name]) / rangeVal
		}
	}
}

// selectFromParetoFront selects the best trajectory from the Pareto front
func (e *AIGuidanceEngine) selectFromParetoFront(front []*ScoredTrajectory, req TrajectoryRequest) *Trajectory {
	if len(front) == 0 {
		return nil
	}

	// Sort by combined metric based on priority
	sort.Slice(front, func(i, j int) bool {
		scoreI := front[i].Score.TotalScore
		scoreJ := front[j].Score.TotalScore

		// Add crowding distance as tiebreaker (prefer diversity)
		if math.Abs(scoreI-scoreJ) < 0.01 {
			return front[i].Score.CrowdingDistance > front[j].Score.CrowdingDistance
		}

		return scoreI > scoreJ
	})

	return front[0].Trajectory
}

// ================================================================================
// Real-Time Threat Adaptation
// ================================================================================

// updateAlertLevel updates the system alert level based on threats
func (a *RealTimeThreatAdapter) updateAlertLevel(threats []ThreatLocation) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(threats) == 0 {
		a.alertLevel = AlertNormal
		return
	}

	maxThreat := 0.0
	for _, threat := range threats {
		threatLevel := threat.Confidence
		if threatLevel > maxThreat {
			maxThreat = threatLevel
		}
	}

	switch {
	case maxThreat > 0.9:
		a.alertLevel = AlertCombat
	case maxThreat > 0.7:
		a.alertLevel = AlertCritical
	case maxThreat > 0.5:
		a.alertLevel = AlertHigh
	case maxThreat > 0.3:
		a.alertLevel = AlertElevated
	default:
		a.alertLevel = AlertNormal
	}
}

// adaptTrajectory modifies trajectory based on real-time threats
func (a *RealTimeThreatAdapter) adaptTrajectory(traj *Trajectory, threatDB map[string]ThreatLocation) *Trajectory {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.alertLevel == AlertNormal {
		return traj
	}

	adapted := &Trajectory{
		ID:          uuid.New().String(),
		PayloadType: traj.PayloadType,
		Waypoints:   make([]Waypoint, len(traj.Waypoints)),
		CreatedAt:   time.Now(),
	}

	copy(adapted.Waypoints, traj.Waypoints)

	// Analyze threats affecting the trajectory
	threatsByType := make(map[string][]ThreatLocation)
	for _, threat := range threatDB {
		// Check if threat affects any waypoint
		for _, wp := range traj.Waypoints {
			dist := math.Sqrt(Distance(wp.Position, threat.Position))
			if dist < threat.EffectRadius*1.5 { // 1.5x buffer
				threatsByType[threat.ThreatType] = append(threatsByType[threat.ThreatType], threat)
				break
			}
		}
	}

	// Apply evasion strategies
	for threatType, threats := range threatsByType {
		strategy := a.evasionStrategies[threatType]
		if strategy == nil {
			continue
		}

		// Modify waypoints based on strategy
		for i := 1; i < len(adapted.Waypoints)-1; i++ {
			wp := &adapted.Waypoints[i]

			// Check proximity to each threat
			for _, threat := range threats {
				dist := math.Sqrt(Distance(wp.Position, threat.Position))
				if dist < threat.EffectRadius {
					// Apply evasion
					switch strategy.ManeuverType {
					case "terrain_mask":
						wp.Position.Z = strategy.PreferredAltitude
					case "high_altitude":
						wp.Position.Z = math.Max(wp.Position.Z, strategy.PreferredAltitude)
					case "speed_burst":
						speedIncrease := 1.5
						wp.Velocity.X *= speedIncrease
						wp.Velocity.Y *= speedIncrease
						wp.Velocity.Z *= speedIncrease
					case "decoy":
						// Add lateral offset
						offset := 500.0
						wp.Position.X += offset * (rand.Float64() - 0.5)
						wp.Position.Y += offset * (rand.Float64() - 0.5)
					}
				}
			}
		}
	}

	// Recalculate trajectory metrics
	adapted.TotalDistance = calculateTrajDistanceStatic(adapted)
	adapted.Confidence = traj.Confidence * 0.9 // Slight confidence reduction for adaptation

	return adapted
}

// PredictThreatMovement predicts future threat positions
func (a *RealTimeThreatAdapter) PredictThreatMovement(threat *ThreatEvent, horizon time.Duration) []Vector3D {
	a.mu.RLock()
	defer a.mu.RUnlock()

	predictions := make([]Vector3D, 0)
	steps := int(horizon.Seconds())

	for i := 0; i <= steps; i++ {
		t := float64(i)
		predicted := Vector3D{
			X: threat.Location.X + threat.Velocity.X*t,
			Y: threat.Location.Y + threat.Velocity.Y*t,
			Z: threat.Location.Z + threat.Velocity.Z*t,
		}
		predictions = append(predictions, predicted)
	}

	return predictions
}

// RecordThreatEvent records a new threat event
func (a *RealTimeThreatAdapter) RecordThreatEvent(event ThreatEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	event.ID = uuid.New().String()
	event.DetectedAt = time.Now()

	a.threatHistory = append(a.threatHistory, event)

	// Keep only recent history
	if len(a.threatHistory) > 1000 {
		a.threatHistory = a.threatHistory[len(a.threatHistory)-1000:]
	}
}

// ================================================================================
// Experience Storage and Learning
// ================================================================================

// storeExperience stores trajectory experience for learning
func (e *AIGuidanceEngine) storeExperience(req TrajectoryRequest, traj *Trajectory) {
	e.experienceReplay.mu.Lock()
	defer e.experienceReplay.mu.Unlock()

	// Calculate reward based on trajectory quality
	reward := traj.Confidence * (1.0 - traj.ThreatExposure) * traj.StealthScore

	experience := Experience{
		State: e.createAgentState(req, e.payloadProfiles[req.PayloadType]),
		Reward:    reward,
		Done:      true,
		Timestamp: time.Now(),
	}

	e.experienceReplay.Experiences = append(e.experienceReplay.Experiences, experience)

	// Maintain buffer size
	if len(e.experienceReplay.Experiences) > e.experienceReplay.MaxSize {
		e.experienceReplay.Experiences = e.experienceReplay.Experiences[1:]
	}
}

// TrainFromExperience performs batch training on stored experiences
func (e *AIGuidanceEngine) TrainFromExperience(batchSize int) error {
	e.experienceReplay.mu.RLock()
	defer e.experienceReplay.mu.RUnlock()

	if len(e.experienceReplay.Experiences) < batchSize {
		return fmt.Errorf("insufficient experiences: %d < %d", len(e.experienceReplay.Experiences), batchSize)
	}

	// Sample random batch
	indices := rand.Perm(len(e.experienceReplay.Experiences))[:batchSize]

	for _, idx := range indices {
		exp := e.experienceReplay.Experiences[idx]

		// Update relevant agent based on state characteristics
		e.agentPool.mu.Lock()
		for _, agent := range e.agentPool.agents {
			// Simple reward-based update
			agent.TotalReward += exp.Reward
			agent.EpisodeCount++
		}
		e.agentPool.mu.Unlock()
	}

	return nil
}

// ================================================================================
// Original Methods (Updated)
// ================================================================================

// UpdateTrajectory recalculates path based on current state
func (e *AIGuidanceEngine) UpdateTrajectory(ctx context.Context, currentState State, traj *Trajectory) (*Trajectory, error) {
	// Find current position in trajectory
	currentWaypointIdx := e.findNearestWaypoint(currentState.Position, traj.Waypoints)

	// If significantly off-course, replan
	deviation := math.Sqrt(Distance(currentState.Position, traj.Waypoints[currentWaypointIdx].Position))

	if deviation > 100.0 { // More than 100m deviation
		// Replan from current position
		req := TrajectoryRequest{
			PayloadType:    traj.PayloadType,
			StartPosition:  currentState.Position,
			TargetPosition: traj.Waypoints[len(traj.Waypoints)-1].Position,
			Priority:       PriorityHigh,
			StealthMode:    StealthModeMedium,
		}

		return e.PlanTrajectory(ctx, req)
	}

	// Check for new threats and adapt if necessary
	e.mu.RLock()
	threats := make([]ThreatLocation, 0, len(e.threatDB))
	for _, t := range e.threatDB {
		threats = append(threats, t)
	}
	e.mu.RUnlock()

	if len(threats) > 0 {
		e.threatAdapter.updateAlertLevel(threats)
		if e.threatAdapter.alertLevel != AlertNormal {
			return e.threatAdapter.adaptTrajectory(traj, e.threatDB), nil
		}
	}

	// Minor adjustments
	return traj, nil
}

// ValidateTrajectory checks physics and constraints
func (e *AIGuidanceEngine) ValidateTrajectory(traj *Trajectory) error {
	if len(traj.Waypoints) < 2 {
		return fmt.Errorf("trajectory must have at least 2 waypoints")
	}

	profile := e.payloadProfiles[traj.PayloadType]

	for i := 1; i < len(traj.Waypoints); i++ {
		prev := traj.Waypoints[i-1]
		curr := traj.Waypoints[i]

		// Check time ordering
		if !curr.Timestamp.After(prev.Timestamp) {
			return fmt.Errorf("waypoint %d has invalid timestamp", i)
		}

		// Check velocity limits
		speed := math.Sqrt(Magnitude(curr.Velocity))
		maxSpeed := curr.Constraints.MaxSpeed
		if profile != nil && profile.MaxSpeed > 0 {
			maxSpeed = profile.MaxSpeed
		}
		if speed > maxSpeed*1.1 { // Allow 10% tolerance
			return fmt.Errorf("waypoint %d exceeds max speed: %.2f > %.2f", i, speed, maxSpeed)
		}

		// Check altitude constraints
		minAlt := curr.Constraints.MinAltitude
		maxAlt := curr.Constraints.MaxAltitude
		if profile != nil {
			minAlt = profile.MinAltitude
			maxAlt = profile.MaxAltitude
		}

		if curr.Position.Z < minAlt {
			return fmt.Errorf("waypoint %d below minimum altitude: %.2f < %.2f", i, curr.Position.Z, minAlt)
		}
		if curr.Position.Z > maxAlt {
			return fmt.Errorf("waypoint %d above maximum altitude: %.2f > %.2f", i, curr.Position.Z, maxAlt)
		}
	}

	return nil
}

// OptimizeForStealth minimizes detection probability
func (e *AIGuidanceEngine) OptimizeForStealth(traj *Trajectory) (*Trajectory, error) {
	if e.stealthModule != nil {
		return e.stealthModule.OptimizeTrajectory(traj, StealthModeMaximum)
	}

	// Fallback: lower altitude and reduce speed
	optimized := &Trajectory{
		ID:          uuid.New().String(),
		PayloadType: traj.PayloadType,
		Waypoints:   make([]Waypoint, len(traj.Waypoints)),
		CreatedAt:   time.Now(),
	}

	copy(optimized.Waypoints, traj.Waypoints)

	profile := e.payloadProfiles[traj.PayloadType]
	targetAlt := 100.0 // Low altitude for stealth
	if profile != nil {
		targetAlt = math.Max(profile.MinAltitude, 100.0)
	}

	for i := range optimized.Waypoints {
		optimized.Waypoints[i].Position.Z = targetAlt
		// Reduce velocity by 30%
		optimized.Waypoints[i].Velocity.X *= 0.7
		optimized.Waypoints[i].Velocity.Y *= 0.7
		optimized.Waypoints[i].Velocity.Z *= 0.7
	}

	optimized.StealthScore = 0.95
	optimized.TotalDistance = e.calculateTrajDistance(optimized)
	optimized.FuelRequired = e.estimateFuel(optimized)

	return optimized, nil
}

// OptimizeForSpeed maximizes arrival time
func (e *AIGuidanceEngine) OptimizeForSpeed(traj *Trajectory) (*Trajectory, error) {
	optimized := &Trajectory{
		ID:          uuid.New().String(),
		PayloadType: traj.PayloadType,
		Waypoints:   make([]Waypoint, 0),
		CreatedAt:   time.Now(),
	}

	profile := e.payloadProfiles[traj.PayloadType]

	// Use fewer waypoints, direct route
	start := traj.Waypoints[0]
	end := traj.Waypoints[len(traj.Waypoints)-1]

	// Calculate optimal velocity
	direction := Vector3D{
		X: end.Position.X - start.Position.X,
		Y: end.Position.Y - start.Position.Y,
		Z: end.Position.Z - start.Position.Z,
	}
	distance := math.Sqrt(Magnitude(direction))

	maxSpeed := start.Constraints.MaxSpeed
	if profile != nil && profile.MaxSpeed > 0 {
		maxSpeed = profile.MaxSpeed
	}
	if maxSpeed == 0 {
		maxSpeed = 100.0 // Default 100 m/s
	}

	travelTime := distance / maxSpeed

	// Create direct waypoint
	midpoint := Waypoint{
		Position: Vector3D{
			X: (start.Position.X + end.Position.X) / 2,
			Y: (start.Position.Y + end.Position.Y) / 2,
			Z: math.Max(start.Position.Z, end.Position.Z),
		},
		Velocity: Vector3D{
			X: (direction.X / distance) * maxSpeed,
			Y: (direction.Y / distance) * maxSpeed,
			Z: (direction.Z / distance) * maxSpeed,
		},
		Timestamp:   start.Timestamp.Add(time.Duration(travelTime/2) * time.Second),
		Constraints: start.Constraints,
	}

	optimized.Waypoints = append(optimized.Waypoints, start, midpoint, end)
	optimized.TotalDistance = distance
	optimized.EstimatedTime = time.Duration(travelTime) * time.Second

	return optimized, nil
}

// OptimizeForFuel minimizes energy consumption
func (e *AIGuidanceEngine) OptimizeForFuel(traj *Trajectory) (*Trajectory, error) {
	optimized := &Trajectory{
		ID:          uuid.New().String(),
		PayloadType: traj.PayloadType,
		Waypoints:   make([]Waypoint, len(traj.Waypoints)),
		CreatedAt:   time.Now(),
	}

	copy(optimized.Waypoints, traj.Waypoints)

	// Smooth accelerations to reduce fuel consumption
	for i := 1; i < len(optimized.Waypoints)-1; i++ {
		prev := optimized.Waypoints[i-1]
		next := optimized.Waypoints[i+1]

		// Average velocities for smoother transitions
		optimized.Waypoints[i].Velocity = Vector3D{
			X: (prev.Velocity.X + next.Velocity.X) / 2,
			Y: (prev.Velocity.Y + next.Velocity.Y) / 2,
			Z: (prev.Velocity.Z + next.Velocity.Z) / 2,
		}
	}

	optimized.FuelRequired = e.estimateFuel(optimized)
	optimized.TotalDistance = e.calculateTrajDistance(optimized)

	return optimized, nil
}

// RegisterThreat adds a threat to the database
func (e *AIGuidanceEngine) RegisterThreat(threat ThreatLocation) {
	e.mu.Lock()
	defer e.mu.Unlock()

	threatID := fmt.Sprintf("threat-%d-%d-%d", int(threat.Position.X), int(threat.Position.Y), int(threat.Position.Z))
	e.threatDB[threatID] = threat

	// Update threat adapter
	threats := make([]ThreatLocation, 0, len(e.threatDB))
	for _, t := range e.threatDB {
		threats = append(threats, t)
	}

	e.threatAdapter.updateAlertLevel(threats)
}

// ClearThreat removes a threat from the database
func (e *AIGuidanceEngine) ClearThreat(threatID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.threatDB, threatID)
}

// GetAlertLevel returns the current alert level
func (e *AIGuidanceEngine) GetAlertLevel() AlertLevel {
	e.threatAdapter.mu.RLock()
	defer e.threatAdapter.mu.RUnlock()
	return e.threatAdapter.alertLevel
}

// GetPayloadProfile returns the profile for a payload type
func (e *AIGuidanceEngine) GetPayloadProfile(payloadType PayloadType) *PayloadProfile {
	return e.payloadProfiles[payloadType]
}

// GetModelVersion returns the engine version
func (e *AIGuidanceEngine) GetModelVersion() string {
	return e.modelVersion
}

// ================================================================================
// Helper functions
// ================================================================================

func (e *AIGuidanceEngine) generateCandidates(req TrajectoryRequest, count int) []*Trajectory {
	candidates := make([]*Trajectory, 0, count)

	for i := 0; i < count; i++ {
		traj := &Trajectory{
			ID:          uuid.New().String(),
			PayloadType: req.PayloadType,
			Waypoints:   e.generateWaypoints(req, i),
			CreatedAt:   time.Now(),
		}

		traj.TotalDistance = e.calculateTrajDistance(traj)
		traj.StealthScore = e.calculateStealthScore(traj)
		traj.ThreatExposure = e.calculateThreatExposure(traj, req.Constraints.MustAvoidThreats)
		traj.FuelRequired = e.estimateFuel(traj)
		traj.Confidence = 0.85 + (float64(i%3) * 0.05) // Vary confidence

		candidates = append(candidates, traj)
	}

	return candidates
}

func (e *AIGuidanceEngine) generateWaypoints(req TrajectoryRequest, variant int) []Waypoint {
	waypoints := make([]Waypoint, 0)

	profile := e.payloadProfiles[req.PayloadType]
	maxSpeed := 100.0
	maxAccel := 10.0
	minAlt := 0.0
	maxAlt := 10000.0

	if profile != nil {
		maxSpeed = profile.MaxSpeed
		maxAccel = profile.MaxAcceleration
		minAlt = profile.MinAltitude
		maxAlt = profile.MaxAltitude
	}

	// Start waypoint
	startWP := Waypoint{
		Position:  req.StartPosition,
		Velocity:  Vector3D{X: 0, Y: 0, Z: 0},
		Timestamp: time.Now(),
		Constraints: WaypointConstraints{
			MaxSpeed:        maxSpeed,
			MaxAcceleration: maxAccel,
			StealthRequired: req.Constraints.StealthRequired,
			MinAltitude:     minAlt,
			MaxAltitude:     maxAlt,
		},
	}
	waypoints = append(waypoints, startWP)

	// Mid waypoints (vary based on variant)
	numMidPoints := 3 + (variant % 5)
	for i := 1; i <= numMidPoints; i++ {
		progress := float64(i) / float64(numMidPoints+1)

		// Vary altitude based on variant and payload domain
		altitude := (minAlt + maxAlt) / 2
		if profile != nil {
			switch profile.OperatingDomain {
			case "ground":
				altitude = 0
			case "underwater":
				altitude = -100 - float64(variant%5)*50
			case "air":
				altitude = 1000 + float64(variant%5)*500
			case "space":
				altitude = 300000 + float64(variant%5)*50000
			case "interstellar":
				altitude = 0 // Position is relative in interstellar
			}
		}

		midWP := Waypoint{
			Position: Vector3D{
				X: req.StartPosition.X + (req.TargetPosition.X-req.StartPosition.X)*progress,
				Y: req.StartPosition.Y + (req.TargetPosition.Y-req.StartPosition.Y)*progress,
				Z: altitude,
			},
			Velocity:  Vector3D{X: maxSpeed / 2, Y: maxSpeed / 2, Z: 0},
			Timestamp: startWP.Timestamp.Add(time.Duration(progress*60) * time.Second),
			Constraints: startWP.Constraints,
		}
		waypoints = append(waypoints, midWP)
	}

	// End waypoint
	endWP := Waypoint{
		Position:    req.TargetPosition,
		Velocity:    Vector3D{X: 0, Y: 0, Z: 0},
		Timestamp:   startWP.Timestamp.Add(60 * time.Second),
		Constraints: startWP.Constraints,
	}
	waypoints = append(waypoints, endWP)

	return waypoints
}

func (e *AIGuidanceEngine) findNearestWaypoint(pos Vector3D, waypoints []Waypoint) int {
	minDist := math.MaxFloat64
	minIdx := 0

	for i, wp := range waypoints {
		dist := math.Sqrt(Distance(pos, wp.Position))
		if dist < minDist {
			minDist = dist
			minIdx = i
		}
	}

	return minIdx
}

func (e *AIGuidanceEngine) calculateTrajDistance(traj *Trajectory) float64 {
	return calculateTrajDistanceStatic(traj)
}

func calculateTrajDistanceStatic(traj *Trajectory) float64 {
	total := 0.0
	for i := 1; i < len(traj.Waypoints); i++ {
		dist := math.Sqrt(Distance(traj.Waypoints[i-1].Position, traj.Waypoints[i].Position))
		total += dist
	}
	return total
}

func (e *AIGuidanceEngine) calculateStealthScore(traj *Trajectory) float64 {
	profile := e.payloadProfiles[traj.PayloadType]
	baseCapability := 0.5
	if profile != nil {
		baseCapability = profile.StealthCapability
	}

	// Simplified: lower altitude and slower speed = higher stealth
	score := baseCapability
	for _, wp := range traj.Waypoints {
		altitudeFactor := 1.0 - (wp.Position.Z / 10000.0)
		if altitudeFactor < 0 {
			altitudeFactor = 0
		}
		if altitudeFactor > 1 {
			altitudeFactor = 1
		}
		speed := math.Sqrt(Magnitude(wp.Velocity))
		maxSpeed := 100.0
		if profile != nil {
			maxSpeed = profile.MaxSpeed
		}
		speedFactor := 1.0 - (speed / maxSpeed)
		if speedFactor < 0 {
			speedFactor = 0
		}
		score *= (altitudeFactor + speedFactor) / 2.0
	}
	return math.Max(0, math.Min(1, score*baseCapability*2))
}

func (e *AIGuidanceEngine) calculateThreatExposure(traj *Trajectory, threats []ThreatLocation) float64 {
	if len(threats) == 0 {
		return 0.0
	}

	totalExposure := 0.0
	for _, wp := range traj.Waypoints {
		for _, threat := range threats {
			dist := math.Sqrt(Distance(wp.Position, threat.Position))
			if dist < threat.EffectRadius {
				exposure := (1.0 - (dist / threat.EffectRadius)) * threat.Confidence
				totalExposure += exposure
			}
		}
	}

	return math.Min(1.0, totalExposure/float64(len(traj.Waypoints)))
}

func (e *AIGuidanceEngine) estimateFuel(traj *Trajectory) float64 {
	profile := e.payloadProfiles[traj.PayloadType]
	consumptionRate := 0.1
	if profile != nil {
		consumptionRate = 1.0 - profile.FuelEfficiency + 0.01
	}

	// Fuel model: sum of velocity changes
	fuel := 0.0
	for i := 1; i < len(traj.Waypoints); i++ {
		deltaV := Vector3D{
			X: traj.Waypoints[i].Velocity.X - traj.Waypoints[i-1].Velocity.X,
			Y: traj.Waypoints[i].Velocity.Y - traj.Waypoints[i-1].Velocity.Y,
			Z: traj.Waypoints[i].Velocity.Z - traj.Waypoints[i-1].Velocity.Z,
		}
		deltaVMag := math.Sqrt(Magnitude(deltaV))
		fuel += deltaVMag * consumptionRate
	}
	return fuel
}

// sigmoid activation function
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}
