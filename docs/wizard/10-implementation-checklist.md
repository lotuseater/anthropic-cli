# 10 — Implementation Checklist

A concrete, file-level checklist that turns [`09-roadmap-v2.md`](09-roadmap-v2.md)
into Monday-morning actions. Each item lists: files to touch, dependencies,
verification command, rollback path. Numbered to match the roadmap weeks.

Effort labels match the catalog: XS ≤ 1 h. S ≤ ½ day. M ≤ 2 days. L ≥ 3 days.

---

## W1.1 — Local plugin marketplace pin (XS, this repo)

**Files**
- `internal/kit/payload/templates/claude-code/plugin-marketplaces.json.template` (new)
- `internal/kit/install.go` — extend `installClaude()` to copy this template into
  `~/.claude/plugins/known_marketplaces.json` if missing, or merge entries if
  present.
- `scripts/install-claude-kit.ps1` — mirror logic in PowerShell.

**Schema reference.** `~/.claude/plugins/known_marketplaces.json` already exists
on this PC with the shape:

```json
{
  "<name>": {
    "source": { "source": "github", "repo": "anthropics/claude-plugins-official" },
    "installLocation": "C:\\Users\\Oleh\\.claude\\plugins\\marketplaces\\<name>",
    "lastUpdated": "<ISO-8601>"
  }
}
```

Local clones can be pointed at via `installLocation` set to the cloned path.
The `source` block describes the canonical origin (used for refresh).

**Verification**

```pwsh
ant kit install --dry-run --claude-only        # diff includes plugin-marketplaces
ant kit install --claude-only                  # actual install
type C:\Users\Oleh\.claude\plugins\known_marketplaces.json
claude /plugin discover                         # offline; lists local entries
```

**Rollback.** `ant kit rollback` restores prior `known_marketplaces.json`.

---

## W1.2 — Per-request trace-id propagation (XS, Wizard_Erasmus)

**Files**
- `Wizard_Erasmus/src/mcp/cognitive_pulse_hook.py` — at top of `main()`,
  `os.environ.setdefault("WIZARD_TRACE_ID", uuid.uuid4().hex)`.
- `Wizard_Erasmus/src/mcp/hook_utils.py` — read env into a span builder
  shared by all hooks.

**Verification**

```bash
WIZARD_TRACE_ID=
claude  # any prompt
echo $WIZARD_TRACE_ID  # set after the first hook fires
grep '"trace_id"' ~/.claude/traces/*.jsonl | head -3  # appears in spans (after W4)
```

**Rollback.** Remove the env-set lines; default back to per-hook anonymous IDs.

---

## W1.3 — `Invoke-Bounded` retry + backoff (XS, PowerShell fork)

**Files**
- `PowerShell/src/Modules/Shared/Microsoft.PowerShell.Wizard/Invoke-Bounded.ps1`
  — add `[int]$Retry = 0`, `[double]$BackoffSeconds = 0.5`. Wrap inner
  invocation in a `for ($i = 0; $i -le $Retry; $i++)` loop with exponential
  backoff on transient exit codes.
- `PowerShell/test/wizard/Invoke-Bounded.Tests.ps1` — Pester case for a
  flaky command that succeeds on attempt 2.

**Verification**

```pwsh
Invoke-Pester PowerShell/test/wizard/Invoke-Bounded.Tests.ps1
# Assert: 1 retry, command succeeds, total wall time covers backoff
```

**Rollback.** Default `-Retry = 0` keeps existing behaviour identical; remove
the new params to revert.

---

## W1.4 — Three first-party skill references (XS, this repo)

**Files**
- `internal/kit/payload/skills/claude-code/_upstream/claude-api/SKILL.md` (pointer + install hint)
- `internal/kit/payload/skills/claude-code/_upstream/mcp-builder/SKILL.md`
- `internal/kit/payload/skills/claude-code/_upstream/skill-creator/SKILL.md`

Each pointer skill contains:

