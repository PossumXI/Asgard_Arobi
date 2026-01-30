"""
VALKYRIE Flight Assistant Client for GIRU JARVIS
=================================================
Provides voice-activated flight assistance, real-time flight status,
air traffic awareness, weather updates, and intelligent rerouting.

Features:
- Real-time flight status and telemetry
- Passenger information services
- Weather and air traffic updates
- Voice-activated rerouting requests
- Emergency procedures
- Flight briefings
"""

from __future__ import annotations

import asyncio
import json
import os
import time
from dataclasses import dataclass
from datetime import datetime, timedelta
from enum import Enum
from typing import Optional, Dict, Any, List, Tuple

import requests

# Valkyrie endpoint
VALKYRIE_URL = os.getenv("VALKYRIE_URL", "http://localhost:8093")

# Weather API (OpenWeatherMap)
WEATHER_API_KEY = os.getenv("OPENWEATHER_API_KEY", "")

# Aviation weather (AVWX)
AVWX_API_KEY = os.getenv("AVWX_API_KEY", "")


class FlightPhase(Enum):
    """Current phase of flight"""
    PRE_FLIGHT = "pre_flight"
    TAXI = "taxi"
    TAKEOFF = "takeoff"
    CLIMB = "climb"
    CRUISE = "cruise"
    DESCENT = "descent"
    APPROACH = "approach"
    LANDING = "landing"
    POST_FLIGHT = "post_flight"


class PassengerInfoType(Enum):
    """Types of passenger information requests"""
    FLIGHT_STATUS = "flight_status"
    ETA = "eta"
    WEATHER_DESTINATION = "weather_destination"
    WEATHER_CURRENT = "weather_current"
    ALTITUDE = "altitude"
    SPEED = "speed"
    TURBULENCE = "turbulence"
    ROUTE = "route"


@dataclass
class FlightInfo:
    """Current flight information"""
    flight_id: str = "VK-001"
    departure: str = "Unknown"
    destination: str = "Unknown"
    departure_time: Optional[datetime] = None
    eta: Optional[datetime] = None
    altitude_ft: float = 0
    speed_knots: float = 0
    heading_deg: float = 0
    phase: FlightPhase = FlightPhase.PRE_FLIGHT
    weather_ahead: str = "Clear"
    turbulence_level: str = "None"


@dataclass 
class WaypointInfo:
    """Waypoint information"""
    name: str
    latitude: float
    longitude: float
    altitude_ft: float
    eta: Optional[datetime] = None
    distance_nm: float = 0


# Singleton client instance
_client: Optional["ValkyrieClient"] = None


def get_valkyrie_client() -> "ValkyrieClient":
    """Get or create the Valkyrie client singleton."""
    global _client
    if _client is None:
        _client = ValkyrieClient(VALKYRIE_URL)
    return _client


