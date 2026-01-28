"""
GIRU Wake Word Engine
=====================
Optional Picovoice Porcupine wake word detection for offline wake word detection.
Requires: pip install pvporcupine and PICOVOICE_ACCESS_KEY environment variable.
"""
from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Callable, Optional, Any

# Try to import pvporcupine, but don't fail if it's not installed
try:
    import pvporcupine
    PORCUPINE_AVAILABLE = True
except ImportError:
    pvporcupine = None
    PORCUPINE_AVAILABLE = False


def _split_csv(value: str) -> list[str]:
    """Split a comma-separated string into a list of stripped values."""
    return [item.strip() for item in value.split(",") if item.strip()]


@dataclass
class WakeWordEngine:
    """Wrapper for Picovoice Porcupine wake word detection engine."""
    porcupine: Any  # pvporcupine.Porcupine

    @property
    def sample_rate(self) -> int:
        return self.porcupine.sample_rate

    @property
    def frame_length(self) -> int:
        return self.porcupine.frame_length

    def process(self, pcm: list[int]) -> bool:
        """Process audio samples and return True if wake word detected."""
        return self.porcupine.process(pcm) >= 0

    def close(self) -> None:
        """Clean up and release resources."""
        self.porcupine.delete()

    @classmethod
    def from_env(
        cls,
        access_key: str | None,
        wake_words: str,
        wake_word_paths: str,
        logger: Optional[Callable[[str, str], None]] = None,
    ) -> "WakeWordEngine | None":
        """
        Create a WakeWordEngine from environment configuration.
        
        Args:
            access_key: Picovoice access key (PICOVOICE_ACCESS_KEY)
            wake_words: Comma-separated built-in wake words (e.g., "jarvis,alexa")
            wake_word_paths: Comma-separated paths to custom .ppn model files
            logger: Optional callback for logging (message, level)
            
        Returns:
            WakeWordEngine instance or None if not available/configured
        """
        # Check if pvporcupine is installed
        if not PORCUPINE_AVAILABLE:
            if logger:
                logger("pvporcupine not installed; wake word engine disabled. Install with: pip install pvporcupine", "warn")
            return None
        
        if not access_key:
            if logger:
                logger("PICOVOICE_ACCESS_KEY not set; wake word engine disabled.", "warn")
            return None

        keywords: list[str] = []
        keyword_paths: list[str] = []

        # Check for built-in wake words
        for word in _split_csv(wake_words):
            if word in pvporcupine.KEYWORDS:
                keywords.append(word)
            else:
                if logger:
                    logger(f"Wake word '{word}' not supported by Porcupine keywords.", "warn")

        # Check for custom wake word model files
        for path in _split_csv(wake_word_paths):
            if os.path.exists(path):
                keyword_paths.append(path)
            else:
                if logger:
                    logger(f"Wake word model not found: {path}", "warn")

        if not keywords and not keyword_paths:
            if logger:
                logger("No valid wake words configured for Porcupine.", "warn")
            return None

        try:
            porcupine = pvporcupine.create(
                access_key=access_key,
                keywords=keywords if keywords else None,
                keyword_paths=keyword_paths if keyword_paths else None,
            )
            return cls(porcupine=porcupine)
        except Exception as e:
            if logger:
                logger(f"Failed to initialize Porcupine wake word engine: {e}", "error")
            return None


def is_available() -> bool:
    """Check if Porcupine wake word detection is available."""
    return PORCUPINE_AVAILABLE
