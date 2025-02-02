package cli

import "github.com/urfave/cli/v3"

func joinFlags(flags ...[]cli.Flag) []cli.Flag {
	var result []cli.Flag
	for _, flag := range flags {
		result = append(result, flag...)
	}
	return result
}
