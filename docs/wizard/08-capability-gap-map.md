# 08 — Capability Gap Map

Concrete gaps between the user's current Claude Code / Codex stack and the
agent-runtime feature set the openai-agents SDK demonstrates. Confirmed by
absence in `Wizard_Erasmus/src/mcp/`, `PowerShell/src/Modules/Shared/Microsoft.PowerShell.Wizard/`,
and the v0 Wizard Kit (`internal/kit/`). No speculation: each row is either
"shipped" (with a file path), "partial" (close but missing a piece), or
"missing" (no implementation found anywhere).

## Gap table

| # | Capability | Status | Where it sits today | Owner if filled | Effort | Blocker |
|---|------------|--------|---------------------|-----------------|--------|---------|
| 1 | First-class **Handoff** with input filter | missing | `team_remind` is closest; pull-only, no LLM-driven routing | Wizard_Erasmus | M | none |
| 2 | Serialisable **RunState** + HITL approvals | missing | Episodic memory persists facts, not run state | Wizard_Erasmus | M | none |
| 3 | **Agents-as-tools** (`as_tool` wrapper) | missing | Team members are addressed by id, not exposed as MCP tools | Wizard_Erasmus | S | none |
| 4 | **Parallel-vs-blocking** guardrail modes | partial | All hooks block today; no `mode: parallel` flag | Wizard_Erasmus | S | per-hook audit |
| 5 | **Session** abstraction with multiple backends | partial | One JSONL session per Claude Code instance; Wizard `wizard_db.py` is SQLite but holds team state, not session history | Wizard_Erasmus + Claude Code config | M | upstream JSONL is fixed |
| 6 | Compaction-aware session backend | missing | No compaction; transcripts grow to 49 MB | Wizard_Erasmus | M | gap #5 first |
| 7 | **Trace** + span grouping | partial | `tool_cache.py` writes ad-hoc latency entries; no parent-trace-id propagation | Wizard_Erasmus | S | none |
| 8 | OTel-compatible span emission | missing | No OTel SDK in `pyproject.toml` | Wizard_Erasmus | M | gap #7 first |
| 9 | Per-request **trace ID** propagation | missing | Session id exists; no request-scope trace id | Wizard_Erasmus | XS | none |
| 10 | Tool-call output **streaming** buffer | missing | Hook output captured whole and returned | Wizard_Erasmus + PowerShell fork | M | needs warm hook host (Phase 6) |
| 11 | **Async / concurrent** tool dispatch | missing | Only `threading` for thread-local state; no asyncio loop | Wizard_Erasmus | M | none |
| 12 | Request **prioritisation / queuing / backpressure** | missing | Calls execute immediately; no priority lanes | Wizard_Erasmus | L | gap #11 first |
| 13 | Multi-provider **credential routing** | missing | Anthropic SDK only; no provider abstraction | anthropic-cli kit | M | low priority — single-provider stack |
| 14 | **Pub/sub** for agent-to-agent status | missing | Members report status by pull; no event bus | Wizard_Erasmus + PowerShell fork (`Publish-WizardSignal` exists) | S | reuse existing signal bus |
| 15 | Tool **output-schema validation** | missing | Tools don't declare output schema | Wizard_Erasmus | S | none |
| 16 | Tool **retry + exponential backoff** | partial | `Invoke-Bounded` has plain timeout, no retry | PowerShell fork | XS | none |
| 17 | **Pre-flight cost estimation** per tool call | missing | Quota-handoff happens after spend, not before | Wizard_Erasmus | M | needs token-cost model |
| 18 | **Local plugin marketplace** mirror | missing | Plugins resolved via network round-trip | this repo (kit templates) | XS | none |

## Effort scale

XS ≤ 1 h. S ≤ ½ day. M ≤ 2 days. L ≥ 3 days.

## Quick-win ranking

The ratio (impact / effort) favours these, in order:

1. **#9 trace-id propagation (XS)** — unblocks #7 and #8 and is a one-line
   `os.environ["WIZARD_TRACE_ID"]` propagation in `cognitive_pulse_hook.py`.
2. **#16 retry + backoff (XS)** — wrap `Invoke-Bounded` retry inside
   `Start-MonitoredProcess` once; cuts flake-induced re-runs.
3. **#18 local plugin marketplace (XS)** — already-cloned `claude-plugins-official`
   and `skills` repos pinned via `.claude/plugin-marketplaces.json`.
4. **#3 agents-as-tools (S)** — small wrapper that exposes a Wizard team
   member as an MCP tool; reuses `wizard_db.py` registry.
5. **#14 pub/sub events (S)** — reuse `Publish-WizardSignal` bus the
   PowerShell fork already provides.

## Slow-but-worth ranking

Largest realised value once shipped, longer to land:

1. **#1 + #2 Handoff with RunState/HITL (M+M)** — the single biggest
   ergonomic upgrade for multi-agent workflows on this PC.
2. **#5 + #6 Session abstraction with compaction (M+M)** — the only path to
   keeping per-project transcripts under 5 MB without losing recall.
3. **#10 streaming output (M)** — depends on Phase 6 warm hook host. Until
   that lands, this gap remains structural.

## Dependencies

```
#9 trace-id ──→ #7 spans ──→ #8 OTel emission
#11 async  ──→ #12 priority/queue
#5 sessions ──→ #6 compaction
Phase 6 hook host (PowerShell fork) ──→ #10 streaming
```

The PowerShell fork's Phase 6 (persistent Python hook host) is on the
critical path for streaming output (gap #10) and substantially improves the
viability of async dispatch (gap #11). It is documented in
`PowerShell/docs/wizard/PLAN.md` §6 and remains the highest-leverage
unshipped item across all three repos.

## Out of scope

- Sandbox / hosted-MCP equivalents — Claude Code is local-first.
- LiteLLM-style provider abstraction (gap #13) — single-provider stack
  doesn't justify the abstraction tax.
- Anthropic-side sandbox / Containers — handled upstream, not here.
