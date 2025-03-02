package usecase

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/thread"
	"github.com/urfave/cli/v3"
)

func (x *UseCases) cmdList(th interfaces.SlackThreadService) *cli.Command {
	var (
		duration time.Duration
		limit    int64
		offset   int64
	)
	return &cli.Command{
		Name:  "list",
		Usage: "List alerts",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:        "duration",
				Aliases:     []string{"d"},
				Usage:       "Duration to list alerts",
				Destination: &duration,
				Value:       time.Hour * 24,
			},
			&cli.IntFlag{
				Name:        "limit",
				Aliases:     []string{"l"},
				Usage:       "Limit the number of alerts to list",
				Destination: &limit,
				Value:       20,
			},
			&cli.IntFlag{
				Name:        "offset",
				Aliases:     []string{"p"},
				Usage:       "Offset to list alerts",
				Destination: &offset,
				Value:       0,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			alerts, err := x.repository.GetLatestAlerts(ctx, time.Now().Add(-duration), int(limit))
			if err != nil {
				return goerr.Wrap(err, "failed to get alerts")
			}

			th.Reply(ctx, "```\n"+fmt.Sprintf("%v", alerts)+"\n```")

			return nil
		},
	}
}

func (x *UseCases) RunCommand(ctx context.Context, args []string, alert *model.Alert, th interfaces.SlackThreadService) error {
	var buf bytes.Buffer
	cmd := cli.Command{
		Name:  "warren",
		Usage: "Slack bot for security monitoring",
		Commands: []*cli.Command{
			x.cmdList(th),
			{
				Name:    "ignore",
				Usage:   "Ignore alerts",
				Aliases: []string{"i"},
				Action: func(ctx context.Context, c *cli.Command) error {
					if alert == nil {
						th.Reply(ctx, "💥 No alert found. Please run the command in the alert thread.")
						return nil
					}

					alerts := []model.Alert{*alert}
					childAlerts, err := x.repository.GetAlertsByParentID(ctx, alert.ID)
					if err != nil {
						return goerr.Wrap(err, "failed to get child alerts")
					}
					alerts = append(alerts, childAlerts...)

					var note string
					if len(args) > 1 {
						note = strings.Join(args[1:], " ")
					}

					newPolicyDiff, err := x.GenerateIgnorePolicy(ctx, alerts, note)
					if err != nil {
						return err
					}

					if err := x.repository.PutPolicyDiff(ctx, newPolicyDiff); err != nil {
						return err
					}

					if err := th.PostPolicyDiff(ctx, newPolicyDiff); err != nil {
						return err
					}

					return nil
				},
			},
		},
		Writer: &buf,
	}

	err := cmd.Run(ctx, args)
	if err != nil {
		return goerr.Wrap(err, "failed to run command", goerr.V("args", args))
	}

	if buf.String() != "" {
		thread.Reply(ctx, "```\n"+buf.String()+"\n```")
	}

	return nil
}
