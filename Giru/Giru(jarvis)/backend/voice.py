"""
GIRU Voice Module
==================
Handles wake word detection (Picovoice), speech recognition, and TTS.
"""

import os
import io
import queue
import struct
import threading
import time
import wave
from typing import Optional, Callable
from pathlib import Path

import pyaudio
import pyttsx3
import requests

from config import get_config

# =============================================================================
# CONSTANTS
# =============================================================================

SAMPLE_RATE = 16000
FRAME_LENGTH = 512
CHANNELS = 1
FORMAT = pyaudio.paInt16


# =============================================================================
# PICOVOICE WAKE WORD DETECTION
# =============================================================================

class PicovoiceWakeWord:
    """
    Offline wake word detection using Picovoice Porcupine.
    
    Free tier: 3 wake words, unlimited usage
    Get access key: https://console.picovoice.ai/
    """
    
    def __init__(self, keywords: list[str] = None, callback: Callable = None):
        self.config = get_config()
        self.access_key = self.config.get_api_key("PICOVOICE_ACCESS_KEY")
        self.keywords = keywords or ["giru"]
        self.callback = callback
        self.porcupine = None
        self.audio = None
        self.stream = None
        self._running = False
        self._thread = None
    
    def is_available(self) -> bool:
        """Check if Picovoice is available."""
        if not self.access_key:
            return False
        try:
            import pvporcupine
            return True
        except ImportError:
            return False
    
    def start(self):
        """Start wake word detection."""
        if not self.is_available():
            raise RuntimeError("Picovoice not available. Install pvporcupine and set PICOVOICE_ACCESS_KEY.")
        
        import pvporcupine
        
        # Create Porcupine instance with custom wake word
        # For custom wake words, you need to train them at console.picovoice.ai
        # Using built-in keywords for now
        try:
            # Try to use "jarvis" as it's a built-in keyword
            self.porcupine = pvporcupine.create(
                access_key=self.access_key,
                keywords=["jarvis"],  # Built-in keyword similar to "giru"
                sensitivities=[self.config.get("picovoice_sensitivity", 0.5)]
            )
        except pvporcupine.PorcupineInvalidArgumentError:
            # Fall back to default keywords
            self.porcupine = pvporcupine.create(
                access_key=self.access_key,
                keywords=["computer"],
                sensitivities=[0.5]
            )
        
        self.audio = pyaudio.PyAudio()
        self.stream = self.audio.open(
            rate=self.porcupine.sample_rate,
            channels=1,
            format=pyaudio.paInt16,
            input=True,
            frames_per_buffer=self.porcupine.frame_length
        )
        
        self._running = True
        self._thread = threading.Thread(target=self._detection_loop, daemon=True)
        self._thread.start()
    
    def stop(self):
        """Stop wake word detection."""
        self._running = False
        if self._thread:
            self._thread.join(timeout=2)
        
        if self.stream:
            self.stream.stop_stream()
            self.stream.close()
        
        if self.audio:
            self.audio.terminate()
        
        if self.porcupine:
            self.porcupine.delete()
    
    def _detection_loop(self):
        """Main detection loop."""
        while self._running:
            try:
                pcm = self.stream.read(self.porcupine.frame_length, exception_on_overflow=False)
                pcm = struct.unpack_from("h" * self.porcupine.frame_length, pcm)
                
                keyword_index = self.porcupine.process(pcm)
                
                if keyword_index >= 0:
                    if self.callback:
                        self.callback()
            except Exception:
                time.sleep(0.1)


# =============================================================================
# TEXT-TO-SPEECH
# =============================================================================

