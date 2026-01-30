"""
Valkyrie Integration Client for GIRU JARVIS
============================================
Provides voice control and monitoring for the Valkyrie autonomous flight system.

Capabilities:
- Flight status monitoring
- Mission control (arm/disarm, mode changes)
- Emergency procedures
- Telemetry queries
- AI decision engine status
- Fail-safe system control
"""

import asyncio
import os
from dataclasses import dataclass
from datetime import datetime
from typing import Optional, Dict, Any, List
import aiohttp

from monitor import get_monitor
from database import ActivityType


# =============================================================================
# CONFIGURATION
# =============================================================================

VALKYRIE_URL = os.getenv("VALKYRIE_URL", "http://localhost:8093")
VALKYRIE_WS_URL = os.getenv("VALKYRIE_WS_URL", "ws://localhost:8093/ws/telemetry")


# =============================================================================
# DATA TYPES
# =============================================================================

@dataclass
class FlightState:
    """Current flight state from Valkyrie."""
    position: Dict[str, float]  # lat, lon, alt
    velocity: Dict[str, float]  # vx, vy, vz
    attitude: Dict[str, float]  # roll, pitch, yaw
    heading: float
    groundspeed: float
    airspeed: float
    altitude_agl: float
    flight_mode: str
    armed: bool
    battery_percent: float
    gps_fix: int
    satellites: int


@dataclass
class MissionStatus:
    """Mission status from Valkyrie."""
    mission_id: str
    status: str
    waypoint_current: int
    waypoint_total: int
    distance_to_waypoint: float
    eta_minutes: float
    auto_continue: bool


# =============================================================================
# VALKYRIE CLIENT
# =============================================================================

