# 06 — `C:\Users\Oleh\Documents\GitHub\Antropic\` Folder Survey

A read-only survey of every repo under `C:\Users\Oleh\Documents\GitHub\Antropic\`
on 2026-04-27. Identifies which ones inform Claude Code / Codex automation on
this PC and which are dead weight.

## Inventory

| Repo | Role | Status |
|------|------|--------|
| `anthropic-cli` | This repo. Stainless-generated Go CLI for the Claude Developer Platform. Now also distributes the Wizard Kit (`internal/kit/`) and these docs. | active |
| `claude-code` | Personal fork of the Claude Code TS/Node binary. Contains 13 in-tree plugins (see below). | active |
| `claude-plugins-official` | Personal fork of the Anthropic plugin marketplace source. ~25 plugins. | active |
| `skills` | Upstream mirror of the Agent Skills standard repo (`agentskills.io`). 17 skill packs + `spec/` + `template/`. | upstream mirror |
| `claude-cookbooks` | Upstream mirror of API patterns (RAG, batch, vision, tool-use). | upstream mirror |
| `anthropic-tokenizer-typescript` | Last commit 2024-03. Token counter for legacy models. | **deprecated** — Claude 4.x reports `usage` directly via API. |
| `anthropic-tools` | Last commit 2024-11. Alpha research preview of tool-use. | **deprecated** — superseded by official tool-use API. |

## Per-repo deep-dive

### `claude-code` (fork)

`plugins/` lists 13 in-tree plugins:

```
agent-sdk-dev    code-review        feature-dev          plugin-dev
claude-opus-4-5-migration           hookify              pr-review-toolkit
commit-commands  explanatory-output-style                ralph-wiggum
                 frontend-design    learning-output-style
                                                          security-guidance
```

Most useful for Claude-Code-on-this-PC:

- `agent-sdk-dev/` — examples for building managed agents end-to-end. The
  cleanest reference for the CMA Memory beta surface in `anthropic-cli` v1.3.x.
- `pr-review-toolkit/` — reference design for chained pre-commit / review hooks.
  Pattern matches what `commit_guard_hook` in Wizard_Erasmus enforces.
- `hookify/` — hook authoring patterns; the user already loads `hookify` as a
  plugin into the daily Claude Code session, so this fork is the source.
- `examples/` — runnable end-to-end examples; mine for prompt shapes.

### `claude-plugins-official` (fork)

`plugins/` carries the upstream plugin set (LSPs for clangd, gopls, csharp,
jdtls, kotlin, lua; marketplace tooling — claude-code-setup, claude-md-
management, mcp-server-dev, plugin-dev; output styles — explanatory,
learning).

Concrete leverage: the user's `~/.claude/settings.json` already lists
`enabledPlugins` for many of these. Pinning `claude-plugins-official` as a
**local marketplace** (`.claude/plugin-marketplaces.json`) eliminates the
network round-trip when Claude Code calls `/plugin discover`.

### `skills` (upstream mirror)

`skills/skills/` carries 17 first-party skill packs (algorithmic-art,
brand-guidelines, canvas-design, **claude-api**, doc-coauthoring, docx,
frontend-design, internal-comms, **mcp-builder**, pdf, pptx, **skill-creator**,
slack-gif-creator, theme-factory, web-artifacts-builder, webapp-testing,
xlsx). `spec/` documents the SKILL.md frontmatter contract; `template/` is
boilerplate for new skills.

Worth installing as-is: `claude-api`, `mcp-builder`, `skill-creator` for the
"author new tooling" loop. The document handlers (docx/pdf/pptx/xlsx) are
already covered by the `document-skills` plugin if enabled.

### `claude-cookbooks` (upstream mirror)

Read-only reference. Most relevant for Wizard_Erasmus reasoning tools:
RAG patterns, batch processing, prompt caching, tool-use orchestration. Cite
patterns from here in future Wizard_Erasmus reasoning-tool implementations
rather than rewriting from scratch.

### Deprecated repos

`anthropic-tokenizer-typescript` and `anthropic-tools` should not be wired
into any local routing. The user's stack does not currently reference them;
audit `~/.claude/settings.json`, Wizard_Erasmus `pyproject.toml`, and
PowerShell `profile.ps1` to confirm and document.

## Top ideas distilled

1. **Pin a local plugin marketplace.** Add `.claude/plugin-marketplaces.json`
   pointing at `file://C:/Users/Oleh/Documents/GitHub/Antropic/claude-plugins-official`
   and `.../skills`. `/plugin discover` then resolves locally; offline-safe.
2. **Install three first-party skills system-wide:** `claude-api`,
   `mcp-builder`, `skill-creator`. Slot them into the v0.2 install via the
   existing `internal/kit/payload/skills/` layout (mirror, don't fork).
3. **Treat `claude-code/plugins/agent-sdk-dev` as the reference for managed
   agents** when the user expands beyond hooks into agent-runtime work
   (CMA Memory + Skills beta in anthropic-cli v1.3.x).
4. **Document `claude-cookbooks/` patterns alongside the corresponding
   Wizard_Erasmus reasoning tools** so the proven shape is one click away
   when fixing rough edges.
5. **Remove deprecated repos** from any tooling that still references them
   (none found yet). Add a README footnote in each `Antropic/<deprecated>`
   so a future agent doesn't waste a turn reading them.
