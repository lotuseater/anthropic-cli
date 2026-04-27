# 01 — Current State

Snapshot of the Claude Code / Codex automation stack on this PC as of
2026-04-27. This grounds the recommendations in `02-token-waste-audit.md` and
`03-improvement-catalog.md`.

## ~/.claude

```
~/.claude/
├── settings.json              107-line permission allowlist; skipAutoPermissionPrompt=true
├── settings.local.json        empty (no per-project overrides today)
├── rules/
│   ├── cognitive-tools.md     mandates analyze_git_diff before commit, smart_test before stop
│   └── no-premature-stop.md   forbids "shall I"/"should I"; cites 210 stops in 604 conversations
├── memory/
│   └── MEMORY.md              one entry total — the index is essentially empty
├── plans/                     65+ plan files; not indexed
├── projects/                  19 active; largest: Wizard_Erasmus (49 MB), SlavaTask (21 MB)
├── sessions/                  1,096+ history entries
├── skills/                    user-installed + plugin skills
├── hooks/                     thin shims; real logic in Wizard_Erasmus/src/mcp
└── plugins/                   10 installed (clangd, pyright, serena, context7,
                               hookify, skill-creator, mcp-server-dev,
                               ralph-loop, plugin-dev, github)
```

### Hook configuration (settings.json)

| Event | Hook count | Examples |
|-------|------------|----------|
| UserPromptSubmit | 4 | `cognitive_pulse`, `claude_route_hint`, `skill_route_hint`, `large_file_slicing_hint` |
| PreToolUse | 3 | bash commit/bloat guard, cache prefetch (Read/Grep/Glob), edit safety |
| PostToolUse | 7 | `visual_verify`, `session_track`, syntax/quality checks, cache save, error recall/triage |
| Stop | 5 | `test_guard`, `premature_stop`, `flaky_dismissal`, `caveat_hedge`, `quota_handoff` |
| PreCompact | 2 | precompact logic, quota handoff |
| SessionStart | 1 | session initialisation, MagicHat seed, episodic memory recall |

Total hook firings per turn (typical): ~14 cold Python spawns, each
re-importing `wizard_mcp` modules.

## ~/.codex (OpenAI Codex CLI)

```
~/.codex/
├── config.toml                Codex CLI config
├── AGENTS.md                  global instructions
├── history.jsonl              ~488 stored prompts (sampled)
├── sessions/                  per-session JSONL transcripts
└── skills/                    Codex-equivalent skill folder (sparse today)
```

The skill opportunities doc (`Wizard_Erasmus/research/codex_skill_opportunities_2026_04_21.md`)
identified five recurring workflow shapes from `history.jsonl` totalling ~313
prompts in repeated families. None of the proposed skills had landed at the
time of the audit.

## C:\Users\Oleh\Documents\GitHub\PowerShell (wizard fork)

Fork of `microsoft/PowerShell` on branch `wizard_power_shell`, three local
commits past upstream master.

| Component | Status |
|-----------|--------|
| Phase 0–1: UTF-8 hardening, PSReadLine gate, native error preferences | shipped |
| Phase 2: Wizard module skeleton, `Get-WizardSession` | shipped |
| Phase 3: `Invoke-Bounded` (16 KB / 80 lines / 120 s defaults) | shipped |
| Phase 4: signal bus (`Publish-WizardSignal`, `Read-WizardSignal`, `Start-MonitoredProcess`) | shipped |
| Phase 5: bash-compat translator (`Invoke-BashCompat`, `bash`/`sh` aliases) | shipped |
| Phase 6: persistent Python hook host (warm child over named-pipe) | **not shipped** |
| Phase 7+: installer (`Install-WizardPwsh.ps1`), repo AI contract templates | partial |

**Activation:** `WIZARD_PWSH_CONTROL=1`. Without the env var the fork behaves
identically to upstream.

**Cmdlets exposed (23):** session info, bounded execution, signal bus, bash
compat, idempotency locks, AI search (`Find-Code`, `Find-Repos`,
`Find-CodeAcrossRepos`, `Get-AIContext`), repo profiling (`Get-RepoProfile`,
`Invoke-RepoBuild`, `Invoke-RepoTest`), digests (`Update-RepoDigest`).

## C:\Users\Oleh\Documents\GitHub\Wizard_Erasmus (MCP server)

156 Python modules under `src/mcp/`. Key files:

| File | Size | Role |
|------|------|------|
| `wizard_mcp_server.py` | 392 KB | main MCP entrypoint, 90+ tools |
| `wizard_db.py` | 214 KB | shared SQLite state: instances, messages, queues |
| `cognitive_pulse_hook.py` | 58 KB | <200 ms micro-workflow on UserPromptSubmit |
| `tool_cache.py` | 55 KB | global cross-project cache, 1,230 reuses / window |
| `commit_guard_hook.py` | 20 KB | structural blocker before `git commit` |
| `pretool_cache_hook.py` | — | cache lookup before model sees prompt |
| `edit_invalidate_hook.py` | — | cache cleanup on Edit/Write |
| `visual_verify_hook.py` | — | OCR scan after Bash / `dab_*` / `smart_*` |
| `claude_cli_wrapper.py` | 105 KB | bridges Claude Code CLI to wizard tools |

## MCP servers loaded into Claude Code

- `wizard` — 200+ tools (`mcp__wizard__*`).
- `serena` — symbolic code analysis.
- `context7` — library docs lookup.
- `claude_ai_Google_Drive` — Drive integration.
- `mempalace` — memory palace storage.

## Plugins loaded into Claude Code

`clangd`, `pyright`, `serena`, `context7`, `hookify`, `skill-creator`,
`mcp-server-dev`, `ralph-loop`, `plugin-dev`, `github`.

## Health summary

The infrastructure is mature but under-tuned. Hooks fire correctly but most
operate in advisory mode (low conversion). The Read-path cache works; the
Grep/Glob/Bash paths are at near-zero hit rate. The memory index is empty.
Plan files accumulate without summary.

The next doc, [02-token-waste-audit.md](02-token-waste-audit.md), quantifies
the cost of these gaps.
