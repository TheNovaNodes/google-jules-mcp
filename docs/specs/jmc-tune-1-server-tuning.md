# JMC-TUNE-1: `google-jules-mcp` Server Tuning Specification

- **Status:** DRAFT — ready for implementation by a separate coding agent (not authored by the implementer, per lab conveyor).
- **Commissioned by:** ЗавЛаб (@DoctorMES), 2026-08-25.
- **Repo:** `TheNovaNodes/google-jules-mcp`.
- **Quality bar (reference):** Google Stitch MCP (cloud, `streamable-http`) — treated by the lab as "works near-ideally".
- **Language note:** code identifiers in English; this document is the single source of truth for the tuning scope.

---

## 0. Summary

The MCP server wraps the Google Jules REST API (v1alpha) so local agents can delegate long-running
coding missions to Jules. Production use on 2026-08-25 exposed four defects/gaps: a hardcoded
`startingBranch="main"` that breaks every repository whose default branch differs (INC-1), zero
lifecycle observability (no way to see a stalled session or answer its questions — INC-2),
documentation drift (README advertises a tool that is not registered — INC-3), and raw-dict
responses that force callers to parse unstructured text (INC-4). This spec defines the minimal,
backwards-compatible upgrade that closes all four classes and matches the operational ergonomics
of the Stitch MCP reference.

---

## 1. Operating context (how the lab actually uses this server)

- Registered in an OpenClaw gateway as two stdio MCP instances
  (`Google-Jules-AI-Agent-{Doctormes,TheNovaNodes}`), sharing `JULES_API_KEY`.
- Used as an independent **validation → adversarial audit → review gate** in a research-conveyor
  workflow on private repositories (example: `thedoctormes-hue/doctorm-unify-protocol`,
  default branch `master`).
- Missions are expected to run **autonomously end-to-end**: Jules publishes her own artifacts
  (audit reports, PRs). Human approval is reserved for merges only.
- Delegation happens from an LLM agent with no human watching; anything that requires an
  interactive answer from a human becomes a silent blocker. This is the core product constraint:
  **the server must make autonomous completion the path of least resistance.**

---

## 2. Incident log (evidence base; all 2026-08-25 UTC)

| ID | What happened | Root cause | Consequence |
|----|---------------|-----------|-------------|
| INC-1 | Session `1161450207223342781` FAILED at 05:02 on `git clone` | `client.create_session()` hardcodes `starting_branch="main"`; target repo default is `master` | Mission lost; had to be retried via direct REST with the correct branch |
| INC-2 | Re-audit session `12444597626949275570` sat in `AWAITING_USER_FEEDBACK` from 06:49, asking "can you confirm I should proceed…" | No tool existed to observe session state or reply; the mission text did not forbid clarifying questions | Silent stall; unblocked manually at ~07:16 via `POST /sessions/{id}:sendMessage` |
| INC-3 | README documents `check_jules_status`; `src/server.py` registers only `list_jules_sources` and `delegate_task_to_jules` | Docs/tool registry drift | Callers trusting README fail at tool-discovery time |
| INC-4 | `delegate_task_to_jules` returns a raw Python dict dump (including the full prompt echo) | No response shaping | Agents regex-parse unstructured text to recover `session id` / `url`; brittle |

---

## 3. Reference model: what makes Stitch MCP the bar

Properties observed in lab production use (verified 2026-08-24/25):

1. **Full lifecycle surface** (~15 tools): submit a job, poll its status, retrieve finished
   artifacts, manage variants — every state a job can be in has a corresponding tool.
2. **Long-operation pattern:** submission returns a handle; recovery from a timeout is
   *retrieve-by-handle*, never blind resubmission (idempotent observation).
3. **Explicit, boring auth:** a single `X-Goog-Api-Key` header; no secrets inside payloads.
4. **Stable transport:** one streamable-http endpoint; behavior identical across clients.
5. **Docs == behavior:** advertised tool set matches the registry exactly.

