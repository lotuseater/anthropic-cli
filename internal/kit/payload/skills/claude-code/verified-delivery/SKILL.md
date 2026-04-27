---
name: verified-delivery
description: Use this skill when the user wants the work verified end-to-end and committed in focused diffs. Trigger phrases include "test and fix till it's working", "run and test yourself, till it actually works", "always commit and push verified changes", "make sure it's actually working", "verify and commit", "ship it". Do NOT trigger for research-only work (use research-to-md) or for premature commits before tests pass.
---

# verified-delivery

You are taking a working tree, running the tests and proving the change
works, splitting the diff into focused commits, and pushing.

## Preconditions

This skill runs **after** the implementation is in place. If the user invokes
it but the change isn't done, complete the implementation first.

## Workflow

1. **Status check.**
   - `git status` — dirty tree expected.
   - `git diff --stat` — get the surface area.
   - Identify any files that should not be committed (`.env`, secrets,
     scratch files).

2. **Run the tests.** Pick the right runner via the wizard PowerShell
   cmdlet: `mcp__wizard__smart_test` if available, else
   `Invoke-RepoTest` from the wizard module. If neither, fall back to the
   project's documented test command (`pytest`, `npm test`, `go test ./...`,
   etc.).
   - If tests fail, **fix them in this skill**. Don't pretend the tree is
     ready when it isn't.
   - Re-run after each fix. Cap at 3 rounds; if still failing, surface the
     failure to the user.

3. **Run the linter / formatter.** `mcp__wizard__smart_lint` and
   `mcp__wizard__smart_format`. Auto-fix what they suggest if it's
   non-controversial; flag the rest for review.

4. **Pre-commit verification.**
   - `mcp__wizard__analyze_git_diff` — required by global cognitive-tools
     rule before any `git commit`.
   - Read its output. If it flags issues, fix them.

5. **Stage by topic.** Don't `git add -A`. Group by concern:
   - tests for one feature → one commit.
   - the implementation → another.
   - generated/build artifacts → only if they belong in the repo.
   Stage each group with `git add <files>` then commit with a focused
   message.

6. **Commit message.** Follow the repo's convention (read `git log -10` to
   match style). Include the Claude trailer:
   ```
   Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
   ```
   No `--amend` unless the user explicitly asks; create a new commit when a
   hook fails.

7. **Push.** Default behaviour:
   - On a feature branch with upstream set: `git push`.
   - On a feature branch without upstream: `git push -u origin <branch>`.
   - On `main` / `master`: **stop and ask** — don't push to a protected
     branch unless the user explicitly authorised it for this turn.

8. **Report.** Output the commit hashes, the push target, and any non-
   committed remainder ("3 modified files left uncommitted: <list>").

## Tooling preferences

- `mcp__wizard__analyze_git_diff` — mandatory before commit.
- `mcp__wizard__smart_test` — preferred test runner.
- `mcp__wizard__generate_commit_msg` — only as a draft starter; revise to
  match repo style.

## Failure modes to avoid

- **Committing without running tests.** This skill verifies; it doesn't
  trust.
- **Skipping pre-commit hooks.** No `--no-verify`. If a hook fails, fix the
  cause.
- **Bundling everything into one mega-commit.** Split by topic.
- **Pushing to main without explicit authorisation.** Pause and ask.
