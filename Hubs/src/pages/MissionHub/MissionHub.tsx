import React, { useState, useEffect, useCallback } from 'react';
import { 
  Shield, Lock, Eye, Radio, MapPin, AlertTriangle, 
  Satellite, Bot, Rocket, Target, Users, Activity,
  Play, Pause, Volume2, VolumeX, Maximize2, Settings,
  ChevronRight, RefreshCw, Clock, Zap
} from 'lucide-react';

// Types
interface ClearanceLevel {
  level: number;
  name: string;
  color: string;
  bgColor: string;
}

interface Mission {
  id: string;
  name: string;
  type: string;
  classification: string;
  clearance: number;
  status: string;
  payloadId: string;
  payloadType: string;
  startTime: string;
  eta?: string;
  progress: number;
  hasLiveFeed: boolean;
  viewers: number;
}

// Reserved for live feed integration
// eslint-disable-next-line @typescript-eslint/no-unused-vars
interface LiveFeedType {
  id: string;
  missionId: string;
  payloadId: string;
  payloadType: string;
  streamType: string;
  clearance: number;
  name: string;
  status: string;
  viewerCount: number;
  quality: string;
}

interface TelemetryFrame {
  payloadId: string;
  position: { x: number; y: number; z: number };
  geoPosition?: { latitude: number; longitude: number; altitude: number };
  velocity: { x: number; y: number; z: number };
  heading: number;
  speed: number;
  fuel: number;
  battery: number;
  status: string;
  missionPhase: string;
  eta?: string;
  distanceRemaining: number;
}

interface Terminal {
  id: string;
  name: string;
  type: string;
  location: string;
  clearance: number;
  status: string;
  capabilities: string[];
}

// Clearance levels configuration
const CLEARANCE_LEVELS: ClearanceLevel[] = [
  { level: 0, name: 'PUBLIC', color: '#22c55e', bgColor: 'rgba(34, 197, 94, 0.2)' },
  { level: 1, name: 'CIVILIAN', color: '#3b82f6', bgColor: 'rgba(59, 130, 246, 0.2)' },
  { level: 2, name: 'MILITARY', color: '#f59e0b', bgColor: 'rgba(245, 158, 11, 0.2)' },
  { level: 3, name: 'GOVERNMENT', color: '#8b5cf6', bgColor: 'rgba(139, 92, 246, 0.2)' },
  { level: 4, name: 'SECRET', color: '#ef4444', bgColor: 'rgba(239, 68, 68, 0.2)' },
  { level: 5, name: 'ULTRA', color: '#ec4899', bgColor: 'rgba(236, 72, 153, 0.2)' },
];

const getClearanceInfo = (level: number): ClearanceLevel => {
  return CLEARANCE_LEVELS[level] || CLEARANCE_LEVELS[0];
};

// API configuration
const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';

// API functions
const fetchMissions = async (clearance: number): Promise<Mission[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/missions?clearance=${clearance}&status=active`);
    if (!response.ok) throw new Error('Failed to fetch missions');
    const data = await response.json();
    return data.missions || [];
  } catch (error) {
    console.error('Error fetching missions:', error);
    return [];
  }
};

const fetchTelemetry = async (payloadId: string): Promise<TelemetryFrame | null> => {
  try {
    const response = await fetch(`${API_BASE_URL}/telemetry/${payloadId}`);
    if (!response.ok) throw new Error('Failed to fetch telemetry');
    return await response.json();
  } catch (error) {
    console.error('Error fetching telemetry:', error);
    return null;
  }
};

const fetchTerminals = async (clearance: number): Promise<Terminal[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/terminals?clearance=${clearance}`);
    if (!response.ok) throw new Error('Failed to fetch terminals');
    const data = await response.json();
    return data.terminals || [];
  } catch (error) {
    console.error('Error fetching terminals:', error);
    return [];
  }
};

// Components
const ClearanceBadge: React.FC<{ clearance: number }> = ({ clearance }) => {
  const info = getClearanceInfo(clearance);
  return (
    <span 
      className="clearance-badge"
      style={{ 
        backgroundColor: info.bgColor, 
        color: info.color,
        border: `1px solid ${info.color}` 
      }}
    >
      <Lock size={12} />
      {info.name}
    </span>
  );
};

