"""
GIRU Skills Module - Extended Capabilities
==========================================
This module provides additional skills and knowledge domains for Giru JARVIS.

Skills included:
- Weather briefings
- News summaries
- Calendar/scheduling
- Smart home control
- Code review/generation
- Document analysis
- Task planning & orchestration
- Research assistant
- System administration
"""

import asyncio
import json
import os
import re
import subprocess
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from pathlib import Path
from typing import Optional, List, Dict, Any
import aiohttp

from database import get_database, ActivityType
from monitor import get_monitor


# =============================================================================
# WEATHER SKILL
# =============================================================================

class WeatherSkill:
    """Weather information and forecasts."""
    
    OPENWEATHER_API_KEY = os.getenv("OPENWEATHER_API_KEY", "")
    BASE_URL = "https://api.openweathermap.org/data/2.5"
    
    @classmethod
    async def get_current_weather(cls, city: str = "New York") -> str:
        """Get current weather for a city."""
        if not cls.OPENWEATHER_API_KEY:
            return f"Weather data unavailable. Current conditions in {city} are typical for this time of year."
        
        try:
            async with aiohttp.ClientSession() as session:
                url = f"{cls.BASE_URL}/weather"
                params = {"q": city, "appid": cls.OPENWEATHER_API_KEY, "units": "metric"}
                async with session.get(url, params=params) as response:
                    if response.status == 200:
                        data = await response.json()
                        temp = data["main"]["temp"]
                        feels_like = data["main"]["feels_like"]
                        condition = data["weather"][0]["description"]
                        humidity = data["main"]["humidity"]
                        wind = data["wind"]["speed"]
                        
                        return (
                            f"Current weather in {city}: {condition.capitalize()}. "
                            f"Temperature is {temp:.1f}°C, feels like {feels_like:.1f}°C. "
                            f"Humidity at {humidity}%, wind speed {wind:.1f} m/s."
                        )
                    return f"Unable to retrieve weather data for {city}."
        except Exception as e:
            return f"Weather service unavailable: {str(e)}"
    
    @classmethod
    async def get_forecast(cls, city: str = "New York", days: int = 3) -> str:
        """Get weather forecast."""
        if not cls.OPENWEATHER_API_KEY:
            return f"Forecast data unavailable. Plan for variable conditions in {city}."
        
        try:
            async with aiohttp.ClientSession() as session:
                url = f"{cls.BASE_URL}/forecast"
                params = {"q": city, "appid": cls.OPENWEATHER_API_KEY, "units": "metric", "cnt": days * 8}
                async with session.get(url, params=params) as response:
                    if response.status == 200:
                        data = await response.json()
                        forecasts = []
                        
                        # Group by day
                        by_day = {}
                        for item in data["list"][:days * 8]:
                            dt = datetime.fromtimestamp(item["dt"])
                            day = dt.strftime("%A")
                            if day not in by_day:
                                by_day[day] = {"temps": [], "conditions": []}
                            by_day[day]["temps"].append(item["main"]["temp"])
                            by_day[day]["conditions"].append(item["weather"][0]["main"])
                        
                        for day, info in list(by_day.items())[:days]:
                            avg_temp = sum(info["temps"]) / len(info["temps"])
                            main_condition = max(set(info["conditions"]), key=info["conditions"].count)
                            forecasts.append(f"{day}: {main_condition}, around {avg_temp:.0f}°C")
                        
                        return f"Forecast for {city}: " + ". ".join(forecasts)
                    return f"Unable to retrieve forecast for {city}."
        except Exception as e:
            return f"Forecast service unavailable: {str(e)}"


# =============================================================================
# NEWS SKILL
# =============================================================================

