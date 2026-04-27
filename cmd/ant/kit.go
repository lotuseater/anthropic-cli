// Wizard Kit subcommand registration. This file is hand-written and is not
// part of the Stainless-generated surface. It attaches the `kit` command to
// the generated `cmd.Command` via init() — Go init order guarantees the
// imported `pkg/cmd` package's init runs first, so cmd.Command is fully
// constructed by the time this init fires.

package main

import (
	"github.com/anthropics/anthropic-cli/pkg/cmd"
	"github.com/urfave/cli/v3"
)

func init() {
	cmd.Command.Commands = append(cmd.Command.Commands, kitCommand)
}

var kitCommand = &cli.Command{
	Name:     "kit",
	Category: "WIZARD KIT",
	Usage:    "Install and manage the Wizard Kit (skills, templates, settings deltas) for Claude Code and Codex CLI.",
	Description: "The Wizard Kit ships a curated bundle of skills, settings deltas, and " +
		"plan/session indexes designed to reduce token usage and improve automation " +
		"reliability. See docs/wizard/ for the full design and rationale.",
	Suggest: true,
	Commands: []*cli.Command{
		kitListCmd,
		kitInstallCmd,
		kitRollbackCmd,
	},
}
