"""
GIRU JARVIS - Advanced AI Assistant Backend v2.0
=================================================
A voice-activated AI assistant with multi-model intelligence,
real-time monitoring, and full ASGARD ecosystem integration.

Features:
- Multi-model AI (Gemini, Claude, GPT, Groq, Together, OpenRouter)
- Flexible wake words: "Giru", "Hey Giru", "Hello Giru"
- Continuous listening with natural conversation flow
- ASGARD integrations: Pricilla, Nysus, Silenus, Hunoid
- Email dictation and sending
- Desktop/file organization
- Real-time activity monitoring
- SQLite database for persistence
- JARVIS-like personality
"""

import asyncio
import json
import os
import queue
import re
import subprocess
import threading
import time
import webbrowser
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Optional

import psutil
import pyaudio
import pyttsx3
import pywhatkit
import requests
import speech_recognition as sr
import websockets
import wikipedia

# Import our modules
from ai_providers import get_ai_manager, MODELS, ModelTier
from database import get_database, ActivityType, ActivityStatus
from monitor import get_monitor, get_monitoring_server


# =============================================================================
# CONFIGURATION
# =============================================================================

# Wake words - any of these will activate Giru
WAKE_WORDS = ["giru", "hey giru", "hello giru", "hi giru", "okay giru"]

# Conversation settings
CONVERSATION_TIMEOUT = 120
ACTIVE_WINDOW_SECONDS = 60
MAX_OUTPUT_CHARS = 2500

# Voice settings
ELEVENLABS_API_KEY = os.getenv("ELEVENLABS_API_KEY")
ELEVENLABS_VOICE_ID = os.getenv("ELEVENLABS_VOICE_ID", "21m00Tcm4TlvDq8ikWAM")
ELEVENLABS_MODEL_ID = os.getenv("ELEVENLABS_MODEL_ID", "eleven_turbo_v2")
ELEVENLABS_OUTPUT_FORMAT = os.getenv("ELEVENLABS_OUTPUT_FORMAT", "pcm_44100")

# ASGARD system endpoints
PRICILLA_URL = os.getenv("PRICILLA_URL", "http://localhost:8092")
NYSUS_URL = os.getenv("NYSUS_URL", "http://localhost:8080")
SILENUS_URL = os.getenv("SILENUS_URL", "http://localhost:9093")
HUNOID_URL = os.getenv("HUNOID_URL", "http://localhost:8090")
GIRU_SECURITY_URL = os.getenv("GIRU_SECURITY_URL", "http://localhost:9090")

# Email settings
SMTP_SERVER = os.getenv("SMTP_SERVER", "smtp.gmail.com")
SMTP_PORT = int(os.getenv("SMTP_PORT", "587"))
SMTP_USER = os.getenv("SMTP_USER", "")
SMTP_PASSWORD = os.getenv("SMTP_PASSWORD", "")

# Workspace
WORKSPACE_ROOT = os.getenv(
    "GIRU_WORKSPACE_ROOT",
    os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))
)


# =============================================================================
# STATE MANAGEMENT
# =============================================================================

@dataclass
class ConversationContext:
    """Maintains conversation history and context."""
    messages: list = field(default_factory=list)
    current_task: Optional[str] = None
    pending_email: Optional[dict] = None
    pending_confirmation: Optional[str] = None
    user_name: str = "Sir"
    last_topic: Optional[str] = None
    conversation_id: Optional[int] = None
    
    def add_message(self, role: str, content: str, model_used: Optional[str] = None):
        self.messages.append({
            "role": role,
            "content": content,
            "timestamp": datetime.now().isoformat(),
            "model_used": model_used
        })
        # Keep last 20 messages
        if len(self.messages) > 20:
            self.messages = self.messages[-20:]
        
        # Store in database
        db = get_database()
        if self.conversation_id:
            db.add_message(self.conversation_id, role, content, model_used)
    
    def get_messages_for_ai(self) -> list[dict]:
        """Get messages formatted for AI API calls."""
        return [{"role": m["role"], "content": m["content"]} for m in self.messages]
    
    def start_session(self):
        """Start a new conversation session."""
        db = get_database()
        session_id = datetime.now().strftime("%Y%m%d_%H%M%S")
        self.conversation_id = db.create_conversation(session_id)
    
    def clear(self):
        if self.conversation_id:
            db = get_database()
            db.end_conversation(self.conversation_id)
        self.messages = []
        self.current_task = None
        self.pending_email = None
        self.pending_confirmation = None
        self.conversation_id = None


STATE = {
    "permissions": {"microphone": False, "camera": False},
    "status": "idle",
    "active_until": 0.0,
    "push_to_talk": False,
    "conversation_mode": False,
    "speaking": False,
    "current_model": None,
}

CONTEXT = ConversationContext()
CLIENTS = set()
EVENT_LOOP = None
TTS_INTERRUPT = threading.Event()

# Get singleton instances
AI_MANAGER = None
MONITOR = None
DB = None


def get_instances():
    """Initialize singleton instances."""
    global AI_MANAGER, MONITOR, DB
    if AI_MANAGER is None:
        AI_MANAGER = get_ai_manager()
    if MONITOR is None:
        MONITOR = get_monitor()
    if DB is None:
        DB = get_database()
    return AI_MANAGER, MONITOR, DB


# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

def now() -> float:
    return time.time()