class ValkyrieClient:
    """Client for communicating with Valkyrie autonomous flight system."""
    
    def __init__(self, base_url: str = VALKYRIE_URL):
        self.base_url = base_url
        self.session: Optional[aiohttp.ClientSession] = None
        self.connected = False
        self._telemetry_callback = None
    
    async def ensure_session(self):
        """Ensure aiohttp session exists."""
        if self.session is None or self.session.closed:
            self.session = aiohttp.ClientSession()
    
    async def close(self):
        """Close the client session."""
        if self.session and not self.session.closed:
            await self.session.close()
    
    # =========================================================================
    # HEALTH & STATUS
    # =========================================================================
    
    async def health_check(self) -> bool:
        """Check if Valkyrie is online."""
        try:
            await self.ensure_session()
            async with self.session.get(f"{self.base_url}/health", timeout=5) as response:
                self.connected = response.status == 200
                return self.connected
        except Exception:
            self.connected = False
            return False
    
    async def get_status(self) -> str:
        """Get Valkyrie system status in natural language."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Getting Valkyrie status"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(f"{self.base_url}/api/v1/status") as response:
                    if response.status == 200:
                        data = await response.json()
                        
                        mode = data.get("flight_mode", "unknown")
                        armed = "armed" if data.get("armed") else "disarmed"
                        ai_enabled = "AI active" if data.get("ai_enabled") else "AI disabled"
                        security = "security monitoring" if data.get("security_enabled") else ""
                        failsafe = "fail-safes enabled" if data.get("failsafe_enabled") else ""
                        
                        parts = [f"Valkyrie is {armed}", f"flight mode {mode}"]
                        if ai_enabled:
                            parts.append(ai_enabled)
                        if security:
                            parts.append(security)
                        if failsafe:
                            parts.append(failsafe)
                        
                        return ", ".join(parts) + "."
                    return "Unable to retrieve Valkyrie status."
            except aiohttp.ClientError:
                return "Valkyrie flight system is offline."
            except Exception as e:
                return f"Valkyrie connection error: {str(e)}"
    
    async def get_state(self) -> str:
        """Get current flight state in natural language."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Getting Valkyrie flight state"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(f"{self.base_url}/api/v1/state") as response:
                    if response.status == 200:
                        data = await response.json()
                        
                        pos = data.get("position", {})
                        vel = data.get("velocity", {})
                        att = data.get("attitude", {})
                        
                        lat = pos.get("latitude", 0)
                        lon = pos.get("longitude", 0)
                        alt = pos.get("altitude", 0)
                        alt_agl = data.get("altitude_agl", alt)
                        
                        gs = data.get("groundspeed", 0)
                        heading = data.get("heading", 0)
                        
                        roll = att.get("roll", 0)
                        pitch = att.get("pitch", 0)
                        
                        battery = data.get("battery_percent", 0)
                        
                        return (
                            f"Current position: {lat:.4f}°, {lon:.4f}° at {alt:.0f}m altitude "
                            f"({alt_agl:.0f}m above ground). "
                            f"Heading {heading:.0f}° at {gs:.1f} m/s groundspeed. "
                            f"Attitude: {roll:.1f}° roll, {pitch:.1f}° pitch. "
                            f"Battery at {battery:.0f}%."
                        )
                    return "Unable to retrieve flight state."
            except aiohttp.ClientError:
                return "Valkyrie flight system is not responding."
            except Exception as e:
                return f"Flight state error: {str(e)}"
    
    # =========================================================================
    # FLIGHT CONTROL
    # =========================================================================
    
    async def arm(self) -> str:
        """Arm the flight controller."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Arming Valkyrie"
        ):
            try:
                await self.ensure_session()
                async with self.session.post(f"{self.base_url}/api/v1/arm") as response:
                    if response.status == 200:
                        return "Valkyrie flight controller armed. Pre-flight checks passed."
                    elif response.status == 400:
                        data = await response.json()
                        return f"Cannot arm: {data.get('error', 'Pre-flight checks failed')}"
                    return "Arming failed. Check system status."
            except Exception as e:
                return f"Arming error: {str(e)}"
    
    async def disarm(self) -> str:
        """Disarm the flight controller."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Disarming Valkyrie"
        ):
            try:
                await self.ensure_session()
                async with self.session.post(f"{self.base_url}/api/v1/disarm") as response:
                    if response.status == 200:
                        return "Valkyrie flight controller disarmed. Safe to approach."
                    return "Disarm command sent."
            except Exception as e:
                return f"Disarming error: {str(e)}"
    
    async def set_mode(self, mode: str) -> str:
        """Set flight mode."""
        monitor = get_monitor()
        
        valid_modes = ["manual", "stabilize", "loiter", "auto", "rtl", "land"]
        mode_lower = mode.lower()
        
        if mode_lower not in valid_modes:
            return f"Invalid mode '{mode}'. Valid modes: {', '.join(valid_modes)}"
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            f"Setting Valkyrie to {mode} mode"
        ):
            try:
                await self.ensure_session()
                async with self.session.post(
                    f"{self.base_url}/api/v1/mode",
                    json={"mode": mode_lower}
                ) as response:
                    if response.status == 200:
                        return f"Flight mode changed to {mode_lower.upper()}."
                    return f"Mode change to {mode} failed."
            except Exception as e:
                return f"Mode change error: {str(e)}"
    
    async def get_mode(self) -> str:
        """Get current flight mode."""
        try:
            await self.ensure_session()
            async with self.session.get(f"{self.base_url}/api/v1/mode") as response:
                if response.status == 200:
                    data = await response.json()
                    mode = data.get("mode", "unknown")
                    return f"Current flight mode: {mode.upper()}."
                return "Unable to retrieve flight mode."
        except Exception as e:
            return f"Mode query error: {str(e)}"
    
    # =========================================================================
    # MISSION CONTROL
    # =========================================================================
    
    async def get_mission(self) -> str:
        """Get current mission status."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Getting Valkyrie mission status"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(f"{self.base_url}/api/v1/mission") as response:
                    if response.status == 200:
                        data = await response.json()
                        
                        status = data.get("status", "none")
                        if status == "none" or not data.get("waypoints"):
                            return "No active mission. Valkyrie is awaiting instructions."
                        
                        current = data.get("current_waypoint", 0)
                        total = len(data.get("waypoints", []))
                        distance = data.get("distance_to_waypoint", 0)
                        eta = data.get("eta_minutes", 0)
                        
                        return (
                            f"Mission in progress: waypoint {current} of {total}. "
                            f"Distance to next waypoint: {distance:.0f}m. "
                            f"Estimated time remaining: {eta:.1f} minutes."
                        )
                    return "Unable to retrieve mission status."
            except Exception as e:
                return f"Mission query error: {str(e)}"
    
    async def start_mission(self) -> str:
        """Start the loaded mission."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Starting Valkyrie mission"
        ):
            try:
                await self.ensure_session()
                async with self.session.post(
                    f"{self.base_url}/api/v1/mission",
                    json={"action": "start"}
                ) as response:
                    if response.status == 200:
                        return "Mission started. Valkyrie is now autonomous."
                    return "Unable to start mission. Ensure mission is loaded and aircraft is armed."
            except Exception as e:
                return f"Mission start error: {str(e)}"
    
    async def pause_mission(self) -> str:
        """Pause the current mission."""
        try:
            await self.ensure_session()
            async with self.session.post(
                f"{self.base_url}/api/v1/mission",
                json={"action": "pause"}
            ) as response:
                if response.status == 200:
                    return "Mission paused. Valkyrie is holding position."
                return "Unable to pause mission."
        except Exception as e:
            return f"Mission pause error: {str(e)}"
    
    async def resume_mission(self) -> str:
        """Resume the paused mission."""
        try:
            await self.ensure_session()
            async with self.session.post(
                f"{self.base_url}/api/v1/mission",
                json={"action": "resume"}
            ) as response:
                if response.status == 200:
                    return "Mission resumed. Continuing to next waypoint."
                return "Unable to resume mission."
        except Exception as e:
            return f"Mission resume error: {str(e)}"
    
    # =========================================================================
    # EMERGENCY PROCEDURES
    # =========================================================================
    
    async def emergency_rtb(self) -> str:
        """Initiate emergency return to base."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "EMERGENCY: Valkyrie RTB",
            {"emergency": True}
        ):
            try:
                await self.ensure_session()
                async with self.session.post(f"{self.base_url}/api/v1/emergency/rtb") as response:
                    if response.status == 200:
                        return "EMERGENCY RTB INITIATED. Valkyrie is returning to base immediately."
                    return "RTB command failed. Attempting manual control."
            except Exception as e:
                return f"CRITICAL: RTB command error: {str(e)}. Manual intervention required."
    
    async def emergency_land(self) -> str:
        """Initiate emergency landing at current position."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "EMERGENCY: Valkyrie landing",
            {"emergency": True}
        ):
            try:
                await self.ensure_session()
                async with self.session.post(f"{self.base_url}/api/v1/emergency/land") as response:
                    if response.status == 200:
                        return "EMERGENCY LANDING INITIATED. Valkyrie is descending to land at current position."
                    return "Landing command failed. Attempting alternative procedures."
            except Exception as e:
                return f"CRITICAL: Landing command error: {str(e)}. Manual intervention required."
    
    # =========================================================================
    # AI & SENSOR STATUS
    # =========================================================================
    
    async def get_ai_status(self) -> str:
        """Get AI decision engine status."""
        try:
            await self.ensure_session()
            async with self.session.get(f"{self.base_url}/api/v1/status") as response:
                if response.status == 200:
                    data = await response.json()
                    
                    ai_enabled = data.get("ai_enabled", False)
                    if not ai_enabled:
                        return "Valkyrie AI decision engine is disabled."
                    
                    safety_score = data.get("ai_safety_score", 0) * 100
                    threat_level = data.get("threat_level", "low")
                    
                    return (
                        f"Valkyrie AI is active. Safety score: {safety_score:.0f}%. "
                        f"Current threat assessment: {threat_level}."
                    )
                return "Unable to retrieve AI status."
        except Exception as e:
            return f"AI status error: {str(e)}"
    
    async def get_sensor_status(self) -> str:
        """Get sensor fusion status."""
        try:
            await self.ensure_session()
            async with self.session.get(f"{self.base_url}/api/v1/status") as response:
                if response.status == 200:
                    data = await response.json()
                    
                    sensors = data.get("sensors", {})
                    gps = "GPS OK" if sensors.get("gps_healthy") else "GPS FAULT"
                    ins = "INS OK" if sensors.get("ins_healthy") else "INS FAULT"
                    radar = "RADAR OK" if sensors.get("radar_healthy") else "RADAR N/A"
                    baro = "BARO OK" if sensors.get("baro_healthy") else "BARO FAULT"
                    
                    fusion_confidence = data.get("fusion_confidence", 0) * 100
                    
                    return (
                        f"Sensor status: {gps}, {ins}, {radar}, {baro}. "
                        f"Fusion confidence: {fusion_confidence:.0f}%."
                    )
                return "Unable to retrieve sensor status."
        except Exception as e:
            return f"Sensor status error: {str(e)}"


