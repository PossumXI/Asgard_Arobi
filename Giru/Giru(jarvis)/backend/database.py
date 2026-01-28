"""
GIRU Database Layer
====================
SQLite-based persistent storage for conversations, activities, and analytics.
"""

import sqlite3
import json
import os
from datetime import datetime, timedelta
from pathlib import Path
from typing import Optional, Any
from dataclasses import dataclass, asdict
from enum import Enum
from contextlib import contextmanager
import threading


# =============================================================================
# CONFIGURATION
# =============================================================================

DB_PATH = Path(os.getenv(
    "GIRU_DB_PATH",
    Path(__file__).parent.parent / "data" / "giru.db"
))


# =============================================================================
# ENUMS
# =============================================================================

class ActivityType(Enum):
    """Types of activities Giru can perform."""
    CONVERSATION = "conversation"
    COMMAND = "command"
    EMAIL = "email"
    FILE_OPERATION = "file_operation"
    TERMINAL = "terminal"
    CODE = "code"
    ASGARD_QUERY = "asgard_query"
    WEB_SEARCH = "web_search"
    SYSTEM = "system"


class ActivityStatus(Enum):
    """Status of an activity."""
    PENDING = "pending"
    IN_PROGRESS = "in_progress"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


# =============================================================================
# DATA CLASSES
# =============================================================================

@dataclass
class Conversation:
    """A conversation session."""
    id: Optional[int] = None
    session_id: str = ""
    started_at: str = ""
    ended_at: Optional[str] = None
    message_count: int = 0
    model_used: Optional[str] = None
    total_tokens: int = 0
    summary: Optional[str] = None


@dataclass
class Message:
    """A single message in a conversation."""
    id: Optional[int] = None
    conversation_id: int = 0
    role: str = ""  # user, assistant, system
    content: str = ""
    timestamp: str = ""
    model_used: Optional[str] = None
    tokens: int = 0
    latency_ms: int = 0


@dataclass
class Activity:
    """An activity performed by Giru."""
    id: Optional[int] = None
    type: str = ""
    status: str = ""
    description: str = ""
    details: Optional[str] = None  # JSON string
    started_at: str = ""
    ended_at: Optional[str] = None
    duration_ms: int = 0
    model_used: Optional[str] = None
    error: Optional[str] = None


@dataclass
class UserPreference:
    """User preferences and settings."""
    key: str = ""
    value: str = ""
    updated_at: str = ""


@dataclass
class ModelUsage:
    """Model usage statistics."""
    model_key: str = ""
    date: str = ""
    request_count: int = 0
    total_input_tokens: int = 0
    total_output_tokens: int = 0
    total_latency_ms: int = 0
    error_count: int = 0


# =============================================================================
# DATABASE MANAGER
# =============================================================================

