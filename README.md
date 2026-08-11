# Google Jules MCP Server

![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
![Python Version](https://img.shields.io/badge/Python-3.10%2B-blue)
![MCP Version](https://img.shields.io/badge/MCP-1.x-brightgreen)

This repository provides a fully functional **Model Context Protocol (MCP)** server for interacting with the **Google Jules AI Agent**. 

Google Jules is a cloud-based autonomous agent capable of resolving GitHub issues and Pull Requests by analyzing the codebase, searching the web, and producing Pull Requests automatically. As a core component of the **Antigravity Agent Ecosystem**, this MCP wrapper exposes Jules's capabilities as tools to other local AI agents (such as Antigravity), allowing them to natively delegate complex tasks to the cloud agent.

## 📚 Documentation

Detailed technical documentation for developers and contributors can be found in the `/docs` directory:

- 🏛️ **[Architecture & Data Flow](docs/architecture.md)**: Visual diagrams and internal logic of the MCP server and API client.
- 🔌 **[API Reference](docs/api-reference.md)**: Detailed specifications for exposed MCP tools and REST client methods.
- 🚀 **[Deployment Guide](docs/deployment.md)**: Step-by-step instructions for running the server and configuring environment variables.

## Quick Start

```bash
# 1. Setup Environment
python -m venv .venv
source .venv/bin/activate
pip install -e .

# 2. Configure API Key
export JULES_API_KEY="your-api-key"

# 3. Run the Server
google-jules-mcp
```
