---
name: conversation-forensics
description: Use this skill to mine prior Codex conversations on this PC and synthesize the findings into a Markdown memo. Trigger when the user says "look at older Codex talks on this PC", "review my recent Codex conversations", "see prior conversations", "mine what I've been doing in Codex", or "produce a summary of recent Codex work". Do NOT trigger for live-runtime inspection (use live-runtime-triage) or single-session continuation (use resume-interrupted-work).
---

# conversation-forensics (Codex parity)

You are mining Codex CLI transcript artifacts on this Windows PC to produce a
durable Markdown memo.

## Inputs

Primary sources, in this order:

1. `C:\Users\Oleh\.codex\history.jsonl` — flat prompt history (~488 entries
   sampled in the original audit).
2. `C:\Users\Oleh\.codex\sessions\` — per-session JSONL transcripts.
3. Repo-level `.codex/` and `.agents/` folders inside any cwd the user names.
4. `C:\Users\Oleh\.codex\AGENTS.md` — global instructions (read for context,
   never modify in this skill).

Do **not** read `~/.claude/` from this skill — there is a sibling
`conversation-forensics` for Claude Code.

## Workflow

1. **Scope.** Confirm the question (single-line clarification only if
   ambiguous). Common shapes:
   - "What have I been working on this week in Codex?"
   - "Find Codex conversations where the wrapper failed."
   - "Recurring keyword families in `history.jsonl`."

2. **Index, don't read whole.** For each `.jsonl`:
   - Use `Get-AIContext` (wizard PowerShell cmdlet) — it slices large files
     by line range with line numbers.
   - Sample first/last 50 lines and search; do not load >5 MB whole.

3. **Search by family.** Keyword passes the original audit used:
   - `(fix|broken|failed|stuck)`
   - `(go on|continue|resume)`
   - `(commit|push)`
   - `(plan|research|study|document findings)`
   - `(test|verify|check)`
   - visual / live-app inspection prompts
   Apply `output_mode: "files_with_matches"` first then drill into top hits.

4. **Cluster and count.** Approximate counts are fine; flag them as such.

5. **Cross-reference.**
   - Wizard_Erasmus telemetry reports if the user wants quantitative backing.
   - Repo notes / docs in any cwd the user mentions.

6. **Synthesize.** Markdown memo with sections: Scope, Sources, Findings,
   Recommendations, Open questions. Save to `~/.codex/memos/<date>-<slug>.md`
   (default) or where the user requested.

7. **Stop.** Output the memo path and a 3-bullet summary. No source edits in
   this skill.

## Tooling preferences

- `Get-AIContext`, `Find-Code`, `Find-CodeAcrossRepos` (wizard PowerShell
  cmdlets) — they handle large files and bound match counts.
- `mcp__wizard__cross_project_search` if available across the wizard MCP
  bridge.

## Failure modes to avoid

- **Reading whole transcripts.**
- **Implementing fixes mid-skill.**
- **Mixing Claude and Codex data.** This skill reads `~/.codex/` only.
