---
name: research-to-md
description: Use this skill when the user wants research-only output ending in a Markdown document, with no code changes. Trigger phrases include "create an md doc with all your findings", "document your findings in an md doc", "don't implement anything yet — just propose changes in md", "write up the research", "produce a research note", "write a design doc". Do NOT trigger when the user wants code edits, or when the request is mainly about reviewing prior conversations (use conversation-forensics).
---

# research-to-md

You are producing a Markdown deliverable from research, with **no code or
config changes**. The risk this skill exists to prevent is drifting into
implementation before the research is on the page.

## Workflow

1. **Confirm research-only mode.** State up front: "Research-only run —
   producing `<path>.md`, no edits to source." If the user wanted edits, they
   will correct you here.

2. **Gather evidence first, conclusions second.**
   - Read what the user pointed at.
   - Use `mcp__wizard__prepare_context` if available to pre-fetch related
     files.
   - For library docs, use `mcp__plugin_context7_context7__query-docs` rather
     than guessing from training data.
   - For prior conversations on this PC, hand off to `conversation-forensics`.

3. **Produce the doc.** Default skeleton:
   ```markdown
   # <Title>
   Date: <YYYY-MM-DD>

   ## Context
   <why this is being asked, what would change with the answer>

   ## Evidence
   <files/commits/transcripts/measurements you actually inspected>

   ## Findings
   <facts derived from evidence; one bullet per fact>

   ## Recommendations
   <ordered, with effort estimate and owner>

   ## Open questions
   ```
   Each fact in Findings cites its evidence (file path, commit hash, line
   number). No claim without backing.

4. **Self-review.** Before saving:
   - Recommendations follow from Findings, not from feel.
   - Effort estimates are XS/S/M/L (≤1 h / ≤½ day / ≤2 days / ≥3 days).
   - No code blocks longer than 25 lines (longer means you're drifting into
     implementation; abstract the snippet).

5. **Save.** Default location: the repo's `docs/research/` folder if it
   exists, else `docs/`, else the user's requested path. Confirm path before
   writing.

6. **Stop.** Output the path and a 3-bullet summary. Do **not** create
   commits, modify settings, or open PRs. Even if the recommendation is
   trivially actionable, leave the action to a follow-up turn.

## Tooling preferences

- `Read`, `Grep`, `Glob` — primary for evidence gathering.
- `WebFetch` / `WebSearch` — only if the user authorised external sources;
  prefer Context7 for library docs.
- `Write` — to the chosen `.md` path only. **Never** to source files in this
  skill.

## Failure modes to avoid

- **Implementing in the same turn.** This skill ends with a saved memo.
- **Conclusions without evidence.** If you couldn't verify it, mark it as an
  Open Question.
- **Burying the lede.** Findings ≠ a wall of text; one fact per bullet.
