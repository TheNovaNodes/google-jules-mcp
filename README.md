# Google Jules MCP Server

![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)

This repository provides a fully functional **Model Context Protocol (MCP)** server for interacting with the **Google Jules AI Agent**. 
Google Jules is a cloud-based autonomous agent capable of resolving GitHub issues and Pull Requests by analyzing the codebase, searching the web, and producing Pull Requests automatically.

As a core component of the **Antigravity Agent Ecosystem**, this MCP wrapper exposes Jules's capabilities as tools to other local AI agents (such as Antigravity), allowing them to natively delegate complex tasks to the cloud agent. This enables seamless integration and collaboration between local and cloud-based agents within the ecosystem.

## Features
- **MCP Tool: `list_jules_sources`**: Retrieve a list of authorized repositories that Jules can interact with.
- **MCP Tool: `delegate_task_to_jules`**: Delegate an architectural or coding task to Jules directly. The local agent will receive a session URL to track Jules's progress.

## Prerequisites
- Python 3.10+
- A valid Google Jules API key (`JULES_API_KEY`) configured in your environment or MCP config.

## Installation
```bash
python -m venv .venv
source .venv/bin/activate
pip install -e .
```

## Running the Server
```bash
export JULES_API_KEY="your-api-key"
google-jules-mcp
```

## Architecture
- `src/server.py`: The MCP standard server implementation (stdio).
- `src/client.py`: The async REST client for the Google Jules `v1alpha` API.