const PayloadIcon: React.FC<{ type: string; size?: number }> = ({ type, size = 20 }) => {
  switch (type) {
    case 'hunoid':
      return <Bot size={size} />;
    case 'uav':
    case 'drone':
      return <Satellite size={size} />;
    case 'rocket':
    case 'spacecraft':
      return <Rocket size={size} />;
    case 'missile':
      return <Target size={size} />;
    default:
      return <Radio size={size} />;
  }
};

const MissionCard: React.FC<{ 
  mission: Mission; 
  onSelect: (mission: Mission) => void;
  selected: boolean;
}> = ({ mission, onSelect, selected }) => {
  const clearanceInfo = getClearanceInfo(mission.clearance);
  
  return (
    <div 
      className={`mission-card ${selected ? 'selected' : ''}`}
      onClick={() => onSelect(mission)}
      style={{ borderColor: selected ? clearanceInfo.color : undefined }}
    >
      <div className="mission-header">
        <div className="mission-icon" style={{ backgroundColor: clearanceInfo.bgColor }}>
          <PayloadIcon type={mission.payloadType} />
        </div>
        <div className="mission-info">
          <h3>{mission.name}</h3>
          <ClearanceBadge clearance={mission.clearance} />
        </div>
        {mission.hasLiveFeed && (
          <div className="live-indicator">
            <span className="live-dot" />
            LIVE
          </div>
        )}
      </div>
      
      <div className="mission-progress">
        <div className="progress-bar">
          <div 
            className="progress-fill" 
            style={{ 
              width: `${mission.progress}%`,
              backgroundColor: clearanceInfo.color 
            }} 
          />
        </div>
        <span className="progress-text">{mission.progress}%</span>
      </div>
      
      <div className="mission-meta">
        <span><Clock size={14} /> {mission.eta || 'In progress'}</span>
        <span><Users size={14} /> {mission.viewers} watching</span>
      </div>
    </div>
  );
};

const LiveFeedViewer: React.FC<{ 
  mission: Mission | null;
  telemetry: TelemetryFrame | null;
}> = ({ mission, telemetry }) => {
  const [isPlaying, setIsPlaying] = useState(true);
  const [isMuted, setIsMuted] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);

  if (!mission) {
    return (
      <div className="feed-placeholder">
        <Radio size={64} className="placeholder-icon" />
        <p>Select a mission to view live feed</p>
      </div>
    );
  }

  const clearanceInfo = getClearanceInfo(mission.clearance);

  return (
    <div className="live-feed-viewer">
      <div className="feed-header">
        <div className="feed-title">
          <PayloadIcon type={mission.payloadType} />
          <span>{mission.name}</span>
        </div>
        <ClearanceBadge clearance={mission.clearance} />
      </div>

      <div className="video-container">
        <div className="video-placeholder" style={{ borderColor: clearanceInfo.color }}>
          <div className="video-overlay">
            <Activity size={48} className="pulse" />
            <p>Live Feed: {mission.payloadId.toUpperCase()}</p>
            <p className="feed-quality">1080p • 30fps • Encrypted</p>
          </div>
          
          {/* Simulated HUD overlay */}
          <div className="hud-overlay">
            <div className="hud-top-left">
              <span className="hud-item">ALT: {telemetry?.geoPosition?.altitude.toFixed(0) || '---'} m</span>
              <span className="hud-item">SPD: {telemetry?.speed.toFixed(1) || '---'} m/s</span>
            </div>
            <div className="hud-top-right">
              <span className="hud-item">FUEL: {telemetry?.fuel.toFixed(0) || '--'}%</span>
              <span className="hud-item">BAT: {telemetry?.battery.toFixed(0) || '--'}%</span>
            </div>
            <div className="hud-bottom">
              <span className="hud-coords">
                {telemetry?.geoPosition 
                  ? `${telemetry.geoPosition.latitude.toFixed(4)}°N, ${telemetry.geoPosition.longitude.toFixed(4)}°W`
                  : 'Acquiring GPS...'}
              </span>
            </div>
          </div>
        </div>

        <div className="video-controls">
          <button onClick={() => setIsPlaying(!isPlaying)}>
            {isPlaying ? <Pause size={20} /> : <Play size={20} />}
          </button>
          <button onClick={() => setIsMuted(!isMuted)}>
            {isMuted ? <VolumeX size={20} /> : <Volume2 size={20} />}
          </button>
          <div className="control-spacer" />
          <button onClick={() => setIsFullscreen(!isFullscreen)}>
            <Maximize2 size={20} />
          </button>
          <button>
            <Settings size={20} />
          </button>
        </div>
      </div>
    </div>
  );
};