class NewsSkill:
    """News and current events."""
    
    NEWS_API_KEY = os.getenv("NEWS_API_KEY", "")
    BASE_URL = "https://newsapi.org/v2"
    
    @classmethod
    async def get_headlines(cls, category: str = "general", country: str = "us") -> str:
        """Get top headlines."""
        if not cls.NEWS_API_KEY:
            return "News service not configured. Headlines unavailable."
        
        try:
            async with aiohttp.ClientSession() as session:
                url = f"{cls.BASE_URL}/top-headlines"
                params = {"category": category, "country": country, "apiKey": cls.NEWS_API_KEY, "pageSize": 5}
                async with session.get(url, params=params) as response:
                    if response.status == 200:
                        data = await response.json()
                        articles = data.get("articles", [])
                        
                        if not articles:
                            return "No news articles available at the moment."
                        
                        headlines = []
                        for article in articles[:5]:
                            title = article.get("title", "").split(" - ")[0]  # Remove source suffix
                            if title:
                                headlines.append(title)
                        
                        return f"Top {category} headlines: " + ". ".join(headlines[:3])
                    return "Unable to retrieve news headlines."
        except Exception as e:
            return f"News service error: {str(e)}"
    
    @classmethod
    async def search_news(cls, query: str) -> str:
        """Search for news on a topic."""
        if not cls.NEWS_API_KEY:
            return f"News search unavailable. Try searching online for '{query}'."
        
        try:
            async with aiohttp.ClientSession() as session:
                url = f"{cls.BASE_URL}/everything"
                params = {"q": query, "apiKey": cls.NEWS_API_KEY, "pageSize": 5, "sortBy": "relevancy"}
                async with session.get(url, params=params) as response:
                    if response.status == 200:
                        data = await response.json()
                        articles = data.get("articles", [])
                        
                        if not articles:
                            return f"No news found for '{query}'."
                        
                        summaries = []
                        for article in articles[:3]:
                            title = article.get("title", "")
                            source = article.get("source", {}).get("name", "")
                            if title:
                                summaries.append(f"{title} ({source})")
                        
                        return f"News about {query}: " + ". ".join(summaries)
                    return f"Unable to search for '{query}'."
        except Exception as e:
            return f"News search error: {str(e)}"


# =============================================================================
# CALENDAR/SCHEDULING SKILL
# =============================================================================

@dataclass
class Reminder:
    """A scheduled reminder."""
    id: str
    message: str
    time: datetime
    recurring: bool = False
    interval_minutes: int = 0


@dataclass
class Task:
    """A task in the task list."""
    id: str
    title: str
    description: str = ""
    priority: str = "medium"  # low, medium, high, critical
    status: str = "pending"  # pending, in_progress, completed, cancelled
    due_date: Optional[datetime] = None
    created_at: datetime = field(default_factory=datetime.now)
    subtasks: List[str] = field(default_factory=list)


