package bigquery

import (
	"context"

	"github.com/urfave/cli/v3"
)

func (x *Action) Helper() *cli.Command {
	return &cli.Command{
		Name:  "bigquery",
		Usage: "BigQuery tool helper",
		Commands: []*cli.Command{
			{
				Name:  "list-datasets",
				Usage: "List datasets",
				Action: func(ctx context.Context, c *cli.Command) error {
					return nil
				},
			},
		},
	}
}
