# 09 — Roadmap v2 (Antropic + openai-agents-python wave)

Six-week roadmap layered on top of [`05-roadmap.md`](05-roadmap.md). The v0
roadmap covered the original Wizard Kit (skills, settings, installer); this
v2 wave addresses the patterns identified in
[`07-openai-agents-sdk-patterns.md`](07-openai-agents-sdk-patterns.md) and
the gaps in [`08-capability-gap-map.md`](08-capability-gap-map.md).

Cross-reference notation: `(P-N)` = pattern N from doc 07; `(G-N)` = gap N
from doc 08.

## Week 1 — Quick wins

Goal: clear the four XS items in one branch.

| Item | Source | Owner | Deliverable |
|------|--------|-------|-------------|
| Local plugin marketplace pin | (G-18) | this repo (kit) | `templates/claude-code/.claude/plugin-marketplaces.json` template + installer step that points at `Antropic/claude-plugins-official` and `Antropic/skills` |
| Per-request trace-id propagation | (G-9) | Wizard_Erasmus | `os.environ.setdefault("WIZARD_TRACE_ID", uuid4())` in `cognitive_pulse_hook.py`; downstream hooks read from env |
| `Invoke-Bounded` retry + backoff | (G-16) | PowerShell fork | `-Retry`, `-BackoffSeconds` parameters; default off, opt-in for known-flaky commands |
| Three first-party skills installed | doc 06 §4 | this repo (kit) | Add `claude-api`, `mcp-builder`, `skill-creator` symlinks to v0.2 install path |

Verification: install runs clean, new env var visible in any hook log,
retry path covered by a Pester test in the PowerShell fork.

## Week 2 — Handoff foundation

Goal: ship the Handoff primitive (P-1, G-1).

| Item | Owner | Deliverable |
|------|-------|-------------|
| `mcp__wizard__handoff_to(agent_id, mode='transfer'|'as_tool', input_filter)` | Wizard_Erasmus | New tool in `wizard_mcp_server.py`; reuses `wizard_db.py` registry |
| `remove_all_tools` and `summarise_oldest(n)` filters | Wizard_Erasmus | Filter library at `src/mcp/wizard_handoff_filters.py`; `summarise_oldest` uses existing `cheap_model_eval.py` |
| Skill template "manager skill" | this repo (kit) | New skill body at `internal/kit/payload/skills/claude-code/manager-pattern/SKILL.md` |

Verification: a manual handoff between two test team members lands cleanly;
filter strips tool calls before transfer.

## Week 3 — RunState + HITL

Goal: ship the approval + resume primitives (P-2, G-2).

| Item | Owner | Deliverable |
|------|-------|-------------|
| `mcp__wizard__checkpoint_run(run_id) → handle` | Wizard_Erasmus | Serialise pending approvals + run metadata into SQLite (`runs` table) |
| `mcp__wizard__resume_run(handle)` | Wizard_Erasmus | Replay captured state into the active session |
| `mcp__wizard__approve_pending(run_id, decision, body?)` | Wizard_Erasmus | Side-channel decision recorder; runner polls before next tool call |
| Skill: `human-in-the-loop` | this repo (kit) | Workflow-compressor skill describing the approval flow |

Verification: a tool flagged `needs_approval` blocks the runner; reboot the
machine; `resume_run` reattaches and `approve_pending` releases it.

## Week 4 — Tracing + spans

Goal: replace ad-hoc telemetry with structured spans (P-7, G-7, G-8).

| Item | Owner | Deliverable |
|------|-------|-------------|
| Span emitter | Wizard_Erasmus | `src/mcp/wizard_traces.py`; one span per hook fire; parent = `WIZARD_TRACE_ID` |
| Sink: JSONL | Wizard_Erasmus | Default sink writes `~/.claude/traces/<date>.jsonl` |
| Sink: OTel exporter (opt-in) | Wizard_Erasmus | If `WIZARD_OTEL_ENDPOINT` is set, dual-emit |
| Migration of `tool_cache.py` latency entries | Wizard_Erasmus | Adapter so existing telemetry consumers still see the old shape during transition |
| Documentation skill: `read-traces` | this repo (kit) | Skill that surfaces `~/.claude/traces/` contents |

Verification: a multi-hook turn produces a single trace with N child spans;
`jq` slice over the JSONL reproduces a Gantt-style summary.

## Week 5 — Sessions abstraction

Goal: make conversation history backend-agnostic (P-6, G-5, G-6).

| Item | Owner | Deliverable |
|------|-------|-------------|
| `WizardSession` Python protocol | Wizard_Erasmus | `src/mcp/wizard_session.py`; methods: append, slice, snapshot, restore |
| SQLite backend | Wizard_Erasmus | Reuses `wizard_db.py`; default backend |
| Compaction backend | Wizard_Erasmus | Wraps SQLite; threshold-driven summarisation via `cheap_model_eval.py` |
| Migration helper | Wizard_Erasmus | One-shot import of existing Claude Code JSONL into SQLite |

Verification: a 20 MB JSONL transcript imports in under 30 s; subsequent
turns persist into SQLite; compaction triggers at 5 MB threshold and
preserves recall over a known prompt-set probe.

## Week 6 — Distribution refresh

Goal: Wizard Kit v0.2.0.

| Item | Owner | Deliverable |
|------|-------|-------------|
| Bump `internal/kit/version.go` and `install-claude-kit.ps1` to `0.2.0` | this repo | Version bump + CHANGELOG entry |
| New skill: `agents-as-tools` | this repo | Workflow-compressor skill describing the manager pattern |
| New skill: `runstate-checkpoint` | this repo | Skill that wraps the approval/checkpoint MCP tools |
| Updated `templates/claude-code/settings.json.template` | this repo | Hook entries for the new tracing/streaming paths |
| Tag and push | this repo | `kit-v0.2.0` tag |

Verification: `ant kit install` lays down the new skills; `ant kit list`
shows v0.2.0 manifest; rollback restores v0.1.0 byte-for-byte.

## Cross-references to other research

- `Wizard_Erasmus/research/mempalace_assessment_2026_04_26.md` — eight
  mempalace ideas (verbatim drawer, query-time dedup, temporal validity
  windows, etc.). The session-abstraction work in week 5 should align with
  the "verbatim drawer + compact index" split rather than reinvent it.
- `PowerShell/docs/wizard/PLAN.md` §6 — the persistent Python hook host
  remains the structural blocker for gap #10 (streaming output). Land it in
  parallel with weeks 2–4 for biggest knock-on benefit.

## Out of scope for v0.2

- Async/concurrent dispatch (G-11) and priority queue (G-12). These are L-
  effort and depend on the warm hook host being mature first.
- Multi-provider routing (G-13). Single-provider stack does not justify the
  abstraction tax yet.
- Anything in the deprecated `anthropic-tools` or `anthropic-tokenizer-typescript`
  repos — covered by [`06-antropic-folder-survey.md`](06-antropic-folder-survey.md).

## Kill switches

If any week produces results below threshold, stop the cascade:

- Week 2: if `handoff_to` adds >100 ms p50 to a handoff, drop the input-
  filter dependency on `cheap_model_eval.py` and inline a regex strip.
- Week 4: if span volume exceeds 1 GB / day on a normal workload, bump the
  default sink to a 7-day rolling buffer.
- Week 5: if compaction loses recall on a probe set, ship the SQLite
  backend without compaction and keep JSONL alive as the recall source of
  truth.
