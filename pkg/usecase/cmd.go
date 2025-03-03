package usecase

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/service/list"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/thread"
	"github.com/urfave/cli/v3"
)

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Try time only format (e.g. "10:00")
	if t, err := time.Parse("15:04", s); err == nil {
		return time.Date(today.Year(), today.Month(), today.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
	}

	// Try date only format (e.g. "2/3")
	if parts := strings.Split(s, "/"); len(parts) == 2 {
		if month, err := time.Parse("1", parts[0]); err == nil {
			if day, err := time.Parse("2", parts[1]); err == nil {
				return time.Date(today.Year(), month.Month(), day.Day(), 0, 0, 0, 0, time.Local), nil
			}
		}
	}

	// Try date+time format (e.g. "02-03T00:00")
	if t, err := time.Parse("01-02T15:04", s); err == nil {
		return time.Date(today.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
	}

	switch s {
	case "today":
		return today, nil
	case "yesterday":
		return today.Add(-24 * time.Hour), nil
	}

	return time.Time{}, goerr.New("invalid time format: expected format: RFC3339, time only (15:04), date only (2/3), date+time (02-03T00:00), today, yesterday", goerr.V("time", s))
}

func (x *UseCases) cmdList(th interfaces.SlackThreadService, user *model.SlackUser) *cli.Command {
	var (
		duration time.Duration
		spanFrom string
		spanTo   string
	)
	return &cli.Command{
		Name:        "list",
		Usage:       "Create a list of alerts",
		UsageText:   "@warren list [options] [$list_id | last]",
		Description: "Create a list of alerts",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:        "duration",
				Aliases:     []string{"d"},
				Usage:       "Duration to list alerts",
				Destination: &duration,
				Value:       time.Hour * 24,
			},
			&cli.StringFlag{
				Name:        "from",
				Aliases:     []string{"f"},
				Usage:       "From time to list alerts",
				Destination: &spanFrom,
			},
			&cli.StringFlag{
				Name:        "to",
				Aliases:     []string{"t"},
				Usage:       "To time to list alerts",
				Destination: &spanTo,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			var src list.Source
			now := clock.Now(ctx)

			switch {
			case spanFrom != "" || spanTo != "":
				from, to := now, now

				if spanFrom != "" {
					if v, err := parseTime(spanFrom); err != nil {
						return err
					} else {
						from = v
					}
				}
				if spanTo != "" {
					if v, err := parseTime(spanTo); err != nil {
						return err
					} else {
						to = v
					}
				}
				src = list.SourceSpan(from, to)

			case c.Args().Len() == 1 && c.Args().First() == "last":
				src = list.SourceLatestAlertList(model.SlackThread{
					ChannelID: th.ChannelID(),
					ThreadID:  th.ThreadID(),
				})

			case c.Args().Len() == 1:
				src = list.SourceAlertListID(model.AlertListID(c.Args().First()))

			case duration != 0:
				src = list.SourceSpan(now.Add(-duration), now)
			}

			svc := list.New(x.repository, list.WithLLM(x.llmClient))
			if err := svc.Run(ctx, th, user, src, c.Args().Slice()); err != nil {
				return err
			}

			return nil
		},
	}
}

func (x *UseCases) RunCommand(ctx context.Context, args []string, alert *model.Alert, th interfaces.SlackThreadService, user *model.SlackUser) error {
	var buf bytes.Buffer
	cmd := cli.Command{
		Name:  "warren",
		Usage: "Slack bot for security monitoring",
		Commands: []*cli.Command{
			x.cmdList(th, user),
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
