# AI Agent Rules for Google Jules MCP

Welcome, Agent! 
This repository contains a Model Context Protocol (MCP) server for Google Jules.

## Overview
- **Goal:** Provide a seamless stdio MCP interface to the Google Jules REST API (`v1alpha`).
- **Tools Provided:** `list_jules_sources` and `delegate_task_to_jules`.

## Rules for Modification
1. **API Keys:** Never hardcode the `JULES_API_KEY` in the source code. It must always be read from the environment variables.
2. **Async Programming:** The client (`src/client.py`) uses `aiohttp`. Make sure to manage `ClientSession` correctly and handle standard HTTP errors.
3. **MCP Standard:** The server uses the `mcp.server.fastmcp` module. Any new tools should be decorated with `@mcp.tool()`.

## Task Delegation Guidelines
When using this MCP server from a parent agent:
- Always use `list_jules_sources` first to find the exact formatted string for the target GitHub repository (e.g., `sources/github/username/repo`).
- When calling `delegate_task_to_jules`, ensure the prompt contains clear and comprehensive instructions. Jules acts autonomously and relies entirely on your prompt.