```markdown
---
name: <skill-name> (upstream pointer)
description: Pointer to the upstream Anthropic skill at C:\Users\Oleh\Documents\GitHub\Antropic\skills\skills\<skill-name>\. Trigger phrases match the upstream skill; this body is intentionally short and just delegates.
---
# <skill-name> (pointer)

Upstream body lives at:
  C:\Users\Oleh\Documents\GitHub\Antropic\skills\skills\<skill-name>\SKILL.md

Read that file directly — do not duplicate. If the upstream skill is missing,
suggest the user clone `anthropics/skills` from GitHub and point Claude Code's
plugin marketplace at it (see W1.1).
```

**Verification**

```pwsh
ant kit list  # shows the three pointers under Claude Code skills
ant kit install --dry-run --claude-only
```

**Rollback.** Same as any kit skill — `ant kit rollback`.

---

## W2.1 — `mcp__wizard__handoff_to` MCP tool (M, Wizard_Erasmus)

**Files**
- `Wizard_Erasmus/src/mcp/wizard_mcp_server.py` — register `handoff_to`.
- `Wizard_Erasmus/src/mcp/wizard_handoffs.py` (new) — orchestrator.
- `Wizard_Erasmus/src/mcp/wizard_handoff_filters.py` (new) — built-in filters
  (`remove_all_tools`, `summarise_oldest`, `anonymise_pii`).
- `Wizard_Erasmus/src/mcp/wizard_db.py` — SELECT helper for team-member
  registry by capability tag.

**Dependency.** None — `wizard_db.py` already has a member registry.

**Verification**

```python
# Manual probe
from wizard_mcp.handoffs import handoff_to
result = handoff_to(
    "research_subagent",
    input_filter="summarise_oldest:5",
    history_mode="filtered",
)
assert result.target == "research_subagent"
assert "tool_use" not in result.transferred_history  # confirms strip
```

**Rollback.** Unregister `handoff_to` from `wizard_mcp_server.py`. New files
are isolated.

---

## W3.1 — Checkpoint / resume / approve trio (M, Wizard_Erasmus)

**Files**
- `Wizard_Erasmus/src/mcp/wizard_runstate.py` (new).
- `Wizard_Erasmus/src/mcp/wizard_db.py` — add `runs` and `pending_approvals`
  tables. SQL migration committed alongside.
- `Wizard_Erasmus/src/mcp/wizard_mcp_server.py` — register
  `checkpoint_run`, `resume_run`, `approve_pending`.

**Verification**

```python
# 1. Tool flagged needs_approval -> pause
state = checkpoint_run("run-001")
# 2. simulate reboot
del wizard_state_in_memory
# 3. resume
resume_run(state.handle)
# 4. approve from another terminal
approve_pending("run-001", decision="approve", body="ok")
# 5. assert run continues to completion
```

**Rollback.** Drop the two SQL tables; unregister the three tools.

---

## W4.1 — Span emitter + JSONL sink (S, Wizard_Erasmus)

**Files**
- `Wizard_Erasmus/src/mcp/wizard_traces.py` (new) — Span class, JSON writer.
- `Wizard_Erasmus/src/mcp/hook_utils.py` — wrap each hook fire in a
  `with span(...)` block.
- `Wizard_Erasmus/src/mcp/tool_cache.py` — emit span instead of ad-hoc
  latency entry; keep adapter for legacy consumers.

**Verification**

```bash
ls ~/.claude/traces/  # one file per day
jq -c 'select(.trace_id != null)' ~/.claude/traces/$(date +%F).jsonl | wc -l
# Expect ≥ N spans for an N-hook turn
```

**Rollback.** Disable `WIZARD_TRACE_ENABLED=1` env; legacy writes remain.

---

## W4.2 — OTel exporter (opt-in, M, Wizard_Erasmus)

**Files**
- `Wizard_Erasmus/pyproject.toml` — add `opentelemetry-sdk`,
  `opentelemetry-exporter-otlp-proto-http`.
- `Wizard_Erasmus/src/mcp/wizard_traces.py` — second sink behind
  `WIZARD_OTEL_ENDPOINT`.