class ElevenLabsTTS:
    """
    High-quality TTS using ElevenLabs API.
    
    Get API key: https://elevenlabs.io/
    """
    
    BASE_URL = "https://api.elevenlabs.io/v1"
    
    def __init__(self):
        self.config = get_config()
        self.api_key = self.config.get_api_key("ELEVENLABS_API_KEY")
        self.voice_id = self.config.get("elevenlabs_voice_id", "21m00Tcm4TlvDq8ikWAM")
        self.model_id = self.config.get("elevenlabs_model_id", "eleven_turbo_v2")
        self._audio = None
    
    def is_available(self) -> bool:
        """Check if ElevenLabs is available."""
        return bool(self.api_key)
    
    def get_voices(self) -> list[dict]:
        """Get available voices from ElevenLabs."""
        if not self.is_available():
            return []
        
        try:
            response = requests.get(
                f"{self.BASE_URL}/voices",
                headers={"xi-api-key": self.api_key},
                timeout=10
            )
            if response.status_code == 200:
                data = response.json()
                return [
                    {"id": v["voice_id"], "name": v["name"]}
                    for v in data.get("voices", [])
                ]
        except Exception:
            pass
        return []
    
    def speak(self, text: str, interrupt_event: threading.Event = None) -> bool:
        """
        Speak text using ElevenLabs.
        
        Returns:
            bool: True if speech completed, False if interrupted or failed
        """
        if not self.is_available():
            raise RuntimeError("ElevenLabs API key not configured")
        
        try:
            url = f"{self.BASE_URL}/text-to-speech/{self.voice_id}/stream"
            
            headers = {
                "xi-api-key": self.api_key,
                "Accept": "audio/mpeg",
                "Content-Type": "application/json",
            }
            
            payload = {
                "text": text,
                "model_id": self.model_id,
                "voice_settings": {
                    "stability": 0.5,
                    "similarity_boost": 0.75,
                    "style": 0.0,
                    "use_speaker_boost": True
                }
            }
            
            params = {
                "output_format": "mp3_44100_128",
                "optimize_streaming_latency": "3"
            }
            
            response = requests.post(
                url,
                headers=headers,
                params=params,
                json=payload,
                stream=True,
                timeout=30
            )
            
            if response.status_code == 401:
                raise RuntimeError("ElevenLabs API key is invalid")
            elif response.status_code == 422:
                error = response.json()
                raise RuntimeError(f"ElevenLabs error: {error.get('detail', {}).get('message', 'Unknown error')}")
            
            response.raise_for_status()
            
            # Stream audio playback
            return self._play_audio_stream(response, interrupt_event)
            
        except requests.exceptions.Timeout:
            raise RuntimeError("ElevenLabs request timed out")
        except requests.exceptions.ConnectionError:
            raise RuntimeError("Could not connect to ElevenLabs API")
    
    def _play_audio_stream(self, response, interrupt_event: threading.Event = None) -> bool:
        """Play audio stream from response."""
        try:
            # For MP3 streaming, we need pygame or similar
            # Fall back to collecting and playing
            audio_data = b""
            for chunk in response.iter_content(chunk_size=4096):
                if interrupt_event and interrupt_event.is_set():
                    return False
                if chunk:
                    audio_data += chunk
            
            if not audio_data:
                return False
            
            # Try pygame first (better MP3 support)
            try:
                import pygame
                pygame.mixer.init()
                sound = pygame.mixer.Sound(io.BytesIO(audio_data))
                sound.play()
                while pygame.mixer.get_busy():
                    if interrupt_event and interrupt_event.is_set():
                        pygame.mixer.stop()
                        return False
                    time.sleep(0.1)
                return True
            except ImportError:
                pass
            
            # Fall back to playsound
            try:
                import tempfile
                with tempfile.NamedTemporaryFile(suffix=".mp3", delete=False) as f:
                    f.write(audio_data)
                    temp_path = f.name
                
                from playsound import playsound
                playsound(temp_path)
                os.unlink(temp_path)
                return True
            except ImportError:
                pass
            
            raise RuntimeError("No audio playback library available (install pygame or playsound)")
            
        except Exception as e:
            raise RuntimeError(f"Audio playback failed: {e}")