class SchedulingSkill:
    """Calendar, reminders, and task management."""
    
    _reminders: Dict[str, Reminder] = {}
    _tasks: Dict[str, Task] = {}
    _counter: int = 0
    
    @classmethod
    def add_reminder(cls, message: str, minutes_from_now: int = 30) -> str:
        """Add a new reminder."""
        cls._counter += 1
        reminder_id = f"reminder-{cls._counter}"
        
        remind_time = datetime.now() + timedelta(minutes=minutes_from_now)
        
        cls._reminders[reminder_id] = Reminder(
            id=reminder_id,
            message=message,
            time=remind_time
        )
        
        time_str = remind_time.strftime("%I:%M %p")
        return f"Reminder set for {time_str}: {message}"
    
    @classmethod
    def list_reminders(cls) -> str:
        """List all active reminders."""
        now = datetime.now()
        active = [r for r in cls._reminders.values() if r.time > now]
        
        if not active:
            return "You have no active reminders."
        
        active.sort(key=lambda r: r.time)
        items = []
        for r in active[:5]:
            time_str = r.time.strftime("%I:%M %p")
            items.append(f"{time_str}: {r.message}")
        
        return f"Your reminders: " + ". ".join(items)
    
    @classmethod
    def check_due_reminders(cls) -> List[str]:
        """Check for reminders that are due."""
        now = datetime.now()
        due = []
        
        for rid, reminder in list(cls._reminders.items()):
            if reminder.time <= now:
                due.append(reminder.message)
                if not reminder.recurring:
                    del cls._reminders[rid]
                else:
                    # Reschedule recurring reminder
                    reminder.time = now + timedelta(minutes=reminder.interval_minutes)
        
        return due
    
    @classmethod
    def add_task(cls, title: str, priority: str = "medium", description: str = "") -> str:
        """Add a new task."""
        cls._counter += 1
        task_id = f"task-{cls._counter}"
        
        cls._tasks[task_id] = Task(
            id=task_id,
            title=title,
            description=description,
            priority=priority
        )
        
        return f"Task added: {title} (Priority: {priority})"
    
    @classmethod
    def list_tasks(cls, status: str = "pending") -> str:
        """List tasks by status."""
        tasks = [t for t in cls._tasks.values() if t.status == status]
        
        if not tasks:
            return f"No {status} tasks."
        
        # Sort by priority
        priority_order = {"critical": 0, "high": 1, "medium": 2, "low": 3}
        tasks.sort(key=lambda t: priority_order.get(t.priority, 2))
        
        items = []
        for t in tasks[:10]:
            items.append(f"[{t.priority.upper()}] {t.title}")
        
        return f"Your {status} tasks: " + ". ".join(items)
    
    @classmethod
    def complete_task(cls, title_keyword: str) -> str:
        """Mark a task as complete."""
        for task in cls._tasks.values():
            if title_keyword.lower() in task.title.lower():
                task.status = "completed"
                return f"Task marked complete: {task.title}"
        return f"No task found matching '{title_keyword}'."


# =============================================================================
# SMART HOME SKILL
# =============================================================================

class SmartHomeSkill:
    """Smart home device control."""
    
    # Simulated devices - in production, integrate with Home Assistant, etc.
    _devices = {
        "living_room_lights": {"type": "light", "state": "off", "brightness": 100},
        "bedroom_lights": {"type": "light", "state": "off", "brightness": 80},
        "thermostat": {"type": "thermostat", "temperature": 22, "mode": "auto"},
        "security_system": {"type": "security", "state": "armed"},
        "front_door": {"type": "lock", "state": "locked"},
    }
    
    @classmethod
    def control_device(cls, device: str, action: str, value: Any = None) -> str:
        """Control a smart home device."""
        device_key = device.lower().replace(" ", "_")
        
        # Find matching device
        matched = None
        for key in cls._devices:
            if device_key in key or key in device_key:
                matched = key
                break
        
        if not matched:
            return f"Device '{device}' not found. Available devices: lights, thermostat, security, front door."
        
        dev = cls._devices[matched]
        
        if dev["type"] == "light":
            if action in ["on", "turn on"]:
                dev["state"] = "on"
                return f"Turning on {matched.replace('_', ' ')}."
            elif action in ["off", "turn off"]:
                dev["state"] = "off"
                return f"Turning off {matched.replace('_', ' ')}."
            elif action == "dim" and value:
                dev["brightness"] = int(value)
                return f"Setting {matched.replace('_', ' ')} brightness to {value}%."
        
        elif dev["type"] == "thermostat":
            if action == "set" and value:
                dev["temperature"] = int(value)
                return f"Setting thermostat to {value}°C."
            elif action == "mode" and value:
                dev["mode"] = value
                return f"Setting thermostat mode to {value}."
        
        elif dev["type"] == "security":
            if action == "arm":
                dev["state"] = "armed"
                return "Security system armed."
            elif action == "disarm":
                dev["state"] = "disarmed"
                return "Security system disarmed."
        
        elif dev["type"] == "lock":
            if action in ["lock", "secure"]:
                dev["state"] = "locked"
                return f"{matched.replace('_', ' ').title()} locked."
            elif action == "unlock":
                dev["state"] = "unlocked"
                return f"{matched.replace('_', ' ').title()} unlocked."
        
        return f"Unknown action '{action}' for {device}."
    
    @classmethod
    def get_status(cls) -> str:
        """Get status of all devices."""
        status_parts = []
        
        lights_on = sum(1 for d in cls._devices.values() if d["type"] == "light" and d["state"] == "on")
        status_parts.append(f"{lights_on} lights on")
        
        thermostat = cls._devices["thermostat"]
        status_parts.append(f"thermostat at {thermostat['temperature']}°C")
        
        security = cls._devices["security_system"]
        status_parts.append(f"security {security['state']}")
        
        door = cls._devices["front_door"]
        status_parts.append(f"front door {door['state']}")
        
        return "Home status: " + ", ".join(status_parts) + "."


