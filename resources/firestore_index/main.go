package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "firestore_index",
		Usage: "Firestore indexes management for Warren",
		Commands: []*cli.Command{
			cmdCreateIndexes(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
}
