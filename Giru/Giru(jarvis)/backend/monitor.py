"""
GIRU Activity Monitor
======================
Real-time activity tracking and monitoring with WebSocket broadcasts.
"""

import asyncio
import json
import logging
import time
from datetime import datetime
from dataclasses import dataclass, field, asdict
from enum import Enum
from typing import Optional, Callable, Any
from contextlib import asynccontextmanager
import websockets
import websockets.exceptions

from database import get_database, ActivityType, ActivityStatus

# Suppress expected WebSocket handshake errors from health checks/probes
# These occur when clients connect without completing the WebSocket handshake
logging.getLogger("websockets").setLevel(logging.CRITICAL)
logging.getLogger("websockets.server").setLevel(logging.CRITICAL)
logging.getLogger("websockets.protocol").setLevel(logging.CRITICAL)


# =============================================================================
# ACTIVITY TRACKING
# =============================================================================

@dataclass
class ActivityEvent:
    """An activity event for real-time monitoring."""
    id: int
    type: str
    status: str
    description: str
    details: Optional[dict] = None
    started_at: str = ""
    ended_at: Optional[str] = None
    duration_ms: int = 0
    model_used: Optional[str] = None
    progress: float = 0.0  # 0.0 to 1.0
    error: Optional[str] = None
    
    def to_dict(self) -> dict:
        return {
            'id': self.id,
            'type': self.type,
            'status': self.status,
            'description': self.description,
            'details': self.details,
            'started_at': self.started_at,
            'ended_at': self.ended_at,
            'duration_ms': self.duration_ms,
            'model_used': self.model_used,
            'progress': self.progress,
            'error': self.error,
        }