# =============================================================================
# CODE ASSISTANT SKILL
# =============================================================================

class CodeAssistantSkill:
    """Code review, generation, and analysis."""
    
    @classmethod
    async def analyze_file(cls, filepath: str) -> str:
        """Analyze a code file."""
        path = Path(filepath)
        
        if not path.exists():
            return f"File not found: {filepath}"
        
        try:
            content = path.read_text(encoding='utf-8')
            lines = content.splitlines()
            
            # Basic analysis
            total_lines = len(lines)
            code_lines = sum(1 for l in lines if l.strip() and not l.strip().startswith('#'))
            blank_lines = sum(1 for l in lines if not l.strip())
            comment_lines = total_lines - code_lines - blank_lines
            
            # Detect language
            ext = path.suffix.lower()
            lang_map = {
                '.py': 'Python', '.js': 'JavaScript', '.ts': 'TypeScript',
                '.go': 'Go', '.java': 'Java', '.cpp': 'C++', '.c': 'C',
                '.rs': 'Rust', '.rb': 'Ruby', '.php': 'PHP'
            }
            language = lang_map.get(ext, 'Unknown')
            
            # Count functions/classes (basic)
            if ext == '.py':
                functions = len(re.findall(r'^def \w+', content, re.MULTILINE))
                classes = len(re.findall(r'^class \w+', content, re.MULTILINE))
            elif ext in ['.js', '.ts']:
                functions = len(re.findall(r'function \w+|const \w+ = (?:async )?\(', content))
                classes = len(re.findall(r'class \w+', content))
            elif ext == '.go':
                functions = len(re.findall(r'^func \w+', content, re.MULTILINE))
                classes = len(re.findall(r'^type \w+ struct', content, re.MULTILINE))
            else:
                functions = 0
                classes = 0
            
            return (
                f"Analysis of {path.name} ({language}):\n"
                f"  Total lines: {total_lines}\n"
                f"  Code lines: {code_lines}\n"
                f"  Comments: {comment_lines}\n"
                f"  Functions: {functions}\n"
                f"  Classes/Types: {classes}"
            )
        except Exception as e:
            return f"Error analyzing file: {str(e)}"
    
    @classmethod
    async def run_linter(cls, filepath: str) -> str:
        """Run appropriate linter on a file."""
        path = Path(filepath)
        
        if not path.exists():
            return f"File not found: {filepath}"
        
        ext = path.suffix.lower()
        
        try:
            if ext == '.py':
                result = subprocess.run(
                    ['python', '-m', 'py_compile', str(path)],
                    capture_output=True, text=True
                )
                if result.returncode == 0:
                    return f"Python syntax check passed for {path.name}."
                return f"Python syntax errors in {path.name}: {result.stderr}"
            
            elif ext == '.go':
                result = subprocess.run(
                    ['go', 'vet', str(path)],
                    capture_output=True, text=True
                )
                if not result.stderr:
                    return f"Go vet passed for {path.name}."
                return f"Go issues in {path.name}: {result.stderr}"
            
            else:
                return f"No linter configured for {ext} files."
        except Exception as e:
            return f"Linter error: {str(e)}"


# =============================================================================
# RESEARCH ASSISTANT SKILL
# =============================================================================