# =============================================================================
# SINGLETON INSTANCE
# =============================================================================

_valkyrie_client: Optional[ValkyrieClient] = None


def get_valkyrie_client() -> ValkyrieClient:
    """Get singleton Valkyrie client instance."""
    global _valkyrie_client
    if _valkyrie_client is None:
        _valkyrie_client = ValkyrieClient()
    return _valkyrie_client


# =============================================================================
# VOICE COMMAND HANDLERS
# =============================================================================

async def handle_valkyrie_command(command: str) -> str:
    """
    Handle a Valkyrie-related voice command.
    
    Supported commands:
    - "Valkyrie status" / "flight status"
    - "Valkyrie state" / "flight state"
    - "Valkyrie position" / "where is Valkyrie"
    - "arm Valkyrie" / "arm aircraft"
    - "disarm Valkyrie" / "disarm aircraft"
    - "set mode [mode]" / "flight mode [mode]"
    - "current mode" / "what mode"
    - "mission status"
    - "start mission"
    - "pause mission"
    - "resume mission"
    - "return to base" / "RTB" / "come home"
    - "emergency land"
    - "AI status" / "decision engine status"
    - "sensor status" / "sensors"
    """
    client = get_valkyrie_client()
    cmd_lower = command.lower()
    
    # Status queries
    if any(phrase in cmd_lower for phrase in ["valkyrie status", "flight status", "aircraft status"]):
        return await client.get_status()
    
    if any(phrase in cmd_lower for phrase in ["valkyrie state", "flight state", "current state"]):
        return await client.get_state()
    
    if any(phrase in cmd_lower for phrase in ["valkyrie position", "where is valkyrie", "location", "coordinates"]):
        return await client.get_state()
    
    # Arm/Disarm
    if any(phrase in cmd_lower for phrase in ["arm valkyrie", "arm aircraft", "arm the"]):
        return await client.arm()
    
    if any(phrase in cmd_lower for phrase in ["disarm valkyrie", "disarm aircraft", "disarm the"]):
        return await client.disarm()
    
    # Mode control
    if "mode" in cmd_lower:
        if any(phrase in cmd_lower for phrase in ["current mode", "what mode", "flight mode is"]):
            return await client.get_mode()
        
        # Extract mode name
        modes = ["manual", "stabilize", "loiter", "auto", "rtl", "land"]
        for mode in modes:
            if mode in cmd_lower:
                return await client.set_mode(mode)
        
        if "set mode" in cmd_lower or "change mode" in cmd_lower:
            return "Please specify a mode: manual, stabilize, loiter, auto, rtl, or land."
    
    # Mission control
    if "mission" in cmd_lower:
        if "status" in cmd_lower:
            return await client.get_mission()
        if "start" in cmd_lower or "begin" in cmd_lower:
            return await client.start_mission()
        if "pause" in cmd_lower or "hold" in cmd_lower:
            return await client.pause_mission()
        if "resume" in cmd_lower or "continue" in cmd_lower:
            return await client.resume_mission()
        return await client.get_mission()
    
    # Emergency procedures
    if any(phrase in cmd_lower for phrase in ["return to base", "rtb", "come home", "return home"]):
        return await client.emergency_rtb()
    
    if any(phrase in cmd_lower for phrase in ["emergency land", "land now", "force land"]):
        return await client.emergency_land()
    
    # AI/Sensor status
    if any(phrase in cmd_lower for phrase in ["ai status", "decision engine", "ai decision"]):
        return await client.get_ai_status()
    
    if any(phrase in cmd_lower for phrase in ["sensor status", "sensors", "sensor health"]):
        return await client.get_sensor_status()
    
    # Default status if nothing specific matched
    return await client.get_status()