**Verification**

```bash
WIZARD_OTEL_ENDPOINT=http://localhost:4318/v1/traces claude  # any prompt
# Inspect collector logs / Jaeger UI
```

**Rollback.** Unset env var; OTel export stops.

---

## W5.1 — `WizardSession` protocol + SQLite backend (M, Wizard_Erasmus)

**Files**
- `Wizard_Erasmus/src/mcp/wizard_session.py` (new) — Python `Protocol`.
- `Wizard_Erasmus/src/mcp/wizard_session_sqlite.py` (new) — backend.
- `Wizard_Erasmus/src/mcp/wizard_db.py` — add `sessions` and `messages`
  tables.

**Verification**

```python
from wizard_mcp.session import WizardSession
from wizard_mcp.session_sqlite import SqliteSession
s = SqliteSession("test-run")
s.append({"role": "user", "content": "hi"})
assert len(s.slice(0, 10)) == 1
snap = s.snapshot()
s2 = SqliteSession.restore(snap)
assert s2.slice(0, 10) == s.slice(0, 10)
```

**Rollback.** Two new tables can be `DROP TABLE`d. New files are isolated.

---

## W5.2 — Compaction backend (M, Wizard_Erasmus)

**Files**
- `Wizard_Erasmus/src/mcp/wizard_session_compacting.py` (new) — wraps
  `SqliteSession`. Threshold-driven summarisation via existing
  `cheap_model_eval.py`.
- `Wizard_Erasmus/tests/test_compacting_session.py` (new) — recall probe.

**Verification**

```python
# 1. Build a 20 MB transcript
# 2. Wrap in compacting session
# 3. Probe recall on a fixed prompt-set; assert >= 90% match vs raw
```

**Rollback.** Default backend stays SQLite; compaction off until opt-in.

---

## W6.1 — Wizard Kit v0.2.0 release (S, this repo)

**Files**
- `internal/kit/version.go` — `Version = "0.2.0"`.
- `scripts/install-claude-kit.ps1` — `$KitVersion = '0.2.0'`.
- `internal/kit/payload/skills/claude-code/agents-as-tools/SKILL.md` (new).
- `internal/kit/payload/skills/claude-code/runstate-checkpoint/SKILL.md` (new).
- `internal/kit/payload/templates/claude-code/settings.json.template` —
  add hook entries for the new tracing/streaming paths (gated by env vars
  so no behaviour change without opt-in).
- `docs/wizard/CHANGELOG.md` (new) — kit-only changelog.

**Verification**

```pwsh
ant kit install --dry-run     # shows new skills + version 0.2.0
ant kit install               # apply
ant kit rollback              # restores v0.1.0 byte-for-byte
git tag kit-v0.2.0
git push origin kit-v0.2.0
```

**Rollback.** Tag the prior commit as `kit-v0.1.0`; `git revert` the v0.2
commit if needed; `ant kit rollback` restores user files.

---

## Dependency graph

```
W1.1 (marketplace) ──┐
W1.2 (trace-id)   ──┼─→ W4.1 (spans) ──→ W4.2 (OTel)
W1.3 (retry)      ──┘                    └─→ Phase 6 PowerShell hook host
W1.4 (skills)
                    W2.1 (handoff) ──┐
                                      ├─→ W6.1 (v0.2 release)
                    W3.1 (HITL)   ───┤
                                      │
                    W5.1 (sessions) ──┴─→ W5.2 (compaction)
```

W1 items are independent and can land in parallel. W2/W3/W5 are independent
of each other but all feed W6.

## Out of scope

- W4.2 OTel exporter — opt-in; ship only if at least one consumer (Honeycomb,
  Jaeger, Anthropic Traces) is configured.
- Async/concurrent dispatch (`08-capability-gap-map.md` G-11/G-12) — depends
  on PowerShell fork Phase 6 hook host being mature first.
- Multi-provider routing (G-13) — single-provider stack does not justify
  the abstraction tax.
