# Google Jules MCP Server

![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)

This repository provides a fully functional **Model Context Protocol (MCP)** server for interacting with the **Google Jules AI Agent**. 
Google Jules is a cloud-based autonomous agent capable of resolving GitHub issues and Pull Requests by analyzing the codebase, searching the web, and producing Pull Requests automatically.

As a core component of the **Antigravity Agent Ecosystem**, this MCP wrapper exposes Jules's capabilities as tools to other local AI agents (such as Antigravity), allowing them to natively delegate complex tasks to the cloud agent. This enables seamless integration and collaboration between local and cloud-based agents within the ecosystem.

## Features
This MCP server provides a comprehensive suite of tools (9 in total) to interact with Google Jules:
- **`jules_list_accounts`**: List all configured Google Jules / GitHub accounts and routing aliases.
- **`jules_list_sources`**: List all GitHub repositories connected to Google Jules.
- **`jules_delegate_task`**: Delegate an autonomous coding, refactoring, or bug-fixing task to Google Jules.
- **`jules_get_session`**: Retrieve status, generated outputs, and PR links for a session.
- **`jules_list_sessions`**: List recent active and completed Jules sessions.
- **`jules_list_activities`**: Get the chronological event timeline of a session (thoughts, plans, bash outputs, diffs).
- **`jules_approve_plan`**: Approve the generated execution plan for a paused session.
- **`jules_send_message`**: Send follow-up guidance or feedback to an ongoing session.
- **`jules_get_diff`**: Extract the unified git patch diff from a completed or active session.

## Prerequisites
- Python 3.10+
- A valid Google Jules API key (`JULES_API_KEY`) configured in your environment or MCP config.
- For multi-account setups, you can configure multiple keys (e.g. `JULES_API_KEY_DOCTORMES`, `JULES_API_KEY_NOVANODES`) which the router will automatically handle.

## Installation
```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -e .
```

## Running the Server
```bash
export JULES_API_KEY="your-api-key"
google-jules-mcp
```

## Architecture
- `src/server.py`: The MCP standard server implementation (stdio). Includes a synchronous CLI wrapper (`main`) that properly invokes the `asyncio` event loop to prevent `RuntimeWarning` crashes.
- `src/router.py`: Handles multi-tenant/multi-account routing. Allows seamlessly switching between multiple Google Jules instances based on the target repository owner.
- `src/client.py`: The async REST client for the Google Jules API.
- `src/booster.py`: Manages prompt injection and context boosting to maximize Jules's output quality.
