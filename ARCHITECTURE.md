# 🏛️ Architecture & System Design — Google Jules MCP Server

[![Protocol: Model Context Protocol](https://img.shields.io/badge/protocol-MCP%20JSON--RPC-green.svg)](https://modelcontextprotocol.io/)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg)](https://golang.org)
[![Specification](https://img.shields.io/badge/spec-JMC--TUNE--1-orange.svg)](docs/specs/jmc-tune-1-server-tuning.md)

This document details the architectural design, component topology, lifecycle workflows, and security guardrails of **`google-jules-mcp`** — the high-performance Go MCP server bridging autonomous AI agents with Google Jules cloud infrastructure.

---

## 1. System Topology & Context

The server operates as a stateless, lightweight bridge using the standard Model Context Protocol (stdio transport). It multiplexes control and observability between local AI agents (such as Antigravity, Tyler Durden, or MCP Router) and Google Jules cloud virtual machines executing in Google Cloud Platform (GCP).

```mermaid
flowchart TD
    subgraph AgentClient["Local AI Agent Tier"]
        CLI["Antigravity CLI (agy)"]
        BOT["Tyler / Collective Bots"]
        ROUTER["MCP Router Gateway"]
    end

    subgraph StdioInterface["Stdio Transport Boundary"]
        STDIN["Stdin (JSON-RPC Requests)"]
        STDOUT["Stdout (JSON-RPC Responses)"]
        STDERR["Stderr (slog Diagnostics / Warnings)"]
    end

    subgraph GoServer["google-jules-mcp Core"]
        SRV["MCP Server Layer (internal/server)"]
        FMT["Response Formatter & Truncator (internal/formatter)"]
        CLIENT["Jules REST API Client (internal/jules)"]
        BRANCH["Dynamic Branch Resolver (internal/jules/branch.go)"]
    end

    subgraph GoogleCloud["Google Cloud Infrastructure"]
        API["Google Jules REST API v1alpha"]
        VM["Cloud Sandbox VM"]
        GIT["Target GitHub Repository (main/master)"]
    end

    AgentClient -->|Calls Tool| STDIN
    STDIN --> SRV
    SRV -->|JSON-RPC Output| STDOUT
    SRV -.->|Structured Logs Only| STDERR

    SRV --> FMT
    SRV --> CLIENT
    CLIENT --> BRANCH
    BRANCH -->|GET /v1alpha/sources| API
    CLIENT -->|HTTPS / X-Goog-Api-Key| API
    API -->|Spawns Sandbox| VM
    VM -->|Proposes Pull Request| GIT
```

---

## 2. Package Decomposition

The codebase is engineered strictly in Go (Go 1.22+ compatible) with zero external CGO dependencies and clean separation of concerns:

| Package | Path | Responsibility |
| :--- | :--- | :--- |
| `main` | `cmd/google-jules-mcp` | Application entry point. Loads `.env`, configures `slog` to `os.Stderr`, initializes dependencies, and binds to `mcpserver.ServeStdio`. |
| `server` | `internal/server` | MCP tool handlers and lifecycle registry. Implements 7 canonical tools, argument validation, and contract tests against `README.md`. |
| `jules` | `internal/jules` | Resilient HTTP REST client for Google Jules API (`https://jules.googleapis.com/v1alpha`). Implements exponential backoff, retry jitter, typed data models, and branch resolution. |
| `formatter` | `internal/formatter` | Context-window hygiene layer. Formats structured Markdown, truncates diffs ($\le 300$ chars) and prompt echoes ($\le 200$ chars), and prevents LLM context exhaustion. |

---

## 3. Tool Lifecycle & Sequence Flow

Google Jules tasks run asynchronously over extended durations (5–15 minutes). The server models execution through a complete 7-tool lifecycle:

```mermaid
sequenceDiagram
    autonumber
    actor Agent as Local AI Agent
    participant MCP as google-jules-mcp
    participant API as Google Jules API (v1alpha)
    participant Cloud as Cloud VM / GitHub

    Note over Agent,MCP: Phase 1: Source Discovery & Branch Resolution
    Agent->>MCP: callTool("list_jules_sources")
    MCP->>API: GET /v1alpha/sources
    API-->>MCP: Sources list with defaultBranch.displayName
    MCP-->>Agent: Formatted source inventory

    Note over Agent,MCP: Phase 2: Autonomous Delegation
    Agent->>MCP: callTool("delegate_task_to_jules", {source, prompt, ...})
    MCP->>MCP: ResolveStartingBranch (R1 invariant)
    MCP->>API: POST /v1alpha/sessions
    API-->>MCP: Session ID & URL
    MCP-->>Agent: Session Handle (e.g. sessions/12345)

    Note over Agent,MCP: Phase 3: Observability & Interactive Gates
    loop Poll Progress & Activities
        Agent->>MCP: callTool("check_jules_status", {session_id})
        MCP->>API: GET /v1alpha/sessions/{id}
        API-->>MCP: State (QUEUED | IN_PROGRESS | COMPLETED | FAILED)
        MCP-->>Agent: Status Summary
    end

    opt Session Requires Plan Approval
        Agent->>MCP: callTool("list_jules_activities", {session_id})
        MCP->>API: GET /v1alpha/sessions/{id}/activities
        API-->>MCP: Activities + Plan Details
        MCP-->>Agent: Formatted Activities & Diff previews
        Agent->>MCP: callTool("approve_jules_plan", {session_id})
        MCP-->>API: POST /v1alpha/sessions/{id}:approvePlan
        Note over MCP: Logs: WARN RELEASING HUMAN-GATE
        API-->>MCP: HTTP 200 OK
        MCP-->>Agent: Plan Approved (Execution Resumed)
    end

    API->>Cloud: Spawns Cloud Sandbox, generates code patch, opens PR
```

---

## 4. Architectural Invariants (JMC-TUNE-1 Standards)

Every modification to this codebase must conform to the formal **JMC-TUNE-1** specification:

1. **R1: Dynamic Branch Resolution:**
   - Literal fallbacks like `"main"` are strictly prohibited.
   - If a repository's default branch is `master`, `ResolveStartingBranch` resolves and passes `"master"`.
   - If the repository is unknown or has no default branch metadata, the `startingBranch` parameter is omitted to let the Google API handle its internal defaults.
2. **R2: Complete Lifecycle Surface:**
   - Exactly 7 tools (`list_jules_sources`, `delegate_task_to_jules`, `check_jules_status`, `get_jules_session`, `list_jules_activities`, `send_jules_message`, `approve_jules_plan`) are maintained.
3. **R3: Context Window Protection:**
   - Unstructured raw dictionary dumps are forbidden.
   - Patch previews are strictly capped at $\le 300$ characters.
   - Echoed prompts are truncated at $\le 200$ characters.
   - Activity streams are capped at $\sim 4$ KB maximum payload.
4. **R4: Human-Actionable Error Taxonomy:**
   - Upstream HTTP error codes are translated into descriptive, diagnostic messages:
     - `400`: Upstream parameter error.
     - `401 / 403`: Invalid or revoked `JULES_API_KEY`.
     - `404`: Session or source not found.
     - `429`: Rate limit encountered, reporting retry attempts.
     - `5xx`: Google Cloud outage or internal server error.
5. **R5: Guardrails on Mutation & Human Gates:**
   - Background auto-watchers and unmonitored polling loops are forbidden.
   - The `approve_jules_plan` action triggers an explicit structured log:  
     `s.logger.WarnContext(ctx, "RELEASING HUMAN-GATE: approve_jules_plan invoked", ...)`
6. **R6: Automated Contract Synchronization:**
   - `TestReadmeMatchesToolRegistry` asserts in CI that `README.md` exactly reflects the registered MCP tool set.

---

## 5. Stdio Transport Hygiene

MCP stdio servers exchange JSON-RPC packets across standard operating system descriptors. A single spurious byte written to `stdout` corrupts the client's JSON framing parser:

- **`os.Stdout`:** Exclusively owned by `github.com/mark3labs/mcp-go/server`. Absolutely no application code may call `fmt.Println`, `os.Stdout.Write`, or standard `log` without redirection.
- **`os.Stderr`:** Exclusively utilized for all diagnostic output, metrics, and structured logging using `log/slog.NewTextHandler(os.Stderr, ...)`.
- **`os.Stdin`:** Exclusively read by the JSON-RPC message loop.

---

## 6. Resilience & Backoff Topology

Network interactions with Google Cloud implement an exponential backoff strategy with jitter:

$$\Delta t = \min(t_{\text{max}}, t_{\text{base}} \times 2^{\text{attempt}}) \pm \text{jitter}$$

- **Base Delay:** 500 ms
- **Max Delay:** 8,000 ms
- **Max Attempts:** 4
- **Deterministic Testing Override:** `JULES_DISABLE_RETRY=true` disables retry loops to prevent artificial latency in test suites.