class GiruDatabase:
    """
    Thread-safe SQLite database manager for Giru.
    """
    
    def __init__(self, db_path: Path = DB_PATH):
        self.db_path = db_path
        self._local = threading.local()
        self._ensure_db_directory()
        self._init_schema()
    
    def _ensure_db_directory(self):
        """Ensure the database directory exists."""
        self.db_path.parent.mkdir(parents=True, exist_ok=True)
    
    @property
    def _conn(self) -> sqlite3.Connection:
        """Get thread-local database connection."""
        if not hasattr(self._local, 'conn') or self._local.conn is None:
            self._local.conn = sqlite3.connect(
                str(self.db_path),
                check_same_thread=False
            )
            self._local.conn.row_factory = sqlite3.Row
        return self._local.conn
    
    @contextmanager
    def _cursor(self):
        """Get a cursor with automatic commit/rollback."""
        cursor = self._conn.cursor()
        try:
            yield cursor
            self._conn.commit()
        except Exception:
            self._conn.rollback()
            raise
        finally:
            cursor.close()
    
    def _init_schema(self):
        """Initialize database schema."""
        with self._cursor() as cursor:
            # Conversations table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS conversations (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    session_id TEXT NOT NULL,
                    started_at TEXT NOT NULL,
                    ended_at TEXT,
                    message_count INTEGER DEFAULT 0,
                    model_used TEXT,
                    total_tokens INTEGER DEFAULT 0,
                    summary TEXT
                )
            """)
            
            # Messages table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS messages (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    conversation_id INTEGER NOT NULL,
                    role TEXT NOT NULL,
                    content TEXT NOT NULL,
                    timestamp TEXT NOT NULL,
                    model_used TEXT,
                    tokens INTEGER DEFAULT 0,
                    latency_ms INTEGER DEFAULT 0,
                    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
                )
            """)
            
            # Activities table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS activities (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    type TEXT NOT NULL,
                    status TEXT NOT NULL,
                    description TEXT NOT NULL,
                    details TEXT,
                    started_at TEXT NOT NULL,
                    ended_at TEXT,
                    duration_ms INTEGER DEFAULT 0,
                    model_used TEXT,
                    error TEXT
                )
            """)
            
            # User preferences table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS user_preferences (
                    key TEXT PRIMARY KEY,
                    value TEXT NOT NULL,
                    updated_at TEXT NOT NULL
                )
            """)
            
            # Model usage statistics table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS model_usage (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    model_key TEXT NOT NULL,
                    date TEXT NOT NULL,
                    request_count INTEGER DEFAULT 0,
                    total_input_tokens INTEGER DEFAULT 0,
                    total_output_tokens INTEGER DEFAULT 0,
                    total_latency_ms INTEGER DEFAULT 0,
                    error_count INTEGER DEFAULT 0,
                    UNIQUE(model_key, date)
                )
            """)
            
            # Email history table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS email_history (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    recipient TEXT NOT NULL,
                    subject TEXT NOT NULL,
                    body TEXT NOT NULL,
                    sent_at TEXT NOT NULL,
                    status TEXT NOT NULL,
                    error TEXT
                )
            """)
            
            # Command history table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS command_history (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    command TEXT NOT NULL,
                    working_dir TEXT,
                    output TEXT,
                    exit_code INTEGER,
                    executed_at TEXT NOT NULL,
                    duration_ms INTEGER DEFAULT 0
                )
            """)
            
            # Create indexes
            cursor.execute("CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id)")
            cursor.execute("CREATE INDEX IF NOT EXISTS idx_activities_type ON activities(type)")
            cursor.execute("CREATE INDEX IF NOT EXISTS idx_activities_status ON activities(status)")
            cursor.execute("CREATE INDEX IF NOT EXISTS idx_activities_started ON activities(started_at)")
            cursor.execute("CREATE INDEX IF NOT EXISTS idx_model_usage_date ON model_usage(date)")
    
    # =========================================================================
    # CONVERSATION METHODS
    # =========================================================================
    
    def create_conversation(self, session_id: str) -> int:
        """Create a new conversation session."""
        with self._cursor() as cursor:
            cursor.execute("""
                INSERT INTO conversations (session_id, started_at)
                VALUES (?, ?)
            """, (session_id, datetime.now().isoformat()))
            return cursor.lastrowid
    
    def end_conversation(self, conversation_id: int, summary: Optional[str] = None):
        """End a conversation session."""
        with self._cursor() as cursor:
            cursor.execute("""
                UPDATE conversations
                SET ended_at = ?, summary = ?
                WHERE id = ?
            """, (datetime.now().isoformat(), summary, conversation_id))
    
    def add_message(
        self,
        conversation_id: int,
        role: str,
        content: str,
        model_used: Optional[str] = None,
        tokens: int = 0,
        latency_ms: int = 0
    ) -> int:
        """Add a message to a conversation."""
        with self._cursor() as cursor:
            cursor.execute("""
                INSERT INTO messages (conversation_id, role, content, timestamp, model_used, tokens, latency_ms)
                VALUES (?, ?, ?, ?, ?, ?, ?)
            """, (conversation_id, role, content, datetime.now().isoformat(), model_used, tokens, latency_ms))
            
            # Update conversation stats
            cursor.execute("""
                UPDATE conversations
                SET message_count = message_count + 1,
                    total_tokens = total_tokens + ?,
                    model_used = COALESCE(?, model_used)
                WHERE id = ?
            """, (tokens, model_used, conversation_id))
            
            return cursor.lastrowid
    
    def get_conversation_messages(self, conversation_id: int, limit: int = 50) -> list[dict]:
        """Get messages from a conversation."""
        with self._cursor() as cursor:
            cursor.execute("""
                SELECT * FROM messages
                WHERE conversation_id = ?
                ORDER BY timestamp DESC
                LIMIT ?
            """, (conversation_id, limit))
            return [dict(row) for row in cursor.fetchall()]
    
    def get_recent_conversations(self, limit: int = 10) -> list[dict]:
        """Get recent conversations."""
        with self._cursor() as cursor:
            cursor.execute("""
                SELECT * FROM conversations
                ORDER BY started_at DESC
                LIMIT ?
            """, (limit,))
            return [dict(row) for row in cursor.fetchall()]
    
    # =========================================================================
    # ACTIVITY METHODS
    # =========================================================================
    
    def start_activity(
        self,
        activity_type: ActivityType,
        description: str,
        details: Optional[dict] = None,
        model_used: Optional[str] = None
    ) -> int:
        """Start tracking a new activity."""
        with self._cursor() as cursor:
            cursor.execute("""
                INSERT INTO activities (type, status, description, details, started_at, model_used)
                VALUES (?, ?, ?, ?, ?, ?)
            """, (
                activity_type.value,
                ActivityStatus.IN_PROGRESS.value,
                description,
                json.dumps(details) if details else None,
                datetime.now().isoformat(),
                model_used
            ))
            return cursor.lastrowid
    
    def complete_activity(
        self,
        activity_id: int,
        status: ActivityStatus = ActivityStatus.COMPLETED,
        error: Optional[str] = None
    ):
        """Mark an activity as completed."""
        with self._cursor() as cursor:
            cursor.execute("SELECT started_at FROM activities WHERE id = ?", (activity_id,))
            row = cursor.fetchone()
            if row:
                started = datetime.fromisoformat(row['started_at'])
                duration_ms = int((datetime.now() - started).total_seconds() * 1000)
            else:
                duration_ms = 0
            
            cursor.execute("""
                UPDATE activities
                SET status = ?, ended_at = ?, duration_ms = ?, error = ?
                WHERE id = ?
            """, (status.value, datetime.now().isoformat(), duration_ms, error, activity_id))
    
    def get_recent_activities(self, limit: int = 50, activity_type: Optional[str] = None) -> list[dict]:
        """Get recent activities."""
        with self._cursor() as cursor:
            if activity_type:
                cursor.execute("""
                    SELECT * FROM activities
                    WHERE type = ?
                    ORDER BY started_at DESC
                    LIMIT ?
                """, (activity_type, limit))
            else:
                cursor.execute("""
                    SELECT * FROM activities
                    ORDER BY started_at DESC
                    LIMIT ?
                """, (limit,))
            return [dict(row) for row in cursor.fetchall()]
    
    def get_active_activities(self) -> list[dict]:
        """Get currently active (in-progress) activities."""
        with self._cursor() as cursor:
            cursor.execute("""
                SELECT * FROM activities
                WHERE status = ?
                ORDER BY started_at DESC
            """, (ActivityStatus.IN_PROGRESS.value,))
            return [dict(row) for row in cursor.fetchall()]
    
    def get_activity_stats(self, days: int = 7) -> dict:
        """Get activity statistics for the past N days."""
        since = (datetime.now() - timedelta(days=days)).isoformat()
        
        with self._cursor() as cursor:
            # Total activities by type
            cursor.execute("""
                SELECT type, COUNT(*) as count, AVG(duration_ms) as avg_duration
                FROM activities
                WHERE started_at >= ?
                GROUP BY type
            """, (since,))
            by_type = {row['type']: {'count': row['count'], 'avg_duration': row['avg_duration']} 
                      for row in cursor.fetchall()}
            
            # Success/failure rates
            cursor.execute("""
                SELECT status, COUNT(*) as count
                FROM activities
                WHERE started_at >= ?
                GROUP BY status
            """, (since,))
            by_status = {row['status']: row['count'] for row in cursor.fetchall()}
            
            # Daily breakdown
            cursor.execute("""
                SELECT DATE(started_at) as date, COUNT(*) as count
                FROM activities
                WHERE started_at >= ?
                GROUP BY DATE(started_at)
                ORDER BY date
            """, (since,))
            daily = [{'date': row['date'], 'count': row['count']} for row in cursor.fetchall()]
            
            return {
                'by_type': by_type,
                'by_status': by_status,
                'daily': daily,
                'total': sum(by_status.values()),
            }
    
    # =========================================================================
    # MODEL USAGE METHODS
    # =========================================================================
    
    def record_model_usage(
        self,
        model_key: str,
        input_tokens: int = 0,
        output_tokens: int = 0,
        latency_ms: int = 0,
        is_error: bool = False
    ):
        """Record model usage statistics."""
        today = datetime.now().strftime("%Y-%m-%d")
        
        with self._cursor() as cursor:
            cursor.execute("""
                INSERT INTO model_usage (model_key, date, request_count, total_input_tokens, total_output_tokens, total_latency_ms, error_count)
                VALUES (?, ?, 1, ?, ?, ?, ?)
                ON CONFLICT(model_key, date) DO UPDATE SET
                    request_count = request_count + 1,
                    total_input_tokens = total_input_tokens + ?,
                    total_output_tokens = total_output_tokens + ?,
                    total_latency_ms = total_latency_ms + ?,
                    error_count = error_count + ?
            """, (
                model_key, today, input_tokens, output_tokens, latency_ms, 1 if is_error else 0,
                input_tokens, output_tokens, latency_ms, 1 if is_error else 0
            ))
    
    def get_model_usage_stats(self, days: int = 30) -> list[dict]:
        """Get model usage statistics for the past N days."""
        since = (datetime.now() - timedelta(days=days)).strftime("%Y-%m-%d")
        
        with self._cursor() as cursor:
            cursor.execute("""
                SELECT model_key,
                       SUM(request_count) as total_requests,
                       SUM(total_input_tokens) as total_input,
                       SUM(total_output_tokens) as total_output,
                       AVG(total_latency_ms / request_count) as avg_latency,
                       SUM(error_count) as errors
                FROM model_usage
                WHERE date >= ?
                GROUP BY model_key
                ORDER BY total_requests DESC
            """, (since,))
            return [dict(row) for row in cursor.fetchall()]
    
    # =========================================================================
    # USER PREFERENCES
    # =========================================================================
    
    def set_preference(self, key: str, value: Any):
        """Set a user preference."""
        with self._cursor() as cursor:
            cursor.execute("""
                INSERT OR REPLACE INTO user_preferences (key, value, updated_at)
                VALUES (?, ?, ?)
            """, (key, json.dumps(value), datetime.now().isoformat()))
    
    def get_preference(self, key: str, default: Any = None) -> Any:
        """Get a user preference."""
        with self._cursor() as cursor:
            cursor.execute("SELECT value FROM user_preferences WHERE key = ?", (key,))
            row = cursor.fetchone()
            if row:
                return json.loads(row['value'])
            return default
    
    def get_all_preferences(self) -> dict:
        """Get all user preferences."""
        with self._cursor() as cursor:
            cursor.execute("SELECT key, value FROM user_preferences")
            return {row['key']: json.loads(row['value']) for row in cursor.fetchall()}
    
    # =========================================================================
    # EMAIL HISTORY
    # =========================================================================
    
    def log_email(
        self,
        recipient: str,
        subject: str,
        body: str,
        status: str = "sent",
        error: Optional[str] = None
    ) -> int:
        """Log an email sent by Giru."""
        with self._cursor() as cursor:
            cursor.execute("""
                INSERT INTO email_history (recipient, subject, body, sent_at, status, error)
                VALUES (?, ?, ?, ?, ?, ?)
            """, (recipient, subject, body, datetime.now().isoformat(), status, error))
            return cursor.lastrowid
    
    def get_email_history(self, limit: int = 50) -> list[dict]:
        """Get email history."""
        with self._cursor() as cursor:
            cursor.execute("""
                SELECT * FROM email_history
                ORDER BY sent_at DESC
                LIMIT ?
            """, (limit,))
            return [dict(row) for row in cursor.fetchall()]
    
    # =========================================================================
    # COMMAND HISTORY
    # =========================================================================
    
    def log_command(
        self,
        command: str,
        working_dir: Optional[str],
        output: Optional[str],
        exit_code: int,
        duration_ms: int
    ) -> int:
        """Log a command executed by Giru."""
        with self._cursor() as cursor:
            cursor.execute("""
                INSERT INTO command_history (command, working_dir, output, exit_code, executed_at, duration_ms)
                VALUES (?, ?, ?, ?, ?, ?)
            """, (command, working_dir, output, exit_code, datetime.now().isoformat(), duration_ms))
            return cursor.lastrowid
    
    def get_command_history(self, limit: int = 50) -> list[dict]:
        """Get command history."""
        with self._cursor() as cursor:
            cursor.execute("""
                SELECT * FROM command_history
                ORDER BY executed_at DESC
                LIMIT ?
            """, (limit,))
            return [dict(row) for row in cursor.fetchall()]
    
    # =========================================================================
    # ANALYTICS QUERIES
    # =========================================================================
    
    def get_dashboard_stats(self) -> dict:
        """Get comprehensive dashboard statistics."""
        today = datetime.now().strftime("%Y-%m-%d")
        week_ago = (datetime.now() - timedelta(days=7)).isoformat()
        
        with self._cursor() as cursor:
            # Today's stats
            cursor.execute("""
                SELECT COUNT(*) as count FROM activities
                WHERE DATE(started_at) = ?
            """, (today,))
            activities_today = cursor.fetchone()['count']
            
            cursor.execute("""
                SELECT COUNT(*) as count FROM messages
                WHERE DATE(timestamp) = ?
            """, (today,))
            messages_today = cursor.fetchone()['count']
            
            cursor.execute("""
                SELECT COUNT(*) as count FROM command_history
                WHERE DATE(executed_at) = ?
            """, (today,))
            commands_today = cursor.fetchone()['count']
            
            cursor.execute("""
                SELECT COUNT(*) as count FROM email_history
                WHERE DATE(sent_at) = ?
            """, (today,))
            emails_today = cursor.fetchone()['count']
            
            # Active activities
            cursor.execute("""
                SELECT COUNT(*) as count FROM activities
                WHERE status = ?
            """, (ActivityStatus.IN_PROGRESS.value,))
            active_activities = cursor.fetchone()['count']
            
            # Most used model today
            cursor.execute("""
                SELECT model_key, request_count
                FROM model_usage
                WHERE date = ?
                ORDER BY request_count DESC
                LIMIT 1
            """, (today,))
            row = cursor.fetchone()
            top_model = {'model': row['model_key'], 'requests': row['request_count']} if row else None
            
            return {
                'today': {
                    'activities': activities_today,
                    'messages': messages_today,
                    'commands': commands_today,
                    'emails': emails_today,
                },
                'active_activities': active_activities,
                'top_model': top_model,
                'timestamp': datetime.now().isoformat(),
            }
    
    def close(self):
        """Close database connection."""
        if hasattr(self._local, 'conn') and self._local.conn:
            self._local.conn.close()
            self._local.conn = None


# =============================================================================
# SINGLETON INSTANCE
# =============================================================================

_db_instance: Optional[GiruDatabase] = None


def get_database() -> GiruDatabase:
    """Get the singleton database instance."""
    global _db_instance
    if _db_instance is None:
        _db_instance = GiruDatabase()
    return _db_instance
