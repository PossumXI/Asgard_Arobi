"""
Sensors Integration Client for GIRU JARVIS
===========================================
Provides voice control and monitoring for WiFi imaging, vision,
triangulation, and sensor fusion across ASGARD systems.

Capabilities:
- WiFi CSI imaging for through-wall detection
- Multi-router triangulation and positioning
- Material classification (walls, obstacles)
- Vision system queries (cameras, detections)
- Sensor fusion status
- Object detection results
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

PRICILLA_URL = os.getenv("PRICILLA_URL", "http://localhost:8089")
SILENUS_URL = os.getenv("SILENUS_URL", "http://localhost:9093")
VALKYRIE_URL = os.getenv("VALKYRIE_URL", "http://localhost:8093")


# =============================================================================
# DATA TYPES
# =============================================================================

@dataclass
class WiFiRouter:
    """A WiFi router used for CSI imaging."""
    id: str
    position: Dict[str, float]
    frequency_ghz: float
    tx_power_dbm: float


@dataclass
class ThroughWallObservation:
    """Estimated target through wall."""
    estimated_position: Dict[str, float]
    material: str
    estimated_depth_m: float
    confidence: float


@dataclass
class VisionDetection:
    """Object detection from vision system."""
    class_name: str
    confidence: float
    bounding_box: Dict[str, int]
    timestamp: int


# =============================================================================
# WIFI IMAGING CLIENT
# =============================================================================

class WiFiImagingClient:
    """Client for WiFi CSI imaging and triangulation."""
    
    def __init__(self, base_url: str = PRICILLA_URL):
        self.base_url = base_url
        self.session: Optional[aiohttp.ClientSession] = None
    
    async def ensure_session(self):
        if self.session is None or self.session.closed:
            self.session = aiohttp.ClientSession()
    
    async def close(self):
        if self.session and not self.session.closed:
            await self.session.close()
    
    # =========================================================================
    # ROUTER MANAGEMENT
    # =========================================================================
    
    async def list_routers(self) -> str:
        """List registered WiFi routers."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Listing WiFi routers"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(f"{self.base_url}/api/v1/wifi/routers") as response:
                    if response.status == 200:
                        data = await response.json()
                        
                        if isinstance(data, list):
                            routers = data
                        else:
                            routers = data.get("routers", [])
                        
                        if not routers:
                            return "No WiFi routers registered. Register routers to enable WiFi imaging."
                        
                        summaries = []
                        for r in routers[:5]:
                            rid = r.get("id", "unknown")
                            freq = r.get("frequencyGhz", 2.4)
                            power = r.get("txPowerDbm", 20)
                            summaries.append(f"{rid} ({freq}GHz, {power}dBm)")
                        
                        return f"WiFi routers registered: {', '.join(summaries)}. Total: {len(routers)}."
                    return "Unable to retrieve WiFi router list."
            except aiohttp.ClientError:
                return "WiFi imaging service is offline."
            except Exception as e:
                return f"WiFi router query error: {str(e)}"
    
    async def register_router(self, router_id: str, x: float, y: float, z: float, 
                              frequency: float = 2.4, power: float = 20.0) -> str:
        """Register a new WiFi router for imaging."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            f"Registering WiFi router {router_id}"
        ):
            try:
                await self.ensure_session()
                payload = {
                    "id": router_id,
                    "position": {"x": x, "y": y, "z": z},
                    "frequencyGhz": frequency,
                    "txPowerDbm": power
                }
                
                async with self.session.post(
                    f"{self.base_url}/api/v1/wifi/routers",
                    json=payload
                ) as response:
                    if response.status in [200, 201]:
                        return f"WiFi router '{router_id}' registered at position ({x}, {y}, {z})."
                    return f"Failed to register router: {response.status}"
            except Exception as e:
                return f"Router registration error: {str(e)}"
    
    # =========================================================================
    # WIFI IMAGING
    # =========================================================================
    
    async def get_imaging_status(self) -> str:
        """Get WiFi imaging system status."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Getting WiFi imaging status"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(f"{self.base_url}/api/v1/wifi/status") as response:
                    if response.status == 200:
                        data = await response.json()
                        
                        enabled = data.get("enabled", False)
                        router_count = data.get("router_count", 0)
                        last_scan = data.get("last_scan", "never")
                        observations = data.get("observation_count", 0)
                        
                        status = "enabled" if enabled else "disabled"
                        return (
                            f"WiFi imaging is {status}. "
                            f"{router_count} routers registered. "
                            f"{observations} observations recorded. "
                            f"Last scan: {last_scan}."
                        )
                    
                    # Fallback to router list check
                    async with self.session.get(f"{self.base_url}/api/v1/wifi/routers") as r2:
                        if r2.status == 200:
                            routers = await r2.json()
                            count = len(routers) if isinstance(routers, list) else len(routers.get("routers", []))
                            return f"WiFi imaging system online. {count} routers registered."
                    
                    return "WiFi imaging system status unknown."
            except aiohttp.ClientError:
                return "WiFi imaging service is offline."
            except Exception as e:
                return f"WiFi imaging status error: {str(e)}"
    
    async def get_through_wall_observations(self) -> str:
        """Get current through-wall observations."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Getting through-wall observations"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(f"{self.base_url}/api/v1/wifi/observations") as response:
                    if response.status == 200:
                        data = await response.json()
                        observations = data.get("observations", [])
                        
                        if not observations:
                            return "No through-wall observations detected."
                        
                        summaries = []
                        for obs in observations[:3]:
                            material = obs.get("material", "unknown")
                            depth = obs.get("estimatedDepthM", 0)
                            confidence = obs.get("confidence", 0) * 100
                            pos = obs.get("estimatedPosition", {})
                            
                            summaries.append(
                                f"{material} at {depth:.1f}m depth "
                                f"(confidence: {confidence:.0f}%)"
                            )
                        
                        return (
                            f"Through-wall detections: {len(observations)} observations. "
                            + ". ".join(summaries)
                        )
                    return "Unable to retrieve through-wall observations."
            except aiohttp.ClientError:
                return "WiFi imaging service is offline."
            except Exception as e:
                return f"Observation query error: {str(e)}"
    
    async def scan_area(self) -> str:
        """Initiate a WiFi imaging scan."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Initiating WiFi imaging scan"
        ):
            try:
                await self.ensure_session()
                async with self.session.post(f"{self.base_url}/api/v1/wifi/imaging") as response:
                    if response.status == 200:
                        data = await response.json()
                        observations = data.get("observations", [])
                        confidence = data.get("confidence", 0) * 100
                        
                        if observations:
                            obs = observations[0]
                            material = obs.get("material", "unknown")
                            depth = obs.get("estimatedDepthM", 0)
                            
                            return (
                                f"WiFi scan complete. Detected {material} material "
                                f"at approximately {depth:.1f} meters. "
                                f"Overall confidence: {confidence:.0f}%."
                            )
                        return "WiFi scan complete. No significant observations."
                    return "Unable to initiate WiFi scan."
            except Exception as e:
                return f"WiFi scan error: {str(e)}"
    
    async def get_material_analysis(self) -> str:
        """Get analysis of detected materials."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Analyzing wall materials"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(f"{self.base_url}/api/v1/wifi/observations") as response:
                    if response.status == 200:
                        data = await response.json()
                        observations = data.get("observations", [])
                        
                        if not observations:
                            return "No material data available. Run a WiFi imaging scan first."
                        
                        # Analyze materials
                        materials = {}
                        for obs in observations:
                            material = obs.get("material", "unknown")
                            materials[material] = materials.get(material, 0) + 1
                        
                        material_loss = {
                            "drywall": 3.0,
                            "glass": 2.0,
                            "wood": 4.0,
                            "composite": 6.0,
                            "brick": 8.0,
                            "concrete": 12.0
                        }
                        
                        summaries = []
                        for mat, count in materials.items():
                            loss = material_loss.get(mat, 5.0)
                            penetration = "excellent" if loss < 4 else "good" if loss < 7 else "limited"
                            summaries.append(f"{mat}: {count} occurrences, {penetration} signal penetration")
                        
                        return "Material analysis: " + ". ".join(summaries)
                    return "Unable to retrieve material data."
            except Exception as e:
                return f"Material analysis error: {str(e)}"


# =============================================================================
# VISION SYSTEM CLIENT
# =============================================================================

class VisionClient:
    """Client for vision systems (Silenus, Valkyrie)."""
    
    def __init__(self, silenus_url: str = SILENUS_URL, valkyrie_url: str = VALKYRIE_URL):
        self.silenus_url = silenus_url
        self.valkyrie_url = valkyrie_url
        self.session: Optional[aiohttp.ClientSession] = None
    
    async def ensure_session(self):
        if self.session is None or self.session.closed:
            self.session = aiohttp.ClientSession()
    
    async def close(self):
        if self.session and not self.session.closed:
            await self.session.close()
    
    # =========================================================================
    # DETECTION QUERIES
    # =========================================================================
    
    async def get_detections(self, source: str = "all") -> str:
        """Get current object detections."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            f"Getting vision detections from {source}"
        ):
            try:
                await self.ensure_session()
                
                # Try Silenus first (satellite/orbital vision)
                url = f"{self.silenus_url}/api/v1/vision/detections"
                
                async with self.session.get(url, timeout=5) as response:
                    if response.status == 200:
                        data = await response.json()
                        detections = data.get("detections", [])
                        
                        if not detections:
                            return "No objects currently detected by vision systems."
                        
                        # Summarize detections by class
                        by_class = {}
                        for det in detections:
                            cls = det.get("class", "unknown")
                            conf = det.get("confidence", 0)
                            if cls not in by_class or conf > by_class[cls]["confidence"]:
                                by_class[cls] = {"confidence": conf, "count": by_class.get(cls, {}).get("count", 0) + 1}
                            else:
                                by_class[cls]["count"] += 1
                        
                        summaries = []
                        for cls, info in sorted(by_class.items(), key=lambda x: -x[1]["confidence"])[:5]:
                            conf_pct = info["confidence"] * 100
                            count = info["count"]
                            summaries.append(f"{count} {cls}{'s' if count > 1 else ''} ({conf_pct:.0f}%)")
                        
                        return f"Vision detections: {', '.join(summaries)}. Total: {len(detections)} objects."
                    
                    return "Unable to retrieve vision detections."
            except asyncio.TimeoutError:
                return "Vision system timed out."
            except aiohttp.ClientError:
                return "Vision systems are offline."
            except Exception as e:
                return f"Vision query error: {str(e)}"
    
    async def get_camera_status(self) -> str:
        """Get camera system status."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Getting camera status"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(f"{self.silenus_url}/api/v1/cameras", timeout=5) as response:
                    if response.status == 200:
                        data = await response.json()
                        cameras = data.get("cameras", [])
                        
                        if not cameras:
                            return "No cameras registered in the system."
                        
                        active = sum(1 for c in cameras if c.get("status") == "active")
                        streaming = sum(1 for c in cameras if c.get("streaming"))
                        
                        return (
                            f"Camera status: {len(cameras)} cameras registered, "
                            f"{active} active, {streaming} streaming."
                        )
                    return "Unable to retrieve camera status."
            except asyncio.TimeoutError:
                return "Camera system timed out."
            except aiohttp.ClientError:
                return "Camera systems offline."
            except Exception as e:
                return f"Camera status error: {str(e)}"
    
    async def detect_specific(self, target_class: str) -> str:
        """Check for specific object class."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            f"Searching for {target_class}"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(
                    f"{self.silenus_url}/api/v1/vision/detections",
                    params={"class": target_class},
                    timeout=5
                ) as response:
                    if response.status == 200:
                        data = await response.json()
                        detections = data.get("detections", [])
                        
                        matches = [d for d in detections if d.get("class", "").lower() == target_class.lower()]
                        
                        if not matches:
                            return f"No {target_class} detected in current view."
                        
                        best = max(matches, key=lambda x: x.get("confidence", 0))
                        conf = best.get("confidence", 0) * 100
                        
                        return f"Detected {len(matches)} {target_class}{'s' if len(matches) > 1 else ''}. Highest confidence: {conf:.0f}%."
                    return f"Unable to search for {target_class}."
            except Exception as e:
                return f"Detection search error: {str(e)}"
    
    # =========================================================================
    # THREAT DETECTION
    # =========================================================================
    
    async def check_threats(self) -> str:
        """Check for visual threats (fire, smoke, etc)."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Checking for visual threats"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(
                    f"{self.silenus_url}/api/v1/vision/alerts",
                    timeout=5
                ) as response:
                    if response.status == 200:
                        data = await response.json()
                        alerts = data.get("alerts", [])
                        
                        threat_types = ["fire", "smoke", "flood", "intrusion"]
                        threats = [a for a in alerts if a.get("type", "").lower() in threat_types]
                        
                        if not threats:
                            return "No visual threats detected. All clear."
                        
                        summaries = []
                        for t in threats[:3]:
                            threat_type = t.get("type", "unknown")
                            severity = t.get("severity", "medium")
                            location = t.get("location", "unknown area")
                            summaries.append(f"{severity} {threat_type} in {location}")
                        
                        return f"ALERT: Visual threats detected. {'. '.join(summaries)}"
                    return "Unable to check for visual threats."
            except Exception as e:
                return f"Threat check error: {str(e)}"


# =============================================================================
# SENSOR FUSION CLIENT
# =============================================================================

class SensorFusionClient:
    """Client for sensor fusion status and triangulation."""
    
    def __init__(self, pricilla_url: str = PRICILLA_URL, valkyrie_url: str = VALKYRIE_URL):
        self.pricilla_url = pricilla_url
        self.valkyrie_url = valkyrie_url
        self.session: Optional[aiohttp.ClientSession] = None
    
    async def ensure_session(self):
        if self.session is None or self.session.closed:
            self.session = aiohttp.ClientSession()
    
    async def close(self):
        if self.session and not self.session.closed:
            await self.session.close()
    
    async def get_fusion_status(self) -> str:
        """Get sensor fusion system status."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Getting sensor fusion status"
        ):
            try:
                await self.ensure_session()
                
                # Try Valkyrie first (has more detailed EKF)
                async with self.session.get(
                    f"{self.valkyrie_url}/api/v1/sensors",
                    timeout=5
                ) as response:
                    if response.status == 200:
                        data = await response.json()
                        
                        sensors = data.get("sensors", {})
                        confidence = data.get("fusion_confidence", 0) * 100
                        
                        active = []
                        for sensor, status in sensors.items():
                            if status.get("healthy", False):
                                active.append(sensor.upper())
                        
                        return (
                            f"Sensor fusion status: {len(active)} sensors active. "
                            f"Active: {', '.join(active[:6])}. "
                            f"Fusion confidence: {confidence:.0f}%."
                        )
                
                return "Sensor fusion status unavailable."
            except asyncio.TimeoutError:
                return "Sensor fusion system timed out."
            except aiohttp.ClientError:
                return "Sensor fusion systems offline."
            except Exception as e:
                return f"Fusion status error: {str(e)}"
    
    async def get_triangulation_result(self) -> str:
        """Get latest triangulation result."""
        monitor = get_monitor()
        
        async with monitor.track_activity(
            ActivityType.ASGARD_QUERY,
            "Getting triangulation result"
        ):
            try:
                await self.ensure_session()
                async with self.session.get(
                    f"{self.pricilla_url}/api/v1/wifi/triangulation",
                    timeout=5
                ) as response:
                    if response.status == 200:
                        data = await response.json()
                        
                        position = data.get("position", {})
                        confidence = data.get("confidence", 0) * 100
                        sources = data.get("source_count", 0)
                        
                        x = position.get("x", 0)
                        y = position.get("y", 0)
                        z = position.get("z", 0)
                        
                        return (
                            f"Triangulation result: Position ({x:.2f}, {y:.2f}, {z:.2f}). "
                            f"Based on {sources} sources. Confidence: {confidence:.0f}%."
                        )
                    return "Triangulation data unavailable."
            except Exception as e:
                return f"Triangulation error: {str(e)}"