const TelemetryPanel: React.FC<{ telemetry: TelemetryFrame | null }> = ({ telemetry }) => {
  if (!telemetry) {
    return (
      <div className="telemetry-panel empty">
        <p>No telemetry data</p>
      </div>
    );
  }

  return (
    <div className="telemetry-panel">
      <h3><Activity size={18} /> Live Telemetry</h3>
      
      <div className="telemetry-grid">
        <div className="telemetry-item">
          <label>Position</label>
          <span>
            ({telemetry.position.x.toFixed(0)}, {telemetry.position.y.toFixed(0)}, {telemetry.position.z.toFixed(0)})
          </span>
        </div>
        <div className="telemetry-item">
          <label>Velocity</label>
          <span>{telemetry.speed.toFixed(1)} m/s</span>
        </div>
        <div className="telemetry-item">
          <label>Heading</label>
          <span>{(telemetry.heading * 180 / Math.PI).toFixed(1)}°</span>
        </div>
        <div className="telemetry-item">
          <label>Altitude</label>
          <span>{telemetry.geoPosition?.altitude.toFixed(0) || '---'} m</span>
        </div>
        <div className="telemetry-item">
          <label>Status</label>
          <span className="status-operational">{telemetry.status}</span>
        </div>
        <div className="telemetry-item">
          <label>Phase</label>
          <span>{telemetry.missionPhase}</span>
        </div>
        <div className="telemetry-item">
          <label>ETA</label>
          <span>{telemetry.eta || 'Calculating...'}</span>
        </div>
        <div className="telemetry-item">
          <label>Distance</label>
          <span>{(telemetry.distanceRemaining / 1000).toFixed(1)} km</span>
        </div>
      </div>

      <div className="telemetry-bars">
        <div className="bar-item">
          <label>Fuel</label>
          <div className="bar-track">
            <div 
              className="bar-fill fuel" 
              style={{ width: `${telemetry.fuel}%` }}
            />
          </div>
          <span>{telemetry.fuel.toFixed(0)}%</span>
        </div>
        <div className="bar-item">
          <label>Battery</label>
          <div className="bar-track">
            <div 
              className="bar-fill battery" 
              style={{ width: `${telemetry.battery}%` }}
            />
          </div>
          <span>{telemetry.battery.toFixed(0)}%</span>
        </div>
      </div>
    </div>
  );
};

const AccessTerminalSelector: React.FC<{
  terminals: Terminal[];
  selectedTerminal: Terminal | null;
  onSelect: (terminal: Terminal) => void;
  userClearance: number;
}> = ({ terminals, selectedTerminal, onSelect, userClearance }) => {
  const accessibleTerminals = terminals.filter(t => t.clearance <= userClearance);

  return (
    <div className="terminal-selector">
      <h3><Shield size={18} /> Access Terminals</h3>
      <div className="terminal-list">
        {accessibleTerminals.map(terminal => (
          <div 
            key={terminal.id}
            className={`terminal-item ${selectedTerminal?.id === terminal.id ? 'selected' : ''}`}
            onClick={() => onSelect(terminal)}
          >
            <div className="terminal-icon">
              <MapPin size={20} />
            </div>
            <div className="terminal-info">
              <span className="terminal-name">{terminal.name}</span>
              <span className="terminal-location">{terminal.location}</span>
            </div>
            <ClearanceBadge clearance={terminal.clearance} />
          </div>
        ))}
      </div>
    </div>
  );
};

