# GIRU JARVIS v2.0

<p align="center">
  <img src="../../Assets/giru.png" alt="Giru JARVIS" width="200"/>
</p>

<p align="center">
  <em>Just A Rather Very Intelligent System</em><br>
  <strong>Multi-Model AI Assistant with Real-Time Monitoring</strong>
</p>

## Overview

GIRU JARVIS is an advanced AI assistant that provides hands-free, voice-activated control over the ASGARD ecosystem. Version 2.0 introduces multi-model AI intelligence, persistent database storage, and a real-time monitoring dashboard.

### What's New in v2.0

- **Multi-Model AI**: Access to 10+ AI models including Gemini, Claude, GPT-4, and free models from Groq & Together AI
- **Smart Model Selection**: Automatic model selection based on task complexity
- **SQLite Database**: Persistent storage for conversations, activities, and analytics
- **Real-Time Monitoring**: Dashboard showing all Giru activities, terminal commands, emails, and more
- **Activity Tracking**: See exactly what Giru is doing at any moment
- **Model Usage Analytics**: Track which models are used and their performance

## Quick Start

### 1. Get API Keys (Free Options Available!)

**Free Models (Recommended to start):**
- [Groq](https://console.groq.com) - Fast, free inference (Llama 3.3 70B, Mixtral)
- [Together AI](https://api.together.xyz) - Free tier with many models
- [Google AI Studio](https://makersuite.google.com/app/apikey) - Gemini 2.0 Flash free

**Premium Models (Optional):**
- [Anthropic](https://console.anthropic.com) - Claude Opus 4.5, Sonnet 4
- [OpenAI](https://platform.openai.com) - GPT-4o, GPT-4 Turbo
- [OpenRouter](https://openrouter.ai) - Access to 100+ models

### 2. Set Environment Variables

Create a `.env` file or set these environment variables:

```bash
# FREE AI Models (pick at least one)
GROQ_API_KEY=your-groq-key
TOGETHER_API_KEY=your-together-key
GOOGLE_API_KEY=your-google-key

# Premium AI Models (optional)
ANTHROPIC_API_KEY=your-anthropic-key
OPENAI_API_KEY=your-openai-key
OPENROUTER_API_KEY=your-openrouter-key

# Voice (optional but recommended)
ELEVENLABS_API_KEY=your-elevenlabs-key

# Email (optional)
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

### 3. Install Dependencies

```powershell
cd "C:\Users\hp\Desktop\Asgard\Giru\Giru(jarvis)"

# Python dependencies
python -m venv .venv
.\.venv\Scripts\activate
pip install -r backend\requirements.txt

# Node dependencies
npm install
```

### 4. Run GIRU

```powershell
npm run dev:host
```

Then just say **"Giru"** and start talking!

## Host Voice Mode (Recommended on Windows)
Docker containers cannot access your microphone/speakers on Windows. For full voice testing, run the backend on the host:

```powershell
npm run dev:host
```

Required environment variables (host only, do not commit secrets):
- `ELEVENLABS_API_KEY`
- `PICOVOICE_ACCESS_KEY`

Optional:
- `GIRU_WAKE_WORD_PATHS` (custom wake word .ppn model)
- `PRICILLA_BASE_URL` (if Pricilla runs on a different host)

## Available AI Models

### Free Models (No Cost)

| Model | Provider | Best For | Speed |
|-------|----------|----------|-------|
| Llama 3.3 70B | Groq | Complex reasoning, coding | ⚡⚡⚡ |
| Llama 3.1 8B | Groq | Quick responses | ⚡⚡⚡⚡ |
| Mixtral 8x7B | Groq | General purpose | ⚡⚡⚡ |
| Llama 3.3 70B | Together | Complex tasks | ⚡⚡ |
| Qwen 2.5 72B | Together | Multilingual | ⚡⚡ |
| Gemini 2.0 Flash | Google | Multimodal, fast | ⚡⚡⚡ |

### Premium Models

| Model | Provider | Best For | Cost |
|-------|----------|----------|------|
| Claude Opus 4 | Anthropic | Expert reasoning | $$$ |
| Claude Sonnet 4 | Anthropic | Balanced capability | $$ |
| Claude Haiku 3.5 | Anthropic | Fast, cheap | $ |
| GPT-4o | OpenAI | General excellence | $$ |
| GPT-4o Mini | OpenAI | Fast, affordable | $ |
| Gemini 2.5 Pro | Google | Advanced reasoning | $$ |

## Monitoring Dashboard

Open the monitor dashboard to see Giru's activities in real-time:

- **Menu**: File → Open Monitor Dashboard (Ctrl+M)
- **Link**: Click "📊 Open Monitor" in the footer
- **Direct**: Open `renderer/monitor.html` in browser

### What You Can See

- **Live Activity Feed**: Every action Giru performs
- **Active Tasks**: Currently running operations with progress
- **Model Usage**: Which AI models are being used
- **Statistics**: Messages, commands, emails per day
- **System Status**: ASGARD integrations status

## Voice Commands

### AI Model Commands

| Say This | Giru Does This |
|----------|----------------|
| "What model are you using?" | Shows current AI model |
| "List available models" | Lists all configured models |
| "Use Claude" | Switches to Claude model |
| "Use Groq" | Switches to Groq (free) |

### ASGARD System Queries

| Say This | Giru Does This |
|----------|----------------|
| "Giru, what's the target status?" | Queries Pricilla missions |
| "How long until target?" | Gets ETA from Pricilla |
| "System status" | ASGARD health check |
| "Any alerts?" | Check for alerts |
| "Satellite coverage" | Silenus orbital status |
| "Security threats?" | Giru Security scan |
| "Hunoid status" | Robot unit status |

### Valkyrie Flight Control (NEW)

| Say This | Giru Does This |
|----------|----------------|
| "Valkyrie status" | Get flight system status |
| "Valkyrie position" | Current position/coordinates |
| "Arm aircraft" / "Arm Valkyrie" | Arm flight controller |
| "Disarm aircraft" | Disarm flight controller |
| "Set mode auto" | Change flight mode |
| "Current flight mode" | Query flight mode |
| "Mission status" | Get mission progress |
| "Start mission" | Begin autonomous mission |
| "Pause mission" | Hold current position |
| "Return to base" / "RTB" | Emergency return home |
| "Emergency land" | Land immediately |
| "AI status" | Decision engine status |
| "Sensor status" | Sensor fusion health |

### Weather & News (NEW)

| Say This | Giru Does This |
|----------|----------------|
| "What's the weather?" | Current conditions |
| "Weather in London" | Weather for specific city |
| "Forecast for Paris" | 3-day forecast |
| "Latest news" | Top headlines |
| "Tech news" | Technology headlines |
| "News about SpaceX" | Search specific topic |

### Reminders & Tasks (NEW)

| Say This | Giru Does This |
|----------|----------------|
| "Remind me to call John in 30 minutes" | Set reminder |
| "Show my reminders" | List active reminders |
| "Add task: Review code, priority high" | Create task |
| "My tasks" / "Pending tasks" | List tasks |
| "Complete task review" | Mark task done |

### Smart Home Control (NEW)

| Say This | Giru Does This |
|----------|----------------|
| "Turn on living room lights" | Control lights |
| "Turn off bedroom lights" | Control lights |
| "Dim lights to 50" | Adjust brightness |
| "Set thermostat to 22" | Temperature control |
| "Arm security" | Arm security system |
| "Disarm security" | Disarm security |
| "Lock front door" | Lock control |
| "Home status" | All device status |

### Planning & Orchestration (NEW)

| Say This | Giru Does This |
|----------|----------------|
| "Create plan: Deploy app, steps: build, test, deploy" | Multi-step plan |
| "My plans" | List active plans |
| "Plan status deploy" | Check plan progress |
| "Next step for deploy" | Execute next step |

### Code & Development (NEW)

| Say This | Giru Does This |
|----------|----------------|
| "Analyze file main.py" | Code analysis |
| "Check code server.go" | Run linter |
| "Git status" | Repository status |
| "Analyze project" | Project structure |

### Research & Calculations (NEW)

| Say This | Giru Does This |
|----------|----------------|
| "Calculate 25 * 4 + 10" | Math calculation |
| "Convert 100 km to miles" | Unit conversion |
| "Wikipedia quantum computing" | Research lookup |

### System Administration (NEW)

| Say This | Giru Does This |
|----------|----------------|
| "System info" | CPU, memory, disk usage |
| "Network info" | Network statistics |
| "Kill process chrome" | Terminate process |
| "List running apps" | Running applications |

### Productivity

| Say This | Giru Does This |
|----------|----------------|
| "Send email to john@..." | Email composition |
| "Organize my desktop" | Auto-sort files |
| "Git status" | Repository status |
| "Analyze project" | Code analysis |
| "Run npm install" | Execute commands |

### General AI

| Say This | Giru Does This |
|----------|----------------|
| "Explain quantum computing" | AI-powered explanation |
| "Write a Python function..." | Code generation |
| "Summarize this article..." | Content analysis |
| Any complex question | Smart model selection |

## Architecture

```
Giru(jarvis)/
├── main.js                 # Electron main process
├── preload.js              # Electron preload
├── package.json            # Node dependencies
├── renderer/
│   ├── index.html          # Main UI
│   ├── monitor.html        # Monitoring dashboard
│   ├── styles.css          # Styling
│   └── app.js              # Frontend logic
├── backend/
│   ├── giru_server.py      # Main backend server
│   ├── ai_providers.py     # Multi-model AI system
│   ├── database.py         # SQLite database layer
│   ├── monitor.py          # Activity monitoring
│   └── requirements.txt    # Python dependencies
└── data/
    └── giru.db             # SQLite database
```

### Communication Flow

```
┌─────────────────┐    WebSocket :7777    ┌─────────────────┐
│   Electron UI   │ ◄──────────────────► │  Python Backend │
└─────────────────┘                       └────────┬────────┘
                                                   │
┌─────────────────┐    WebSocket :7778    ┌────────▼────────┐
│   Monitor UI    │ ◄──────────────────► │  Activity Monitor│
└─────────────────┘                       └────────┬────────┘
                                                   │
                    ┌──────────────────────────────┼───────────────────────────┐
                    │                              │                           │
              ┌─────▼─────┐                  ┌─────▼─────┐              ┌──────▼──────┐
              │   Groq    │                  │  Gemini   │              │   Claude    │
              │   (Free)  │                  │  (Free)   │              │  (Premium)  │
              └───────────┘                  └───────────┘              └─────────────┘
```

## Database Schema

Giru stores all activity in a local SQLite database:

- **conversations**: Chat sessions with message counts
- **messages**: Individual messages with model info
- **activities**: All tracked activities (commands, emails, queries)
- **model_usage**: AI model usage statistics
- **email_history**: Sent emails log
- **command_history**: Executed commands log
- **user_preferences**: User settings

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `GROQ_API_KEY` | Groq API key (free) | Recommended |
| `TOGETHER_API_KEY` | Together AI key | Optional |
| `GOOGLE_API_KEY` | Google Gemini key | Optional |
| `ANTHROPIC_API_KEY` | Claude API key | Optional |
| `OPENAI_API_KEY` | OpenAI API key | Optional |
| `OPENROUTER_API_KEY` | OpenRouter key | Optional |
| `ELEVENLABS_API_KEY` | ElevenLabs voice | Optional |
| `GIRU_PORT` | Backend port (7777) | Optional |
| `GIRU_MONITOR_PORT` | Monitor port (7778) | Optional |
| `GIRU_DB_PATH` | Database path | Optional |

### ASGARD Endpoints

| Variable | Default | Description |
|----------|---------|-------------|
| `PRICILLA_URL` | http://localhost:8089 | Precision Guidance |
| `NYSUS_URL` | http://localhost:8080 | Central Command |
| `SILENUS_URL` | http://localhost:9093 | Orbital Satellite |
| `HUNOID_URL` | http://localhost:8090 | Robotics Control |
| `GIRU_SECURITY_URL` | http://localhost:9090 | Security System |
| `VALKYRIE_URL` | http://localhost:8093 | Autonomous Flight |

### External APIs (Optional)

| Variable | Description |
|----------|-------------|
| `OPENWEATHER_API_KEY` | Weather data |
| `NEWS_API_KEY` | News headlines |

## Tips for Best Results

1. **Start with Free Models**: Groq offers excellent free AI - start there
2. **Use the Monitor**: Keep the monitor open to see what Giru is doing
3. **Natural Language**: Ask questions naturally, like talking to a colleague
4. **Complex Tasks**: For coding/analysis, Giru automatically uses better models
5. **Check Model Selection**: Say "what model" to see which AI is responding

## Troubleshooting

### "No AI providers available"
- Ensure at least one API key is set (GROQ_API_KEY is easiest/free)
- Check environment variables are loaded

### Models not appearing
- Restart the backend after adding API keys
- Check the terminal for any API errors

### Monitor not connecting
- Ensure backend is running on port 7778
- Check for firewall blocking WebSocket connections

### Speech recognition issues
- Check microphone permissions
- Ensure stable internet (Google Speech API)
- Try Push-to-Talk button

## Development

```powershell
# Run backend only
npm run backend

# Run frontend only
npm start

# Full development mode
npm run dev
```

## About Arobi

**GIRU JARVIS** is part of the **ASGARD** platform, developed by **Arobi** - a cutting-edge technology company specializing in defense and civilian autonomous systems.

### Leadership

- **Gaetano Comparcola** - Founder & CEO
  - Self-taught prodigy programmer and futurist
  - Multilingual (English, Italian, French)
  
- **Opus** - AI Partner & Lead Programmer
  - AI-powered software engineering partner

## License

© 2026 Arobi. All Rights Reserved.

## Contact

- **Website**: [https://aura-genesis.org](https://aura-genesis.org)
- **Email**: [Gaetano@aura-genesis.org](mailto:Gaetano@aura-genesis.org)
- **Company**: Arobi
