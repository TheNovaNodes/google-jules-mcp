# Deployment Guide

This document provides a step-by-step guide to deploying and running the Google Jules MCP Server.

## Environment Variables

The server requires specific environment variables to function correctly. You can provide these by exporting them in your terminal or using a `.env` file.

| Variable Name | Required | Description |
| :--- | :--- | :--- |
| `JULES_API_KEY` | **Yes** | Your valid Google Jules API key. Required for authenticating requests to `https://jules.googleapis.com`. |

## Local Development Deployment

Follow these instructions to deploy the MCP server locally for development or integration with a local AI agent.

1. **Clone the repository and navigate to the root directory.**
2. **Create a virtual environment (recommended):**
   ```bash
   python -m venv .venv
   source .venv/bin/activate
   ```
3. **Install the package in editable mode:**
   ```bash
   pip install -e .
   ```
4. **Export your API Key:**
   ```bash
   export JULES_API_KEY="your-api-key-here"
   ```
5. **Start the MCP Server:**
   ```bash
   google-jules-mcp
   ```
   *Note: The server communicates via standard I/O (`stdio`). You will not see typical web server logs unless an error occurs. Local AI agents configured to use this server will invoke the `google-jules-mcp` command directly.*

## Docker Deployment

### TODO
Docker containerization is not yet fully implemented in the codebase. When implemented, it should include:

- A `Dockerfile` defining a minimal Python runtime (e.g., `python:3.10-slim`).
- Configuration for copying source files and installing dependencies (`pip install .`).
- Safe injection of the `JULES_API_KEY` environment variable at runtime.
- Proper handling of I/O streams within the Docker container to ensure MCP communication works seamlessly (e.g., using `docker run -i` to keep stdin open).

**Proposed Docker Run Command (Future):**
```bash
docker run -i --rm -e JULES_API_KEY="your-api-key-here" google-jules-mcp:latest
```
