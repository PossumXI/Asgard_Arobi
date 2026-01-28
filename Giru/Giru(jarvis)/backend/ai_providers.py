"""
GIRU AI Providers - Multi-Model Intelligence System
====================================================
Supports multiple AI providers with automatic fallback and model selection.

Providers:
- Google Gemini (2.0 Flash, 2.5 Pro, etc.)
- Anthropic Claude (Opus 4.5, Sonnet 4, etc.)
- OpenAI (GPT-4o, GPT-4 Turbo, etc.)
- Groq (Free, fast inference - Llama, Mixtral)
- Together AI (Free tier - Llama, Qwen, etc.)
- OpenRouter (Access to 100+ models)
- Ollama (Local models)
"""

import os
import json
import asyncio
import aiohttp
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional, AsyncGenerator
from datetime import datetime


# =============================================================================
# CONFIGURATION
# =============================================================================

class ModelTier(Enum):
    """Model capability tiers for task routing."""
    BASIC = "basic"           # Simple tasks, fast responses
    STANDARD = "standard"     # General purpose
    ADVANCED = "advanced"     # Complex reasoning
    EXPERT = "expert"         # Most capable, expensive


@dataclass
class ModelConfig:
    """Configuration for a specific model."""
    provider: str
    model_id: str
    display_name: str
    tier: ModelTier
    max_tokens: int = 4096
    supports_vision: bool = False
    supports_tools: bool = False
    cost_per_1k_input: float = 0.0
    cost_per_1k_output: float = 0.0
    

