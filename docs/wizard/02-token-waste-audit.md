# 02 — Token Waste Audit

Eight token-waste patterns observed across this PC's Claude Code stack, with
telemetry numbers and concrete pointers to evidence. Each pattern maps to one
or more levers in [03-improvement-catalog.md](03-improvement-catalog.md).

## 1. Cold Python hook spawns

**Observation.** Every UserPromptSubmit and PostToolUse hook is invoked as a
fresh `powershell → & py -3.14 hook.py` process. Each re-imports `wizard_mcp`
modules from scratch.

**Magnitude.** ~14 cold spawns per turn. ~500 ms per spawn (import + DB
connect). Aggregate budget consumed: ~7 s per turn before any model work.

**Evidence.** `PowerShell/docs/wizard/RESEARCH.md` §2.3 ("seven friction
signals"); `Wizard_Erasmus/src/mcp/cognitive_pulse_hook.py` import block at
top is ~120 LOC of `from wizard_mcp.* import` statements.

**Why it matters.** Latency hides the cost: the user perceives Claude as
slower without seeing where the time goes. Cumulatively this displaces the
prompt cache TTL window (5 min) — long pauses miss the cache entirely.

## 2. Cognitive pulse injected into prompt

**Observation.** `cognitive_pulse_hook.py` produces a 10–20 KB context block
(memory scan, compiled procedures, MagicHat status, recent episodic memories)
and prepends it to **every** UserPromptSubmit.

**Magnitude.** ~10–20 KB × every turn = ~5–10% of prompt tokens spent on
context that is mostly unused for trivial reads.

**Evidence.**
- `Wizard_Erasmus/research/Hooks_Tools_Skills_Usage_Report_2026_04_19.md`
  — `cognitive_pulse` should fire 8,927 times in audit window, fired 9 times
  (99.9% miss). When it does fire, the full block lands.
- `<cognitive-pulse-hook>` blocks visible in this very session's system
  reminders.

**Why it matters.** The pulse is computationally cheap (<200 ms) but its
output is paid for in tokens, not CPU. Publishing to a signal bus topic and
including only a one-line pointer in the prompt would preserve the value
without the per-turn token cost.

## 3. Visual-verify on every Bash

**Observation.** `visual_verify_hook.py` matches **every** Bash, `smart_*`,
and `dab_*` PostToolUse. It runs OCR (`dab_visual_scan`) with a 12 s timeout.

**Magnitude.** 12 s budget × every shell call. For a turn with 5 shell calls,
that's 60 s spent OCRing the screen — even when the command is `ls`,
`echo "done"`, `git log`, or any other read-only / safe operation.

**Evidence.** `~/.claude/settings.json` PostToolUse matchers list ~15 patterns
all routed to `visual_verify`. No skip heuristic.

**Why it matters.** Most output discrepancies are caught by exit codes and
text streams; OCR adds value only when a GUI subprocess produced visible
output that the model otherwise wouldn't see (e.g., a crashed Team App
window). The matcher should narrow to write/build/test commands.

## 4. Grep/Glob have ~0% cache hit rate

**Observation.** The cross-project tool cache is healthy on Read calls but
near-zero on Grep/Glob/Bash.

**Magnitude.** Cache hit rate 19.9% overall, 18.47 MB saved over a 2-day
window. If Grep/Glob/Bash matched Read's hit rate, expected savings would
roughly triple.

**Evidence.** `Wizard_Erasmus/research/Cache_Effectiveness_Report_2026_04_19.md`,
SQL aggregates per tool family.

**Why it matters.** Within a session, the same `rg -i "foo"` against the same
codebase is run 3+ times as the model rediscovers context. A hash-keyed cache
on (cwd, args, file mtimes) would deduplicate without semantic risk.

## 5. Tool output not compacted in history

**Observation.** Bash/Read/Grep results are stored verbatim in the session
JSONL. No size cap, no hash-with-tail compaction.

**Magnitude.** Largest project transcript (`projects/Wizard_Erasmus/`)
weighs 49 MB. SlavaTask weighs 21 MB. Even if 60% is genuine signal, that
leaves ~30 MB of redundant tool output per project.

**Evidence.** Direct `du -sh ~/.claude/projects/*` (or PowerShell equivalent).

**Why it matters.** Large transcripts inflate context-recovery costs across
sessions and slow PreCompact / quota-handoff hooks. A "≥50 KB outputs are
stored as hash + first 5 KB + last 5 KB; full body in side-archive" rule
recovers 20–30% history bloat without losing recall.

## 6. Memory not indexed

**Observation.** `~/.claude/memory/MEMORY.md` has one entry. There are 65+
plan files in `~/.claude/plans/`. The model rediscovers prior plans by
filesystem walk on each session.

**Magnitude.** 5–8% per session in unnecessary `Read` and `Glob` calls during
session start while the model orients itself.

**Evidence.** Direct `ls C:\Users\Oleh\.claude\plans\ | wc -l` and inspection
of MEMORY.md.

**Why it matters.** A one-line-per-plan index in MEMORY.md (auto-built by
the installer) lets the in-context memory system surface relevant prior work
in a single read. The index regenerates idempotently.

## 7. Sequential Stop guard chain

**Observation.** Five Stop hooks fire in order: `test_guard`,
`premature_stop`, `flaky_dismissal`, `caveat_hedge`, `claude_quota_handoff`.
Each runs even when an earlier one already blocked.

**Magnitude.** ~5–10% of Stop events spend all five guard budgets when only
the first decision matters.

**Evidence.** `~/.claude/settings.json` Stop matcher array; no
`SKIP_DOWNSTREAM` short-circuit.

**Why it matters.** A guard that blocks should publish a signal that
downstream guards check. This is a one-line change in each guard plus a
shared signal name (`wizard.stop.blocked`).

## 8. Permission prompts still leak through allowlist

**Observation.** `~/.claude/settings.json` has a 107-line permission
allowlist with `skipAutoPermissionPrompt=true`. Despite this, conversation
transcripts show 149 instances of "please go on" / "please continue" across
604 conversations.

**Magnitude.** Per `~/.claude/rules/no-premature-stop.md`: 210 premature stops
in 604 conversations. Many trace to project-specific commands not in the
allowlist (custom build scripts, `mcp__*` tools introduced after the allowlist
was last edited).

**Evidence.** `~/.claude/rules/no-premature-stop.md` evidence section;
`~/.claude/projects/SlavaTask/*.jsonl` "please go on - btw, before continue,
see if you can free up some disk space" (timestamp 1776686842 = 2026-04-27).

**Why it matters.** Per-project `.claude/settings.local.json` files for the
three active families (PHP/SlavaTask, C++/Wizard_Erasmus, Python/Serial_to_Google_Doc)
covers the long tail without bloating the global allowlist.

---

## Combined impact estimate

If all eight patterns are addressed:

| Bucket | Saving |
|--------|--------|
| Latency (cold spawns + visual verify) | 30–40% of per-turn wall time |
| Prompt tokens (pulse-as-signal + memory index) | 10–20% |
| History bloat (tool-output compaction) | 20–30% of project JSONL size |
| Stop-event budget (guard short-circuit) | 5–10% |
| Allowlist coverage | qualitative — fewer interruptions |

These are upper bounds; realised savings depend on workload mix. The
[improvement catalog](03-improvement-catalog.md) breaks each lever down with
effort estimates and ownership.