def clamp_text(text: str) -> str:
    if len(text) <= MAX_OUTPUT_CHARS:
        return text
    return text[: MAX_OUTPUT_CHARS - 3] + "..."


def normalize_text(text: str) -> str:
    return " ".join(text.lower().strip().split())


def log_sync(message: str, level: str = "info") -> None:
    if EVENT_LOOP is None:
        return
    payload = {"type": "log", "message": message, "level": level}
    asyncio.run_coroutine_threadsafe(broadcast(payload), EVENT_LOOP)


async def broadcast(payload: dict) -> None:
    if not CLIENTS:
        return
    message = json.dumps(payload)
    await asyncio.gather(*(client.send(message) for client in CLIENTS), return_exceptions=True)


async def set_status(value: str) -> None:
    if STATE["status"] == value:
        return
    STATE["status"] = value
    await broadcast({"type": "status", "value": value})


async def send_utterance(role: str, text: str) -> None:
    await broadcast({"type": "utterance", "role": role, "text": text})


async def send_model_info(model_key: str) -> None:
    """Send current model info to UI."""
    STATE["current_model"] = model_key
    config = MODELS.get(model_key)
    if config:
        await broadcast({
            "type": "model_info",
            "model": model_key,
            "display_name": config.display_name,
            "provider": config.provider,
            "tier": config.tier.value,
        })


# =============================================================================
# WAKE WORD DETECTION
# =============================================================================

def contains_wake_word(transcript: str) -> tuple[bool, str]:
    normalized = normalize_text(transcript)
    for wake_word in WAKE_WORDS:
        if wake_word in normalized:
            idx = normalized.find(wake_word)
            remaining = normalized[idx + len(wake_word):].strip()
            return True, remaining
    return False, ""


def is_conversation_active() -> bool:
    return (
        STATE["push_to_talk"] or 
        STATE["active_until"] > now() or 
        STATE["conversation_mode"]
    )


# =============================================================================
# TEXT-TO-SPEECH
# =============================================================================

def play_elevenlabs(text: str) -> None:
    if not ELEVENLABS_API_KEY:
        raise RuntimeError("ELEVENLABS_API_KEY is not set.")
    
    url = f"https://api.elevenlabs.io/v1/text-to-speech/{ELEVENLABS_VOICE_ID}/stream"
    headers = {
        "xi-api-key": ELEVENLABS_API_KEY,
        "Accept": "audio/pcm",
        "Content-Type": "application/json",
    }
    payload = {
        "text": text,
        "model_id": ELEVENLABS_MODEL_ID,
        "voice_settings": {"stability": 0.5, "similarity_boost": 0.75},
    }
    params = {
        "output_format": ELEVENLABS_OUTPUT_FORMAT,
        "optimize_streaming_latency": 3,
    }
    
    response = requests.post(url, headers=headers, params=params, json=payload, stream=True, timeout=30)
    response.raise_for_status()

    audio = pyaudio.PyAudio()
    stream = audio.open(format=pyaudio.paInt16, channels=1, rate=44100, output=True)
    
    try:
        for chunk in response.iter_content(chunk_size=4096):
            if TTS_INTERRUPT.is_set():
                TTS_INTERRUPT.clear()
                break
            if chunk:
                stream.write(chunk)
    finally:
        stream.stop_stream()
        stream.close()
        audio.terminate()


def tts_worker(tts_queue: "queue.Queue[str]") -> None:
    engine = None
    if not ELEVENLABS_API_KEY:
        engine = pyttsx3.init()
        engine.setProperty("rate", 175)
        log_sync("ElevenLabs not configured; using pyttsx3 fallback.", "warn")

    while True:
        text = tts_queue.get()
        if text is None:
            break
        
        STATE["speaking"] = True
        if EVENT_LOOP:
            asyncio.run_coroutine_threadsafe(set_status("speaking"), EVENT_LOOP)
        
        try:
            if ELEVENLABS_API_KEY:
                play_elevenlabs(text)
            else:
                engine.say(text)
                engine.runAndWait()
        except Exception as exc:
            log_sync(f"TTS error: {exc}", "error")
        finally:
            STATE["speaking"] = False
            if EVENT_LOOP and is_conversation_active():
                asyncio.run_coroutine_threadsafe(set_status("listening"), EVENT_LOOP)


# =============================================================================
# AI RESPONSE GENERATION
# =============================================================================

async def get_ai_response(messages: list[dict], complexity: str = "standard") -> tuple[str, str]:
    """Get AI response using the multi-model system."""
    ai_manager, monitor, db = get_instances()
    
    # Track the AI query
    async with monitor.track_activity(
        ActivityType.CONVERSATION,
        f"Generating AI response ({complexity})",
        {"complexity": complexity, "message_count": len(messages)}
    ) as tracker:
        start_time = time.time()
        
        try:
            response, model_used = await ai_manager.chat(
                messages=messages,
                task_complexity=complexity,
            )
            
            latency_ms = int((time.time() - start_time) * 1000)
            
            # Record usage
            db.record_model_usage(
                model_key=model_used,
                latency_ms=latency_ms,
            )
            
            await send_model_info(model_used)
            await tracker.log(f"Response generated in {latency_ms}ms using {model_used}")
            
            return response, model_used
            
        except Exception as e:
            log_sync(f"AI error: {e}", "error")
            return f"I apologize, but I encountered an error: {str(e)}", "fallback"


