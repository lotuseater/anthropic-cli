---
name: verified-delivery
description: Use this skill when the user wants the work verified end-to-end and committed in focused diffs. Trigger phrases include "test and fix till it's working", "run and test yourself, till it actually works", "always commit and push verified changes", "make sure it's actually working", "verify and commit", "ship it". Do NOT trigger for research-only work (use research-to-md) or for premature commits before tests pass.
---

# verified-delivery (Codex parity)

Take a working tree, run the tests, prove the change works, split the diff
into focused commits, push.

## Preconditions

The implementation is in place. If the user invokes this but the change
isn't done, complete the implementation first.

## Workflow

1. **Status check.**
   - `git status`, `git diff --stat`.
   - Identify files that should not be committed (`.env`, secrets,
     scratch).

2. **Run the tests.** Prefer `mcp__wizard__smart_test` if available;
   otherwise `Invoke-RepoTest` (wizard PowerShell cmdlet); else the project's
   documented test command (`pytest`, `npm test`, `go test ./...`, etc.).
   - Failing tests → fix in this skill, re-run, cap at 3 rounds.

3. **Lint / format.** `mcp__wizard__smart_lint`, `mcp__wizard__smart_format`.
   Auto-fix non-controversial; flag the rest.

4. **Pre-commit verification.** `mcp__wizard__analyze_git_diff` — required by
   global cognitive-tools rule.

5. **Stage by topic.** No `git add -A`. Group by concern, commit each group
   with a focused message matching repo style (`git log -10`).

6. **Commit message** with the trailer:
   ```
   Co-Authored-By: Codex CLI / Wizard Kit
   ```
   No `--amend` unless the user asks; new commit on hook failure.

7. **Push.**
   - Feature branch with upstream → `git push`.
   - Without upstream → `git push -u origin <branch>`.
   - `main` / `master` → **stop and ask**.

8. **Report.** Commit hashes, push target, non-committed remainder.

## Tooling preferences

- `mcp__wizard__analyze_git_diff` — mandatory before commit.
- `mcp__wizard__smart_test`.
- `mcp__wizard__generate_commit_msg` — draft starter only.

## Failure modes to avoid

- **Committing without running tests.**
- **`--no-verify` to skip pre-commit hooks.**
- **Mega-commit instead of topical splits.**
- **Pushing to main without explicit authorisation.**
