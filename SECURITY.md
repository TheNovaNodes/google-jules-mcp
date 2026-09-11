# Security Policy

TheNovaNodes Collective treats autonomous AI agent infrastructure security, credential protection, and operational integrity with the highest priority. **google-jules-mcp** bridges local AI agent environments with Google Jules cloud sandboxes.

---

## 🔒 Supported Versions

| Version | Supported          | Runtime    | Status             |
| ------- | ------------------ | ---------- | ------------------ |
| 1.0.x   | :white_check_mark: | Go 1.22+   | Production Current |

---

## 🛡️ Security Architecture & Threat Model

### 1. Zero-Leakage Credential Policy & Redaction
- **Header-Based Auth:** Upstream API communication transmits `JULES_API_KEY` strictly via the `X-Goog-Api-Key` HTTP header. The key is never appended to URLs or query parameters.
- **Payload Redaction:** All HTTP response bodies are sanitized through active string replacement (`[REDACTED_API_KEY]`) before being returned or cached in error structures.
- **Dynamic Secret Loading:** Credentials must be supplied via environment variables (`JULES_API_KEY`) or local untracked `.env` files. Hardcoding secrets in source code, tests, or scripts is strictly forbidden.

### 2. Stdio Transport & IPC Stream Isolation
- **`os.Stdout` Integrity:** The standard output stream is exclusively owned by the MCP JSON-RPC protocol transport (`mark3labs/mcp-go`).
- **Logging Redirection:** All diagnostic messages, traces, and warnings are strictly routed to `os.Stderr` using `log/slog`. No application code may write plain text to `stdout`, preventing protocol frame corruption.

### 3. Human Gate & Mutation Guardrails (HITL)
- **No Autonomous Watchers:** The server implements no background polling loops, recurring timers, or auto-approval workers.
- **Explicit Approval:** The `approve_jules_plan` tool is a deliberate, one-shot invocation that emits a structured `WARN` log (`RELEASING HUMAN-GATE`) to ensure human-in-the-loop oversight before code generation begins.
- **Remote Isolation:** Jules executes code changes inside ephemeral Google Cloud virtual machines. The local MCP server neither mounts host volumes nor touches local Git working trees.

### 4. Dynamic Branch Resolution & Injection Defense
- Starting branches are resolved dynamically against repository metadata (`defaultBranch.displayName`) or validated against user input.
- Literal branch fallbacks (e.g. hardcoded `main`) are prohibited to eliminate stale checkouts and unexpected branch target divergences.

---

## 🚨 Reporting a Vulnerability

If you discover a security vulnerability in `google-jules-mcp`:
1. **Do not** create a public GitHub issue.
2. Report the vulnerability privately to **ЗавЛаб** or via the secure Telegram operational contour.
3. Include reproduction steps, environment details, and relevant MCP client configurations.