def determine_complexity(text: str) -> str:
    """Determine task complexity from the request."""
    text_lower = text.lower()
    
    # Expert tasks
    if any(word in text_lower for word in ["analyze", "complex", "detailed", "explain thoroughly", "deep dive"]):
        return "expert"
    
    # Advanced tasks
    if any(word in text_lower for word in ["code", "debug", "refactor", "architecture", "design", "strategy"]):
        return "advanced"
    
    # Basic tasks
    if any(word in text_lower for word in ["what time", "hello", "hi", "thanks", "bye", "yes", "no"]):
        return "basic"
    
    return "standard"


# =============================================================================
# ASGARD SYSTEM INTEGRATIONS
# =============================================================================

async def pricilla_get_missions() -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.ASGARD_QUERY,
        "Querying Pricilla missions"
    ):
        try:
            response = await asyncio.to_thread(
                requests.get, f"{PRICILLA_URL}/api/v1/missions", timeout=5
            )
            if response.status_code == 200:
                missions = response.json()
                if not missions:
                    return "There are no active missions at the moment."
                
                summary = []
                for m in missions[:5]:
                    status = m.get("status", "unknown")
                    payload_type = m.get("payload_type", "unknown")
                    summary.append(f"{payload_type} mission: {status}")
                
                return f"I found {len(missions)} active missions. " + ". ".join(summary)
            return "I couldn't reach Pricilla at the moment."
        except Exception as e:
            log_sync(f"Pricilla error: {e}", "error")
            return "Pricilla appears to be offline."


async def pricilla_get_target_eta(mission_id: str = None) -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.ASGARD_QUERY,
        "Getting target ETA from Pricilla"
    ):
        try:
            url = f"{PRICILLA_URL}/api/v1/missions"
            if mission_id:
                url = f"{PRICILLA_URL}/api/v1/missions/{mission_id}"
            
            response = await asyncio.to_thread(requests.get, url, timeout=5)
            if response.status_code == 200:
                data = response.json()
                missions = [data] if mission_id else data
                
                if not missions:
                    return "No active targeting missions found."
                
                active = [m for m in missions if m.get("status") in ["active", "in_progress"]]
                if not active:
                    return "No missions currently in progress."
                
                mission = active[0]
                eta = mission.get("eta_seconds", 0)
                if eta > 0:
                    minutes = int(eta // 60)
                    seconds = int(eta % 60)
                    if minutes > 0:
                        return f"Estimated time to target: {minutes} minutes and {seconds} seconds."
                    return f"Estimated time to target: {seconds} seconds."
                return "Target engagement is imminent."
            return "Unable to retrieve targeting information."
        except Exception as e:
            log_sync(f"Pricilla ETA error: {e}", "error")
            return "Pricilla targeting system is not responding."


async def pricilla_get_targeting_metrics() -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.ASGARD_QUERY,
        "Getting targeting metrics"
    ):
        try:
            response = await asyncio.to_thread(
                requests.get, f"{PRICILLA_URL}/api/v1/metrics/targeting", timeout=5
            )
            if response.status_code == 200:
                metrics = response.json()
                hit_prob = metrics.get("hit_probability", 0) * 100
                cep = metrics.get("cep_meters", 0)
                stealth = metrics.get("stealth_score", 0) * 100
                
                return (
                    f"Targeting metrics: {hit_prob:.1f}% hit probability, "
                    f"circular error probable of {cep:.1f} meters, "
                    f"stealth rating at {stealth:.1f}%."
                )
            return "Unable to retrieve targeting metrics."
        except Exception as e:
            log_sync(f"Pricilla metrics error: {e}", "error")
            return "Targeting metrics unavailable."


async def nysus_get_status() -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.ASGARD_QUERY,
        "Checking Nysus status"
    ):
        try:
            response = await asyncio.to_thread(
                requests.get, f"{NYSUS_URL}/health", timeout=5
            )
            if response.status_code == 200:
                data = response.json()
                status = data.get("status", "unknown")
                return f"ASGARD central systems are {status}."
            return "Unable to reach central command."
        except Exception as e:
            log_sync(f"Nysus error: {e}", "error")
            return "Nysus central systems appear to be offline."


async def nysus_get_dashboard_stats() -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.ASGARD_QUERY,
        "Getting dashboard statistics"
    ):
        try:
            response = await asyncio.to_thread(
                requests.get, f"{NYSUS_URL}/api/dashboard/stats", timeout=5
            )
            if response.status_code == 200:
                stats = response.json()
                alerts = stats.get("active_alerts", 0)
                missions = stats.get("active_missions", 0)
                satellites = stats.get("satellites_online", 0)
                hunoids = stats.get("hunoids_active", 0)
                
                parts = []
                if alerts > 0:
                    parts.append(f"{alerts} active alerts")
                if missions > 0:
                    parts.append(f"{missions} missions in progress")
                if satellites > 0:
                    parts.append(f"{satellites} satellites online")
                if hunoids > 0:
                    parts.append(f"{hunoids} Hunoid units active")
                
                if parts:
                    return "Current ASGARD status: " + ", ".join(parts) + "."
                return "All ASGARD systems are quiet."
            return "Dashboard data unavailable."
        except Exception as e:
            log_sync(f"Dashboard error: {e}", "error")
            return "Unable to retrieve dashboard statistics."