class ActivityMonitor:
    """
    Real-time activity monitor that tracks all Giru operations
    and broadcasts updates via WebSocket.
    """
    
    def __init__(self):
        self.db = get_database()
        self.active_activities: dict[int, ActivityEvent] = {}
        self.subscribers: set = set()
        self._lock = asyncio.Lock()
    
    async def subscribe(self, callback: Callable):
        """Subscribe to activity updates."""
        async with self._lock:
            self.subscribers.add(callback)
    
    async def unsubscribe(self, callback: Callable):
        """Unsubscribe from activity updates."""
        async with self._lock:
            self.subscribers.discard(callback)
    
    async def _broadcast(self, event_type: str, data: dict):
        """Broadcast an event to all subscribers."""
        message = {
            'event': event_type,
            'data': data,
            'timestamp': datetime.now().isoformat(),
        }
        
        async with self._lock:
            for callback in list(self.subscribers):
                try:
                    await callback(message)
                except Exception:
                    pass
    
    @asynccontextmanager
    async def track_activity(
        self,
        activity_type: ActivityType,
        description: str,
        details: Optional[dict] = None,
        model_used: Optional[str] = None
    ):
        """
        Context manager for tracking an activity with automatic status updates.
        
        Usage:
            async with monitor.track_activity(ActivityType.COMMAND, "Running git status") as activity:
                # Do work
                activity.update_progress(0.5)
        """
        # Start the activity
        activity_id = self.db.start_activity(
            activity_type=activity_type,
            description=description,
            details=details,
            model_used=model_used
        )
        
        event = ActivityEvent(
            id=activity_id,
            type=activity_type.value,
            status=ActivityStatus.IN_PROGRESS.value,
            description=description,
            details=details,
            started_at=datetime.now().isoformat(),
            model_used=model_used,
            progress=0.0,
        )
        
        self.active_activities[activity_id] = event
        
        # Broadcast start
        await self._broadcast('activity_started', event.to_dict())
        
        tracker = ActivityTracker(activity_id, event, self)
        
        try:
            yield tracker
            
            # Mark as completed
            event.status = ActivityStatus.COMPLETED.value
            event.ended_at = datetime.now().isoformat()
            event.progress = 1.0
            started = datetime.fromisoformat(event.started_at)
            event.duration_ms = int((datetime.now() - started).total_seconds() * 1000)
            
            self.db.complete_activity(activity_id, ActivityStatus.COMPLETED)
            await self._broadcast('activity_completed', event.to_dict())
            
        except Exception as e:
            # Mark as failed
            event.status = ActivityStatus.FAILED.value
            event.ended_at = datetime.now().isoformat()
            event.error = str(e)
            started = datetime.fromisoformat(event.started_at)
            event.duration_ms = int((datetime.now() - started).total_seconds() * 1000)
            
            self.db.complete_activity(activity_id, ActivityStatus.FAILED, error=str(e))
            await self._broadcast('activity_failed', event.to_dict())
            raise
        
        finally:
            # Remove from active
            self.active_activities.pop(activity_id, None)
    
    async def start_activity(
        self,
        activity_type: ActivityType,
        description: str,
        details: Optional[dict] = None,
        model_used: Optional[str] = None
    ) -> int:
        """Start an activity without using context manager."""
        activity_id = self.db.start_activity(
            activity_type=activity_type,
            description=description,
            details=details,
            model_used=model_used
        )
        
        event = ActivityEvent(
            id=activity_id,
            type=activity_type.value,
            status=ActivityStatus.IN_PROGRESS.value,
            description=description,
            details=details,
            started_at=datetime.now().isoformat(),
            model_used=model_used,
            progress=0.0,
        )
        
        self.active_activities[activity_id] = event
        await self._broadcast('activity_started', event.to_dict())
        
        return activity_id
    
    async def update_activity(
        self,
        activity_id: int,
        progress: Optional[float] = None,
        details: Optional[dict] = None,
        description: Optional[str] = None
    ):
        """Update an in-progress activity."""
        if activity_id not in self.active_activities:
            return
        
        event = self.active_activities[activity_id]
        
        if progress is not None:
            event.progress = max(0.0, min(1.0, progress))
        if details is not None:
            event.details = details
        if description is not None:
            event.description = description
        
        await self._broadcast('activity_updated', event.to_dict())
    
    async def complete_activity(
        self,
        activity_id: int,
        status: ActivityStatus = ActivityStatus.COMPLETED,
        error: Optional[str] = None
    ):
        """Complete an activity."""
        if activity_id not in self.active_activities:
            return
        
        event = self.active_activities[activity_id]
        event.status = status.value
        event.ended_at = datetime.now().isoformat()
        event.error = error
        event.progress = 1.0 if status == ActivityStatus.COMPLETED else event.progress
        
        started = datetime.fromisoformat(event.started_at)
        event.duration_ms = int((datetime.now() - started).total_seconds() * 1000)
        
        self.db.complete_activity(activity_id, status, error)
        
        event_type = 'activity_completed' if status == ActivityStatus.COMPLETED else 'activity_failed'
        await self._broadcast(event_type, event.to_dict())
        
        self.active_activities.pop(activity_id, None)
    
    def get_active_activities(self) -> list[dict]:
        """Get all currently active activities."""
        return [event.to_dict() for event in self.active_activities.values()]
    
    def get_recent_activities(self, limit: int = 50) -> list[dict]:
        """Get recent activities from database."""
        return self.db.get_recent_activities(limit)
    
    def get_activity_stats(self, days: int = 7) -> dict:
        """Get activity statistics."""
        return self.db.get_activity_stats(days)
    
    async def get_dashboard_data(self) -> dict:
        """Get comprehensive dashboard data."""
        db_stats = self.db.get_dashboard_stats()
        model_stats = self.db.get_model_usage_stats(7)
        activity_stats = self.db.get_activity_stats(7)
        
        return {
            'summary': db_stats,
            'active_activities': self.get_active_activities(),
            'recent_activities': self.get_recent_activities(20),
            'model_usage': model_stats,
            'activity_stats': activity_stats,
            'timestamp': datetime.now().isoformat(),
        }


class ActivityTracker:
    """Helper class for tracking activity progress."""
    
    def __init__(self, activity_id: int, event: ActivityEvent, monitor: ActivityMonitor):
        self.activity_id = activity_id
        self.event = event
        self.monitor = monitor
    
    async def update_progress(self, progress: float, description: Optional[str] = None):
        """Update activity progress (0.0 to 1.0)."""
        await self.monitor.update_activity(
            self.activity_id,
            progress=progress,
            description=description
        )
    
    async def update_details(self, details: dict):
        """Update activity details."""
        await self.monitor.update_activity(
            self.activity_id,
            details=details
        )
    
    async def log(self, message: str):
        """Log a message for this activity."""
        current_details = self.event.details or {}
        logs = current_details.get('logs', [])
        logs.append({
            'timestamp': datetime.now().isoformat(),
            'message': message
        })
        current_details['logs'] = logs[-20:]  # Keep last 20 logs
        await self.monitor.update_activity(
            self.activity_id,
            details=current_details
        )