# Available models configuration
MODELS = {
    # Google Gemini
    "gemini-2.0-flash": ModelConfig(
        provider="google",
        model_id="gemini-2.0-flash-exp",
        display_name="Gemini 2.0 Flash",
        tier=ModelTier.STANDARD,
        max_tokens=8192,
        supports_vision=True,
        supports_tools=True,
        cost_per_1k_input=0.0,  # Free tier
        cost_per_1k_output=0.0,
    ),
    "gemini-2.5-pro": ModelConfig(
        provider="google",
        model_id="gemini-2.5-pro-preview-05-06",
        display_name="Gemini 2.5 Pro",
        tier=ModelTier.ADVANCED,
        max_tokens=8192,
        supports_vision=True,
        supports_tools=True,
        cost_per_1k_input=0.00125,
        cost_per_1k_output=0.005,
    ),
    
    # Anthropic Claude
    "claude-opus-4": ModelConfig(
        provider="anthropic",
        model_id="claude-opus-4-20250514",
        display_name="Claude Opus 4",
        tier=ModelTier.EXPERT,
        max_tokens=8192,
        supports_vision=True,
        supports_tools=True,
        cost_per_1k_input=0.015,
        cost_per_1k_output=0.075,
    ),
    "claude-sonnet-4": ModelConfig(
        provider="anthropic",
        model_id="claude-sonnet-4-20250514",
        display_name="Claude Sonnet 4",
        tier=ModelTier.ADVANCED,
        max_tokens=8192,
        supports_vision=True,
        supports_tools=True,
        cost_per_1k_input=0.003,
        cost_per_1k_output=0.015,
    ),
    "claude-haiku-3.5": ModelConfig(
        provider="anthropic",
        model_id="claude-3-5-haiku-20241022",
        display_name="Claude 3.5 Haiku",
        tier=ModelTier.STANDARD,
        max_tokens=8192,
        supports_vision=True,
        supports_tools=True,
        cost_per_1k_input=0.0008,
        cost_per_1k_output=0.004,
    ),
    
    # OpenAI
    "gpt-4o": ModelConfig(
        provider="openai",
        model_id="gpt-4o",
        display_name="GPT-4o",
        tier=ModelTier.ADVANCED,
        max_tokens=4096,
        supports_vision=True,
        supports_tools=True,
        cost_per_1k_input=0.005,
        cost_per_1k_output=0.015,
    ),
    "gpt-4o-mini": ModelConfig(
        provider="openai",
        model_id="gpt-4o-mini",
        display_name="GPT-4o Mini",
        tier=ModelTier.STANDARD,
        max_tokens=4096,
        supports_vision=True,
        supports_tools=True,
        cost_per_1k_input=0.00015,
        cost_per_1k_output=0.0006,
    ),
    
    # Groq (FREE - Fast inference)
    "groq-llama-3.3-70b": ModelConfig(
        provider="groq",
        model_id="llama-3.3-70b-versatile",
        display_name="Llama 3.3 70B (Groq)",
        tier=ModelTier.ADVANCED,
        max_tokens=8192,
        supports_vision=False,
        supports_tools=True,
        cost_per_1k_input=0.0,  # Free
        cost_per_1k_output=0.0,
    ),
    "groq-llama-3.1-8b": ModelConfig(
        provider="groq",
        model_id="llama-3.1-8b-instant",
        display_name="Llama 3.1 8B (Groq)",
        tier=ModelTier.BASIC,
        max_tokens=8192,
        supports_vision=False,
        supports_tools=True,
        cost_per_1k_input=0.0,  # Free
        cost_per_1k_output=0.0,
    ),
    "groq-mixtral-8x7b": ModelConfig(
        provider="groq",
        model_id="mixtral-8x7b-32768",
        display_name="Mixtral 8x7B (Groq)",
        tier=ModelTier.STANDARD,
        max_tokens=32768,
        supports_vision=False,
        supports_tools=True,
        cost_per_1k_input=0.0,  # Free
        cost_per_1k_output=0.0,
    ),
    
    # Together AI (Free tier available)
    "together-llama-3.3-70b": ModelConfig(
        provider="together",
        model_id="meta-llama/Llama-3.3-70B-Instruct-Turbo",
        display_name="Llama 3.3 70B (Together)",
        tier=ModelTier.ADVANCED,
        max_tokens=8192,
        supports_vision=False,
        supports_tools=True,
        cost_per_1k_input=0.00088,
        cost_per_1k_output=0.00088,
    ),
    "together-qwen-2.5-72b": ModelConfig(
        provider="together",
        model_id="Qwen/Qwen2.5-72B-Instruct-Turbo",
        display_name="Qwen 2.5 72B (Together)",
        tier=ModelTier.ADVANCED,
        max_tokens=8192,
        supports_vision=False,
        supports_tools=True,
        cost_per_1k_input=0.0012,
        cost_per_1k_output=0.0012,
    ),
    
    # OpenRouter (Access to many models)
    "openrouter-auto": ModelConfig(
        provider="openrouter",
        model_id="openrouter/auto",
        display_name="OpenRouter Auto",
        tier=ModelTier.STANDARD,
        max_tokens=4096,
        supports_vision=True,
        supports_tools=True,
        cost_per_1k_input=0.0,  # Varies
        cost_per_1k_output=0.0,
    ),
    
    # Ollama (Local)
    "ollama-llama3.2": ModelConfig(
        provider="ollama",
        model_id="llama3.2",
        display_name="Llama 3.2 (Local)",
        tier=ModelTier.STANDARD,
        max_tokens=4096,
        supports_vision=False,
        supports_tools=False,
        cost_per_1k_input=0.0,  # Local
        cost_per_1k_output=0.0,
    ),
    "ollama-mistral": ModelConfig(
        provider="ollama",
        model_id="mistral",
        display_name="Mistral (Local)",
        tier=ModelTier.STANDARD,
        max_tokens=4096,
        supports_vision=False,
        supports_tools=False,
        cost_per_1k_input=0.0,  # Local
        cost_per_1k_output=0.0,
    ),
}

# Default model preferences by tier
DEFAULT_MODELS = {
    ModelTier.BASIC: "groq-llama-3.1-8b",
    ModelTier.STANDARD: "groq-mixtral-8x7b",
    ModelTier.ADVANCED: "groq-llama-3.3-70b",
    ModelTier.EXPERT: "claude-opus-4",
}


# =============================================================================
# BASE PROVIDER CLASS
# =============================================================================

