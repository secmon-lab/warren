package cli

import "github.com/urfave/cli/v3"

func cmdTool() *cli.Command {
	var helpers []*cli.Command
	for _, tool := range tools {
		helper := tool.Helper()
		if helper != nil {
			helpers = append(helpers, helper)
		}
	}

	return &cli.Command{
		Name:     "tool",
		Usage:    "Call tool helper",
		Commands: helpers,
	}
}