# =============================================================================
# SINGLETON INSTANCES
# =============================================================================

_wifi_client: Optional[WiFiImagingClient] = None
_vision_client: Optional[VisionClient] = None
_fusion_client: Optional[SensorFusionClient] = None


def get_wifi_client() -> WiFiImagingClient:
    global _wifi_client
    if _wifi_client is None:
        _wifi_client = WiFiImagingClient()
    return _wifi_client


def get_vision_client() -> VisionClient:
    global _vision_client
    if _vision_client is None:
        _vision_client = VisionClient()
    return _vision_client


def get_fusion_client() -> SensorFusionClient:
    global _fusion_client
    if _fusion_client is None:
        _fusion_client = SensorFusionClient()
    return _fusion_client


# =============================================================================
# VOICE COMMAND HANDLERS
# =============================================================================

async def handle_wifi_imaging_command(command: str) -> str:
    """Handle WiFi imaging voice commands."""
    client = get_wifi_client()
    cmd_lower = command.lower()
    
    # Router management
    if any(phrase in cmd_lower for phrase in ["list routers", "wifi routers", "show routers"]):
        return await client.list_routers()
    
    # Status
    if any(phrase in cmd_lower for phrase in ["wifi status", "imaging status", "wifi imaging status"]):
        return await client.get_imaging_status()
    
    # Observations
    if any(phrase in cmd_lower for phrase in ["through wall", "wall scan", "observations", "what's behind"]):
        return await client.get_through_wall_observations()
    
    # Scan
    if any(phrase in cmd_lower for phrase in ["wifi scan", "scan area", "imaging scan", "scan walls"]):
        return await client.scan_area()
    
    # Material analysis
    if any(phrase in cmd_lower for phrase in ["material", "wall type", "what material"]):
        return await client.get_material_analysis()
    
    return await client.get_imaging_status()


