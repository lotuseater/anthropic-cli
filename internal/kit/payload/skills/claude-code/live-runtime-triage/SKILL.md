---
name: live-runtime-triage
description: Use this skill when the user reports something running on this PC is misbehaving — a stuck launcher, a frozen Team App instance, a process that won't exit, a window that won't focus, an AI agent instance that stopped responding. Trigger phrases include "see currently running", "the app stuck while launching", "launch and capture screenshot", "what's the running instance doing", "why won't this finish". Do NOT trigger for source-only debugging (open the file and read it instead) or for cross-session forensics (use conversation-forensics).
---

# live-runtime-triage

You are inspecting **live runtime state** on this Windows PC before touching
source. The premise: many failures are restart, restore, or stale-artifact
issues, not code bugs.

## Workflow

1. **Snapshot before changing anything.** No code edits in this turn until
   step 5 at the earliest.

2. **Enumerate processes and windows.**
   - `mcp__wizard__terminal_session_inventory` — all wizard-tracked sessions.
   - `mcp__wizard__list_instances` — agent instances.
   - `mcp__wizard__dab_list_windows` — top-level windows.
   - `mcp__wizard__dab_tree` (if a specific window was named) — UI tree.

3. **Read signals, don't poll.** Prefer `Read-WizardSignal` (or
   `mcp__wizard__terminal_session_monitor`) over OCR. The signal bus carries
   `process.heartbeat`, `process.exited`, etc. If the process publishes, read
   from there first.

4. **Capture visible state if needed.** If the question genuinely requires
   pixels (UI claim, dialog text, freeze symptom):
   - `mcp__wizard__dab_screenshot` for the specific window.
   - `mcp__wizard__dab_ocr` over a region, not the whole screen.
   - One capture, not a burst, unless the user asked for time-series.

5. **Diagnose.** State the hypothesis explicitly:
   - "Hypothesis: process P is stuck on I/O because <evidence>."
   - "Hypothesis: window W is unresponsive because <evidence>."
   Do not jump to code edits.

6. **Recover or hand off.**
   - If recoverable in-place (focus, click, send keystroke), do it via
     `dab_smart_click` or `dab_send_keys` — confirm with the user first if
     the action is destructive (close, restart).
   - If the user wants a code fix, **switch out of this skill** explicitly:
     "Diagnosis complete. Recommended source change: <X>. Should I implement?"
     Wait for confirmation before opening source files.

7. **Preserve evidence.** Save screenshots and signal snapshots into
   `~/.claude/triage/<timestamp>/` so the next session can see them. Do not
   delete prior triage runs.

## Tooling preferences

- `mcp__wizard__dab_*` — primary; respect bounded budgets (do not burst-
  capture).
- `Read-WizardSignal` — replaces OCR-polling.
- `mcp__wizard__detect_impasse` — sanity check before declaring stuck.
- `Get-WizardLog -Tail 100` — for processes started via `Invoke-Bounded`.

## Failure modes to avoid

- **OCR-storming the screen.** A single targeted capture beats ten full-
  screen scans.
- **Killing a process to "fix" it without diagnosis.** That's a workaround,
  not a triage outcome. State the hypothesis first.
- **Editing source mid-triage.** Diagnosis ends; the next skill begins.
