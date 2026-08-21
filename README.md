---
module_type: mcp-server
status: active
protocol: stdio
primary_capability: Task delegation to Google Jules
requires: JULES_API_KEY
works_with: GitHub repositories, Google Jules
last_verified: 2026-08-21
---

# Google Jules MCP Server
*Delegate complex, long-running coding tasks from local agents to the cloud-based Google Jules AI agent.*

## Status and last verified date
Status: Active
Last verified: 2026-08-21

## What it does / does not do
**What it does:**
- Wraps the Google Jules v1alpha REST API into an MCP (Model Context Protocol) server.
- Allows listing connected GitHub repositories (sources).
- Allows delegating coding/refactoring tasks to Jules by creating new remote sessions.

**What it does not do:**
- Does not run Jules locally (Jules runs on Google Cloud VMs).
- Does not automatically approve or merge Jules's Pull Requests.
- Does not monitor the real-time stream of the session (it creates the session and returns).

## Why an agent would use it
Local agents (like Antigravity) use this MCP server to offload significant refactoring, large-scale testing, or complex feature implementation across multiple files. This prevents the local chat and context window from blocking, letting Jules handle heavy lifting in the cloud and submit a Pull Request upon completion.

## Architecture and dependencies
- **Architecture**: A FastMCP-based server communicating via standard input/output (stdio). It uses lazy client initialization to load environment variables at request time, ensuring fresh instances and proper resource cleanup.
- **Dependencies**: 
  - `mcp` (>=1.2.0, <2.0.0)
  - `aiohttp` (>=3.0.0)
  - `pydantic` (>=2.0.0)
  - `python-dotenv` (>=1.0.0)
  - `tenacity` (>=8.0.0)

## Compatibility
- Requires Python 3.10 or higher.
- Implements MCP (Model Context Protocol) stdio transport.

## Quick start and health check
**Quick Start:**
```bash
python -m venv .venv
source .venv/bin/activate
pip install -e .
cp .env.example .env # Add your JULES_API_KEY
python src/main.py
```
**Health Check:** Call the `check_jules_status` MCP tool.

## Configuration and environment variables
- `JULES_API_KEY`: The API key for accessing Google Jules. (No default, required).

## Complete MCP tool/API table with side effects
| Tool | Description | Side Effects |
|------|-------------|--------------|
| `list_jules_sources` | List connected GitHub repositories | None (Read-only) |
| `delegate_task_to_jules` | Start a remote session on a source | **Creates remote session** |
| `check_jules_status` | Get status of a Jules session | None (Read-only) |

## Security model and trust boundaries
- **Authentication**: Requires a valid `JULES_API_KEY`.
- **Trust Boundaries**: The server executes locally but interacts directly with Google APIs. It should run in a trusted environment to prevent key leakage.

## Tests and exact commands
```bash
pytest tests/
```

## Operations, logs, backup/restore, rollback
- **Logs**: Handled via standard MCP logging or standard error stream.
- **Backup**: Stateless. Back up the API key elsewhere.

## Generic MCP-client example
```json
{
  "mcpServers": {
    "google-jules": {
      "command": "python",
      "args": ["-m", "google_jules_mcp.server"],
      "env": {
        "JULES_API_KEY": "YOUR_KEY"
      }
    }
  }
}
```

## Limitations and roadmap
- Only creates sessions; streaming logs is unsupported.

## Related TheNovaNodes modules
- antigravity-telegram-agent

## License
MIT
