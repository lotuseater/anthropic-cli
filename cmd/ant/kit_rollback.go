package main

import (
	"context"
	"errors"
	"os"

	"github.com/anthropics/anthropic-cli/internal/kit"
	"github.com/urfave/cli/v3"
)

var kitRollbackCmd = &cli.Command{
	Name:  "rollback",
	Usage: "Restore the most recent Wizard Kit backup snapshot.",
	Description: "Reads .wizard-kit-backup/<timestamp>/ under each enabled target " +
		"and restores files over the current state.",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "dry-run", Usage: "Show what would restore. No writes."},
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
		return kit.Rollback(home, kit.Options{
			DryRun:     c.Bool("dry-run"),
			ClaudeOnly: c.Bool("claude-only"),
			CodexOnly:  c.Bool("codex-only"),
			Out:        c.Writer,
		})
	},
}