**Mapping to this repo:** we stay on stdio transport (local gateway requirement; switching is out
of scope) but adopt properties 1, 2, 3 and 5 verbatim for the Jules domain.

---

## 4. Current state (code facts, audited 2026-08-25 @ `a3248ea`)

- `src/server.py`: FastMCP app `"google-jules-mcp"`; lazy `JulesClient` init; registered tools:
  `list_jules_sources()`, `delegate_task_to_jules(source_name, prompt)`. Nothing else.
- `src/client.py`: `JulesClient` with `BASE_URL=https://jules.googleapis.com/v1alpha`;
  `list_sources()`; `create_session(source_name, prompt, starting_branch="main")` ← defect;
  tenacity retry (3×, jittered) on 429/5xx/connection errors; lowercase `x-goog-api-key` header.
- Existing test suite under `tests/` must be **extended**, not replaced.
- Repo conventions: ruff, type-hinted Python ≥3.10, MIT, standardized README.

---

## 5. Requirements

Priority: **P0** blocks release, **P1** same release strongly desired, **P2** may be deferred.

### R1 — Session creation correctness (P0)

- `delegate_task_to_jules(source_name, prompt, starting_branch: str | None = None,
  title: str | None = None, require_plan_approval: bool = False)`.
- Branch resolution order:
  1. explicit `starting_branch` argument;
  2. **auto policy:** resolve the source's default branch from `list_sources()`
     (`defaultBranch.displayName`) and send it;
  3. if the source cannot be resolved, omit `githubRepoContext.startingBranch` entirely and let
     Google pick the repo default.
- It is **forbidden** for the literal string `"main"` to appear as a fallback default anywhere in
  the client.
- `require_plan_approval=True` may only be sent when the caller passes it explicitly; the default
  request payload must omit the flag entirely.
- **Acceptance:** unit tests assert the exact payload for a `master`-default repo resolves to
  `"master"` (or omits the field); dedicated regression test
  `test_no_hardcoded_main_for_master_default_repo`.

### R2 — Lifecycle tools (P0)

Expose the full session lifecycle as MCP tools (mirroring REST 1:1):

| Tool | REST backing |
|------|--------------|
| `check_jules_status(session_id)` *(restores the README-promised tool)* | `GET /sessions/{id}` |
| `get_jules_session(session_id)` — richer alias (state, updateTime, url, title) | `GET /sessions/{id}` |
| `list_jules_activities(session_id, page_size=None, page_token=None)` | `GET /sessions/{id}/activities` |
| `send_jules_message(session_id, prompt)` | `POST /sessions/{id}:sendMessage` body `{prompt}` |
| `approve_jules_plan(session_id)` | `POST /sessions/{id}:approvePlan` |

- `approve_jules_plan` logs a WARNING on every invocation (it releases a human-gate) and its
  docstring states it is intended **only within the scope of the delegated mission**.
- **Acceptance:** mocked-transport tests assert method + path + payload for each tool; tool count
  and names match README (see R6).

### R3 — Response hygiene (P1)

- Every tool returns compact, deterministic text: always includes `session id` and `url` when
  known; prompts echoed back are truncated to ≤200 chars; raw API dicts are never dumped.
- Activity listing collapses each activity to `{createTime, originator, type, one-line-summary}`
  with patch bodies truncated to ≤300 chars and total output capped (default ~4 KB).
- **Acceptance:** golden-snapshot tests for each tool's output shape.

### R4 — Error taxonomy (P1)

Map upstream failures to actionable messages (table-driven, tested):

| Upstream | Returned guidance |
|----------|-------------------|
| 404 `NOT_FOUND` | "Session/source not found — check session_id / source_name (use list_jules_sources)" |
| 401/403 | "JULES_API_KEY invalid or lacks scope" |
| 400 `INVALID_ARGUMENT` | Pass through Google's reason string verbatim |
| 429 after retries exhausted | "Rate-limited; retry after backoff" + attempts made |
| 5xx after retries exhausted | "Jules API unavailable; attempts=N" |

