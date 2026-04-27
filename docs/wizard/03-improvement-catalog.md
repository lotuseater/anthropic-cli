# 03 — Improvement Catalog

Ten ranked levers for reducing token spend and improving automation. Each entry
gives the problem, proposed change, estimated saving, effort, and the owner
repo (where the implementation actually lands).

| # | Lever | Saving | Effort | Owner |
|---|-------|--------|--------|-------|
| 1 | Persistent Python hook host | 30–40% latency | M | PowerShell fork (Phase 6) |
| 2 | `cognitive_pulse` → signal bus | 5–10% prompt tokens | S | Wizard_Erasmus |
| 3 | First-wave skill pack (5 skills) | compresses ~313 repeated workflows | M | This repo (kit) |
| 4 | `visual_verify` skip heuristic | ~10% turn time | XS | Wizard_Erasmus + this repo (template) |
| 5 | Plan/memory index in MEMORY.md | 5–8% per session | XS | This repo (kit) |
| 6 | Grep/Glob result hash cache | 10–15% on grep-heavy work | S | Wizard_Erasmus (`tool_cache.py`) |
| 7 | Tool-output compaction in history | 20–30% history bloat | M | Wizard_Erasmus + Claude Code core |
| 8 | Per-project `settings.local.json` | qualitative | XS | This repo (kit templates) |
| 9 | Stop guard chain short-circuit | 5–10% Stop events | XS | Wizard_Erasmus |
| 10 | Distributable installer | enables system-wide rollout | M | This repo |

Effort scale: XS ≤ 1 h, S ≤ half day, M ≤ 2 days, L ≥ 3 days.

---

## 1. Persistent Python hook host *(Phase 6, owner: PowerShell fork)*

