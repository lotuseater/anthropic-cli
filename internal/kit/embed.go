package kit

import "embed"

// Payload is the embedded Wizard Kit asset tree. Contains:
//   - skills/claude-code/<skill>/SKILL.md
//   - skills/codex/<skill>/SKILL.md
//   - templates/claude-code/{settings.json.template, CLAUDE.md.template, MEMORY.md.template, hooks/, per-project/}
//   - templates/codex/{config.toml.template, AGENTS.md.template}
//
// The same tree is consumed by scripts/install-claude-kit.ps1 (which reads
// from internal/kit/payload/ on disk). The Go binary uses the embedded copy
// so `ant kit install` works without the source tree being present.
//
//go:embed all:payload
var Payload embed.FS
