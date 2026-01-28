/**
 * GIRU JARVIS - Frontend Application
 * Advanced AI Assistant with ASGARD Integration
 */

// =============================================================================
// DOM ELEMENTS
// =============================================================================

const core = document.getElementById("core");
const statusText = document.getElementById("status-text");
const voiceVisualizer = document.getElementById("voice-visualizer");
const connectionPill = document.getElementById("connection-pill");
const wakeHintText = document.getElementById("wake-hint-text");
const conversation = document.getElementById("conversation");
const systemLog = document.getElementById("system-log");
const micToggle = document.getElementById("mic-toggle");
const pushToTalk = document.getElementById("push-to-talk");
const interruptBtn = document.getElementById("interrupt-btn");
const clearChat = document.getElementById("clear-chat");
const toggleText = document.getElementById("toggle-text");
const micState = document.getElementById("mic-state");
const camState = document.getElementById("cam-state");
const filesystemState = document.getElementById("filesystem-state");
const emailState = document.getElementById("email-state");
const networkState = document.getElementById("network-state");
const gitState = document.getElementById("git-state");
const systemState = document.getElementById("system-state");
const textForm = document.getElementById("text-form");
const textInput = document.getElementById("text-input");

// ASGARD Systems
const sysPricilla = document.getElementById("sys-pricilla");
const sysNysus = document.getElementById("sys-nysus");
const sysSilenus = document.getElementById("sys-silenus");
const sysHunoid = document.getElementById("sys-hunoid");
const sysSecurity = document.getElementById("sys-security");

// =============================================================================
// STATE
// =============================================================================

let socket;
let connected = false;
let permissions = {
  microphone: false,
  camera: false,
  filesystem: false,
  email: false,
  network: false,
  git: false,
  system: false,
};

const PERMISSION_LABELS = {
  microphone: "Microphone",
  camera: "Camera",
  filesystem: "Filesystem",
  email: "Email",
  network: "Network",
  git: "Git",
  system: "System",
};
let currentStatus = "idle";
let pushToTalkActive = false;
let textVisible = false;

// =============================================================================
// UI UPDATES
// =============================================================================

const STATUS_MAP = {
  idle: { text: "STANDBY", hint: 'Say <strong>"Hello"</strong>, <strong>"Hello Giru"</strong>, or <strong>"Giru"</strong> to activate' },
  listening: { text: "LISTENING", hint: 'Say <strong>"Giru"</strong> or speak a command' },
  active: { text: "PROCESSING", hint: "Processing your request..." },
  speaking: { text: "SPEAKING", hint: "Giru is responding..." },
};

const updateStatus = (value) => {
  currentStatus = value;
  
  // Update reactor visualization
  core.className = `arc-reactor ${value}`;
  
  // Update status text
  const config = STATUS_MAP[value] || STATUS_MAP.idle;
  statusText.textContent = config.text;
  wakeHintText.innerHTML = config.hint;
  
  // Update voice visualizer
  voiceVisualizer.classList.toggle("active", value === "listening" || value === "speaking");
  voiceVisualizer.classList.toggle("speaking", value === "speaking");
};

const updateConnection = (value) => {
  connected = value;
  connectionPill.classList.toggle("connected", value);
  connectionPill.classList.toggle("disconnected", !value);
  connectionPill.querySelector(".indicator-label").textContent = value ? "ONLINE" : "OFFLINE";
};

const updatePermission = (key, value) => {
  permissions[key] = value;
  const badgeMap = {
    microphone: micState,
    camera: camState,
    filesystem: filesystemState,
    email: emailState,
    network: networkState,
    git: gitState,
    system: systemState,
  };
  const badge = badgeMap[key];
  if (!badge) return;
  
  badge.textContent = value ? "On" : "Off";
  badge.classList.toggle("on", value);
  badge.classList.toggle("off", !value);
  
  if (key === "microphone") {
    const btnLabel = micToggle.querySelector(".btn-label");
    btnLabel.textContent = value ? "Disable Microphone" : "Enable Microphone";
    micToggle.classList.toggle("active", value);
  }

  const toggleButton = document.querySelector(`[data-permission-toggle="${key}"]`);
  if (toggleButton) {
    const label = toggleButton.querySelector(".btn-label");
    if (label) {
      const display = PERMISSION_LABELS[key] || key;
      label.textContent = value ? `Disable ${display}` : `Enable ${display}`;
    }
  }
};

const updateSystemStatus = (system, online) => {
  const el = document.getElementById(`sys-${system}`);
  if (el) {
    el.classList.toggle("online", online);
    el.classList.toggle("offline", !online);
    el.querySelector(".system-status").textContent = online ? "ONLINE" : "OFFLINE";
  }
};

// =============================================================================
// LOGGING & CONVERSATION
// =============================================================================

const formatTime = () => {
  const now = new Date();
  return now.toLocaleTimeString("en-US", { hour12: false });
};