async def nysus_get_alerts() -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.ASGARD_QUERY,
        "Checking alerts"
    ):
        try:
            response = await asyncio.to_thread(
                requests.get, f"{NYSUS_URL}/api/alerts", timeout=5
            )
            if response.status_code == 200:
                alerts = response.json()
                if not alerts:
                    return "No active alerts at this time."
                
                high_priority = [a for a in alerts if a.get("priority") == "high"]
                if high_priority:
                    return f"Warning: {len(high_priority)} high-priority alerts detected. " + \
                           high_priority[0].get("message", "Details unavailable.")
                
                return f"There are {len(alerts)} alerts in the system."
            return "Alert system unavailable."
        except Exception as e:
            log_sync(f"Alerts error: {e}", "error")
            return "Unable to retrieve alert information."


async def silenus_get_coverage() -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.ASGARD_QUERY,
        "Checking Silenus satellite coverage"
    ):
        try:
            response = await asyncio.to_thread(
                requests.get, f"{SILENUS_URL}/healthz", timeout=5
            )
            if response.status_code == 200:
                return "Silenus orbital systems are online and operational."
            return "Silenus systems status unknown."
        except Exception as e:
            log_sync(f"Silenus error: {e}", "error")
            return "Silenus satellite systems are not responding."


async def hunoid_get_status() -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.ASGARD_QUERY,
        "Checking Hunoid status"
    ):
        try:
            response = await asyncio.to_thread(
                requests.get, f"{HUNOID_URL}/api/status", timeout=5
            )
            if response.status_code == 200:
                data = response.json()
                state = data.get("state", "unknown")
                battery = data.get("battery_percent", 0)
                mission = data.get("current_mission", "none")
                
                return (
                    f"Hunoid unit status: {state}, "
                    f"battery at {battery}%, "
                    f"current mission: {mission}."
                )
            return "Hunoid control systems unavailable."
        except Exception as e:
            log_sync(f"Hunoid error: {e}", "error")
            return "Unable to reach Hunoid units."


async def giru_security_get_threats() -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.ASGARD_QUERY,
        "Checking security threats"
    ):
        try:
            response = await asyncio.to_thread(
                requests.get, f"{GIRU_SECURITY_URL}/api/threats", timeout=5
            )
            if response.status_code == 200:
                threats = response.json()
                if not threats:
                    return "No active security threats detected. All systems are secure."
                
                critical = [t for t in threats if t.get("severity") == "critical"]
                if critical:
                    return f"Alert: {len(critical)} critical security threats detected. Immediate attention required."
                
                return f"Security monitoring shows {len(threats)} potential threats under observation."
            return "Security systems status unknown."
        except Exception as e:
            log_sync(f"Giru Security error: {e}", "error")
            return "Security monitoring systems are offline."


# =============================================================================
# EMAIL FUNCTIONALITY
# =============================================================================

async def send_email(to: str, subject: str, body: str) -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.EMAIL,
        f"Sending email to {to}",
        {"recipient": to, "subject": subject}
    ) as tracker:
        if not SMTP_USER or not SMTP_PASSWORD:
            return "Email is not configured. Please set SMTP_USER and SMTP_PASSWORD."
        
        try:
            import smtplib
            from email.mime.text import MIMEText
            from email.mime.multipart import MIMEMultipart
            
            await tracker.update_progress(0.3, "Composing email...")
            
            def _send():
                msg = MIMEMultipart()
                msg['From'] = SMTP_USER
                msg['To'] = to
                msg['Subject'] = subject
                msg.attach(MIMEText(body, 'plain'))
                
                server = smtplib.SMTP(SMTP_SERVER, SMTP_PORT)
                server.starttls()
                server.login(SMTP_USER, SMTP_PASSWORD)
                server.send_message(msg)
                server.quit()
            
            await tracker.update_progress(0.6, "Connecting to mail server...")
            await asyncio.to_thread(_send)
            
            # Log to database
            db.log_email(to, subject, body, "sent")
            
            await tracker.update_progress(1.0, "Email sent successfully")
            return f"Email sent successfully to {to}."
            
        except Exception as e:
            db.log_email(to, subject, body, "failed", str(e))
            log_sync(f"Email error: {e}", "error")
            return f"Failed to send email: {str(e)}"


def parse_email_request(text: str) -> dict:
    email = {"to": None, "subject": None, "body": None}
    
    to_patterns = [
        r"to\s+(\S+@\S+)",
        r"send (?:an )?email to\s+(\S+)",
        r"email\s+(\S+@\S+)",
    ]
    for pattern in to_patterns:
        match = re.search(pattern, text, re.IGNORECASE)
        if match:
            email["to"] = match.group(1).strip()
            break
    
    subject_patterns = [
        r"subject[:\s]+[\"']?([^\"'\n]+)[\"']?",
        r"about\s+[\"']?([^\"'\n]+)[\"']?",
        r"regarding\s+[\"']?([^\"'\n]+)[\"']?",
    ]
    for pattern in subject_patterns:
        match = re.search(pattern, text, re.IGNORECASE)
        if match:
            email["subject"] = match.group(1).strip()
            break
    
    body_patterns = [
        r"(?:saying|message)[:\s]+[\"']?(.+)[\"']?$",
        r"(?:body|content)[:\s]+[\"']?(.+)[\"']?$",
    ]
    for pattern in body_patterns:
        match = re.search(pattern, text, re.IGNORECASE | re.DOTALL)
        if match:
            email["body"] = match.group(1).strip()
            break
    
    return email


