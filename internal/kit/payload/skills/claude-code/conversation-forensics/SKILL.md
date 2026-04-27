---
name: conversation-forensics
description: Use this skill to mine prior Claude Code conversations on this PC, find recurring patterns or unfinished threads, and synthesize the findings into a Markdown memo. Trigger when the user says "look at older Claude talks on this PC", "review my recent conversations", "see prior conversations", "mine what I've been doing in Claude", or "produce a summary of recent Claude work". Do NOT trigger for live-runtime inspection (use live-runtime-triage) or single-session continuation (use resume-interrupted-work).
---

# conversation-forensics

You are mining Claude Code transcript artifacts on this Windows PC to produce a
durable Markdown memo.

## Inputs

Primary sources, in this order:

1. `C:\Users\Oleh\.claude\projects\<project>\*.jsonl` — per-project transcripts
   (this is where Claude Code stores session logs).
2. `C:\Users\Oleh\.claude\sessions\` — global session history index.
3. `C:\Users\Oleh\.claude\plans\` — saved plan files.
4. `C:\Users\Oleh\.claude\memory\MEMORY.md` — and any linked memory files.
5. Repo-level `.claude/` folders inside any cwd the user names.

Do **not** read `~/.codex/` from this skill — there is a sibling
`conversation-forensics` for Codex.

## Workflow

1. **Scope.** Confirm the question. Common shapes:
   - "What have I been working on this week?"
   - "Find conversations where X failed."
   - "Recurring pain points across projects."
   If unclear, ask one short question via AskUserQuestion before reading.

2. **Index, don't read whole.** For each `.jsonl`:
   - Use `Get-AIContext` (wizard PowerShell cmdlet) or `Read` with `offset` /
     `limit` to sample first/last 50 lines.
   - Extract the conversation summary line (first user message), timestamp,
     and any error / "please go on" markers.
   - Never load a >5 MB transcript whole into context.

3. **Search, don't reread.** Use `Grep` with patterns specific to the
   question:
   - failures: `(Error|failed|broken|stuck|exception)`
   - continuations: `(go on|continue|resume|please go)`
   - commits: `git commit -m`
   Apply `head_limit: 250` and use `output_mode: "files_with_matches"` first,
   then drill into the top hits.

4. **Cluster.** Group findings by project, by topic, or by time-window —
   whichever the user's question wants. Note frequencies (approximate is
   fine; flag them as such).

5. **Cross-reference.**
   - Plans in `~/.claude/plans/` that match the topic.
   - Memories in `~/.claude/memory/MEMORY.md`.
   - Linked Wizard_Erasmus telemetry reports if the user wants quantitative
     backing.

6. **Synthesize.** Write a Markdown memo. Default sections:
   ```markdown
   # Conversation Forensics — <topic>
   Date: <YYYY-MM-DD>

   ## Scope
   ## Sources
   ## Findings (ranked)
   ## Recommendations
   ## Open questions
   ```
   Save to either `~/.claude/memos/<date>-<slug>.md` (default) or wherever
   the user requested. Do **not** modify any source file.

7. **Stop.** Output the memo path and a 3-bullet summary. Do not implement
   anything from the recommendations — that's for `research-to-md` or another
   workflow.

## Tooling preferences

- `Glob` with concrete patterns (`*.jsonl` not `**/*`) when listing transcripts.
- `mcp__wizard__recall_memories` if available — checks the memory graph
  before reading raw files.
- `mcp__wizard__cross_project_search` if the question spans multiple projects.

## Failure modes to avoid

- **Reading whole transcripts.** They can be 49 MB. Sample, search, drill in.
- **Implementing fixes.** Stay in research mode; emit a memo only.
- **Mixing Codex and Claude data.** This skill reads `~/.claude/` only.
