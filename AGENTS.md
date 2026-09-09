# 🛸 AGENTS.md — Directives for Autonomous & Pair-Programming AI Agents

> **Context:** This document is the primary machine-readable operational manifest for all AI coding agents (Google Jules, Antigravity, Cursor, Claude Code, Devin, and NovaNodes collective bots) operating within this repository.

---

## 🏛️ PART 1: NOVANODES CORE INVARIANTS (Universal Standard)

### 1. Strict Git Flow (ПРАВИЛА КРОВИ)
- 🛑 **NEVER push directly to `main` or `master` branches.**
- All changes (features, bugfixes, refactorings, documentation) **MUST** go through dedicated branch naming (`feat/...`, `fix/...`, `refactor/...`, `docs/...`) and a formal Pull Request (PR).
- **NEVER merge PRs without explicit approval from ЗавЛаб.**
- **No Force-Push:** Never rewrite public commit history (`git push -f`) on upstream branches.

### 2. Security & Credential Hygiene
- 🛑 **Zero-Leakage Policy:** NEVER hardcode API keys, tokens, passwords, or credentials into source code, unit tests, commit messages, or PR bodies.
- All secrets must be loaded dynamically from environment variables (e.g., `os.Getenv("JULES_API_KEY")`).
- Never echo or log secret values in command stdout/stderr.

### 3. Deadlock, Timeout & Process Safety
- **Network Timeouts:** Every outbound network call or HTTP client must configure explicit timeouts (default: 60s total, connection pooling with 90s idle timeout).
- **Command Timeouts:** When running auxiliary shell commands, always enforce hard timeouts (`timeout -k 2s 15s ...`) to prevent zombie processes.
- **Stdio Redirection:** When spawning background processes or smoke-testing MCP stdio servers, always redirect stdin (`< /dev/null`) to avoid terminal `SIGTTIN` hangs.
- **No Infinite Polling:** Never loop endlessly (`while true: sleep(1)`) waiting for external events. Use scheduled checks or event-driven wakeups.

### 4. Continuous Verification (No Blind Reports)
- **Pre-PR Native Minimum:** Never claim a task is complete without running the project's native test suite and linters locally.
- **Verification without Absurdity:** Use native project tools directly (`go test`, `go vet`, `make`); do not pull heavy emulator containers.
- **Post-PR Cloud CI:** After creating a PR, monitor the actual GitHub Actions CI (`gh pr checks <id>`). Do not report success until all cloud checks are green.

---

## 🔬 PART 2: REPOSITORY PROFILE & SPECIFIC DIRECTIVES

### 1. Project Overview & Tech Stack
- **Repository:** `TheNovaNodes/google-jules-mcp`
- **Primary Language:** Go 1.22+ (compatible with Go 1.22, 1.23, 1.24, 1.25)
- **Key Frameworks / Libraries:**
  - `github.com/mark3labs/mcp-go` (Model Context Protocol Go SDK)
  - `net/http` standard library for REST API client
- **Entry Points:** `cmd/google-jules-mcp/main.go`
- **Artifact:** Single static binary compiled to `bin/google-jules-mcp`

### 2. The Golden Loop (Mandatory Verification Commands)
*Every agent must execute and pass these commands before opening a Pull Request:*

```bash
# 1. Static Analysis & Vet
go vet ./...

# 2. Complete Test Suite with Race Detector & Coverage (100% PASS required)
go test -v -race -cover ./...

# 3. Clean Binary Build
make build
# Binary is produced at ./bin/google-jules-mcp
```

### 3. Architectural Invariants & Taboos (JMC-TUNE-1 Spec)
- **R1 (Branch Resolution):** It is **FORBIDDEN** to hardcode `"main"` or any literal fallback branch in client payloads. Always use `client.ResolveStartingBranch()` which auto-detects `defaultBranch.displayName` from `GET /v1alpha/sources` or omits the field.
- **R2 (Lifecycle Tools):** All 7 tools (`list_jules_sources`, `delegate_task_to_jules`, `check_jules_status`, `get_jules_session`, `list_jules_activities`, `send_jules_message`, `approve_jules_plan`) must remain registered and functional.
- **R3 (Response Hygiene):** Never dump raw JSON dictionaries to calling agents. Always use `formatter.Format*` functions: prompt echoes $\le 200$ chars, unidiff patches $\le 300$ chars, activities capped at $\sim 4$ KB.
- **R4 (Error Taxonomy):** Map upstream Google HTTP errors to human-actionable messages (400 pass-through, 401/403 invalid key, 404 not found, 429 backoff notice with attempts, 5xx unavailable with attempts).
- **R5 (Guardrails):** No background auto-approvers or watchers. `approve_jules_plan` must log a mandatory WARNING (`RELEASING HUMAN-GATE`).
- **R6 (Docs-to-Registry Contract):** Whenever tools are modified, `README.md` must be updated and `TestReadmeMatchesToolRegistry` must pass.
- **Stdio Protocol Hygiene:** In MCP stdio servers, `os.Stdout` is strictly reserved for JSON-RPC messages. **ALL logs must go exclusively to `os.Stderr` (`slog.NewTextHandler(os.Stderr, ...)`).** Any output to `os.Stdout` breaks JSON-RPC parsing.

### 4. PR & Commit Conventions
- **Commit Format:** Conventional Commits (`feat(...)`, `fix(...)`, `refactor(...)`, `docs(...)`, `test(...)`).
- **PR Description:** Must detail: (1) What changed, (2) JMC-TUNE-1 requirement addressed, (3) Verification output (`go test -race`).
