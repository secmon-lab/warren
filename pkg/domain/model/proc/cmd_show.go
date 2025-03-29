package proc

/*
func cmdShow(c *clients) *cli.Command {
	var (
		source string
		limit  int64
		offset int64
	)

	return &cli.Command{
		Name:        "show",
		Usage:       "Show a list of alerts",
		UsageText:   "@warren show [-t last|thread|${list_id}]",
		Description: "Show a list of alerts",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "source",
				Aliases:     []string{"s"},
				Usage:       "Source alerts to show [root|last|${list_id}]",
				Destination: &source,
				Value:       "last",
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
			src := sourceFromTarget(ctx, targetAlerts, th, alerts)
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
					alerts = alert.Alerts{}
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
*/
