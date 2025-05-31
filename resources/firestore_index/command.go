package main

import (
	"context"

	"github.com/urfave/cli/v3"
)

func cmdCreateIndexes() *cli.Command {
	var (
		projectID  string
		databaseID string
		dryrun     bool
	)

	return &cli.Command{
		Name:    "create",
		Aliases: []string{"c"},
		Usage:   "Create missing Firestore indexes",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "project",
				Aliases:     []string{"p"},
				Usage:       "Firestore project ID",
				Required:    true,
				Destination: &projectID,
				Category:    "Firestore",
				Sources:     cli.EnvVars("WARREN_FIRESTORE_PROJECT_ID"),
			},
			&cli.StringFlag{
				Name:        "database",
				Aliases:     []string{"d"},
				Usage:       "Firestore database ID",
				Destination: &databaseID,
				Category:    "Firestore",
				Sources:     cli.EnvVars("WARREN_FIRESTORE_DATABASE_ID"),
				Value:       "(default)",
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "Check required indexes without creating them",
				Destination: &dryrun,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runCreateIndexes(ctx, projectID, databaseID, dryrun)
		},
	}
}
