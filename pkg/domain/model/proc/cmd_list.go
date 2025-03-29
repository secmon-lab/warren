package proc

/*
func cmdList(alerts []alert.Alert, th interfaces.SlackThreadService, user *slack.SlackUser) *cli.Command {
	var (
		duration   time.Duration
		spanFrom   string
		spanTo     string
		unresolved bool
	)
	return &cli.Command{
		Name:        "list",
		Usage:       "Call alert list and create another list",
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
			&cli.BoolFlag{
				Name:        "unresolved",
				Aliases:     []string{"u"},
				Usage:       "Show only unresolved alerts",
				Destination: &unresolved,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			var src source.Source
			now := clock.Now(ctx)

			args := c.Args().Slice()
			switch {
			case unresolved:
				src = source.Unresolved()

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
				src = source.LatestAlertList(slack.SlackThread{
					ChannelID: th.ChannelID(),
					ThreadID:  th.ThreadID(),
				})
				args = args[1:]

			case len(args) > 0:
				src = source.AlertListID(alert.ListID(args[0]))
				args = args[1:]

			case duration != 0:
				src = source.Span(now.Add(-duration), now)

			case len(alerts) > 0:
				src = source.Static(alerts)

			default:
				src = source.Span(now.Add(-time.Hour*24), now)
			}

			if unresolved {
				args = append(args, "status", "new", "ack", "blocked")
			}

			svc := list.New(x.repository, list.WithLLM(x.llmClient))
			if err := svc.Run(ctx, th, user, src, args); err != nil {
				return err
			}

			return nil
		},
	}
}
*/
