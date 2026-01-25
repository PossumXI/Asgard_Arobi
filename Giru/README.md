# Giru 2.0 - AI Defense System

<p align="center">
  <img src="../Assets/giru.png" alt="Giru - AI Defense System" width="200"/>
</p>

<p align="center">
  <em>Giru - The Guardian Intelligence of ASGARD</em>
</p>

## Overview
Giru is the immune system of ASGARD, providing continuous threat detection, network monitoring, and autonomous defense capabilities. It integrates advanced security features including shadow stack zero-day detection, red/blue team automated testing, and secure steganographic communication.

## Architecture
- **Network Scanner**: Real-time packet capture and log ingestion
- **Threat Detector**: Anomaly detection and threat classification
- **Mitigation Responder**: Automated response actions
- **Shadow Stack**: Parallel execution for zero-day detection
- **Red/Blue Team Agents**: Automated security testing
- **Gaga Chat**: Linguistic steganography for secure communication
- **Event Publisher**: NATS-based security event broadcasting

## Directory Structure
```
Giru/
├── README.md                    # This file
cmd/giru/
├── main.go                      # Main entry point
internal/security/
├── events/
│   ├── publisher.go             # NATS event publisher
│   └── schema.go                # Event type definitions
├── mitigation/
│   └── responder.go             # Mitigation action executor
├── scanner/
│   ├── interface.go             # Scanner interface
│   ├── analyzer.go              # Traffic analysis
│   ├── capture.go               # Packet capture utilities
│   ├── log_ingestion.go         # Log file scanning
│   └── realtime_scanner.go      # Real-time pcap scanner
├── threat/
│   └── detector.go              # Threat detection logic
├── shadow/
│   └── executor.go              # Shadow stack zero-day detection
├── redteam/
│   └── agent.go                 # Red team attack simulation
├── blueteam/
│   └── agent.go                 # Blue team defensive agent
└── gagachat/
    └── stego.go                 # Linguistic steganography
```

## Features Implemented

### Network Scanning
| Mode | Description |
|------|-------------|
| Real-time (pcap) | Live packet capture via Npcap/libpcap |
| Log Ingestion | Parse security logs (syslog, etc.) |
| API-only | Demo mode without scanning |

### Shadow Stack (NEW)
Zero-day detection through parallel execution monitoring:
- **Execution Tracking**: Monitor process behavior in isolation
- **Behavior Profiles**: Define expected behavior for known processes
- **Anomaly Detection**: Detect deviations from normal patterns
- **File Access Monitoring**: Track unauthorized file operations
- **Network Access Monitoring**: Detect suspicious network activity
- **Syscall Analysis**: Flag suspicious system calls

Anomaly Types Detected:
- Process injection
- Privilege escalation
- Suspicious syscalls
- Network exfiltration
- File integrity violations
- Behavioral deviations
- Memory corruption

### Red Team Agent (NEW)
Automated penetration testing with MITRE ATT&CK mapping:
- **Reconnaissance**: Port scanning, service detection
- **Exploitation**: Vulnerability checks (safe mode)
- **Persistence**: Check for persistence mechanisms
- **Lateral Movement**: Assess lateral movement risks
- **Exfiltration**: Test data egress controls
- **Privilege Escalation**: Check for privesc paths
- **Denial of Service**: Test rate limiting and resilience

### Blue Team Agent (NEW)
Automated defensive monitoring and response:
- **Rule-Based Detection**: Configurable detection rules
- **Auto-Response**: Automatic threat mitigation
- **IP Blocklisting**: Dynamic block management
- **Incident Response**: Containment, eradication, recovery
- **Threat Correlation**: Multi-source threat analysis

Built-in Detection Rules:
- Brute force attacks
- Port scanning
- SQL injection attempts
- XSS attempts
- Traffic anomalies

### Gaga Chat (NEW)
Linguistic steganography for covert communication:
- **Zero-Width Encoding**: Hide data in zero-width Unicode characters
- **Synonym Substitution**: Encode bits via word synonyms
- **Whitespace Patterns**: Use spacing patterns for encoding
- **Punctuation Encoding**: Hide data in punctuation
- **Hybrid Mode**: Combine multiple techniques
- **AES-256 Encryption**: Optional message encryption

### API Endpoints
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Service health check |
| `/api/threat-zones` | GET | Geographic threat zones for Pricilla |
| `/api/threats` | GET | Active threats list |
| `/api/scans` | POST | Initiate security scan |

## Build Status
**Phase: OPERATIONAL** (Full security suite including Shadow Stack, Red/Blue Team, Gaga Chat)

## Usage

### Run with All Features
```powershell
# Set encryption key for Gaga Chat
$env:GAGA_ENCRYPTION_KEY = "your-secure-key"

# Run in API-only mode (for demos)
go run ./cmd/giru/main.go -api-only

# Run with real-time scanning (requires admin + Npcap)
go run ./cmd/giru/main.go -interface "\Device\NPF_{YOUR-GUID}"
```

### Command-Line Flags
| Flag | Default | Description |
|------|---------|-------------|
| `-interface` | "" | Network interface to monitor |
| `-list-interfaces` | false | List available interfaces |
| `-api-only` | false | Run API server only |
| `-nats` | nats://localhost:4222 | NATS server URL |
| `-metrics-addr` | :9091 | Metrics server address |
| `-api-addr` | :9090 | API server address |

### Environment Variables
| Variable | Description |
|----------|-------------|
| `SECURITY_SCANNER_MODE` | Scanner mode: pcap, log |
| `SECURITY_LOG_SOURCES` | Log sources (path:type) |
| `GAGA_ENCRYPTION_KEY` | Encryption key for Gaga Chat |

## Dependencies
- Go 1.21+
- gopacket (packet capture library)
- Npcap (Windows) or libpcap (Linux/macOS)
- NATS JetStream (optional)

## Integration Points
- **Pricilla**: Provides threat zones via `/api/threat-zones`
- **Nysus**: Security events forwarded to control plane
- **Silenus**: Can receive satellite-detected threats
- **Hunoid**: Provides hazard data for mission planning
- **Shadow Stack → Blue Team**: Anomalies trigger blue team responses

## Security Notes
- Red Team Agent runs in **SAFE MODE** by default (no actual exploitation)
- Shadow Stack monitors but does not interfere with process execution
- Gaga Chat encryption uses AES-256-GCM
- All security events are logged and auditable