const addLog = (message, level = "info") => {
  const entry = document.createElement("div");
  entry.className = `log-entry ${level}`;
  entry.textContent = `[${formatTime()}] ${message}`;
  systemLog.appendChild(entry);
  systemLog.scrollTop = systemLog.scrollHeight;
  
  // Keep log manageable
  while (systemLog.children.length > 100) {
    systemLog.removeChild(systemLog.firstChild);
  }
};

const addUtterance = (role, text) => {
  const bubble = document.createElement("div");
  bubble.className = `bubble ${role}`;
  bubble.textContent = text;
  conversation.appendChild(bubble);
  conversation.scrollTop = conversation.scrollHeight;
};

const clearConversation = () => {
  conversation.innerHTML = "";
  addLog("Conversation cleared.");
};

// =============================================================================
// WEBSOCKET COMMUNICATION
// =============================================================================

const send = (payload) => {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    addLog("Backend not connected.", "warn");
    return;
  }
  socket.send(JSON.stringify(payload));
};

const connect = () => {
  const port = 7777; // Match backend default
  socket = new WebSocket(`ws://127.0.0.1:${port}`);

  socket.addEventListener("open", () => {
    updateConnection(true);
    addLog("Connected to Giru backend.");
    send({ type: "client_hello" });
    
    // Check ASGARD systems after connection
    checkAsgardSystems();
  });

  socket.addEventListener("close", () => {
    updateConnection(false);
    updateStatus("idle");
    addLog("Backend disconnected. Reconnecting...", "warn");
    setTimeout(connect, 2000);
  });

  socket.addEventListener("error", (error) => {
    addLog(`Connection error: ${error.message || "Unknown"}`, "error");
  });

  socket.addEventListener("message", (event) => {
    try {
      const data = JSON.parse(event.data);
      handleMessage(data);
    } catch (error) {
      addLog(`Parse error: ${error.message}`, "error");
    }
  });
};

const handleMessage = (data) => {
  switch (data.type) {
    case "status":
      updateStatus(data.value);
      break;
      
    case "permission":
      updatePermission(data.key, data.value);
      break;
      
    case "log":
      addLog(data.message, data.level || "info");
      break;
      
    case "utterance":
      addUtterance(data.role, data.text);
      break;
      
    case "error":
      addLog(data.message, "error");
      break;
      
    case "system_status":
      updateSystemStatus(data.system, data.online);
      break;
    
    case "models_list":
      handleModelsList(data.models);
      break;
    
    case "model_info":
      handleModelInfo(data);
      break;
  }
};

// =============================================================================
// ASGARD SYSTEMS CHECK
// =============================================================================

const checkAsgardSystems = async () => {
  const systems = [
    { name: "pricilla", url: "http://localhost:8092/health" },
    { name: "nysus", url: "http://localhost:8080/health" },
    { name: "silenus", url: "http://localhost:9093/healthz" },
    { name: "hunoid", url: "http://localhost:8090/api/status" },
    { name: "security", url: "http://localhost:9090/health" },
  ];
  
  for (const system of systems) {
    try {
      const response = await fetch(system.url, { 
        method: "GET",
        mode: "no-cors",
        signal: AbortSignal.timeout(2000)
      });
      // With no-cors, we can't actually read the response
      // but if we get here without error, the server is likely up
      updateSystemStatus(system.name, true);
    } catch (error) {
      updateSystemStatus(system.name, false);
    }
  }
};

// Periodically check systems
setInterval(checkAsgardSystems, 30000);

// =============================================================================
// EVENT HANDLERS
// =============================================================================

// Microphone toggle
micToggle.addEventListener("click", async () => {
  if (!permissions.microphone) {
    const ok = await window.giruShell.confirm(
      "Allow Giru to access the microphone for wake word and speech recognition?"
    );
    if (!ok) return;
  }
  send({
    type: "permission",
    key: "microphone",
    value: !permissions.microphone,
  });
});

// Additional permission toggles
const permissionMessages = {
  camera: "Allow Giru to access the camera for vision tasks?",
  filesystem: "Allow Giru to access your files and desktop for organization?",
  email: "Allow Giru to send emails on your behalf?",
  network: "Allow Giru to access the internet and ASGARD systems?",
  git: "Allow Giru to run git commands in the workspace?",
  system: "Allow Giru to execute system commands?",
};

document.querySelectorAll("[data-permission-toggle]").forEach((button) => {
  const key = button.dataset.permissionToggle;
  button.addEventListener("click", async () => {
    if (!permissions[key]) {
      const message = permissionMessages[key] || `Allow ${key} access?`;
      const ok = await window.giruShell.confirm(message);
      if (!ok) return;
    }
    send({
      type: "permission",
      key,
      value: !permissions[key],
    });
  });
});

// Push-to-talk
pushToTalk.addEventListener("mousedown", () => {
  pushToTalkActive = true;
  pushToTalk.classList.add("active");
  send({ type: "push_to_talk", active: true });
});

pushToTalk.addEventListener("mouseup", () => {
  pushToTalkActive = false;
  pushToTalk.classList.remove("active");
  send({ type: "push_to_talk", active: false });
});