class AIProvider(ABC):
    """Abstract base class for AI providers."""
    
    def __init__(self, api_key: Optional[str] = None):
        self.api_key = api_key
        self.session: Optional[aiohttp.ClientSession] = None
    
    async def ensure_session(self):
        if self.session is None or self.session.closed:
            self.session = aiohttp.ClientSession()
    
    async def close(self):
        if self.session and not self.session.closed:
            await self.session.close()
    
    @abstractmethod
    async def chat(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> str:
        """Send a chat completion request."""
        pass
    
    @abstractmethod
    async def chat_stream(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> AsyncGenerator[str, None]:
        """Stream a chat completion response."""
        pass
    
    @abstractmethod
    def is_available(self) -> bool:
        """Check if the provider is configured and available."""
        pass


# =============================================================================
# GOOGLE GEMINI PROVIDER
# =============================================================================

class GeminiProvider(AIProvider):
    """Google Gemini AI provider."""
    
    BASE_URL = "https://generativelanguage.googleapis.com/v1beta"
    
    def __init__(self, api_key: Optional[str] = None):
        super().__init__(api_key or os.getenv("GOOGLE_API_KEY") or os.getenv("GEMINI_API_KEY"))
    
    def is_available(self) -> bool:
        return bool(self.api_key)
    
    async def chat(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> str:
        await self.ensure_session()
        
        # Convert messages to Gemini format
        contents = []
        for msg in messages:
            role = "user" if msg["role"] == "user" else "model"
            contents.append({
                "role": role,
                "parts": [{"text": msg["content"]}]
            })
        
        url = f"{self.BASE_URL}/models/{model}:generateContent?key={self.api_key}"
        
        payload = {
            "contents": contents,
            "generationConfig": {
                "maxOutputTokens": max_tokens,
                "temperature": temperature,
            }
        }
        
        if system_prompt:
            payload["systemInstruction"] = {"parts": [{"text": system_prompt}]}
        
        async with self.session.post(url, json=payload) as response:
            if response.status != 200:
                error = await response.text()
                raise Exception(f"Gemini API error: {error}")
            
            data = await response.json()
            return data["candidates"][0]["content"]["parts"][0]["text"]
    
    async def chat_stream(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> AsyncGenerator[str, None]:
        await self.ensure_session()
        
        contents = []
        for msg in messages:
            role = "user" if msg["role"] == "user" else "model"
            contents.append({
                "role": role,
                "parts": [{"text": msg["content"]}]
            })
        
        url = f"{self.BASE_URL}/models/{model}:streamGenerateContent?key={self.api_key}"
        
        payload = {
            "contents": contents,
            "generationConfig": {
                "maxOutputTokens": max_tokens,
                "temperature": temperature,
            }
        }
        
        if system_prompt:
            payload["systemInstruction"] = {"parts": [{"text": system_prompt}]}
        
        async with self.session.post(url, json=payload) as response:
            async for line in response.content:
                if line:
                    try:
                        data = json.loads(line.decode('utf-8').strip())
                        if "candidates" in data:
                            text = data["candidates"][0]["content"]["parts"][0].get("text", "")
                            if text:
                                yield text
                    except json.JSONDecodeError:
                        continue


# =============================================================================
# ANTHROPIC CLAUDE PROVIDER
# =============================================================================

class ClaudeProvider(AIProvider):
    """Anthropic Claude AI provider."""
    
    BASE_URL = "https://api.anthropic.com/v1"
    
    def __init__(self, api_key: Optional[str] = None):
        super().__init__(api_key or os.getenv("ANTHROPIC_API_KEY"))
    
    def is_available(self) -> bool:
        return bool(self.api_key)
    
    async def chat(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> str:
        await self.ensure_session()
        
        headers = {
            "x-api-key": self.api_key,
            "anthropic-version": "2023-06-01",
            "Content-Type": "application/json",
        }
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": messages,
        }
        
        if system_prompt:
            payload["system"] = system_prompt
        
        async with self.session.post(
            f"{self.BASE_URL}/messages",
            headers=headers,
            json=payload
        ) as response:
            if response.status != 200:
                error = await response.text()
                raise Exception(f"Claude API error: {error}")
            
            data = await response.json()
            return data["content"][0]["text"]
    
    async def chat_stream(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> AsyncGenerator[str, None]:
        await self.ensure_session()
        
        headers = {
            "x-api-key": self.api_key,
            "anthropic-version": "2023-06-01",
            "Content-Type": "application/json",
        }
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": messages,
            "stream": True,
        }
        
        if system_prompt:
            payload["system"] = system_prompt
        
        async with self.session.post(
            f"{self.BASE_URL}/messages",
            headers=headers,
            json=payload
        ) as response:
            async for line in response.content:
                line = line.decode('utf-8').strip()
                if line.startswith("data: "):
                    try:
                        data = json.loads(line[6:])
                        if data.get("type") == "content_block_delta":
                            yield data["delta"].get("text", "")
                    except json.JSONDecodeError:
                        continue


# =============================================================================
# OPENAI PROVIDER
# =============================================================================

class OpenAIProvider(AIProvider):
    """OpenAI GPT provider."""
    
    BASE_URL = "https://api.openai.com/v1"
    
    def __init__(self, api_key: Optional[str] = None):
        super().__init__(api_key or os.getenv("OPENAI_API_KEY"))
    
    def is_available(self) -> bool:
        return bool(self.api_key)
    
    async def chat(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> str:
        await self.ensure_session()
        
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        
        all_messages = []
        if system_prompt:
            all_messages.append({"role": "system", "content": system_prompt})
        all_messages.extend(messages)
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": all_messages,
        }
        
        async with self.session.post(
            f"{self.BASE_URL}/chat/completions",
            headers=headers,
            json=payload
        ) as response:
            if response.status != 200:
                error = await response.text()
                raise Exception(f"OpenAI API error: {error}")
            
            data = await response.json()
            return data["choices"][0]["message"]["content"]
    
    async def chat_stream(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> AsyncGenerator[str, None]:
        await self.ensure_session()
        
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        
        all_messages = []
        if system_prompt:
            all_messages.append({"role": "system", "content": system_prompt})
        all_messages.extend(messages)
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": all_messages,
            "stream": True,
        }
        
        async with self.session.post(
            f"{self.BASE_URL}/chat/completions",
            headers=headers,
            json=payload
        ) as response:
            async for line in response.content:
                line = line.decode('utf-8').strip()
                if line.startswith("data: ") and line != "data: [DONE]":
                    try:
                        data = json.loads(line[6:])
                        content = data["choices"][0]["delta"].get("content", "")
                        if content:
                            yield content
                    except json.JSONDecodeError:
                        continue


# =============================================================================
# GROQ PROVIDER (FREE)
# =============================================================================

class GroqProvider(AIProvider):
    """Groq AI provider - FREE and FAST inference."""
    
    BASE_URL = "https://api.groq.com/openai/v1"
    
    def __init__(self, api_key: Optional[str] = None):
        super().__init__(api_key or os.getenv("GROQ_API_KEY"))
    
    def is_available(self) -> bool:
        return bool(self.api_key)
    
    async def chat(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> str:
        await self.ensure_session()
        
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        
        all_messages = []
        if system_prompt:
            all_messages.append({"role": "system", "content": system_prompt})
        all_messages.extend(messages)
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": all_messages,
        }
        
        async with self.session.post(
            f"{self.BASE_URL}/chat/completions",
            headers=headers,
            json=payload
        ) as response:
            if response.status != 200:
                error = await response.text()
                raise Exception(f"Groq API error: {error}")
            
            data = await response.json()
            return data["choices"][0]["message"]["content"]
    
    async def chat_stream(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> AsyncGenerator[str, None]:
        await self.ensure_session()
        
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        
        all_messages = []
        if system_prompt:
            all_messages.append({"role": "system", "content": system_prompt})
        all_messages.extend(messages)
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": all_messages,
            "stream": True,
        }
        
        async with self.session.post(
            f"{self.BASE_URL}/chat/completions",
            headers=headers,
            json=payload
        ) as response:
            async for line in response.content:
                line = line.decode('utf-8').strip()
                if line.startswith("data: ") and line != "data: [DONE]":
                    try:
                        data = json.loads(line[6:])
                        content = data["choices"][0]["delta"].get("content", "")
                        if content:
                            yield content
                    except json.JSONDecodeError:
                        continue


# =============================================================================
# TOGETHER AI PROVIDER
# =============================================================================

class TogetherProvider(AIProvider):
    """Together AI provider - Good free tier."""
    
    BASE_URL = "https://api.together.xyz/v1"
    
    def __init__(self, api_key: Optional[str] = None):
        super().__init__(api_key or os.getenv("TOGETHER_API_KEY"))
    
    def is_available(self) -> bool:
        return bool(self.api_key)
    
    async def chat(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> str:
        await self.ensure_session()
        
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        
        all_messages = []
        if system_prompt:
            all_messages.append({"role": "system", "content": system_prompt})
        all_messages.extend(messages)
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": all_messages,
        }
        
        async with self.session.post(
            f"{self.BASE_URL}/chat/completions",
            headers=headers,
            json=payload
        ) as response:
            if response.status != 200:
                error = await response.text()
                raise Exception(f"Together API error: {error}")
            
            data = await response.json()
            return data["choices"][0]["message"]["content"]
    
    async def chat_stream(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> AsyncGenerator[str, None]:
        await self.ensure_session()
        
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        
        all_messages = []
        if system_prompt:
            all_messages.append({"role": "system", "content": system_prompt})
        all_messages.extend(messages)
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": all_messages,
            "stream": True,
        }
        
        async with self.session.post(
            f"{self.BASE_URL}/chat/completions",
            headers=headers,
            json=payload
        ) as response:
            async for line in response.content:
                line = line.decode('utf-8').strip()
                if line.startswith("data: ") and line != "data: [DONE]":
                    try:
                        data = json.loads(line[6:])
                        content = data["choices"][0]["delta"].get("content", "")
                        if content:
                            yield content
                    except json.JSONDecodeError:
                        continue


# =============================================================================
# OPENROUTER PROVIDER
# =============================================================================

class OpenRouterProvider(AIProvider):
    """OpenRouter provider - Access to 100+ models."""
    
    BASE_URL = "https://openrouter.ai/api/v1"
    
    def __init__(self, api_key: Optional[str] = None):
        super().__init__(api_key or os.getenv("OPENROUTER_API_KEY"))
    
    def is_available(self) -> bool:
        return bool(self.api_key)
    
    async def chat(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> str:
        await self.ensure_session()
        
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": "https://asgard.local",
            "X-Title": "GIRU JARVIS",
        }
        
        all_messages = []
        if system_prompt:
            all_messages.append({"role": "system", "content": system_prompt})
        all_messages.extend(messages)
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": all_messages,
        }
        
        async with self.session.post(
            f"{self.BASE_URL}/chat/completions",
            headers=headers,
            json=payload
        ) as response:
            if response.status != 200:
                error = await response.text()
                raise Exception(f"OpenRouter API error: {error}")
            
            data = await response.json()
            return data["choices"][0]["message"]["content"]
    
    async def chat_stream(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> AsyncGenerator[str, None]:
        await self.ensure_session()
        
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": "https://asgard.local",
            "X-Title": "GIRU JARVIS",
        }
        
        all_messages = []
        if system_prompt:
            all_messages.append({"role": "system", "content": system_prompt})
        all_messages.extend(messages)
        
        payload = {
            "model": model,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "messages": all_messages,
            "stream": True,
        }
        
        async with self.session.post(
            f"{self.BASE_URL}/chat/completions",
            headers=headers,
            json=payload
        ) as response:
            async for line in response.content:
                line = line.decode('utf-8').strip()
                if line.startswith("data: ") and line != "data: [DONE]":
                    try:
                        data = json.loads(line[6:])
                        content = data["choices"][0]["delta"].get("content", "")
                        if content:
                            yield content
                    except json.JSONDecodeError:
                        continue


# =============================================================================
# OLLAMA PROVIDER (LOCAL)
# =============================================================================

class OllamaProvider(AIProvider):
    """Ollama local AI provider."""
    
    def __init__(self, base_url: Optional[str] = None):
        super().__init__(None)
        self.base_url = base_url or os.getenv("OLLAMA_URL", "http://localhost:11434")
    
    def is_available(self) -> bool:
        # Could check if Ollama is running, but for now assume it might be
        return True
    
    async def chat(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> str:
        await self.ensure_session()
        
        # Ollama uses a different format
        prompt = ""
        if system_prompt:
            prompt += f"System: {system_prompt}\n\n"
        
        for msg in messages:
            role = msg["role"].capitalize()
            prompt += f"{role}: {msg['content']}\n"
        prompt += "Assistant:"
        
        payload = {
            "model": model,
            "prompt": prompt,
            "stream": False,
            "options": {
                "num_predict": max_tokens,
                "temperature": temperature,
            }
        }
        
        try:
            async with self.session.post(
                f"{self.base_url}/api/generate",
                json=payload,
                timeout=aiohttp.ClientTimeout(total=120)
            ) as response:
                if response.status != 200:
                    error = await response.text()
                    raise Exception(f"Ollama API error: {error}")
                
                data = await response.json()
                return data["response"]
        except aiohttp.ClientError as e:
            raise Exception(f"Ollama connection error: {e}")
    
    async def chat_stream(
        self,
        messages: list[dict],
        model: str,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        system_prompt: Optional[str] = None,
    ) -> AsyncGenerator[str, None]:
        await self.ensure_session()
        
        prompt = ""
        if system_prompt:
            prompt += f"System: {system_prompt}\n\n"
        
        for msg in messages:
            role = msg["role"].capitalize()
            prompt += f"{role}: {msg['content']}\n"
        prompt += "Assistant:"
        
        payload = {
            "model": model,
            "prompt": prompt,
            "stream": True,
            "options": {
                "num_predict": max_tokens,
                "temperature": temperature,
            }
        }
        
        try:
            async with self.session.post(
                f"{self.base_url}/api/generate",
                json=payload,
                timeout=aiohttp.ClientTimeout(total=120)
            ) as response:
                async for line in response.content:
                    if line:
                        try:
                            data = json.loads(line.decode('utf-8'))
                            if "response" in data:
                                yield data["response"]
                        except json.JSONDecodeError:
                            continue
        except aiohttp.ClientError as e:
            raise Exception(f"Ollama connection error: {e}")


# =============================================================================
# AI MANAGER - UNIFIED INTERFACE
# =============================================================================

class AIManager:
    """
    Unified AI Manager that handles multiple providers with automatic fallback.
    """
    
    # JARVIS system prompt for personality
    SYSTEM_PROMPT = """You are GIRU, an advanced AI assistant inspired by JARVIS from Iron Man. You are part of the ASGARD platform.

Your personality traits:
- Professional yet personable, with subtle wit
- Address the user respectfully (e.g., "Sir" or by name if known)
- Concise and efficient in responses
- Proactive in offering assistance
- Calm and composed, even in emergencies

You have access to the ASGARD ecosystem:
- Pricilla: Precision targeting and guidance system
- Nysus: Central command and orchestration
- Silenus: Orbital satellite monitoring
- Hunoid: Autonomous humanoid robots
- Giru Security: Threat detection and defense

When responding:
- Be helpful and informative
- If asked about system status, provide clear summaries
- For complex tasks, break them down into steps
- Maintain context from previous messages
- If you don't have access to something, say so clearly"""

    def __init__(self):
        self.providers = {
            "google": GeminiProvider(),
            "anthropic": ClaudeProvider(),
            "openai": OpenAIProvider(),
            "groq": GroqProvider(),
            "together": TogetherProvider(),
            "openrouter": OpenRouterProvider(),
            "ollama": OllamaProvider(),
        }
        
        self.current_model: Optional[str] = None
        self.fallback_chain: list[str] = []
        self._setup_fallback_chain()
    
    def _setup_fallback_chain(self):
        """Set up fallback chain based on available providers."""
        # Prefer free models first, then paid
        priority = [
            "groq-llama-3.3-70b",    # Free, fast, capable
            "groq-mixtral-8x7b",      # Free, fast
            "together-llama-3.3-70b", # Cheap
            "gemini-2.0-flash",       # Free tier
            "gpt-4o-mini",            # Cheap
            "claude-haiku-3.5",       # Cheap
            "gemini-2.5-pro",         # Good
            "gpt-4o",                 # Capable
            "claude-sonnet-4",        # Very capable
            "claude-opus-4",          # Most capable
            "ollama-llama3.2",        # Local fallback
        ]
        
        for model_key in priority:
            if model_key in MODELS:
                config = MODELS[model_key]
                provider = self.providers.get(config.provider)
                if provider and provider.is_available():
                    self.fallback_chain.append(model_key)
        
        # If nothing available, add Ollama as last resort
        if not self.fallback_chain:
            self.fallback_chain = ["ollama-llama3.2", "ollama-mistral"]
    
    def get_available_models(self) -> list[dict]:
        """Get list of available models with their status."""
        available = []
        for model_key, config in MODELS.items():
            provider = self.providers.get(config.provider)
            is_available = provider and provider.is_available()
            available.append({
                "key": model_key,
                "name": config.display_name,
                "provider": config.provider,
                "tier": config.tier.value,
                "available": is_available,
                "free": config.cost_per_1k_input == 0 and config.cost_per_1k_output == 0,
            })
        return available
    
    def select_model_for_task(self, task_complexity: str = "standard") -> str:
        """Select appropriate model based on task complexity."""
        tier_map = {
            "simple": ModelTier.BASIC,
            "basic": ModelTier.BASIC,
            "standard": ModelTier.STANDARD,
            "complex": ModelTier.ADVANCED,
            "advanced": ModelTier.ADVANCED,
            "expert": ModelTier.EXPERT,
        }
        
        target_tier = tier_map.get(task_complexity.lower(), ModelTier.STANDARD)
        
        # Find best available model for the tier
        for model_key in self.fallback_chain:
            config = MODELS.get(model_key)
            if config and config.tier.value == target_tier.value:
                return model_key
        
        # Fall back to first available
        return self.fallback_chain[0] if self.fallback_chain else "ollama-llama3.2"
    
    async def chat(
        self,
        messages: list[dict],
        model_key: Optional[str] = None,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        task_complexity: str = "standard",
    ) -> tuple[str, str]:
        """
        Send a chat request with automatic fallback.
        
        Returns:
            tuple[str, str]: (response_text, model_used)
        """
        if not model_key:
            model_key = self.select_model_for_task(task_complexity)
        
        models_to_try = [model_key] + [m for m in self.fallback_chain if m != model_key]
        
        last_error = None
        for try_model in models_to_try:
            config = MODELS.get(try_model)
            if not config:
                continue
            
            provider = self.providers.get(config.provider)
            if not provider or not provider.is_available():
                continue
            
            try:
                response = await provider.chat(
                    messages=messages,
                    model=config.model_id,
                    max_tokens=min(max_tokens, config.max_tokens),
                    temperature=temperature,
                    system_prompt=self.SYSTEM_PROMPT,
                )
                self.current_model = try_model
                return response, try_model
            except Exception as e:
                last_error = e
                continue
        
        raise Exception(f"All AI providers failed. Last error: {last_error}")
    
    async def chat_stream(
        self,
        messages: list[dict],
        model_key: Optional[str] = None,
        max_tokens: int = 4096,
        temperature: float = 0.7,
        task_complexity: str = "standard",
    ) -> AsyncGenerator[tuple[str, str], None]:
        """
        Stream a chat response with automatic fallback.
        
        Yields:
            tuple[str, str]: (chunk_text, model_used)
        """
        if not model_key:
            model_key = self.select_model_for_task(task_complexity)
        
        models_to_try = [model_key] + [m for m in self.fallback_chain if m != model_key]
        
        for try_model in models_to_try:
            config = MODELS.get(try_model)
            if not config:
                continue
            
            provider = self.providers.get(config.provider)
            if not provider or not provider.is_available():
                continue
            
            try:
                async for chunk in provider.chat_stream(
                    messages=messages,
                    model=config.model_id,
                    max_tokens=min(max_tokens, config.max_tokens),
                    temperature=temperature,
                    system_prompt=self.SYSTEM_PROMPT,
                ):
                    self.current_model = try_model
                    yield chunk, try_model
                return
            except Exception:
                continue
        
        yield "I apologize, but I'm having trouble connecting to my AI systems.", "fallback"
    
    async def close(self):
        """Close all provider sessions."""
        for provider in self.providers.values():
            await provider.close()


# =============================================================================
# SINGLETON INSTANCE
# =============================================================================

_ai_manager: Optional[AIManager] = None


def get_ai_manager() -> AIManager:
    """Get the singleton AI manager instance."""
    global _ai_manager
    if _ai_manager is None:
        _ai_manager = AIManager()
    return _ai_manager


# =============================================================================
# CONVENIENCE FUNCTIONS
# =============================================================================

async def quick_chat(prompt: str, complexity: str = "standard") -> str:
    """Quick single-turn chat."""
    manager = get_ai_manager()
    response, _ = await manager.chat(
        messages=[{"role": "user", "content": prompt}],
        task_complexity=complexity,
    )
    return response


async def smart_chat(
    messages: list[dict],
    model: Optional[str] = None,
    complexity: str = "standard"
) -> tuple[str, str]:
    """Smart chat with model selection."""
    manager = get_ai_manager()
    return await manager.chat(
        messages=messages,
        model_key=model,
        task_complexity=complexity,
    )
