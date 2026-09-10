---
module_type: mcp-server
status: active
protocol: stdio
primary_capability: Task delegation to Google Jules
requires: JULES_API_KEY
works_with: GitHub repositories, Google Jules
last_verified: 2026-09-08
---

# Google Jules MCP Server (Go)
*Delegate complex, long-running coding tasks from local agents to the cloud-based Google Jules AI agent.*

📚 **Documentation Suite:** [Architecture](ARCHITECTURE.md) • [Agent Directives](AGENTS.md) • [Contributing](CONTRIBUTING.md) • [Specification](docs/specs/jmc-tune-1-server-tuning.md) • [License](LICENSE)

## Status and last verified date
Status: Active  
Language: Go (Golang)  
Protocol: Model Context Protocol (stdio transport)  
Last verified: 2026-09-08  

## What it does / does not do
**What it does:**
- Exposes Google Jules v1alpha REST API as a high-performance Model Context Protocol (stdio) server.
- Lists connected GitHub repositories with their resolved default branches.
- Autonomously delegates tasks to Jules, creating remote sessions in Google Cloud VMs.
- Provides complete lifecycle observability: status checks, activity logs with unidiff patch previews, interactive message replies, and human plan approvals.
- Implements smart branch resolution (R1): resolves default branch automatically from source metadata without hardcoded fallbacks.
- Compact response formatting (R3): protects LLM context windows by truncating patch previews (<=300 chars) and prompt echoes (<=200 chars).

**What it does not do:**
- Does not run Jules locally (Jules runs inside Google Cloud VMs).
- Does not automatically approve or merge Jules's Pull Requests on GitHub.
- Does not maintain unmonitored background watcher loops (one-shot actions only).

## Quick start
```bash
# 1. Clone repository
git clone https://github.com/TheNovaNodes/google-jules-mcp.git
cd google-jules-mcp

# 2. Configure environment
cp .env.example .env
# Edit .env and set your JULES_API_KEY

# 3. Build binary
make build
# or: go build -o bin/google-jules-mcp ./cmd/google-jules-mcp

# 4. Run server (stdio)
./bin/google-jules-mcp
```

## Configuration and environment variables
- `JULES_API_KEY`: The API key for accessing Google Jules REST API (Required).
- `JULES_DISABLE_RETRY`: Set to `1` or `true` to disable exponential backoff retries (Optional, for deterministic testing).

## Complete MCP tool/API table with side effects
| Tool | Description | Side Effects |
|------|-------------|--------------|
| `list_jules_sources` | List all GitHub repositories and default branches connected to Jules | None (Read-only) |
| `delegate_task_to_jules` | Delegate a long-running coding/refactoring mission to Jules | **Creates remote cloud session** |
| `check_jules_status` | Check the current execution state and status of a Jules session | None (Read-only) |
| `get_jules_session` | Retrieve comprehensive details and metadata of a Jules session | None (Read-only) |
| `list_jules_activities` | List execution activity events, progress steps, and code patch previews | None (Read-only) |
| `send_jules_message` | Send an interactive message or instructions to a Jules session | **Sends message to remote session** |
| `approve_jules_plan` | Approve a proposed execution plan for a session paused in plan approval | **Releases human gate; starts execution** |
| `get_jules_patch` | Extract the latest Git unidiff patch produced by Jules for a session | None (Read-only) |

## Security model and trust boundaries
- **Authentication**: All upstream requests require a valid `JULES_API_KEY` sent via the `X-Goog-Api-Key` header.
- **Guardrails (R5)**: The server implements no background auto-approvers or watchers. `approve_jules_plan` is an explicit, one-shot action that logs a mandatory warning.
- **Git Safety**: The server never pushes directly, merges, or touches branches outside of what the Jules API executes remotely.

## Generic MCP-client example
Add to your Claude Desktop, Antigravity, Cursor, or OpenClaw MCP configuration:
```json
{
  "mcpServers": {
    "google-jules": {
      "command": "/absolute/path/to/google-jules-mcp/bin/google-jules-mcp",
      "args": [],
      "env": {
        "JULES_API_KEY": "YOUR_JULES_API_KEY"
      }
    }
  }
}
```

## Testing
```bash
make test
# or: go test -v -race ./...
```

## License
MIT