# =============================================================================
# DESKTOP/FILE ORGANIZATION
# =============================================================================

async def organize_desktop() -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.FILE_OPERATION,
        "Organizing desktop files"
    ) as tracker:
        desktop = Path.home() / "Desktop"
        if not desktop.exists():
            return "Desktop folder not found."
        
        categories = {
            "Documents": [".pdf", ".doc", ".docx", ".txt", ".md", ".rtf", ".odt"],
            "Images": [".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp"],
            "Videos": [".mp4", ".avi", ".mov", ".mkv", ".wmv", ".webm"],
            "Audio": [".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a"],
            "Archives": [".zip", ".rar", ".7z", ".tar", ".gz"],
            "Code": [".py", ".js", ".ts", ".go", ".java", ".cpp", ".c", ".h"],
            "Spreadsheets": [".xlsx", ".xls", ".csv"],
            "Presentations": [".ppt", ".pptx", ".key"],
        }
        
        def _organize():
            moved = 0
            for file in desktop.iterdir():
                if file.is_file() and not file.name.startswith("."):
                    ext = file.suffix.lower()
                    for category, extensions in categories.items():
                        if ext in extensions:
                            category_dir = desktop / category
                            category_dir.mkdir(exist_ok=True)
                            file.rename(category_dir / file.name)
                            moved += 1
                            break
            return moved
        
        await tracker.update_progress(0.5, "Scanning files...")
        moved = await asyncio.to_thread(_organize)
        
        if moved > 0:
            return f"Desktop organized. Moved {moved} files into categorized folders."
        return "Desktop is already organized. No files needed to be moved."


async def list_folder(path: str) -> str:
    folder = Path(path).expanduser()
    if not folder.exists():
        return f"Folder {path} does not exist."
    if not folder.is_dir():
        return f"{path} is not a folder."
    
    items = list(folder.iterdir())[:20]
    
    folders = [i.name for i in items if i.is_dir()]
    files = [i.name for i in items if i.is_file()]
    
    result = []
    if folders:
        result.append(f"Folders: {', '.join(folders[:10])}")
    if files:
        result.append(f"Files: {', '.join(files[:10])}")
    
    if not result:
        return f"The folder {path} is empty."
    
    return ". ".join(result)


# =============================================================================
# TERMINAL / COMMAND EXECUTION
# =============================================================================

async def run_subprocess(command: str, cwd: str | None = None) -> tuple[str, int, int]:
    """Run a subprocess command with monitoring."""
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.TERMINAL,
        f"Running: {command[:50]}...",
        {"command": command, "cwd": cwd}
    ) as tracker:
        start_time = time.time()
        
        def _run():
            result = subprocess.run(
                command, shell=True, capture_output=True, text=True, cwd=cwd
            )
            return result.stdout.strip() or result.stderr.strip() or "(no output)", result.returncode
        
        output, exit_code = await asyncio.to_thread(_run)
        duration_ms = int((time.time() - start_time) * 1000)
        
        # Log to database
        db.log_command(command, cwd, output[:1000], exit_code, duration_ms)
        
        return clamp_text(output), exit_code, duration_ms


async def git_status(path: str = WORKSPACE_ROOT) -> str:
    output, exit_code, _ = await run_subprocess("git status --short", cwd=path)
    if "not a git repository" in output.lower():
        return "This is not a git repository."
    if not output or output == "(no output)":
        return "Working directory is clean. No changes detected."
    
    lines = output.strip().split("\n")
    return f"Git status shows {len(lines)} changed files."


async def analyze_project(path: str = WORKSPACE_ROOT) -> str:
    ai_manager, monitor, db = get_instances()
    
    async with monitor.track_activity(
        ActivityType.CODE,
        "Analyzing project structure"
    ):
        project_path = Path(path)
        if not project_path.exists():
            return f"Project path {path} does not exist."
        
        def _analyze():
            stats = {"files": 0, "folders": 0, "extensions": {}}
            
            for item in project_path.rglob("*"):
                parts = item.parts
                if any(p.startswith(".") or p in ["node_modules", "__pycache__", "venv", ".venv"] for p in parts):
                    continue
                
                if item.is_file():
                    stats["files"] += 1
                    ext = item.suffix.lower() or "no extension"
                    stats["extensions"][ext] = stats["extensions"].get(ext, 0) + 1
                elif item.is_dir():
                    stats["folders"] += 1
            
            return stats
        
        stats = await asyncio.to_thread(_analyze)
        
        top_types = sorted(stats["extensions"].items(), key=lambda x: x[1], reverse=True)[:5]
        type_summary = ", ".join([f"{count} {ext}" for ext, count in top_types])
        
        return (
            f"Project analysis: {stats['files']} files across {stats['folders']} folders. "
            f"Primary file types: {type_summary}."
        )


# =============================================================================
# COMMAND HANDLER
# =============================================================================

def get_jarvis_greeting() -> str:
    hour = datetime.now().hour
    if hour < 12:
        time_greeting = "Good morning"
    elif hour < 17:
        time_greeting = "Good afternoon"
    else:
        time_greeting = "Good evening"
    
    return f"{time_greeting}, {CONTEXT.user_name}. How may I assist you?"