**Problem.** ~14 cold spawns per turn × ~500 ms each = ~7 s wasted before any
model work. See [02-token-waste-audit.md §1](02-token-waste-audit.md#1-cold-python-hook-spawns).

**Proposed change.** Add `hook.register / hook.invoke / hook.list /
hook.unregister` verbs to `WizardControlServer` (the JSON-RPC named-pipe
server already booting before runspace init). One warm Python child holds
hooks hot. First call ~50 ms, subsequent calls <10 ms. Spec: `PowerShell/docs/wizard/PLAN.md`
§6.

**Pilot.** Migrate `pretool_cache_hook` first (1,230 reuses / window — high
signal). Measure with `Hooks_Tools_Skills_Usage_Report` on a 48 h window
before mass migration.

**Why this lands in PowerShell, not here.** The hook host needs to start
before the first prompt, which means it has to live inside the shell host.
This repo's role is documenting it and pointing users to the spec.

## 2. `cognitive_pulse` → signal bus *(owner: Wizard_Erasmus)*

**Problem.** 10–20 KB context block injected into every turn's prompt; 99.9%
of intended firings already miss. See [02 §2](02-token-waste-audit.md#2-cognitive-pulse-injected-into-prompt).

**Proposed change.** Modify `cognitive_pulse_hook.py` to:

1. Compute the pulse as today.
2. `Publish-WizardSignal -Topic cognitive.pulse -Body <json>`.
3. Inject only a **one-line pointer** in the prompt:
   `<cognitive-pulse-pointer topic="cognitive.pulse" id="<sig>"/>`.
4. Add a `recall_cognitive_pulse` skill / MCP tool that fetches the body when
   the model decides it's relevant (defeasible-on-demand).

**Side benefit.** The agent can decide to skip the pulse entirely on trivial
turns; today it pays even when the prompt is `ls`.

## 3. First-wave skill pack *(owner: this repo)*

**Problem.** ~313 repeated workflow prompts in local Codex history alone.
Five distinct shapes account for them: conversation forensics, resume
interrupted work, research-to-md, live runtime triage, verified delivery.
See `Wizard_Erasmus/research/codex_skill_opportunities_2026_04_21.md`.

**Proposed change.** Ship five skills (under `internal/kit/payload/skills/`)
in two parallel copies: `claude-code/<skill>/SKILL.md` and `codex/<skill>/SKILL.md`.
Each skill is a workflow compressor (steps, tool sequence, deliverable
shape), not a reference manual.

**Skills:**

- `conversation-forensics` — mine `~/.claude/projects/*` and `~/.codex/sessions/*`,
  produce a Markdown memo. Recovers context after interruption; cross-project
  pattern recognition.
- `resume-interrupted-work` — locate the most recent unfinished session
  (possibly in a different cwd), restore context, continue.
- `research-to-md` — research-only deliverable: evidence → findings →
  recommendations, no implementation drift.
- `live-runtime-triage` — inspect running processes / windows before code
  edits. Uses `Get-WizardSession`, `Read-WizardSignal`, `dab_*` tools.
- `verified-delivery` — run tests, verify, commit focused diffs in dirty trees,
  push.

## 4. `visual_verify` skip heuristic *(owners: Wizard_Erasmus + this repo)*

**Problem.** OCR scan after **every** Bash, even safe read-only commands.
12 s budget × 5 calls = 60 s wasted. See [02 §3](02-token-waste-audit.md#3-visual-verify-on-every-bash).

**Proposed change.** Two parts:

1. **In Wizard_Erasmus:** narrow the matcher inside `visual_verify_hook.py`
   to commands that change visible state:
   `(npm|yarn|pnpm|cargo|go|make|pytest|jest|dotnet|build|test|migrate|push|deploy|install|rm|chmod|mv|cp -r)`.
   Pass-through (no OCR) for everything else.
2. **In this repo's template:** ship a `settings.json.template` with the
   narrowed matcher list so a fresh install picks it up automatically.

## 5. Plan/memory index in MEMORY.md *(owner: this repo)*

**Problem.** 65+ plans in `~/.claude/plans/`, MEMORY.md has one entry. Each
session re-walks the filesystem. See [02 §6](02-token-waste-audit.md#6-memory-not-indexed).

**Proposed change.** Installer builds a one-line-per-plan index by reading
each plan's first H1 / H2 and frontmatter. Format:

```markdown
- [Plan title](../plans/<file>.md) — one-line hook (date)
```

Idempotent: rerunning the installer rebuilds without duplicates. The
generated section is delimited by `<!-- wizard-kit:plan-index BEGIN -->` /
`<!-- wizard-kit:plan-index END -->` so user-authored entries above and below
are preserved.

## 6. Grep/Glob result hash cache *(owner: Wizard_Erasmus)*

**Problem.** Cache hit rate ~0% on Grep/Glob/Bash; same `rg` query repeated
3+ times in a session. See [02 §4](02-token-waste-audit.md#4-grepglob-have-0-cache-hit-rate).

**Proposed change.** Extend `tool_cache.py` to key on
`(tool_name, sorted_args, cwd, max(file_mtime for files matched))`. Invalidate
on Edit/Write of any matching file via the existing `edit_invalidate_hook`.
TTL 1 h within session.

## 7. Tool-output compaction in history *(owners: Wizard_Erasmus + Claude Code core)*

**Problem.** 49 MB transcript / project. See [02 §5](02-token-waste-audit.md#5-tool-output-not-compacted-in-history).

**Proposed change.** PostToolUse hook checks tool-result size; if >50 KB,
rewrites the in-history record to:

```json
{ "summary": "<first 5 KB>...<last 5 KB>",
  "full_hash": "sha256:...",
  "archive_path": "~/.claude/cache/tool-results/<hash>.txt" }
```

Full body remains queryable via a new `recall_tool_result` MCP tool.

This needs Claude Code core's cooperation (or a postprocessing pass on the
JSONL when sessions close). Documented here, owned externally.

## 8. Per-project `settings.local.json` *(owner: this repo)*

**Problem.** "please go on" leaks past the global allowlist. See
[02 §8](02-token-waste-audit.md#8-permission-prompts-still-leak-through-allowlist).

**Proposed change.** Ship template `settings.local.json` files for the three
active project families:

- `templates/claude-code/per-project/php/.claude/settings.local.json`
- `templates/claude-code/per-project/cpp/.claude/settings.local.json`
- `templates/claude-code/per-project/python/.claude/settings.local.json`

Installer detects project root by looking for marker files (`composer.json`,
`CMakeLists.txt`, `pyproject.toml`) and offers to drop the matching template.

## 9. Stop guard chain short-circuit *(owner: Wizard_Erasmus)*

**Problem.** Five Stop guards run sequentially even when an earlier one
already blocked. See [02 §7](02-token-waste-audit.md#7-sequential-stop-guard-chain).

**Proposed change.** First blocking guard publishes
`Publish-WizardSignal -Topic wizard.stop.blocked -Body <reason>`. Each
downstream guard checks `Read-WizardSignal -Topic wizard.stop.blocked` and
exits early if a block is already pending in this Stop event.

## 10. Distributable installer *(owner: this repo)*

**Problem.** Even if all of the above shipped, propagating them to a second
PC (or restoring after a clean reinstall) is manual. The user's note says
"custom version of cli may be built and spread across the system."

**Proposed change.** Ship from this repo:

- `scripts/install-claude-kit.ps1` — PowerShell installer with `-DryRun`,
  `-Install`, `-Rollback`, `-ClaudeOnly`, `-CodexOnly`, `-Force`.
- `ant kit install / list / rollback` — Go subcommand wrapping the same
  payload via `embed.FS`. After `go install ./cmd/ant` (or Homebrew install
  of `ant`), `ant kit install` works without this repo on disk.

Both readers consume the same `internal/kit/payload/` tree, so there is one
source of truth.

---

## What we are explicitly NOT doing

- **Forking Claude Code itself.** Out of scope. Claude Code core is closed
  source; we work around it via hooks, skills, and templates.
- **Replacing Wizard_Erasmus's hook system.** It works; we're tuning the
  loudest losers (cognitive pulse, visual verify, stop chain).
- **Rewriting the Stainless-generated `pkg/cmd/` tree.** Anything that lands
  in this repo lives in `cmd/ant/` (custom main-package files), `internal/kit/`,
  `docs/`, `scripts/`, `templates/`, or `skills/`.

The next doc, [04-distribution-strategy.md](04-distribution-strategy.md),
explains the asset layout, embed strategy, and PowerShell parity.