async def handle_vision_command(command: str) -> str:
    """Handle vision system voice commands."""
    client = get_vision_client()
    cmd_lower = command.lower()
    
    # Detections
    if any(phrase in cmd_lower for phrase in ["detections", "what do you see", "objects detected", "vision scan"]):
        return await client.get_detections()
    
    # Camera status
    if any(phrase in cmd_lower for phrase in ["camera status", "cameras", "camera health"]):
        return await client.get_camera_status()
    
    # Visual threats
    if any(phrase in cmd_lower for phrase in ["visual threat", "fire detected", "smoke detected"]):
        return await client.check_threats()
    
    # Specific detections
    targets = ["person", "vehicle", "aircraft", "ship", "fire", "smoke"]
    for target in targets:
        if f"detect {target}" in cmd_lower or f"any {target}" in cmd_lower or f"find {target}" in cmd_lower:
            return await client.detect_specific(target)
    
    return await client.get_detections()


async def handle_sensor_fusion_command(command: str) -> str:
    """Handle sensor fusion voice commands."""
    client = get_fusion_client()
    cmd_lower = command.lower()
    
    # Fusion status
    if any(phrase in cmd_lower for phrase in ["fusion status", "sensor fusion", "sensors active"]):
        return await client.get_fusion_status()
    
    # Triangulation
    if any(phrase in cmd_lower for phrase in ["triangulation", "triangulate", "position estimate"]):
        return await client.get_triangulation_result()
    
    return await client.get_fusion_status()
