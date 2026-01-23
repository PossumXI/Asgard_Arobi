# Nysus - Central Nerve Center

## Overview
Nysus is the orchestration hub of ASGARD, coordinating all subsystems through Model Context Protocol (MCP).

## Architecture
- **Context Aggregator**: Pulls data streams from Silenus and Hunoid
- **MCP Server**: Exposes system capabilities as LLM-accessible tools
- **Command Dispatcher**: Routes commands via Control_net

## Directory Structure
```
Nysus/
├── cmd/                 # Main Nysus service
├── internal/
│   ├── mcp/            # Model Context Protocol server
│   ├── aggregator/     # Multi-source data fusion
│   └── dispatcher/     # Command routing
└── agents/             # Specialized AI agents
```

## Build Status
Phase 2 - In progress (MCP server implementation)

## Dependencies
- Go 1.21+
- NATS JetStream
- LLM integration (OpenAI/Claude compatible)