- **Acceptance:** one test per row using mocked statuses.

### R5 — Policy guardrails (P0, non-negotiable)

- The server implements **no auto-approve loops and no watchers**; `approve_jules_plan` is a
  one-shot explicit action.
- The server never merges, never pushes, never touches branches beyond what the Jules API does
  natively on Jules-owned working branches.
- These boundaries are documented in the README "Security model" section.

### R6 — Docs ↔ registry contract (P1)

- README tool table is regenerated from the actual `@mcp.tool()` registry.
- A unit test asserts `set(tools in server registry) == set(rows in README table)` and fails on
  drift (this would have caught INC-3).
- **Acceptance:** `test_readme_matches_tool_registry` green; README updated with all new tools.

### R7 — Configuration (P2, keep minimal)

- Add `JULES_DISABLE_RETRY=1` env switch for deterministic tests.
- Document existing retry knobs; add nothing else without a demonstrated need.

### R8 — Testing & CI (P0)

- Unit tests mock the HTTP layer (no network in CI).
- Optional live contract suite behind `RUN_LIVE_JULES_TESTS=1` (skipped by default).
- Coverage target ≥90% on `JulesClient` public methods and all tool functions.
- **Acceptance:** full suite green on a clean venv; ruff clean.

---

## 6. Rollout plan

| PR | Contents | Notes |
|----|----------|-------|
| PR-1 | R1 (branch resolution + payload builder) + regression tests | Smallest safe diff; immediately unblocks master-default repos |
| PR-2 | R2 + R3 + R4 + R6 (lifecycle tools, hygiene, taxonomy, docs-sync test) | Version minor bump |
| PR-3 (optional) | R7 + live-contract suite | May be deferred |

Each PR: full suite green, ruff clean, maintainer smoke-test against the real API before merge.
Backwards compatibility: tool signatures grow additively only; `check_jules_status` regains its
documented meaning.

---

## 7. Out of scope

Auto-merge/auto-approve automation, activity streaming/websockets, transport migration away from
stdio, GitHub-side PR review management, scheduling.

---

## 8. Open questions (for implementer/reviewer)

1. Add a convenience `wait_for_jules_terminal_state(session_id, timeout_s)` polling helper now or
   defer (P2 candidate)? Lab currently solves this with external cron.
2. Cache `list_sources()` for auto branch resolution (TTL?) — freshness vs latency tradeoff.
3. Should `send_jules_message` refuse when session state is terminal (`COMPLETED`/`FAILED`)?

---

## Appendix A — REST v1alpha quick reference (empirically verified 2026-08-25)

```
POST /v1alpha/sessions
     {prompt, sourceContext:{source, githubRepoContext:{startingBranch}},
      title?, requirePlanApproval?}
GET  /v1alpha/sessions            GET /v1alpha/sources
GET  /v1alpha/sessions/{id}       GET /v1alpha/sessions/{id}/activities
POST /v1alpha/sessions/{id}:sendMessage   body {"prompt": "..."}
POST /v1alpha/sessions/{id}:approvePlan   body {}
```

Observed session states: `QUEUED`, `IN_PROGRESS`, `COMPLETED`, `FAILED`,
`AWAITING_USER_FEEDBACK`.
Observed activity kinds: `progressUpdated`, `agentMessaged`, `sessionCompleted`,
`sessionFailed`, plus `artifacts[].changeSet.gitPatch` carrying unidiff output.

Sources corroborating endpoints beyond first-party use:
`raycast/extensions` (`extensions/jules-agents/src/jules.ts`),
`GatienBoquet/jules_mcp` (`src/client/jules-client.ts`).

## Appendix B — Incident session references

- `1161450207223342781` — FAILED (clone; INC-1)
- `15074687531563124415` — COMPLETED; published audit report + PR autonomously (healthy baseline)
- `12444597626949275570` — AWAITING_USER_FEEDBACK stall, resumed after manual `:sendMessage` (INC-2)
