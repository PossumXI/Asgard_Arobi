"""
GIRU Configuration Manager
===========================
Handles user settings, API keys, and preferences with persistence.
"""

import os
import json
from pathlib import Path
from typing import Optional, Any
from dataclasses import dataclass, asdict
from datetime import datetime


# =============================================================================
# PATHS
# =============================================================================

CONFIG_DIR = Path(os.getenv(
    "GIRU_CONFIG_DIR",
    Path(__file__).parent.parent / "data"
))

CONFIG_FILE = CONFIG_DIR / "config.json"
CREDENTIALS_FILE = CONFIG_DIR / "credentials.json"


# =============================================================================
# DEFAULT CONFIGURATION
# =============================================================================

DEFAULT_CONFIG = {
    # AI Settings
    "default_model": "groq-llama-3.3-70b",  # Free by default
    "fallback_to_free": True,
    "max_tokens": 4096,
    "temperature": 0.7,
    
    # Voice Settings
    "wake_words": ["giru", "hey giru", "hello giru", "hi giru"],
    "use_picovoice": False,  # Offline wake word detection
    "picovoice_sensitivity": 0.5,
    
    # TTS Settings
    "tts_provider": "pyttsx3",  # pyttsx3, elevenlabs
    "tts_rate": 175,
    "elevenlabs_voice_id": "21m00Tcm4TlvDq8ikWAM",  # Rachel
    "elevenlabs_model_id": "eleven_turbo_v2",
    
    # Conversation Settings
    "conversation_timeout": 120,
    "active_window_seconds": 60,
    "auto_enable_mic": False,
    
    # ASGARD Integration
    "pricilla_url": "http://localhost:8092",
    "nysus_url": "http://localhost:8080",
    "silenus_url": "http://localhost:9093",
    "hunoid_url": "http://localhost:8090",
    "giru_security_url": "http://localhost:9090",
    
    # UI Settings
    "theme": "dark",
    "show_model_info": True,
    "compact_mode": False,
}

# API keys that can be configured (stored separately for security)
API_KEYS = [
    "GROQ_API_KEY",
    "TOGETHER_API_KEY",
    "GOOGLE_API_KEY",
    "ANTHROPIC_API_KEY",
    "OPENAI_API_KEY",
    "OPENROUTER_API_KEY",
    "ELEVENLABS_API_KEY",
    "PICOVOICE_ACCESS_KEY",
]


# =============================================================================
# CONFIGURATION MANAGER
# =============================================================================

