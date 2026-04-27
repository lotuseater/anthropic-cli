---
name: live-runtime-triage
description: Use this skill when the user reports something running on this PC is misbehaving — a stuck launcher, a frozen app, a process that won't exit, a window that won't focus. Trigger phrases include "see currently running", "the app stuck while launching", "launch and capture screenshot", "what's the running instance doing", "why won't this finish". Do NOT trigger for source-only debugging (open the file and read it instead) or for cross-session forensics (use conversation-forensics).
---

# live-runtime-triage (Codex parity)

Inspect live runtime state before touching source. Many failures are
restart, restore, or stale-artifact issues, not code bugs.

## Workflow

1. **Snapshot before changing anything.** No code edits in this turn until
   step 5 at the earliest.

2. **Enumerate processes and windows.**
   - `mcp__wizard__terminal_session_inventory` — wizard-tracked sessions.
   - `mcp__wizard__list_instances` — agent instances.
   - `mcp__wizard__dab_list_windows`, `mcp__wizard__dab_tree`.

3. **Read signals, don't poll.** `Read-WizardSignal` (PowerShell cmdlet)
   carries `process.heartbeat`, `process.exited`, etc. Prefer over OCR.

4. **Capture pixels only if needed.** `dab_screenshot` for one window;
   `dab_ocr` over a region. One capture per question.

5. **Diagnose.** State the hypothesis explicitly before any action.

6. **Recover or hand off.**
   - In-place focus / click / send-keys via `dab_*` — confirm before any
     destructive action.
   - For source fixes: hand off ("Diagnosis complete. Recommended source
     change: <X>. Should I implement?") and wait.

7. **Preserve evidence.** Save to `~/.codex/triage/<timestamp>/`.

## Tooling preferences

- `mcp__wizard__dab_*` — primary; respect bounded budgets.
- `Read-WizardSignal` — replaces OCR-polling.
- `mcp__wizard__detect_impasse` — sanity check.

## Failure modes to avoid

- **OCR-storming.**
- **Killing without diagnosis.**
- **Editing source mid-triage.**