async def handle_command(text: str) -> str:
    """Handle a user command with JARVIS personality and AI intelligence."""
    lower = normalize_text(text)
    complexity = determine_complexity(text)
    
    # Add to context
    CONTEXT.add_message("user", text)
    
    # Handle pending confirmations
    if CONTEXT.pending_confirmation:
        if any(word in lower for word in ["yes", "confirm", "do it", "send it", "go ahead"]):
            if CONTEXT.pending_confirmation == "email" and CONTEXT.pending_email:
                result = await send_email(
                    CONTEXT.pending_email["to"],
                    CONTEXT.pending_email["subject"],
                    CONTEXT.pending_email["body"]
                )
                CONTEXT.pending_confirmation = None
                CONTEXT.pending_email = None
                return result
        elif any(word in lower for word in ["no", "cancel", "nevermind", "stop"]):
            CONTEXT.pending_confirmation = None
            CONTEXT.pending_email = None
            return "Understood. Request cancelled."
        CONTEXT.pending_confirmation = None
    
    # =========================================================================
    # QUICK RESPONSES (No AI needed)
    # =========================================================================
    
    if lower in ["hello", "hi", "hey", "greetings"]:
        return get_jarvis_greeting()
    
    if any(phrase in lower for phrase in ["how are you", "how do you do", "what's up", "whats up"]):
        return f"All systems are operational, {CONTEXT.user_name}. Ready to assist."
    
    if any(phrase in lower for phrase in ["thank you", "thanks", "appreciate"]):
        return f"My pleasure, {CONTEXT.user_name}. Is there anything else?"
    
    if any(phrase in lower for phrase in ["goodbye", "bye", "see you", "that's all", "thats all"]):
        STATE["conversation_mode"] = False
        STATE["active_until"] = 0
        return f"Very well, {CONTEXT.user_name}. I'll be here when you need me."
    
    if any(phrase in lower for phrase in ["what time", "current time", "time is it"]):
        return f"The current time is {datetime.now().strftime('%I:%M %p on %A, %B %d')}."
    
    if any(phrase in lower for phrase in ["what date", "today's date", "what day"]):
        return f"Today is {datetime.now().strftime('%A, %B %d, %Y')}."
    
    # =========================================================================
    # MODEL MANAGEMENT
    # =========================================================================
    
    if any(phrase in lower for phrase in ["what model", "which model", "current model"]):
        if STATE["current_model"]:
            config = MODELS.get(STATE["current_model"])
            if config:
                return f"I'm currently using {config.display_name} from {config.provider}."
        return "I'll select the most appropriate model for each task."
    
    if any(phrase in lower for phrase in ["available models", "list models", "show models"]):
        ai_manager, _, _ = get_instances()
        available = ai_manager.get_available_models()
        available_list = [m for m in available if m["available"]]
        free_models = [m for m in available_list if m["free"]]
        
        return (
            f"I have access to {len(available_list)} AI models. "
            f"{len(free_models)} are free to use. "
            f"Available providers: Groq, Together AI, Google Gemini, OpenAI, Anthropic, and OpenRouter."
        )
    
    if "use model" in lower or "switch to" in lower:
        # Extract model name
        model_patterns = [
            r"use (?:model\s+)?(\S+)",
            r"switch to (\S+)",
        ]
        for pattern in model_patterns:
            match = re.search(pattern, lower)
            if match:
                model_name = match.group(1)
                # Find matching model
                for key, config in MODELS.items():
                    if model_name in key.lower() or model_name in config.display_name.lower():
                        STATE["current_model"] = key
                        return f"Switching to {config.display_name}. I'll use this model for subsequent requests."
                return f"I couldn't find a model matching '{model_name}'. Try 'list models' to see available options."
    
    # =========================================================================
    # ASGARD SYSTEM QUERIES
    # =========================================================================
    
    if any(phrase in lower for phrase in ["target", "targeting", "mission", "pricilla"]):
        if any(phrase in lower for phrase in ["eta", "time", "how long", "when"]):
            return await pricilla_get_target_eta()
        if any(phrase in lower for phrase in ["metric", "accuracy", "probability"]):
            return await pricilla_get_targeting_metrics()
        return await pricilla_get_missions()
    
    if any(phrase in lower for phrase in ["status", "system", "nysus", "dashboard"]):
        if "alert" in lower:
            return await nysus_get_alerts()
        if "dashboard" in lower or "stat" in lower:
            return await nysus_get_dashboard_stats()
        return await nysus_get_status()
    
    if any(phrase in lower for phrase in ["satellite", "silenus", "orbital", "coverage"]):
        return await silenus_get_coverage()
    
    if any(phrase in lower for phrase in ["hunoid", "robot", "unit"]):
        return await hunoid_get_status()
    
    if any(phrase in lower for phrase in ["security", "threat", "attack", "breach"]):
        return await giru_security_get_threats()
    
    # =========================================================================
    # EMAIL
    # =========================================================================
    
    if any(phrase in lower for phrase in ["send email", "send an email", "email to", "compose email"]):
        email_data = parse_email_request(text)
        
        if not email_data["to"]:
            CONTEXT.current_task = "email"
            return "Who would you like me to send the email to?"
        
        if not email_data["subject"]:
            CONTEXT.pending_email = email_data
            return f"What should the subject line be for the email to {email_data['to']}?"
        
        if not email_data["body"]:
            CONTEXT.pending_email = email_data
            return "What would you like the email to say?"
        
        CONTEXT.pending_email = email_data
        CONTEXT.pending_confirmation = "email"
        return (
            f"Ready to send email to {email_data['to']} with subject '{email_data['subject']}'. "
            f"Shall I send it?"
        )
    
    # =========================================================================
    # DESKTOP AND FILE ORGANIZATION
    # =========================================================================
    
    if any(phrase in lower for phrase in ["organize desktop", "clean desktop", "tidy desktop"]):
        return await organize_desktop()
    
    if any(phrase in lower for phrase in ["list folder", "show folder", "what's in", "whats in"]):
        path_match = re.search(r"(?:in|folder)\s+[\"']?([^\"']+)[\"']?", text, re.IGNORECASE)
        if path_match:
            return await list_folder(path_match.group(1).strip())
        return await list_folder("~/Desktop")
    
    # =========================================================================
    # PROJECT AND GIT COMMANDS
    # =========================================================================
    
    if lower.startswith("git "):
        log_sync(f"Running git command in {WORKSPACE_ROOT}")
        output, exit_code, _ = await run_subprocess(text, cwd=WORKSPACE_ROOT)
        return f"Git output: {output}"
    
    if any(phrase in lower for phrase in ["project status", "repo status", "git status"]):
        return await git_status()
    
    if any(phrase in lower for phrase in ["analyze project", "project analysis", "project info"]):
        return await analyze_project()
    
    # =========================================================================
    # SYSTEM COMMANDS
    # =========================================================================
    
    if lower.startswith("open "):
        target = text[5:].strip()
        if target.startswith("http://") or target.startswith("https://"):
            webbrowser.open(target)
            return f"Opening {target} in your browser."
        pywhatkit.search(target)
        return f"Searching for {target}."
    
    if lower.startswith("search "):
        query = text[7:].strip()
        pywhatkit.search(query)
        return f"Searching for {query}."
    
    if lower.startswith("wikipedia "):
        query = text[len("wikipedia "):].strip()
        try:
            summary = wikipedia.summary(query, sentences=2)
            return summary
        except Exception:
            return f"I couldn't find information about {query} on Wikipedia."
    
    if lower in {"list apps", "list applications", "what's running", "whats running", "running apps"}:
        names = []
        for proc in psutil.process_iter(["name"]):
            name = proc.info.get("name")
            if name:
                names.append(name)
        unique = sorted(set(names))[:25]
        if not unique:
            return "No running applications detected."
        return f"Currently running applications include: {', '.join(unique[:15])}."
    
    if lower.startswith("run "):
        command = text[4:].strip()
        log_sync(f"Executing command: {command}")
        output, exit_code, _ = await run_subprocess(command)
        return f"Command output: {output}"
    
    # =========================================================================
    # AI-POWERED RESPONSES
    # =========================================================================
    
    # For anything else, use the AI system
    messages = CONTEXT.get_messages_for_ai()
    response, model_used = await get_ai_response(messages, complexity)
    CONTEXT.add_message("assistant", response, model_used)
    
    return response