pushToTalk.addEventListener("mouseleave", () => {
  if (pushToTalkActive) {
    pushToTalkActive = false;
    pushToTalk.classList.remove("active");
    send({ type: "push_to_talk", active: false });
  }
});

// Interrupt button
interruptBtn.addEventListener("click", () => {
  send({ type: "interrupt" });
  addLog("Speech interrupt requested.");
});

// Clear chat
clearChat.addEventListener("click", clearConversation);

// Toggle text input (voice-first UI)
if (toggleText) {
  toggleText.addEventListener("click", () => {
    textVisible = !textVisible;
    textForm.classList.toggle("hidden", !textVisible);
  });
}

// Text input form
textForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const value = textInput.value.trim();
  if (!value) return;
  
  addUtterance("user", value);
  send({ type: "text", text: value });
  textInput.value = "";
});

// Keyboard shortcut for push-to-talk (Space key)
document.addEventListener("keydown", (event) => {
  if (event.code === "Space" && document.activeElement !== textInput) {
    event.preventDefault();
    if (!pushToTalkActive) {
      pushToTalkActive = true;
      pushToTalk.classList.add("active");
      send({ type: "push_to_talk", active: true });
    }
  }
});

document.addEventListener("keyup", (event) => {
  if (event.code === "Space" && pushToTalkActive) {
    pushToTalkActive = false;
    pushToTalk.classList.remove("active");
    send({ type: "push_to_talk", active: false });
  }
});

// =============================================================================
// MODEL MANAGEMENT
// =============================================================================

const modelModal = document.getElementById("model-modal");
const modelSelector = document.getElementById("model-selector");
const closeModal = document.getElementById("close-modal");

let availableModels = [];
let currentModel = null;

// Provider icons
const PROVIDER_ICONS = {
  google: "🌐",
  anthropic: "🧠",
  openai: "🤖",
  groq: "⚡",
  together: "🔗",
  openrouter: "🔀",
  ollama: "💻",
};

const renderModelSelector = () => {
  if (!availableModels.length) {
    modelSelector.innerHTML = '<div style="text-align: center; color: var(--text-muted);">No models available</div>';
    return;
  }
  
  modelSelector.innerHTML = availableModels.map(model => `
    <div class="model-option ${model.available ? '' : 'unavailable'} ${currentModel === model.key ? 'selected' : ''}" 
         data-model="${model.key}" ${model.available ? '' : 'title="API key not configured"'}>
      <div class="model-icon">${PROVIDER_ICONS[model.provider] || '🤖'}</div>
      <div class="model-details">
        <div class="model-name">${model.name}</div>
        <div class="model-provider">${model.provider}</div>
      </div>
      <div class="model-tags">
        ${model.free ? '<span class="model-tag free">FREE</span>' : ''}
        <span class="model-tag ${model.tier}">${model.tier}</span>
      </div>
    </div>
  `).join('');
  
  // Add click handlers
  modelSelector.querySelectorAll('.model-option:not(.unavailable)').forEach(el => {
    el.addEventListener('click', () => {
      const modelKey = el.dataset.model;
      send({ type: 'select_model', model: modelKey });
      modelModal.classList.add('hidden');
    });
  });
};

const showModelModal = () => {
  renderModelSelector();
  modelModal.classList.remove('hidden');
};

closeModal?.addEventListener('click', () => {
  modelModal.classList.add('hidden');
});

modelModal?.addEventListener('click', (e) => {
  if (e.target === modelModal) {
    modelModal.classList.add('hidden');
  }
});

// Add model info bar to wake hint
const addModelInfoBar = () => {
  const wakeHint = document.querySelector('.wake-hint');
  if (!wakeHint || document.getElementById('model-info-bar')) return;
  
  const modelBar = document.createElement('div');
  modelBar.id = 'model-info-bar';
  modelBar.className = 'model-info-bar';
  modelBar.innerHTML = `
    <span>🤖 AI Model:</span>
    <span id="current-model-name">Auto-Select</span>
    <span class="model-badge" id="model-tier">SMART</span>
  `;
  modelBar.addEventListener('click', showModelModal);
  wakeHint.after(modelBar);
};

// Handle model info updates
const handleModelInfo = (data) => {
  currentModel = data.model;
  const nameEl = document.getElementById('current-model-name');
  const tierEl = document.getElementById('model-tier');
  
  if (nameEl) nameEl.textContent = data.display_name || 'Auto-Select';
  if (tierEl) {
    tierEl.textContent = (data.tier || 'smart').toUpperCase();
    tierEl.className = `model-badge ${data.tier || ''}`;
  }
};

// Handle models list
const handleModelsList = (models) => {
  availableModels = models;
  addModelInfoBar();
};


// =============================================================================
// INITIALIZATION
// =============================================================================

// Welcome message
addLog("Giru JARVIS v2.0 initialized.");
addLog("Multi-model AI system ready.");
addLog('Say "Giru" to activate voice control.');

// Connect to backend
connect();

// Add model info bar after a short delay
setTimeout(addModelInfoBar, 500);

// Focus on text input for quick typing
textInput.focus();
