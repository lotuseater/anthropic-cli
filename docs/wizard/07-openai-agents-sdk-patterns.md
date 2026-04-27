# 07 — Patterns from `openai-agents-python` SDK

Translation report from `C:\Users\Oleh\Documents\GitHub\open_ai\openai-agents-python`
(read 2026-04-27) to the user's Wizard_Erasmus + PowerShell + anthropic-cli
stack. The goal is **patterns to port**, not the SDK itself.

## A) Project purpose

OpenAI Agents Python is a lightweight, provider-agnostic framework for
building multi-agent workflows. It ships declarative composition (Agent +
Tool + Handoff + Guardrail) over a runtime loop (Runner) with persistent
state (Session, RunState) and observability (Trace + Span). The 0.14 release
adds `SandboxAgent` for hosted-execution workflows, but the core
abstractions are stable, runtime-neutral, and small.

## B) Core abstractions ↔ existing equivalents

| Abstraction | Kind | One-line role | Existing equivalent on this PC |
|-------------|------|---------------|--------------------------------|
| `Agent` | authoring | LLM + instructions + tools + handoffs + guardrails + output schema | Wizard subagent definition (loose); Claude Code skills (instructions only) |
| `Runner` | runtime | Turn loop: invoke model → resolve tools → check guardrails → handoff → persist session | `claude run` loop + Wizard team coordination (loose) |
| `Handoff` | both | First-class transfer-to-X tool with input filtering | Wizard `team_remind` (no LLM-driven routing); Claude Code skill chaining (manual) |
| `Tool` | authoring | Function/agent/MCP/hosted call with schema and approval gates | MCP tools + Claude Code tool defs |
| `Guardrail` | authoring | Input/output/tool validator with parallel-vs-blocking modes | `commit_guard_hook`, security governor (always blocking, no parallel mode) |
| `Session` | runtime | Conversation history backend (SQLite, Redis, MongoDB, Dapr, encrypted) | Claude Code session JSONL (one shape, no compaction) |
| `Trace` + `Span` | runtime | Observable workflow container; spans cover model + tool + handoff + guardrail | `tool_cache.py` telemetry + ad-hoc file writes (no OTel-style spans) |
| `RunState` | runtime | Serialisable checkpoint of agent state, history, pending interruptions | Wizard episodic memory (no run-state checkpoint) |
| `RunContext` | runtime | Per-run carrier for app state, approval decisions, run metadata | Wizard `run_context` + working memory |
| `InputFilter` | authoring | Function transforming history before a handoff (strip tools, anonymise, summarise) | None — handoff filtering is ad-hoc |
| `ModelSettings` | authoring | Temperature, top_p, tool_choice, thinking budgets | Per-call overrides in anthropic-cli |
| `HostedMCPTool` | runtime | MCP server executed by OpenAI's Responses API | None — Claude Code only knows local MCP |

## C) Patterns worth porting (ranked)

### 1. ⭐⭐⭐⭐⭐ Handoff-driven agent composition

**What.** Specialist agents declared via `Handoff(...)`. Each becomes a
synthesised `transfer_to_<name>` tool exposed to the parent. The LLM picks
the specialist; the runner transfers conversation cleanly.

**Why it helps.** Per-specialist prompt isolation; per-specialist model
config; routing decided at call time, not in code.

**Where to land.** Wizard_Erasmus: new MCP tool
`mcp__wizard__handoff_to(agent, input_filter, history_mode)`. Reuses the
existing team-member registry in `wizard_db.py`.

**Reference files in `openai-agents-python`:**
- `src/agents/handoffs/__init__.py`
- `src/agents/handoffs/history.py`
- `examples/handoffs/message_filter.py`

### 2. ⭐⭐⭐⭐⭐ HITL with serialisable RunState

**What.** Tools mark `needs_approval=True` (or a callable). Runner pauses on
approval-pending; surfaces `interruptions` in the result. Convert to
`RunState`, persist (JSON/pickle), reload, call
`state.approve(interruption)` / `state.reject(...)`, then `Runner.run(agent,
state)` resumes from the same point. Survives restarts; nested approvals
inside handoffs bubble up correctly.

**Why it helps.** Approval gates that don't break the run loop and that
survive a reboot or instance kill.

**Where to land.** Wizard_Erasmus tool trio:
- `mcp__wizard__checkpoint_run(run_id) → handle`
- `mcp__wizard__resume_run(handle)`
- `mcp__wizard__approve_pending(run_id, decision, body?)`

Use the existing SQLite store in `wizard_db.py` for persistence; layer over
the Claude Code session JSONL.

**Reference files:**
- `src/agents/run_state.py`
- `src/agents/items.py` — see `ToolApprovalItem`
- `examples/agent_patterns/human_in_the_loop.py`
- `examples/agent_patterns/human_in_the_loop_conditional.py`

### 3. ⭐⭐⭐⭐ Input-filter pipelines for handoffs

**What.** Before transfer, filter conversation history with composable
filters: `remove_all_tools`, custom lambdas, anonymise PII, summarise N
oldest turns. SDK ships `HandoffInputData` wrapping pre-handoff items + new
items + input history.

**Why it helps.** Cuts token cost on the receiving side and prevents
information leakage. Especially valuable when the receiving specialist
runs a cheaper / smaller model.

