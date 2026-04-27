package main

import (
	"context"
	"errors"
	"os"

	"github.com/anthropics/anthropic-cli/internal/kit"
	"github.com/urfave/cli/v3"
)

var kitInstallCmd = &cli.Command{
	Name:  "install",
	Usage: "Install the Wizard Kit into ~/.claude and/or ~/.codex.",
	Description: "Merges settings.json / config.toml deltas, appends managed " +
		"sections to CLAUDE.md / AGENTS.md, copies skills, and rebuilds plan/session " +
		"indexes. Existing files are backed up to .wizard-kit-backup/<timestamp>/ " +
		"before any write. Idempotent.",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "dry-run", Usage: "Show what would change. No writes."},
		&cli.BoolFlag{Name: "claude-only", Usage: "Skip ~/.codex."},
		&cli.BoolFlag{Name: "codex-only", Usage: "Skip ~/.claude."},
	},
	Action: func(_ context.Context, c *cli.Command) error {
		if c.Bool("claude-only") && c.Bool("codex-only") {
			return errors.New("cannot pass --claude-only and --codex-only together")
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return kit.Install(home, kit.Options{
			DryRun:     c.Bool("dry-run"),
			ClaudeOnly: c.Bool("claude-only"),
			CodexOnly:  c.Bool("codex-only"),
			Out:        c.Writer,
		})
	},
}
