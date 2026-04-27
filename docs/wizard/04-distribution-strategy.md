# 04 — Distribution Strategy

How the kit ships, where the assets live, and how the two installers (Go +
PowerShell) stay in sync.

## Why this repo

`anthropic-cli` is the natural home for distribution because:

1. It already builds and ships a Windows `ant.exe` (Stainless-generated Go
   binary, GoReleaser, Homebrew tap, deb/rpm/apk).
2. Its release pipeline (notarisation, signing) is solid as of v1.3.2 — we
   inherit it for free.
3. The kit payload is small (KBs of skills + templates) so embedding it in
   the binary is trivial.

The kit does **not** modify the Stainless-generated wrapper. It adds a
sibling subcommand and a parallel PowerShell installer.

## Asset layout (single source of truth)

```
anthropic-cli/
├── internal/kit/
│   ├── embed.go              //go:embed all:payload
│   ├── install.go            install logic (used by ant kit + tests)
│   ├── rollback.go           rollback logic
│   ├── merge.go              JSON / TOML / Markdown merge helpers
│   ├── index.go              MEMORY.md / AGENTS.md plan-index builder
│   └── payload/
│       ├── skills/
│       │   ├── claude-code/
│       │   │   ├── conversation-forensics/SKILL.md
│       │   │   ├── resume-interrupted-work/SKILL.md
│       │   │   ├── research-to-md/SKILL.md
│       │   │   ├── live-runtime-triage/SKILL.md
│       │   │   └── verified-delivery/SKILL.md
│       │   └── codex/                        # parallel bodies, Codex-specific
│       │       ├── conversation-forensics/SKILL.md
│       │       ├── resume-interrupted-work/SKILL.md
│       │       ├── research-to-md/SKILL.md
│       │       ├── live-runtime-triage/SKILL.md
│       │       └── verified-delivery/SKILL.md
│       └── templates/
│           ├── claude-code/
│           │   ├── settings.json.template
│           │   ├── CLAUDE.md.template
│           │   ├── MEMORY.md.template
│           │   ├── hooks/                    # symlink-friendly stubs
│           │   └── per-project/{php,cpp,python}/.claude/settings.local.json
│           └── codex/
│               ├── config.toml.template
│               └── AGENTS.md.template
├── cmd/ant/
│   ├── main.go               (untouched, Stainless-generated)
│   ├── kit.go                NEW — registers `kit` command on cmd.Command
│   ├── kit_install.go        NEW
│   ├── kit_rollback.go       NEW
│   ├── kit_list.go           NEW
│   └── kit_dry_run.go        NEW
├── scripts/
│   └── install-claude-kit.ps1  PowerShell installer (reads internal/kit/payload/)
└── docs/wizard/                this folder
```

**Decision: assets live under `internal/kit/payload/`, not at repo root.**
Reasoning:

- Go's `//go:embed` directive only embeds files in or below the source file's
  directory. Putting `embed.go` in `internal/kit/` and assets in
  `internal/kit/payload/` avoids the workaround of declaring a package at
  module root just to satisfy embed.
- The PowerShell installer reads from the same path
  (`$RepoRoot/internal/kit/payload/...`), so both consumers see one tree.
- The plan summary mentioned `templates/claude-code/` at repo root; that was
  flexibility for the asset path. Moving it under `internal/kit/payload/`
  preserves the same logical layout and is more idiomatic Go.

## Go binary path: `ant kit ...`

### Registration without touching generated code

`cmd/ant/main.go` is Stainless-generated. We avoid editing it. Instead, a
new file `cmd/ant/kit.go` (in the same `package main`) declares an `init()`
function that mutates `cmd.Command.Commands`:

```go
// cmd/ant/kit.go — NOT generated
package main

import (
    "github.com/anthropics/anthropic-cli/internal/kit"
    "github.com/anthropics/anthropic-cli/pkg/cmd"
    "github.com/urfave/cli/v3"
)

func init() {
    cmd.Command.Commands = append(cmd.Command.Commands, kitCommand)
}

var kitCommand = &cli.Command{
    Name:     "kit",
    Category: "WIZARD KIT",
    Usage:    "Install / manage the Wizard automation kit for Claude Code and Codex CLI",
    Commands: []*cli.Command{
        kitListCmd,
        kitInstallCmd,
        kitRollbackCmd,
    },
}
```