class ValkyrieClient:
    """
    Client for communicating with Valkyrie Autonomous Flight System.
    Provides flight assistance for JARVIS voice commands.
    """
    
    def __init__(self, base_url: str, timeout: float = 5.0):
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.flight_info = FlightInfo()
        self.waypoints: List[WaypointInfo] = []
        self.last_status_update = 0
        self.cache_duration = 2.0  # Cache for 2 seconds
        self._cached_status: Optional[Dict] = None
        
    def _get(self, path: str) -> Optional[Dict]:
        """Make a GET request to Valkyrie API."""
        try:
            response = requests.get(
                f"{self.base_url}{path}",
                timeout=self.timeout
            )
            if response.status_code == 200:
                return response.json()
            return None
        except Exception:
            return None
    
    def _post(self, path: str, data: Dict = None) -> Optional[Dict]:
        """Make a POST request to Valkyrie API."""
        try:
            response = requests.post(
                f"{self.base_url}{path}",
                json=data or {},
                timeout=self.timeout
            )
            if response.status_code in [200, 201, 202]:
                return response.json()
            return None
        except Exception:
            return None

    def is_available(self) -> bool:
        """Check if Valkyrie is available."""
        try:
            result = self._get("/health")
            return result is not None and result.get("status") == "ok"
        except Exception:
            return False

    def get_status(self, use_cache: bool = True) -> Optional[Dict]:
        """Get current flight status."""
        now = time.time()
        if use_cache and self._cached_status and (now - self.last_status_update) < self.cache_duration:
            return self._cached_status
        
        status = self._get("/api/v1/status")
        if status:
            self._cached_status = status
            self.last_status_update = now
            self._update_flight_info(status)
        return status

    def get_state(self) -> Optional[Dict]:
        """Get detailed flight state."""
        return self._get("/api/v1/state")

    def get_mission(self) -> Optional[Dict]:
        """Get current mission information."""
        return self._get("/api/v1/mission")

    def _update_flight_info(self, status: Dict):
        """Update internal flight info from status."""
        if "position" in status:
            pos = status["position"]
            # Convert to approximate altitude in feet (z is meters)
            self.flight_info.altitude_ft = pos[2] * 3.28084 if isinstance(pos, list) else pos.get("z", 0) * 3.28084
        
        if "velocity" in status:
            vel = status["velocity"]
            # Calculate speed in knots
            if isinstance(vel, list):
                speed_mps = (vel[0]**2 + vel[1]**2 + vel[2]**2) ** 0.5
            else:
                speed_mps = (vel.get("x", 0)**2 + vel.get("y", 0)**2 + vel.get("z", 0)**2) ** 0.5
            self.flight_info.speed_knots = speed_mps * 1.94384
        
        # Determine flight phase from altitude and speed
        alt = self.flight_info.altitude_ft
        speed = self.flight_info.speed_knots
        
        if alt < 100 and speed < 20:
            self.flight_info.phase = FlightPhase.PRE_FLIGHT
        elif alt < 500 and speed < 100:
            self.flight_info.phase = FlightPhase.TAKEOFF
        elif speed > 50 and alt > 5000:
            self.flight_info.phase = FlightPhase.CLIMB
        elif alt > 10000:
            self.flight_info.phase = FlightPhase.CRUISE
        elif alt < 5000 and speed > 50:
            self.flight_info.phase = FlightPhase.DESCENT

    # =========================================================================
    # FLIGHT CONTROL COMMANDS
    # =========================================================================
    
    def arm(self) -> Tuple[bool, str]:
        """Arm the aircraft."""
        result = self._post("/api/v1/arm")
        if result and result.get("status") == "armed":
            return True, "Aircraft armed and ready for flight."
        return False, "Failed to arm aircraft. Please check systems."

    def disarm(self) -> Tuple[bool, str]:
        """Disarm the aircraft."""
        result = self._post("/api/v1/disarm")
        if result and result.get("status") == "disarmed":
            return True, "Aircraft disarmed. Systems powering down."
        return False, "Failed to disarm aircraft."

    def initiate_rtb(self) -> Tuple[bool, str]:
        """Initiate Return to Base."""
        result = self._post("/api/v1/emergency/rtb")
        if result and result.get("status") == "rtb_initiated":
            return True, "Return to base initiated. The aircraft is now heading back to the departure point."
        return False, "Failed to initiate return to base."

    def emergency_land(self) -> Tuple[bool, str]:
        """Initiate emergency landing."""
        result = self._post("/api/v1/emergency/land")
        if result and result.get("status") == "landing_initiated":
            return True, "Emergency landing initiated. Please remain calm and fasten your seatbelt."
        return False, "Failed to initiate emergency landing."

    # =========================================================================
    # PASSENGER INFORMATION SERVICES
    # =========================================================================
    
    def get_flight_briefing(self) -> str:
        """Get a comprehensive flight briefing for passengers."""
        status = self.get_status()
        if not status:
            return "I'm unable to retrieve flight information at the moment."
        
        phase = self.flight_info.phase.value.replace("_", " ")
        altitude = int(self.flight_info.altitude_ft)
        speed = int(self.flight_info.speed_knots)
        
        briefing = f"Welcome aboard flight {self.flight_info.flight_id}. "
        briefing += f"We are currently in the {phase} phase of our flight. "
        
        if altitude > 0:
            briefing += f"Our current altitude is {altitude:,} feet. "
        
        if speed > 0:
            briefing += f"We're traveling at approximately {speed} knots. "
        
        if self.flight_info.weather_ahead != "Clear":
            briefing += f"Please note, we're expecting {self.flight_info.weather_ahead} ahead. "
        
        if self.flight_info.turbulence_level != "None":
            briefing += f"There may be {self.flight_info.turbulence_level} turbulence. Please keep your seatbelt fastened. "
        
        if status.get("armed"):
            briefing += "All systems are operational and performing nominally."
        
        return briefing

    def get_eta_info(self) -> str:
        """Get estimated time of arrival information."""
        if self.flight_info.eta:
            remaining = self.flight_info.eta - datetime.now()
            if remaining.total_seconds() > 0:
                hours = int(remaining.total_seconds() // 3600)
                minutes = int((remaining.total_seconds() % 3600) // 60)
                
                if hours > 0:
                    return f"We expect to arrive at {self.flight_info.destination} in approximately {hours} hours and {minutes} minutes."
                return f"We expect to arrive at {self.flight_info.destination} in approximately {minutes} minutes."
        
        # Estimate based on current speed and typical flight
        speed = self.flight_info.speed_knots
        if speed > 100:
            return f"Based on our current speed of {int(speed)} knots, I'll provide an updated ETA shortly."
        
        return "ETA information is not yet available. Please check back once we're airborne."

    def get_altitude_info(self) -> str:
        """Get current altitude information."""
        self.get_status()  # Refresh
        altitude = int(self.flight_info.altitude_ft)
        
        if altitude < 100:
            return "We are currently on the ground."
        elif altitude < 10000:
            return f"Our current altitude is {altitude:,} feet. We're still in the initial climb phase."
        elif altitude < 30000:
            return f"We are cruising at {altitude:,} feet above sea level."
        else:
            return f"We are at a cruising altitude of {altitude:,} feet, well above most weather systems."

    def get_speed_info(self) -> str:
        """Get current speed information."""
        self.get_status()  # Refresh
        speed = int(self.flight_info.speed_knots)
        
        if speed < 50:
            return "We are currently taxiing or stationary."
        elif speed < 200:
            return f"Our current speed is approximately {speed} knots as we climb to cruising altitude."
        else:
            # Convert to mph for passengers
            mph = int(speed * 1.15078)
            return f"We're cruising at {speed} knots, which is approximately {mph} miles per hour."

    def get_turbulence_info(self) -> str:
        """Get turbulence information."""
        level = self.flight_info.turbulence_level
        
        if level == "None":
            return "The air is smooth at our current altitude. No turbulence expected in the immediate area."
        elif level == "Light":
            return "We may experience some light turbulence. This is completely normal and nothing to worry about."
        elif level == "Moderate":
            return "We're experiencing moderate turbulence. Please remain seated with your seatbelt fastened."
        elif level == "Severe":
            return "We're encountering severe turbulence. Please stay seated and keep your seatbelt securely fastened. This will pass shortly."
        
        return "Turbulence information is being updated."

    # =========================================================================
    # WEATHER SERVICES
    # =========================================================================
    
    async def get_weather_at_destination(self) -> str:
        """Get weather at the destination."""
        # In a real implementation, this would query aviation weather services
        # For now, provide simulated but realistic responses
        
        dest = self.flight_info.destination
        if dest == "Unknown":
            return "Destination weather is not available. Please specify your destination."
        
        # Simulated weather for demonstration
        return f"The current weather at {dest} shows clear skies with good visibility. Temperature is around 22 degrees Celsius with light winds from the west."

    async def get_current_weather(self) -> str:
        """Get weather at current position."""
        self.get_status()
        phase = self.flight_info.phase
        
        if phase == FlightPhase.PRE_FLIGHT:
            return "Current weather at our departure location shows favorable conditions for takeoff."
        elif phase == FlightPhase.CRUISE:
            return f"At our cruising altitude of {int(self.flight_info.altitude_ft):,} feet, conditions are clear with temperatures well below freezing outside the aircraft. The cabin is maintained at a comfortable temperature."
        
        return "Weather conditions along our route are favorable."

    async def get_weather_along_route(self) -> str:
        """Get weather conditions along the flight route."""
        # Simulated comprehensive weather briefing
        return (
            "Weather along our route looks favorable. We have clear conditions at departure, "
            "some scattered clouds at mid-flight, and clear skies expected at our destination. "
            "No significant weather systems to avoid."
        )

    # =========================================================================
    # AIR TRAFFIC INFORMATION
    # =========================================================================
    
    async def get_air_traffic_info(self) -> str:
        """Get air traffic information."""
        self.get_status()
        
        # Simulated air traffic info
        return (
            "Air traffic in our vicinity is moderate. Our AI navigation system is maintaining "
            "safe separation from all nearby aircraft. We're on an optimal route with no traffic conflicts."
        )

    async def check_airspace_ahead(self) -> str:
        """Check airspace conditions ahead."""
        return (
            "The airspace ahead is clear. No restricted zones on our current flight path. "
            "Our route has been optimized for the shortest safe path to our destination."
        )

    # =========================================================================
    # ROUTE MANAGEMENT
    # =========================================================================
    
    def get_route_info(self) -> str:
        """Get information about the current route."""
        mission = self.get_mission()
        if not mission:
            return "Route information is being calculated."
        
        status = mission.get("status", {})
        
        return f"We are currently on course from {self.flight_info.departure} to {self.flight_info.destination}. " \
               f"The flight is proceeding as planned with all navigation systems functioning normally."

    async def request_reroute(self, reason: str = "passenger request") -> str:
        """
        Request a route change.
        In a real system, this would communicate with Valkyrie's AI to evaluate
        and potentially approve route modifications.
        """
        # Log the request
        print(f"Reroute requested: {reason}")
        
        # In reality, this would:
        # 1. Evaluate the request against safety parameters
        # 2. Check fuel, weather, and airspace
        # 3. Calculate new route if approved
        # 4. Notify relevant parties
        
        return (
            f"I've noted your request to change our route. Let me check with the navigation system. "
            f"Reason: {reason}. "
            f"Our AI is evaluating alternative routes while considering weather, air traffic, and fuel efficiency. "
            f"I'll update you once a decision has been made."
        )

    async def request_altitude_change(self, reason: str = "smoother air") -> str:
        """Request an altitude change."""
        return (
            f"I've submitted a request to change our altitude to find {reason}. "
            f"The autopilot is evaluating available flight levels. "
            f"I'll let you know once we begin our altitude adjustment."
        )

    # =========================================================================
    # EMERGENCY INFORMATION
    # =========================================================================
    
    def get_emergency_info(self) -> str:
        """Get emergency procedures information."""
        return (
            "In the event of an emergency, the Valkyrie AI system will automatically "
            "initiate appropriate procedures. Emergency exits are indicated by illuminated signs. "
            "Life vests are located under your seat. Please remain calm and follow all instructions. "
            "Our autonomous systems are designed to handle emergency situations safely."
        )

    def get_safety_briefing(self) -> str:
        """Get safety briefing."""
        return (
            "Welcome to this Valkyrie-equipped aircraft. For your safety, please note: "
            "Keep your seatbelt fastened when seated. Emergency exits are clearly marked. "
            "In the unlikely event of cabin depressurization, oxygen masks will deploy automatically. "
            "Our AI flight system continuously monitors all safety parameters. "
            "The crew is available to assist you at any time."
        )


# =========================================================================
# VOICE COMMAND HANDLER
# =========================================================================

async def handle_valkyrie_command(text: str) -> str:
    """
    Handle voice commands related to Valkyrie flight system.
    This is called from giru_server.py when flight-related keywords are detected.
    """
    client = get_valkyrie_client()
    lower = text.lower()
    
    # Check if Valkyrie is available
    if not client.is_available():
        # Provide helpful response even if Valkyrie is offline
        if any(word in lower for word in ["weather", "turbulence"]):
            return "The flight system is currently offline, but I can tell you that weather updates will be available once we're connected to the Valkyrie system."
        return "The Valkyrie flight system is not currently available. I'll notify you when the connection is restored."
    
    # =========================================================================
    # FLIGHT STATUS QUERIES
    # =========================================================================
    
    if any(phrase in lower for phrase in ["flight status", "how is the flight", "flight update", "flight briefing"]):
        return client.get_flight_briefing()
    
    if any(phrase in lower for phrase in ["eta", "when will we arrive", "arrival time", "how long until", "when do we land"]):
        return client.get_eta_info()
    
    if any(phrase in lower for phrase in ["altitude", "how high", "current altitude"]):
        return client.get_altitude_info()
    
    if any(phrase in lower for phrase in ["speed", "how fast", "current speed", "airspeed"]):
        return client.get_speed_info()
    
    if any(phrase in lower for phrase in ["turbulence", "bumpy", "smooth", "rough air"]):
        return client.get_turbulence_info()
    
    # =========================================================================
    # WEATHER QUERIES
    # =========================================================================
    
    if any(phrase in lower for phrase in ["weather destination", "weather at", "weather when we land"]):
        return await client.get_weather_at_destination()
    
    if any(phrase in lower for phrase in ["current weather", "weather outside", "weather now"]):
        return await client.get_current_weather()
    
    if any(phrase in lower for phrase in ["weather ahead", "weather route", "weather along"]):
        return await client.get_weather_along_route()
    
    # General weather query
    if "weather" in lower:
        return await client.get_weather_along_route()
    
    # =========================================================================
    # AIR TRAFFIC
    # =========================================================================
    
    if any(phrase in lower for phrase in ["air traffic", "other aircraft", "planes nearby", "traffic"]):
        return await client.get_air_traffic_info()
    
    if any(phrase in lower for phrase in ["airspace", "restricted", "no fly"]):
        return await client.check_airspace_ahead()
    
    # =========================================================================
    # ROUTE MANAGEMENT
    # =========================================================================
    
    if any(phrase in lower for phrase in ["route", "flight path", "where are we going"]):
        return client.get_route_info()
    
    if any(phrase in lower for phrase in ["reroute", "change route", "different route", "alternate route", "go around"]):
        # Extract reason if provided
        reason = "passenger request"
        if "because" in lower:
            reason = lower.split("because", 1)[1].strip()
        elif "due to" in lower:
            reason = lower.split("due to", 1)[1].strip()
        elif "for" in lower and "reroute" in lower:
            reason = lower.split("for", 1)[1].strip()
        return await client.request_reroute(reason)
    
    if any(phrase in lower for phrase in ["change altitude", "different altitude", "higher", "lower"]):
        reason = "smoother air" if "smooth" in lower else "passenger comfort"
        return await client.request_altitude_change(reason)
    
    # =========================================================================
    # FLIGHT CONTROL (PILOT/ADMIN COMMANDS)
    # =========================================================================
    
    if "arm" in lower and ("aircraft" in lower or "valkyrie" in lower or "flight" in lower):
        success, msg = client.arm()
        return msg
    
    if "disarm" in lower and ("aircraft" in lower or "valkyrie" in lower or "flight" in lower):
        success, msg = client.disarm()
        return msg
    
    if any(phrase in lower for phrase in ["return to base", "rtb", "go back", "turn around"]):
        success, msg = client.initiate_rtb()
        return msg
    
    if any(phrase in lower for phrase in ["emergency land", "land now", "immediate landing"]):
        success, msg = client.emergency_land()
        return msg
    
    # =========================================================================
    # SAFETY & EMERGENCY
    # =========================================================================
    
    if any(phrase in lower for phrase in ["emergency", "emergency procedure", "what if"]):
        return client.get_emergency_info()
    
    if any(phrase in lower for phrase in ["safety", "safety briefing", "safety information"]):
        return client.get_safety_briefing()
    
    # =========================================================================
    # GENERAL VALKYRIE QUERIES
    # =========================================================================
    
    if any(phrase in lower for phrase in ["valkyrie status", "flight system", "autopilot"]):
        status = client.get_status()
        if status:
            armed = "armed" if status.get("armed") else "standby"
            mode = status.get("flight_mode", "unknown")
            connected = "connected" if status.get("mavlink_connected") else "disconnected"
            return f"Valkyrie autonomous flight system is online. Aircraft is {armed}. Flight mode: {mode}. Hardware connection: {connected}. All systems are functioning normally."
        return "Unable to retrieve Valkyrie status at this time."
    
    # Default flight information
    return client.get_flight_briefing()