class ConfigManager:
    """Manages Giru configuration and credentials."""
    
    def __init__(self):
        self._ensure_config_dir()
        self._config = self._load_config()
        self._credentials = self._load_credentials()
    
    def _ensure_config_dir(self):
        """Ensure configuration directory exists."""
        CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    
    def _load_config(self) -> dict:
        """Load configuration from file."""
        if CONFIG_FILE.exists():
            try:
                with open(CONFIG_FILE, 'r') as f:
                    loaded = json.load(f)
                    # Merge with defaults to handle new settings
                    return {**DEFAULT_CONFIG, **loaded}
            except Exception:
                pass
        return DEFAULT_CONFIG.copy()
    
    def _load_credentials(self) -> dict:
        """Load credentials from file."""
        if CREDENTIALS_FILE.exists():
            try:
                with open(CREDENTIALS_FILE, 'r') as f:
                    return json.load(f)
            except Exception:
                pass
        return {}
    
    def _save_config(self):
        """Save configuration to file."""
        with open(CONFIG_FILE, 'w') as f:
            json.dump(self._config, f, indent=2)
    
    def _save_credentials(self):
        """Save credentials to file."""
        with open(CREDENTIALS_FILE, 'w') as f:
            json.dump(self._credentials, f, indent=2)
    
    # =========================================================================
    # CONFIG ACCESS
    # =========================================================================
    
    def get(self, key: str, default: Any = None) -> Any:
        """Get a configuration value."""
        return self._config.get(key, default)
    
    def set(self, key: str, value: Any):
        """Set a configuration value."""
        self._config[key] = value
        self._save_config()
    
    def get_all(self) -> dict:
        """Get all configuration values."""
        return self._config.copy()
    
    def reset(self):
        """Reset configuration to defaults."""
        self._config = DEFAULT_CONFIG.copy()
        self._save_config()
    
    # =========================================================================
    # API KEY ACCESS
    # =========================================================================
    
    def get_api_key(self, key: str) -> Optional[str]:
        """Get an API key (checks environment first, then stored credentials)."""
        # Environment variable takes precedence
        env_value = os.getenv(key)
        if env_value:
            return env_value
        return self._credentials.get(key)
    
    def set_api_key(self, key: str, value: str):
        """Set an API key."""
        if key in API_KEYS:
            self._credentials[key] = value
            self._save_credentials()
            # Also set in environment for immediate use
            os.environ[key] = value
    
    def get_all_api_keys(self) -> dict:
        """Get all API key statuses (masked values)."""
        result = {}
        for key in API_KEYS:
            value = self.get_api_key(key)
            if value:
                # Mask the key for display
                result[key] = f"{value[:4]}...{value[-4:]}" if len(value) > 8 else "****"
            else:
                result[key] = None
        return result
    
    def has_api_key(self, key: str) -> bool:
        """Check if an API key is configured."""
        return bool(self.get_api_key(key))
    
    def clear_api_key(self, key: str):
        """Clear an API key."""
        if key in self._credentials:
            del self._credentials[key]
            self._save_credentials()
    
    # =========================================================================
    # CONVENIENCE METHODS
    # =========================================================================
    
    def get_default_model(self) -> str:
        """Get the default AI model."""
        return self.get("default_model", "groq-llama-3.3-70b")
    
    def set_default_model(self, model: str):
        """Set the default AI model."""
        self.set("default_model", model)
    
    def get_tts_provider(self) -> str:
        """Get the TTS provider."""
        # Check if ElevenLabs is available
        if self.has_api_key("ELEVENLABS_API_KEY"):
            return self.get("tts_provider", "elevenlabs")
        return "pyttsx3"
    
    def is_picovoice_available(self) -> bool:
        """Check if Picovoice is available."""
        return self.has_api_key("PICOVOICE_ACCESS_KEY") and self.get("use_picovoice", False)
    
    def get_available_providers(self) -> list[dict]:
        """Get list of available AI providers based on API keys."""
        providers = [
            {"name": "Groq", "key": "GROQ_API_KEY", "free": True},
            {"name": "Together AI", "key": "TOGETHER_API_KEY", "free": True},
            {"name": "Google Gemini", "key": "GOOGLE_API_KEY", "free": True},
            {"name": "Anthropic Claude", "key": "ANTHROPIC_API_KEY", "free": False},
            {"name": "OpenAI", "key": "OPENAI_API_KEY", "free": False},
            {"name": "OpenRouter", "key": "OPENROUTER_API_KEY", "free": False},
        ]
        
        for p in providers:
            p["available"] = self.has_api_key(p["key"])
        
        return providers
    
    def export_config(self) -> dict:
        """Export configuration (without credentials) for backup."""
        return {
            "config": self._config,
            "exported_at": datetime.now().isoformat(),
            "version": "2.0",
        }
    
    def import_config(self, data: dict):
        """Import configuration from backup."""
        if "config" in data:
            self._config = {**DEFAULT_CONFIG, **data["config"]}
            self._save_config()


# =============================================================================
# SINGLETON INSTANCE
# =============================================================================

_config_manager: Optional[ConfigManager] = None


def get_config() -> ConfigManager:
    """Get the singleton configuration manager."""
    global _config_manager
    if _config_manager is None:
        _config_manager = ConfigManager()
    return _config_manager