class ResearchSkill:
    """Research and information gathering."""
    
    @classmethod
    async def search_wikipedia(cls, query: str) -> str:
        """Search Wikipedia for information."""
        try:
            import wikipedia
            summary = wikipedia.summary(query, sentences=3)
            return summary
        except Exception as e:
            return f"Wikipedia search error: {str(e)}"
    
    @classmethod
    async def calculate(cls, expression: str) -> str:
        """Evaluate a mathematical expression safely."""
        try:
            # Remove potentially dangerous characters
            safe_expr = re.sub(r'[^0-9+\-*/().%^ ]', '', expression)
            safe_expr = safe_expr.replace('^', '**')
            
            # Evaluate
            result = eval(safe_expr, {"__builtins__": {}}, {"abs": abs, "round": round, "min": min, "max": max})
            return f"The result of {expression} is {result}."
        except Exception as e:
            return f"Unable to calculate: {str(e)}"
    
    @classmethod
    async def unit_convert(cls, value: float, from_unit: str, to_unit: str) -> str:
        """Convert between units."""
        conversions = {
            # Length
            ("km", "miles"): lambda x: x * 0.621371,
            ("miles", "km"): lambda x: x * 1.60934,
            ("m", "ft"): lambda x: x * 3.28084,
            ("ft", "m"): lambda x: x * 0.3048,
            ("cm", "inches"): lambda x: x * 0.393701,
            ("inches", "cm"): lambda x: x * 2.54,
            # Weight
            ("kg", "lbs"): lambda x: x * 2.20462,
            ("lbs", "kg"): lambda x: x * 0.453592,
            # Temperature
            ("c", "f"): lambda x: (x * 9/5) + 32,
            ("f", "c"): lambda x: (x - 32) * 5/9,
            # Volume
            ("l", "gal"): lambda x: x * 0.264172,
            ("gal", "l"): lambda x: x * 3.78541,
        }
        
        key = (from_unit.lower(), to_unit.lower())
        if key in conversions:
            result = conversions[key](value)
            return f"{value} {from_unit} = {result:.2f} {to_unit}"
        
        return f"Cannot convert from {from_unit} to {to_unit}."


# =============================================================================
# SYSTEM ADMINISTRATION SKILL
# =============================================================================

class SysAdminSkill:
    """System administration and monitoring."""
    
    @classmethod
    async def get_system_info(cls) -> str:
        """Get system information."""
        try:
            import psutil
            
            cpu_percent = psutil.cpu_percent(interval=1)
            memory = psutil.virtual_memory()
            disk = psutil.disk_usage('/')
            
            return (
                f"System status: CPU at {cpu_percent}%, "
                f"Memory {memory.percent}% used ({memory.used // (1024**3)}GB of {memory.total // (1024**3)}GB), "
                f"Disk {disk.percent}% used ({disk.used // (1024**3)}GB of {disk.total // (1024**3)}GB)."
            )
        except ImportError:
            return "System monitoring unavailable - psutil not installed."
        except Exception as e:
            return f"System info error: {str(e)}"
    
    @classmethod
    async def get_network_info(cls) -> str:
        """Get network information."""
        try:
            import psutil
            
            net_io = psutil.net_io_counters()
            bytes_sent = net_io.bytes_sent / (1024**2)
            bytes_recv = net_io.bytes_recv / (1024**2)
            
            connections = len(psutil.net_connections())
            
            return (
                f"Network: {bytes_sent:.1f} MB sent, {bytes_recv:.1f} MB received, "
                f"{connections} active connections."
            )
        except Exception as e:
            return f"Network info error: {str(e)}"
    
    @classmethod
    async def kill_process(cls, process_name: str) -> str:
        """Kill a process by name."""
        try:
            import psutil
            
            killed = 0
            for proc in psutil.process_iter(['name', 'pid']):
                if process_name.lower() in proc.info['name'].lower():
                    proc.kill()
                    killed += 1
            
            if killed > 0:
                return f"Terminated {killed} process(es) matching '{process_name}'."
            return f"No processes found matching '{process_name}'."
        except Exception as e:
            return f"Process termination error: {str(e)}"