**Where to land.** Wizard_Erasmus: filter parameter on the new
`handoff_to` tool. Reuse the existing summariser in `cheap_model_eval.py`
for the "summarise older turns" filter.

### 4. ⭐⭐⭐⭐ Agents-as-tools manager pattern

**What.** Convert an agent to a tool via
`agent.as_tool(name, description, is_enabled=fn, needs_approval=fn,
parameters=Pydantic)`. The manager keeps the conversation; specialists are
called like functions and their results merge into the manager's working set.

**Why it helps.** Different from handoff: the manager retains control and
can call several specialists in one turn. Ideal when synthesis from
multiple sources is needed.

**Where to land.** Wizard_Erasmus: extend the team-coordination layer so any
team member can be exposed as an MCP tool by id. anthropic-cli kit: skill
template for "manager skills" that compose specialist skills.

**Reference files:**
- `src/agents/agent.py` — `as_tool(...)` method
- `examples/agent_patterns/agents_as_tools.py`
- `examples/agent_patterns/agents_as_tools_conditional.py`

### 5. ⭐⭐⭐ Guardrails with parallel-vs-blocking modes

**What.** Input guardrails can run **parallel** (concurrent with model — low
latency, may waste tokens if guardrail tripwires late) or **blocking**
(guardrail first, model only if it passes). Tool guardrails wrap a single
call. Output guardrails always block.

**Why it helps.** Cost/latency lever per rule. The user's
`commit_guard_hook` is unconditionally blocking; some content guards could
be parallel without losing strictness.

**Where to land.** Wizard_Erasmus: extend hook front-matter with a `mode`
field (`blocking` | `parallel`); update `apply_team_recovery` to honour it.

### 6. ⭐⭐⭐ Session abstraction with multi-backend support

**What.** `Session` interface plus shipped backends:
`sqlite_session.py`, `openai_conversations_session.py`,
`openai_responses_compaction_session.py`. Async variants in `extensions/`.

**Why it helps.** Swap backend without touching runner code. Compaction is
opt-in and lives behind the same interface.

**Where to land.** Wizard_Erasmus: define a thin `WizardSession` Python
protocol; default backend = SQLite via `wizard_db.py`. Compaction backend
later.

**Reference files:**
- `src/agents/memory/session.py`
- `src/agents/memory/sqlite_session.py`
- `src/agents/memory/openai_responses_compaction_session.py`

### 7. ⭐⭐⭐ Trace + span grouping

**What.** Wrap a workflow in `with trace("name", group_id, metadata): ...`.
Every Runner call inside emits spans (model call, tool exec, handoff,
guardrail). Group ID correlates related workflows.

**Why it helps.** Single unified view of multi-agent execution. Replaces ad-
hoc file logs in `tool_cache.py` with structured spans suitable for any
OTel collector.

**Where to land.** Wizard_Erasmus: emit one span per hook fire; parent-trace
ID propagated via env var (`WIZARD_TRACE_ID`). Sink: JSONL under
`~/.claude/traces/<date>.jsonl` + optional Honeycomb / Anthropic Traces UI.

### 8. ⭐⭐ Structured tool inputs / output types

**What.** Pydantic models for tool parameters; agents return strict-mode
JSON via the Responses API.

**Why it helps.** Type safety; LLM constraint; downstream code treats
outputs as objects.

**Where to land.** anthropic-cli kit: tool stubs ship with declared
schemas; Wizard_Erasmus tools surface declared schemas through MCP.

## D) Patterns NOT to port

- **`SandboxAgent` + Blaxel manifest.** Designed for hosted execution;
  Claude Code is local-first.
- **Responses-API-coupled abstractions.** Tightly coupled to OpenAI's
  message shape; Anthropic SDK has its own.
- **`HostedMCPTool`.** Only meaningful when the model runs on OpenAI's
  servers.
- **GPT-5 reasoning content shape.** Anthropic has its own thinking mode.
- **LiteLLM / multi-provider routing.** Stack is Anthropic-native; a
  provider-abstraction layer is overkill.

## E) File pointers for deeper dives

| Purpose | File |
|---------|------|
| Runner entry point | `src/agents/run.py` |
| Agent class + `as_tool` | `src/agents/agent.py` |
| Handoff design + input filter | `src/agents/handoffs/__init__.py` |
| Handoff history | `src/agents/handoffs/history.py` |
| Session interface | `src/agents/memory/session.py` |
| SQLite session backend | `src/agents/memory/sqlite_session.py` |
| Compaction backend | `src/agents/memory/openai_responses_compaction_session.py` |
| RunState serialisation | `src/agents/run_state.py` |
| Approval items | `src/agents/items.py` |
| Tracing | `src/agents/tracing/traces.py` |
| Examples — handoff w/ filter | `examples/handoffs/message_filter.py` |
| Examples — HITL | `examples/agent_patterns/human_in_the_loop.py` |
| Examples — HITL conditional | `examples/agent_patterns/human_in_the_loop_conditional.py` |
| Examples — agents as tools | `examples/agent_patterns/agents_as_tools.py` |
| Examples — agents as tools conditional | `examples/agent_patterns/agents_as_tools_conditional.py` |

All paths are relative to `C:\Users\Oleh\Documents\GitHub\open_ai\openai-agents-python\`.