class Pyttsx3TTS:
    """
    Offline TTS using pyttsx3.
    Works without any API keys.
    """
    
    def __init__(self):
        self.config = get_config()
        self.engine = None
        self._init_engine()
    
    def _init_engine(self):
        """Initialize the TTS engine."""
        try:
            self.engine = pyttsx3.init()
            self.engine.setProperty("rate", self.config.get("tts_rate", 175))
            
            # Try to set a better voice if available
            voices = self.engine.getProperty("voices")
            for voice in voices:
                # Prefer female voices for JARVIS-like experience
                if "female" in voice.name.lower() or "zira" in voice.name.lower():
                    self.engine.setProperty("voice", voice.id)
                    break
        except Exception:
            self.engine = None
    
    def is_available(self) -> bool:
        """Check if pyttsx3 is available."""
        return self.engine is not None
    
    def speak(self, text: str, interrupt_event: threading.Event = None) -> bool:
        """Speak text using pyttsx3."""
        if not self.is_available():
            self._init_engine()
            if not self.is_available():
                raise RuntimeError("pyttsx3 engine failed to initialize")
        
        try:
            self.engine.say(text)
            self.engine.runAndWait()
            return True
        except Exception as e:
            raise RuntimeError(f"pyttsx3 error: {e}")


class TTSManager:
    """
    Manages TTS with automatic fallback.
    """
    
    def __init__(self):
        self.config = get_config()
        self.elevenlabs = ElevenLabsTTS()
        self.pyttsx3 = Pyttsx3TTS()
        self._interrupt = threading.Event()
    
    def interrupt(self):
        """Interrupt current speech."""
        self._interrupt.set()
    
    def speak(self, text: str) -> tuple[bool, str]:
        """
        Speak text using the best available TTS.
        
        Returns:
            tuple[bool, str]: (success, provider_used)
        """
        self._interrupt.clear()
        
        # Determine which provider to use
        preferred = self.config.get_tts_provider()
        
        if preferred == "elevenlabs" and self.elevenlabs.is_available():
            try:
                success = self.elevenlabs.speak(text, self._interrupt)
                return success, "elevenlabs"
            except Exception as e:
                # Log error and fall back
                print(f"ElevenLabs TTS failed: {e}")
        
        # Fall back to pyttsx3
        if self.pyttsx3.is_available():
            try:
                success = self.pyttsx3.speak(text, self._interrupt)
                return success, "pyttsx3"
            except Exception as e:
                print(f"pyttsx3 TTS failed: {e}")
        
        return False, "none"


# =============================================================================
# SPEECH RECOGNITION
# =============================================================================

class SpeechRecognizer:
    """
    Speech recognition using Google Speech API or offline alternatives.
    """
    
    def __init__(self):
        import speech_recognition as sr
        self.recognizer = sr.Recognizer()
        self.recognizer.dynamic_energy_threshold = True
        self.recognizer.pause_threshold = 0.6
        self.recognizer.energy_threshold = 300
    
    def listen(self, timeout: int = 3, phrase_time_limit: int = 8) -> Optional[str]:
        """
        Listen for speech and return transcript.
        
        Returns:
            Optional[str]: Transcript or None if nothing heard
        """
        import speech_recognition as sr
        
        try:
            with sr.Microphone() as source:
                self.recognizer.adjust_for_ambient_noise(source, duration=0.3)
                audio = self.recognizer.listen(
                    source,
                    timeout=timeout,
                    phrase_time_limit=phrase_time_limit
                )
        except sr.WaitTimeoutError:
            return None
        
        try:
            # Use Google Speech API (free, requires internet)
            transcript = self.recognizer.recognize_google(audio)
            return transcript
        except sr.UnknownValueError:
            return None
        except sr.RequestError as e:
            print(f"Speech recognition error: {e}")
            return None


# =============================================================================
# SINGLETON INSTANCES
# =============================================================================

_tts_manager: Optional[TTSManager] = None
_wake_word: Optional[PicovoiceWakeWord] = None
_recognizer: Optional[SpeechRecognizer] = None


def get_tts() -> TTSManager:
    """Get TTS manager singleton."""
    global _tts_manager
    if _tts_manager is None:
        _tts_manager = TTSManager()
    return _tts_manager


def get_wake_word() -> Optional[PicovoiceWakeWord]:
    """Get wake word detector if available."""
    global _wake_word
    if _wake_word is None:
        _wake_word = PicovoiceWakeWord()
    return _wake_word if _wake_word.is_available() else None


def get_recognizer() -> SpeechRecognizer:
    """Get speech recognizer singleton."""
    global _recognizer
    if _recognizer is None:
        _recognizer = SpeechRecognizer()
    return _recognizer
