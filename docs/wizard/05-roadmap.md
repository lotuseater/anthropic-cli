# 05 — Roadmap

Four-week phased rollout. Cross-repo work is flagged with the owning repo;
items in **this repo** ship in branch `wizard-kit-v0`.

## Week 1 (this branch)

Goal: land the kit infrastructure + the cheapest token wins.

| Item | Lever # | Owner | Status in `wizard-kit-v0` |
|------|---------|-------|----------------------------|
| `docs/wizard/*` (six docs) | — | this repo | shipping |
| Skill pack (5 × 2 CLIs) | 3 | this repo | shipping |
| Settings + CLAUDE.md + MEMORY.md templates | 4, 5, 8 | this repo | shipping |
| `scripts/install-claude-kit.ps1` | 10 | this repo | shipping |
| `ant kit list / install / rollback` Go subcommand | 10 | this repo | shipping |

**Dogfood:** install on this PC via `ant kit install --dry-run` then
`ant kit install`. Verify `~/.claude/skills/<5 new>` symlinks resolve, MEMORY.md
contains the plan index, and `ant kit rollback` restores the previous state
byte-for-byte.

**Verification target:**
- `go build ./...` green.
- `pwsh -File scripts/install-claude-kit.ps1 -DryRun` reports clean diff.
- After install, `claude --version` and `codex --version` both still launch.
- A trivial new Claude Code session loads without errors and the new skills
  appear in the skill list.

## Week 2 (PowerShell fork)

Goal: persistent Python hook host (Lever #1, biggest latency win).

| Item | Owner | Notes |
|------|-------|-------|
| Add `hook.register / invoke / list / unregister` verbs to `WizardControlServer` | PowerShell fork | spec: `PowerShell/docs/wizard/PLAN.md` §6 |
| Implement warm Python child process | PowerShell fork | new file `src/Modules/Shared/Microsoft.PowerShell.Wizard/HookHost.ps1` |
| Migrate `pretool_cache_hook` as pilot | Wizard_Erasmus | `src/mcp/hook_host.py` |
| Telemetry: 48 h baseline + post-migration | Wizard_Erasmus | re-run `Hooks_Tools_Skills_Usage_Report` |

**Verification target:**
- Cold spawn rate <2 / turn (down from ~14).
- `pretool_cache_hook` p50 latency <50 ms (down from ~500 ms).
- No regression in cache hit rate.

## Week 3 (Wizard_Erasmus)

Goal: signal-bus migration of cognitive pulse + visual verify (Levers #2 + #4).

| Item | Owner |
|------|-------|
| Rewrite `cognitive_pulse_hook.py` to publish to `cognitive.pulse` topic; emit one-line pointer in prompt | Wizard_Erasmus |
| Add `recall_cognitive_pulse` MCP tool / skill | Wizard_Erasmus |
| Narrow `visual_verify_hook.py` matcher | Wizard_Erasmus |
| Stop guard chain short-circuit (Lever #9) | Wizard_Erasmus |
| Grep/Glob hash cache (Lever #6) | Wizard_Erasmus |

**Verification target:**
- Cognitive pulse miss rate <70% (down from 99.9%).
- Visual-verify scan count drops 50%+ on a 24 h sample.
- Cache hit rate 30%+ (up from 19.9%).

## Week 4 (this repo)

Goal: distribution polish + token-output compaction proposal.

| Item | Owner |
|------|-------|
| Tag `kit-v0.1.0`; ship via GoReleaser to Homebrew tap | this repo |
| Update top-level README with `ant kit install` quick-start | this repo |
| Write Lever #7 (tool-output compaction in history) RFC and submit upstream | this repo + Claude Code core |
| Telemetry rerun: full 8-pattern audit; archive in `Wizard_Erasmus/research/` | Wizard_Erasmus |

**Verification target:**
- A second PC (or fresh user account on this PC) installs the kit via
  `brew install anthropics/tap/ant && ant kit install` and reaches a working
  state without manual fixup.
- Aggregate token saving across the 8 patterns: 25–35% on a representative
  workload (defined as: 2 h of mixed read/edit/build/test/commit work).

---

## Decision log (kept here for posterity)

- **2026-04-27.** Decided: assets live under `internal/kit/payload/` (not
  repo root). Go embed simplicity wins; PowerShell installer reads same path.
- **2026-04-27.** Decided: `ant kit` registers via `init()` in
  `cmd/ant/kit.go`, mutating `cmd.Command.Commands`. Avoids editing
  Stainless-generated `main.go` / `cmd.go`.
- **2026-04-27.** Decided: ship Codex parity in v0.1. Both `~/.claude/skills/`
  and `~/.codex/skills/` populated; both `MEMORY.md` and `~/.codex/AGENTS.md`
  get plan/conversation indexes.
- **2026-04-27.** Decided: branch `wizard-kit-v0`, push to origin, no PR auto-
  created (user opens it).

## Out of scope

- Forking Claude Code core.
- A standalone `wizard-kit` repo (the kit lives here as long as it's small;
  re-evaluate at >500 KB payload).
- Codex-specific MCP tooling (Codex parity here is **skill bodies + config
  templates only** — no Codex-specific runtime helpers).
- iOS / Android / non-x64 distribution (the kit is Windows-first; Linux/macOS
  via the existing GoReleaser pipeline once tested).
