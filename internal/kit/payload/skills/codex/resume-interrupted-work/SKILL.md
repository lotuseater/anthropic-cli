---
name: resume-interrupted-work
description: Use this skill when the user wants to continue a Codex CLI task interrupted by a reboot, network drop, context handoff, or simple time gap — possibly in a different working directory than the current cwd. Trigger phrases include "continue where I stopped in Codex", "resume the work from yesterday", "the latest Codex session may be in a different repo", "pick up where we left off". Do NOT trigger for fresh tasks or for cross-project research summaries (use conversation-forensics).
---

# resume-interrupted-work (Codex parity)

You are reconstructing the most recent unfinished Codex CLI thread.

## Workflow

1. **Locate candidates.**
   - `C:\Users\Oleh\.codex\sessions\` — `.jsonl` files with mtime in the
     last 72 h, sorted descending.
   - `C:\Users\Oleh\.codex\history.jsonl` — last ~30 prompts, scan for
     interrupted-style endings.

2. **Rank.** Score each candidate by:
   - mtime (recent wins, 0.3× weight).
   - last entry is a tool error / "please go on" / mid-step.
   - matches user's resume keywords if any.
   Surface the top 3.

3. **Pick.** If one is clearly ahead, proceed silently. If two are close,
   one short clarifying question.

4. **Restore context.** Read the last ~150 messages of the chosen session.
   Extract the active task list, files most recently edited, last failing
   command + its error.

5. **Resume.** Open the last-edited files via your standard Codex flow.
   - If the cwd of the interrupted session differs from the current cwd,
     surface this to the user before any write.

6. **Continue.** Don't re-execute completed steps; check the transcript
   tail before redoing.

## Tooling preferences

- `mcp__wizard__terminal_session_inventory` — preferred if available.
- `mcp__wizard__terminal_session_handoff` — to mark the resumed session.
- `Get-WizardSession` — confirms wizard session id.
- `Get-AIContext` — to slice the transcript without re-reading whole.

## Failure modes to avoid

- **Resuming the wrong session** because mtime alone was used.
- **Re-doing completed steps** — read the tail first.
- **Silently switching cwd.**
