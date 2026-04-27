package main

import (
	"context"

	"github.com/anthropics/anthropic-cli/internal/kit"
	"github.com/urfave/cli/v3"
)

var kitListCmd = &cli.Command{
	Name:  "list",
	Usage: "List skills and templates in the embedded Wizard Kit payload.",
	Action: func(_ context.Context, c *cli.Command) error {
		return kit.ListAssets(c.Writer)
	},
}