# =============================================================================
# SPEECH PROCESSING
# =============================================================================

async def process_text(text: str, tts_queue: "queue.Queue[str]") -> None:
    await send_utterance("user", text)
    await set_status("active")
    
    response = await handle_command(text)
    
    await send_utterance("assistant", response)
    tts_queue.put(response)
    
    STATE["active_until"] = now() + ACTIVE_WINDOW_SECONDS
    STATE["conversation_mode"] = True


def speech_loop(tts_queue: "queue.Queue[str]") -> None:
    recognizer = sr.Recognizer()
    recognizer.dynamic_energy_threshold = True
    recognizer.pause_threshold = 0.6
    recognizer.energy_threshold = 300

    while True:
        if not STATE["permissions"]["microphone"]:
            time.sleep(0.3)
            continue
        
        if STATE["speaking"]:
            time.sleep(0.1)
            continue

        try:
            with sr.Microphone() as source:
                recognizer.adjust_for_ambient_noise(source, duration=0.3)
                
                timeout = 2 if is_conversation_active() else 3
                phrase_limit = 8 if is_conversation_active() else 5
                
                audio = recognizer.listen(
                    source, 
                    timeout=timeout, 
                    phrase_time_limit=phrase_limit
                )
        except sr.WaitTimeoutError:
            if STATE["conversation_mode"] and STATE["active_until"] < now():
                STATE["conversation_mode"] = False
                asyncio.run_coroutine_threadsafe(set_status("listening"), EVENT_LOOP)
            continue

        try:
            transcript = recognizer.recognize_google(audio)
        except sr.UnknownValueError:
            continue
        except sr.RequestError as exc:
            log_sync(f"Speech service error: {exc}", "error")
            time.sleep(1.0)
            continue

        if not transcript:
            continue
        
        log_sync(f"Heard: {transcript}")
        
        wake_detected, remaining_text = contains_wake_word(transcript)
        
        if wake_detected:
            STATE["active_until"] = now() + ACTIVE_WINDOW_SECONDS
            STATE["conversation_mode"] = True
            
            # Start new conversation session
            if not CONTEXT.conversation_id:
                CONTEXT.start_session()
            
            asyncio.run_coroutine_threadsafe(set_status("active"), EVENT_LOOP)
            log_sync("Wake word detected. Listening for commands.")
            
            if remaining_text:
                asyncio.run_coroutine_threadsafe(
                    process_text(remaining_text, tts_queue), EVENT_LOOP
                )
            else:
                greeting = get_jarvis_greeting()
                asyncio.run_coroutine_threadsafe(
                    send_utterance("assistant", greeting), EVENT_LOOP
                )
                tts_queue.put(greeting)
            continue
        
        if is_conversation_active():
            asyncio.run_coroutine_threadsafe(
                process_text(transcript, tts_queue), EVENT_LOOP
            )


