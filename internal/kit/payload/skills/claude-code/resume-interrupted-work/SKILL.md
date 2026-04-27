---
name: resume-interrupted-work
description: Use this skill when the user wants to continue a Claude Code task that was interrupted by a reboot, network drop, context handoff, or simple time gap — possibly in a different working directory than the current cwd. Trigger phrases include "continue where I stopped", "resume the work from yesterday", "the latest Claude session, possibly in a different repo", "pick up where we left off". Do NOT trigger for fresh tasks or for cross-project research summaries (use conversation-forensics for those).
---

# resume-interrupted-work

You are reconstructing the most recent unfinished Claude Code thread and
restoring enough context to continue without re-asking the user.

## Workflow

1. **Locate candidates.** Use `mcp__wizard__terminal_session_inventory` if
   available; otherwise scan:
   - `C:\Users\Oleh\.claude\projects\` — find `.jsonl` with mtime in the last
     72 h, sorted by mtime descending.
   - `C:\Users\Oleh\.claude\plans\` — plans with mtime in the same window.

2. **Rank.** For each candidate session, score:
   - mtime (most recent wins, but only by 0.3× weight).
   - has open in-progress task list at the end of the transcript.
   - last message is a tool error, a user "ok", or an assistant mid-thought
     (`incomplete: true` flag in JSONL).
   - matches keywords from the user's resume request (if any given).
   Combine into one ranking; surface the top 3.

3. **Pick.** If one candidate is strongly ahead, proceed silently. If two are
   close, ask the user via AskUserQuestion (single multi-select-style prompt
   with the top 2-3 options).

4. **Restore context.** Read the **last** ~150 messages of the chosen
   transcript using `Read` with `offset`. Extract:
   - The active TODO / task list at the end.
   - Files most recently edited.
   - Last failing command + its error.
   - Any plan file the session was working from.

5. **Resume.** Open the last-edited files. If the cwd of the interrupted
   session differs from the current cwd, surface that to the user before any
   tool-call that would write — let them decide whether to switch.

6. **Continue.** Pick up where the prior session stopped. Do not re-execute
   already-completed work; check the transcript to see what landed before
   redoing.

## Tooling preferences

- `mcp__wizard__terminal_session_inventory` — preferred if available.
- `mcp__wizard__terminal_session_handoff` — to formally mark a session as
  resumed and prevent two threads from racing.
- `Glob` with `mtime` sort hints (this is what the Glob tool returns by
  default).
- `Get-WizardSession` — confirms the wizard session id; useful for cross-
  referencing.

## Failure modes to avoid

- **Resuming the wrong session** because mtime alone was used. Score on
  state, not just time.
- **Re-doing completed steps.** Always read the tail of the transcript first.
- **Silently switching cwd.** Tell the user when their resume target lives
  somewhere else.
