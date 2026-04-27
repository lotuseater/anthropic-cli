# Wizard Kit hooks

This directory is intentionally a placeholder.

The kit does **not** install fresh Python hook implementations into
`~/.claude/hooks/`. The real hook code lives in
`C:\Users\Oleh\Documents\GitHub\Wizard_Erasmus\src\mcp\` and is invoked via
`py -3.14 -m wizard_mcp.hooks.<name>` directly from `settings.json` matchers
(see `../settings.json.template`).

When Phase 6 of the PowerShell wizard fork lands (persistent Python hook
host over `WizardControlServer`), this directory will gain a thin shim that
routes hook calls through the named-pipe instead of cold spawning Python.
That shim will be added in a follow-up release of the kit.

For now, the only kit-managed change to `~/.claude/hooks/` is the
`settings.json` matcher narrowing — no hook code is copied here.
