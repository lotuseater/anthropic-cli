# Wizard Docs — Claude Code & Codex Automation Kit

This folder documents the **Wizard Kit** that ships from this repo: a small
distributable bundle of skills, hooks, config templates, and an installer
designed to reduce token usage and improve automation reliability when running
Claude Code (and OpenAI Codex CLI) on this PC.

The kit is **not** the `ant` CLI itself. `ant` is a thin Stainless-generated
wrapper for the Anthropic Developer Platform API. The kit reuses this repo as
a distribution channel: it ships an embedded payload of skills/templates and
two installers (PowerShell + an in-tree `ant kit` Go subcommand).

## Contents

| File | Purpose |
|------|---------|
| [01-current-state.md](01-current-state.md) | Snapshot of what's already installed under `~/.claude/`, `~/.codex/`, the PowerShell wizard fork, and Wizard_Erasmus. |
| [02-token-waste-audit.md](02-token-waste-audit.md) | Eight observed token-waste patterns with telemetry numbers and transcript evidence. |
| [03-improvement-catalog.md](03-improvement-catalog.md) | Ten ranked levers with problem / proposed change / saving / effort / owner repo. |
| [04-distribution-strategy.md](04-distribution-strategy.md) | Why this repo is the distribution home, asset layout (`internal/kit/payload/`), Go embed approach, and PowerShell parity. |
| [05-roadmap.md](05-roadmap.md) | Four-week phased rollout: pulse-as-signal → hook host → skill pack → distribution package. |
| [06-antropic-folder-survey.md](06-antropic-folder-survey.md) | Inventory of the seven repos under `C:\Users\Oleh\Documents\GitHub\Antropic\` and what to do with each. |
| [07-openai-agents-sdk-patterns.md](07-openai-agents-sdk-patterns.md) | Translation report from `openai-agents-python` to this stack: 8 ranked patterns to port + 5 NOT to port + file pointers. |
| [08-capability-gap-map.md](08-capability-gap-map.md) | 18 capability gaps vs. an agent-runtime feature set, with status / owner / effort / dependencies. |
| [09-roadmap-v2.md](09-roadmap-v2.md) | Six-week v0.2 roadmap layered on `05-roadmap.md`: handoffs → RunState/HITL → tracing → sessions → distribution. |

## Companion docs (other repos)

The numbers and concepts here are grounded in research from two sister repos.
Read alongside:

- `C:\Users\Oleh\Documents\GitHub\PowerShell\docs\wizard\` — wizard PowerShell
  fork: `WizardControlServer`, cmdlets, Phase 6 hook host spec.
  - `RESEARCH.md` — friction signals, hook-spawn cost, agent-ecosystem map.
  - `PLAN.md` — phased work plan; Phase 6 (persistent Python hook host) is the
    biggest unrealized win.
  - `USAGE.md` — full cmdlet API and signal contract.
- `C:\Users\Oleh\Documents\GitHub\Wizard_Erasmus\research\` — telemetry and skill
  opportunity research.
  - `codex_skill_opportunities_2026_04_21.md` — origin of the five-skill batch.
  - `Hooks_Tools_Skills_Usage_Report_2026_04_19.md` — 99.9% miss rate on
    `cognitive_pulse`; structural vs. advisory effectiveness.
  - `Cache_Effectiveness_Report_2026_04_19.md` — 19.9% cache hit rate; Read-path
    healthy, Grep/Glob/Bash at ~0%.
  - `agent_token_efficiency_research_2026_04_24.md` — token waste comes from
    rediscovery, not single large prompts.
  - `claude_repetitive_tasks_stats_2026_04_27.md` — top families:
    `unclassified` (24.9%), `team_app_orchestration` (22%).

## How the kit installs

Two equivalent paths. Both read assets from `internal/kit/payload/`.

```powershell
# PowerShell route (works without rebuilding the binary)
pwsh -File scripts/install-claude-kit.ps1 -DryRun
pwsh -File scripts/install-claude-kit.ps1 -Install
pwsh -File scripts/install-claude-kit.ps1 -Rollback
```

```bash
# Go binary route (after `go build ./cmd/ant`)
ant kit list
ant kit install --dry-run
ant kit install
ant kit rollback
```

Both honour `--claude-only` / `--codex-only` (`-ClaudeOnly` / `-CodexOnly`) to
limit scope.

## Status

Phase 1–5 land in branch `wizard-kit-v0`. Cross-repo work (Phase 6: persistent
Python hook host in the PowerShell fork; signal-bus migration of cognitive
pulse and visual verify in Wizard_Erasmus) is **described** here and **executed
elsewhere**. See [05-roadmap.md](05-roadmap.md) for the schedule.
