---
name: research-to-md
description: Use this skill when the user wants research-only output ending in a Markdown document, with no code changes. Trigger phrases include "create an md doc with all your findings", "document your findings in an md doc", "don't implement anything yet — just propose changes in md", "write up the research", "produce a research note", "write a design doc". Do NOT trigger when the user wants code edits, or when the request is mainly about reviewing prior conversations (use conversation-forensics).
---

# research-to-md (Codex parity)

Produce a Markdown deliverable from research, with **no code or config
changes**.

## Workflow

1. **Confirm research-only mode.** State up front: "Research-only run —
   producing `<path>.md`, no edits."

2. **Gather evidence first.** Read what's pointed at; for library docs use
   `mcp__plugin_context7_context7__query-docs` if available, else inspect the
   library's installed source. Avoid guessing from training data.

3. **Produce the doc** with sections: Context, Evidence, Findings,
   Recommendations, Open questions. Each fact in Findings cites its evidence
   (file path, commit, line).

4. **Self-review.** Recommendations follow from Findings; effort sized as
   XS/S/M/L; no code blocks longer than 25 lines.

5. **Save.** Default: `docs/research/` if it exists, else `docs/`, else the
   user's path. Confirm before writing.

6. **Stop.** Output the path and a 3-bullet summary. No commits, no settings
   changes.

## Tooling preferences

- `Read`, `Grep`, `Glob`.
- `Find-Code`, `Get-AIContext` — wizard PowerShell cmdlets for large files.
- `WebFetch` / `WebSearch` only if user authorised; prefer Context7 for
  libraries.

## Failure modes to avoid

- **Implementing in the same turn.**
- **Conclusions without evidence.**
- **Burying the lede** — one fact per bullet.
