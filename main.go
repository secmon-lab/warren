package main

import (
	"context"
	"os"

	"github.com/secmon-lab/warren/pkg/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
}
