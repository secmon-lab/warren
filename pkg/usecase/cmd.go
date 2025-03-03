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
	"github.com/secmon-lab/warren/pkg/service/source"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/thread"
	"github.com/urfave/cli/v3"
)

func (x *UseCases) RunCommand(ctx context.Context, args []string, alert *model.Alert, th interfaces.SlackThreadService, user *model.SlackUser) error {
	ctx = thread.WithReplyFunc(ctx, th.Reply)

	var buf bytes.Buffer
	cmd := cli.Command{
		Name:  "warren",
		Usage: "Slack bot for security monitoring",
		Commands: []*cli.Command{
			x.cmdList(alert, th, user),
			x.cmdIgnore(alert, th),
			x.cmdShow(alert, th),
		},
		Writer: &buf,
	}

	err := cmd.Run(ctx, args)
	if err != nil {
		thread.Reply(ctx, "💥 Failed to run command: "+err.Error())
	}

	if buf.String() != "" {
		thread.Reply(ctx, "```\n"+buf.String()+"\n```")
	}

	return nil
}

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

func (x *UseCases) cmdList(alert *model.Alert, th interfaces.SlackThreadService, user *model.SlackUser) *cli.Command {
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
			var src source.Source
			now := clock.Now(ctx)

			args := c.Args().Slice()
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
				src = source.Span(from, to)

			case len(args) > 0 && args[0] == "last":
				src = source.LatestAlertList(model.SlackThread{
					ChannelID: th.ChannelID(),
					ThreadID:  th.ThreadID(),
				})
				args = args[1:]

			case len(args) > 0:
				src = source.AlertListID(model.AlertListID(args[0]))
				args = args[1:]

			case duration != 0:
				src = source.Span(now.Add(-duration), now)

			case alert != nil:
				src = source.Alert(alert)

			default:
				src = source.Span(now.Add(-time.Hour*24), now)
			}

			svc := list.New(x.repository, list.WithLLM(x.llmClient))
			if err := svc.Run(ctx, th, user, src, args); err != nil {
				return err
			}

			return nil
		},
	}
}

func sourceFromTarget(ctx context.Context, target string, th interfaces.SlackThreadService, alert *model.Alert) source.Source {
	switch target {
	case "last":
		return source.LatestAlertList(model.SlackThread{
			ChannelID: th.ChannelID(),
			ThreadID:  th.ThreadID(),
		})

	case "thread":
		if alert == nil {
			th.Reply(ctx, "💥 No alert found. Please run the command in the alert thread.")
			return nil
		}
		return source.Alert(alert)

	case "":
		if alert != nil {
			return source.Alert(alert)
		}
		return source.LatestAlertList(model.SlackThread{
			ChannelID: th.ChannelID(),
			ThreadID:  th.ThreadID(),
		})

	default:
		return source.AlertListID(model.AlertListID(target))
	}
}

func (x *UseCases) cmdIgnore(alert *model.Alert, th interfaces.SlackThreadService) *cli.Command {
	var targetAlerts string

	return &cli.Command{
		Name:        "ignore",
		Usage:       "Create a ignore policy",
		UsageText:   "@warren ignore [-t last|thread|${list_id}] [query...]",
		Description: "Create a ignore policy",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "target",
				Aliases:     []string{"t"},
				Usage:       "Target alerts to ignore",
				Destination: &targetAlerts,
				Value:       "",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			src := sourceFromTarget(ctx, targetAlerts, th, alert)
			if src == nil {
				return nil
			}

			var note string
			if c.Args().Len() > 0 {
				note = strings.Join(c.Args().Slice(), " ")
			}

			newPolicyDiff, err := x.GenerateIgnorePolicy(ctx, src, note)
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
	}
}

func (x *UseCases) cmdShow(alert *model.Alert, th interfaces.SlackThreadService) *cli.Command {
	var (
		targetAlerts string
		limit        int64
		offset       int64
	)

	return &cli.Command{
		Name:        "show",
		Usage:       "Show a list of alerts",
		UsageText:   "@warren show [-t last|thread|${list_id}]",
		Description: "Show a list of alerts",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "target",
				Aliases:     []string{"t"},
				Usage:       "Target alerts to show",
				Destination: &targetAlerts,
				Value:       "",
			},
			&cli.IntFlag{
				Name:        "limit",
				Aliases:     []string{"l"},
				Usage:       "Limit the number of alerts to show",
				Destination: &limit,
			},
			&cli.IntFlag{
				Name:        "offset",
				Aliases:     []string{"o"},
				Usage:       "Offset the number of alerts to show",
				Destination: &offset,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			src := sourceFromTarget(ctx, targetAlerts, th, alert)
			if src == nil {
				return nil
			}

			alerts, err := src(ctx, x.repository)
			if err != nil {
				return err
			}

			if limit > 0 && int64(len(alerts)) > limit {
				alerts = alerts[:limit]
			}
			if offset > 0 {
				if offset > int64(len(alerts)) {
					alerts = []model.Alert{}
				} else {
					alerts = alerts[offset:]
				}
			}

			if err := th.PostAlerts(ctx, alerts); err != nil {
				return err
			}

			return nil
		},
	}
}
