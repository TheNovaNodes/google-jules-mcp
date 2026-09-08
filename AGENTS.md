# AI Agent Directives for Google Jules MCP (Go)

Welcome, Agent. This repository contains the Model Context Protocol (MCP) server for Google Jules written in Go (Golang).

## Architectural Guidelines
1. **Language & Architecture:** Pure Go. Uses `github.com/mark3labs/mcp-go` stdio transport. Single static binary output.
2. **API Keys:** Never hardcode or log `JULES_API_KEY`. Read exclusively from environment / `.env`.
3. **Branch Resolution (R1):** Never hardcode `"main"` or any literal branch. Always use `ResolveStartingBranch` (explicit -> source default branch -> omit field).
4. **Lifecycle Tools (R2):** All 7 tools (`list_jules_sources`, `delegate_task_to_jules`, `check_jules_status`, `get_jules_session`, `list_jules_activities`, `send_jules_message`, `approve_jules_plan`) must remain in sync with `README.md`.
5. **Docs Sync Contract (R6):** Adding or removing tools requires updating `README.md` and keeping `TestReadmeMatchesToolRegistry` passing.
6. **Testing:** All new features must include unit tests with `httptest.Server` and pass `go test -race ./...`.