// Main Component
export const MissionHub: React.FC = () => {
  const [userClearance, setUserClearance] = useState<number>(0);
  const [missions, setMissions] = useState<Mission[]>([]);
  const [selectedMission, setSelectedMission] = useState<Mission | null>(null);
  const [telemetry, setTelemetry] = useState<TelemetryFrame | null>(null);
  const [terminals, setTerminals] = useState<Terminal[]>([]);
  const [selectedTerminal, setSelectedTerminal] = useState<Terminal | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  // Initialize data - fetch terminals
  useEffect(() => {
    const loadTerminals = async () => {
      const terminalsData = await fetchTerminals(userClearance);
      setTerminals(terminalsData);
    };
    loadTerminals();
  }, [userClearance]);

  // Update missions when clearance changes
  useEffect(() => {
    const loadMissions = async () => {
      const missionsData = await fetchMissions(userClearance);
      setMissions(missionsData);
    };
    loadMissions();
  }, [userClearance]);

  // Real-time telemetry updates via polling
  useEffect(() => {
    if (!selectedMission) {
      setTelemetry(null);
      return;
    }

    // Initial fetch
    const loadTelemetry = async () => {
      const telemData = await fetchTelemetry(selectedMission.payloadId);
      if (telemData) setTelemetry(telemData);
    };
    loadTelemetry();

    // Poll for updates every second
    const interval = setInterval(loadTelemetry, 1000);

    return () => clearInterval(interval);
  }, [selectedMission]);

  const handleLogin = (clearance: number) => {
    setUserClearance(clearance);
    setIsAuthenticated(true);
  };

  const handleMissionSelect = useCallback(async (mission: Mission) => {
    setSelectedMission(mission);
    const telemData = await fetchTelemetry(mission.payloadId);
    if (telemData) setTelemetry(telemData);
  }, []);

  const refreshMissions = useCallback(async () => {
    const missionsData = await fetchMissions(userClearance);
    setMissions(missionsData);
  }, [userClearance]);

  // Login screen
  if (!isAuthenticated) {
    return (
      <div className="mission-hub login-screen">
        <div className="login-container">
          <div className="login-header">
            <Shield size={48} className="login-icon" />
            <h1>ASGARD MISSION HUB</h1>
            <p>Select Access Level</p>
          </div>
          
          <div className="clearance-selector">
            {CLEARANCE_LEVELS.map(level => (
              <button
                key={level.level}
                className="clearance-button"
                style={{ 
                  backgroundColor: level.bgColor,
                  borderColor: level.color,
                  color: level.color
                }}
                onClick={() => handleLogin(level.level)}
              >
                <Lock size={20} />
                <span className="clearance-name">{level.name}</span>
                <ChevronRight size={20} />
              </button>
            ))}
          </div>
          
          <p className="login-note">
            <AlertTriangle size={16} />
            Access is logged and monitored
          </p>
        </div>
      </div>
    );
  }

  const clearanceInfo = getClearanceInfo(userClearance);

  return (
    <div className="mission-hub">
      {/* Header */}
      <header className="hub-header" style={{ borderBottomColor: clearanceInfo.color }}>
        <div className="header-left">
          <Shield size={32} style={{ color: clearanceInfo.color }} />
          <h1>ASGARD MISSION HUB</h1>
        </div>
        <div className="header-center">
          <ClearanceBadge clearance={userClearance} />
          {selectedTerminal && (
            <span className="terminal-badge">
              <MapPin size={14} />
              {selectedTerminal.name}
            </span>
          )}
        </div>
        <div className="header-right">
          <button className="refresh-btn" onClick={refreshMissions}>
            <RefreshCw size={18} />
          </button>
          <button className="logout-btn" onClick={() => setIsAuthenticated(false)}>
            <Lock size={18} />
            Logout
          </button>
        </div>
      </header>

      {/* Main Content */}
      <main className="hub-main">
        {/* Left Panel - Mission List */}
        <aside className="missions-panel">
          <div className="panel-header">
            <h2><Zap size={20} /> Active Missions</h2>
            <span className="mission-count">{missions.length}</span>
          </div>
          <div className="missions-list">
            {missions.map(mission => (
              <MissionCard
                key={mission.id}
                mission={mission}
                onSelect={handleMissionSelect}
                selected={selectedMission?.id === mission.id}
              />
            ))}
            {missions.length === 0 && (
              <div className="no-missions">
                <Eye size={32} />
                <p>No missions available at your clearance level</p>
              </div>
            )}
          </div>
        </aside>

        {/* Center - Live Feed */}
        <section className="feed-panel">
          <LiveFeedViewer mission={selectedMission} telemetry={telemetry} />
        </section>

        {/* Right Panel - Telemetry & Terminals */}
        <aside className="info-panel">
          <TelemetryPanel telemetry={telemetry} />
          <AccessTerminalSelector
            terminals={terminals}
            selectedTerminal={selectedTerminal}
            onSelect={setSelectedTerminal}
            userClearance={userClearance}
          />
        </aside>
      </main>

      {/* Footer */}
      <footer className="hub-footer">
        <span>PERCILA Guidance System v1.0.0</span>
        <span>|</span>
        <span>{new Date().toLocaleString()}</span>
        <span>|</span>
        <span style={{ color: clearanceInfo.color }}>
          CLEARANCE: {clearanceInfo.name}
        </span>
      </footer>
    </div>
  );
};

export default MissionHub;