Go init order guarantees: `pkg/cmd` is imported by `cmd/ant`, so
`pkg/cmd.init()` (which sets `cmd.Command`) runs before any `init()` in
`package main`. Our `init()` therefore sees a fully-formed `cmd.Command` and
appends to it. No race, no need to modify `main.go`.

### Subcommands

| Command | Action |
|---------|--------|
| `ant kit list` | enumerate skills + templates in the embedded payload |
| `ant kit install [--claude-only \| --codex-only] [--dry-run] [--force]` | install kit into `~/.claude/` and/or `~/.codex/` |
| `ant kit rollback` | restore latest backup |

### Embed

```go
// internal/kit/embed.go
package kit

import "embed"

//go:embed all:payload
var Payload embed.FS
```

The `all:` prefix includes hidden files (e.g., `.claude/settings.local.json`
under `per-project/`).

### Install flow

1. Determine target dirs:
   - Claude Code: `os.UserHomeDir() + "/.claude"`.
   - Codex: `os.UserHomeDir() + "/.codex"`.
2. Skip targets per `--claude-only` / `--codex-only`.
3. Snapshot existing files into `<target>/.wizard-kit-backup/<timestamp>/`.
4. Walk embedded payload; for each file:
   - `*.template` → merge into target (JSON three-way for `settings.json`,
     TOML merge for `config.toml`, append-only for `*.md`).
   - SKILL.md / hook stub → copy to target with `0644` perms.
5. Build plan/conversation index, write into MEMORY.md (Claude) /
   AGENTS.md (Codex) under delimited section.
6. Print summary: N files added, M merged, K backed up.

### Rollback flow

1. Read `<target>/.wizard-kit-backup/` to find latest timestamp.
2. For each file in that snapshot, restore over the current target.
3. Delete kit-installed files that did not exist in the backup (tracked via
   `<target>/.wizard-kit-manifest.json` written at install time).
4. Print summary.

## PowerShell parity path

`scripts/install-claude-kit.ps1` uses the same `internal/kit/payload/`
directory and the same backup/manifest scheme. It is the path users hit
when they're working in this repo's clone (they can run the script without
rebuilding `ant`).

### Why ship both

- **Go binary:** users on a fresh PC who installed `ant` via Homebrew /
  GoReleaser-built installer. They get the kit without cloning this repo.
- **PowerShell script:** Windows-first users (Codex / Claude on Windows
  Terminal). They already have the wizard fork (`WIZARD_PWSH_CONTROL=1`) on
  PATH. PowerShell-native operations (symlinks, `Compare-Object`, three-way
  merge with `pwsh`'s built-in cmdlets) are simpler to read and audit.

### Manifest

`<target>/.wizard-kit-manifest.json` records, per install:

```json
{
  "version": "0.1.0",
  "timestamp": "2026-04-27T12:34:56Z",
  "installed_by": "ant kit install" | "install-claude-kit.ps1",
  "files_added":   ["skills/conversation-forensics/SKILL.md", ...],
  "files_merged":  ["settings.json", "CLAUDE.md", ...],
  "files_backed_up": ["settings.json", ...]
}
```

Both installers read and write the same file format. `ant kit rollback` and
`-Rollback` produce byte-identical results on the same starting state.

## Versioning

- Kit version is independent of `ant` version.
- Stored in `internal/kit/version.go` and `scripts/install-claude-kit.ps1`
  header.
- v0.x while in `wizard-kit-v0` branch; bumps to v1.0 once Phase 6 hook host
  is dogfooded for a week.

## Distribution channels

| Channel | Status |
|---------|--------|
| Local clone of this repo + `pwsh -File scripts/install-claude-kit.ps1 -Install` | works after Phase 4 lands |
| `ant kit install` after `go build ./cmd/ant` | works after Phase 5 lands |
| `ant kit install` after Homebrew install of `ant` | works once a release tagged with kit-bearing branch is published |
| Standalone `wizard-kit-v0.zip` in GitHub Releases | not in scope for this branch |

The next doc, [05-roadmap.md](05-roadmap.md), schedules the work.