# =============================================================================
# MONITORING WEBSOCKET SERVER
# =============================================================================

class MonitoringServer:
    """
    WebSocket server for real-time monitoring dashboard.
    """
    
    def __init__(self, monitor: ActivityMonitor, port: int = 7778):
        self.monitor = monitor
        self.port = port
        self.clients: set = set()
        self._running = False
    
    async def _broadcast_to_clients(self, message: dict):
        """Broadcast message to all connected monitoring clients."""
        if not self.clients:
            return
        
        data = json.dumps(message)
        await asyncio.gather(
            *[client.send(data) for client in list(self.clients)],
            return_exceptions=True
        )
    
    async def _handle_client(self, websocket):
        """Handle a monitoring client connection with proper error handling."""
        self.clients.add(websocket)

        try:
            # Send initial dashboard data
            dashboard = await self.monitor.get_dashboard_data()
            await websocket.send(json.dumps({
                'event': 'dashboard_init',
                'data': dashboard,
                'timestamp': datetime.now().isoformat(),
            }))

            # Subscribe to activity updates
            await self.monitor.subscribe(self._broadcast_to_clients)

            # Handle incoming messages
            async for message in websocket:
                try:
                    data = json.loads(message)
                    await self._handle_message(websocket, data)
                except json.JSONDecodeError:
                    pass

        except websockets.exceptions.ConnectionClosedError:
            # Client disconnected - expected behavior
            pass
        except websockets.exceptions.ConnectionClosedOK:
            # Client closed connection gracefully
            pass
        except Exception as e:
            # Log unexpected errors but don't crash
            print(f"Monitor WebSocket error: {type(e).__name__}: {e}")

        finally:
            self.clients.discard(websocket)
    
    async def _handle_message(self, websocket, data: dict):
        """Handle incoming message from monitoring client."""
        msg_type = data.get('type')
        
        if msg_type == 'get_dashboard':
            dashboard = await self.monitor.get_dashboard_data()
            await websocket.send(json.dumps({
                'event': 'dashboard_update',
                'data': dashboard,
                'timestamp': datetime.now().isoformat(),
            }))
        
        elif msg_type == 'get_activities':
            limit = data.get('limit', 50)
            activities = self.monitor.get_recent_activities(limit)
            await websocket.send(json.dumps({
                'event': 'activities_list',
                'data': activities,
                'timestamp': datetime.now().isoformat(),
            }))
        
        elif msg_type == 'get_stats':
            days = data.get('days', 7)
            stats = self.monitor.get_activity_stats(days)
            await websocket.send(json.dumps({
                'event': 'stats_update',
                'data': stats,
                'timestamp': datetime.now().isoformat(),
            }))
    
    async def start(self):
        """Start the monitoring server."""
        import os
        self._running = True

        # Bind to 0.0.0.0 to accept connections from outside Docker container
        bind_host = "0.0.0.0" if os.getenv("GIRU_DOCKER") else "127.0.0.1"
        async with websockets.serve(self._handle_client, bind_host, self.port):
            print(f"Monitoring server started on ws://{bind_host}:{self.port}")
            
            # Periodically send dashboard updates
            while self._running:
                await asyncio.sleep(5)  # Update every 5 seconds
                if self.clients:
                    dashboard = await self.monitor.get_dashboard_data()
                    await self._broadcast_to_clients({
                        'event': 'dashboard_update',
                        'data': dashboard,
                        'timestamp': datetime.now().isoformat(),
                    })
    
    def stop(self):
        """Stop the monitoring server."""
        self._running = False


# =============================================================================
# SINGLETON INSTANCES
# =============================================================================

_monitor_instance: Optional[ActivityMonitor] = None
_server_instance: Optional[MonitoringServer] = None


def get_monitor() -> ActivityMonitor:
    """Get the singleton activity monitor instance."""
    global _monitor_instance
    if _monitor_instance is None:
        _monitor_instance = ActivityMonitor()
    return _monitor_instance


def get_monitoring_server(port: int = 7778) -> MonitoringServer:
    """Get the singleton monitoring server instance."""
    global _server_instance
    if _server_instance is None:
        _server_instance = MonitoringServer(get_monitor(), port)
    return _server_instance