# =============================================================================
# WEBSOCKET HANDLERS
# =============================================================================

async def handle_message(message: str, tts_queue: "queue.Queue[str]") -> None:
    data = json.loads(message)

    if data.get("type") == "client_hello":
        await broadcast({"type": "status", "value": STATE["status"]})
        for key, value in STATE["permissions"].items():
            await broadcast({"type": "permission", "key": key, "value": value})
        
        # Send available models info
        ai_manager, _, _ = get_instances()
        models = ai_manager.get_available_models()
        await broadcast({"type": "models_list", "models": models})
        return

    if data.get("type") == "permission":
        key = data.get("key")
        value = bool(data.get("value"))
        if key in STATE["permissions"]:
            STATE["permissions"][key] = value
            await broadcast({"type": "permission", "key": key, "value": value})
            
            if key == "microphone":
                if value:
                    await set_status("listening")
                    log_sync("Microphone enabled. Say 'Giru' to activate.")
                else:
                    await set_status("idle")
                    STATE["conversation_mode"] = False
            return

    if data.get("type") == "push_to_talk":
        STATE["push_to_talk"] = bool(data.get("active"))
        if STATE["push_to_talk"]:
            STATE["conversation_mode"] = True
            if not CONTEXT.conversation_id:
                CONTEXT.start_session()
            await set_status("active")
        else:
            await set_status("listening")
        return

    if data.get("type") == "text":
        text = data.get("text", "").strip()
        if text:
            if not CONTEXT.conversation_id:
                CONTEXT.start_session()
            await process_text(text, tts_queue)

    if data.get("type") == "interrupt":
        TTS_INTERRUPT.set()
        log_sync("Speech interrupted.")
    
    if data.get("type") == "select_model":
        model_key = data.get("model")
        if model_key in MODELS:
            STATE["current_model"] = model_key
            await send_model_info(model_key)


async def handler(websocket) -> None:
    CLIENTS.add(websocket)
    await broadcast({"type": "log", "message": "Client connected."})
    try:
        async for message in websocket:
            await handle_message(message, handler.tts_queue)
    finally:
        CLIENTS.discard(websocket)
        await broadcast({"type": "log", "message": "Client disconnected."})


# =============================================================================
# MAIN ENTRY POINT
# =============================================================================

async def main() -> None:
    global EVENT_LOOP
    EVENT_LOOP = asyncio.get_running_loop()
    tts_queue: "queue.Queue[str]" = queue.Queue()
    handler.tts_queue = tts_queue

    # Initialize singletons
    ai_manager, monitor, db = get_instances()
    
    # Start TTS worker
    tts_thread = threading.Thread(target=tts_worker, args=(tts_queue,), daemon=True)
    tts_thread.start()

    # Start speech recognition loop
    listener_thread = threading.Thread(target=speech_loop, args=(tts_queue,), daemon=True)
    listener_thread.start()

    # Get available models
    available_models = ai_manager.get_available_models()
    free_models = [m for m in available_models if m["available"] and m["free"]]
    paid_models = [m for m in available_models if m["available"] and not m["free"]]

    port = int(os.getenv("GIRU_PORT", "7777"))
    monitor_port = int(os.getenv("GIRU_MONITOR_PORT", "7778"))
    
    print(f"""
╔══════════════════════════════════════════════════════════════════════════╗
║                        GIRU JARVIS v2.0                                 ║
║              Advanced Multi-Model AI Assistant                          ║
║                     ASGARD Platform                                     ║
╠══════════════════════════════════════════════════════════════════════════╣
║  Wake Words: "Giru", "Hey Giru", "Hello Giru"                           ║
║  Backend:    ws://127.0.0.1:{port:<5}                                      ║
║  Monitor:    ws://127.0.0.1:{monitor_port:<5} | file:///.../monitor.html       ║
╠══════════════════════════════════════════════════════════════════════════╣
║  AI Models Available:                                                    ║
║    Free:  {len(free_models):>2} models (Groq, Together AI, Gemini Free Tier)           ║
║    Paid:  {len(paid_models):>2} models (Claude, GPT-4, Gemini Pro)                     ║
╠══════════════════════════════════════════════════════════════════════════╣
║  ASGARD Integrations:                                                    ║
║    • Pricilla (Targeting): {PRICILLA_URL:<40} ║
║    • Nysus (Command):      {NYSUS_URL:<40} ║
║    • Silenus (Orbital):    {SILENUS_URL:<40} ║
║    • Hunoid (Robots):      {HUNOID_URL:<40} ║
╚══════════════════════════════════════════════════════════════════════════╝
    """)
    
    # Start monitoring server in background
    monitoring_server = get_monitoring_server(monitor_port)
    monitor_task = asyncio.create_task(monitoring_server.start())
    
    # Start main WebSocket server
    async with websockets.serve(handler, "127.0.0.1", port):
        log_sync(f"Giru JARVIS backend listening on ws://127.0.0.1:{port}")
        await asyncio.Future()


if __name__ == "__main__":
    asyncio.run(main())