# =============================================================================
# PLANNING & ORCHESTRATION SKILL
# =============================================================================

@dataclass
class Plan:
    """A multi-step plan."""
    id: str
    name: str
    description: str
    steps: List[Dict[str, Any]] = field(default_factory=list)
    current_step: int = 0
    status: str = "pending"
    created_at: datetime = field(default_factory=datetime.now)


class PlanningSkill:
    """Task planning and orchestration."""
    
    _plans: Dict[str, Plan] = {}
    _counter: int = 0
    
    @classmethod
    def create_plan(cls, name: str, description: str, steps: List[str]) -> str:
        """Create a new multi-step plan."""
        cls._counter += 1
        plan_id = f"plan-{cls._counter}"
        
        step_objs = [
            {"id": i + 1, "action": step, "status": "pending"}
            for i, step in enumerate(steps)
        ]
        
        cls._plans[plan_id] = Plan(
            id=plan_id,
            name=name,
            description=description,
            steps=step_objs
        )
        
        return f"Plan '{name}' created with {len(steps)} steps. Ready to execute."
    
    @classmethod
    def get_plan_status(cls, plan_name: str) -> str:
        """Get status of a plan."""
        for plan in cls._plans.values():
            if plan_name.lower() in plan.name.lower():
                completed = sum(1 for s in plan.steps if s["status"] == "completed")
                total = len(plan.steps)
                current = plan.steps[plan.current_step] if plan.current_step < total else None
                
                status = f"Plan '{plan.name}': {completed}/{total} steps complete."
                if current:
                    status += f" Current step: {current['action']}"
                return status
        
        return f"No plan found matching '{plan_name}'."
    
    @classmethod
    def execute_next_step(cls, plan_name: str) -> str:
        """Execute the next step in a plan."""
        for plan in cls._plans.values():
            if plan_name.lower() in plan.name.lower():
                if plan.current_step >= len(plan.steps):
                    plan.status = "completed"
                    return f"Plan '{plan.name}' is already complete."
                
                step = plan.steps[plan.current_step]
                step["status"] = "completed"
                plan.current_step += 1
                
                if plan.current_step >= len(plan.steps):
                    plan.status = "completed"
                    return f"Step completed: {step['action']}. Plan '{plan.name}' is now complete!"
                
                next_step = plan.steps[plan.current_step]
                return f"Step completed: {step['action']}. Next: {next_step['action']}"
        
        return f"No plan found matching '{plan_name}'."
    
    @classmethod
    def list_plans(cls) -> str:
        """List all plans."""
        if not cls._plans:
            return "No active plans."
        
        items = []
        for plan in cls._plans.values():
            completed = sum(1 for s in plan.steps if s["status"] == "completed")
            total = len(plan.steps)
            items.append(f"{plan.name} ({completed}/{total} complete)")
        
        return "Active plans: " + ". ".join(items)


# =============================================================================
# SKILL REGISTRY
# =============================================================================

SKILLS = {
    "weather": WeatherSkill,
    "news": NewsSkill,
    "scheduling": SchedulingSkill,
    "smart_home": SmartHomeSkill,
    "code": CodeAssistantSkill,
    "research": ResearchSkill,
    "sysadmin": SysAdminSkill,
    "planning": PlanningSkill,
}


async def invoke_skill(skill_name: str, method: str, *args, **kwargs) -> str:
    """Invoke a skill method."""
    skill = SKILLS.get(skill_name)
    if not skill:
        return f"Unknown skill: {skill_name}"
    
    method_func = getattr(skill, method, None)
    if not method_func:
        return f"Unknown method: {method}"
    
    if asyncio.iscoroutinefunction(method_func):
        return await method_func(*args, **kwargs)
    return method_func(*args, **kwargs)
