package proc

/*
func cmdClustering(alerts []alert.Alert, th interfaces.SlackThreadService, user *slack.SlackUser) *cli.Command {
	var target string
	var topN int64
	var similarityThreshold float64

	return &cli.Command{
		Name:        "cluster",
		Usage:       "Create new lists of alert by clustering",
		UsageText:   "@warren cluster [-t last|thread|${list_id}]",
		Description: "Create new lists of alert by clustering",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "target",
				Aliases:     []string{"t"},
				Usage:       "Target list of alerts to cluster [last|thread|${list_id}]",
				Destination: &target,
			},
			&cli.IntFlag{
				Name:        "top-n",
				Aliases:     []string{"n"},
				Usage:       "Top N clusters to show",
				Destination: &topN,
				Value:       6,
			},
			&cli.FloatFlag{
				Name:        "similarity-threshold",
				Aliases:     []string{"s"},
				Usage:       "Similarity threshold for clustering",
				Destination: &similarityThreshold,
				Value:       0.99,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			src := sourceFromTarget(ctx, target, th, alerts)
			if src == nil {
				return nil
			}

			alerts, err := src(ctx, x.repository)
			if err != nil {
				return err
			}

			threadData := slack.SlackThread{
				ChannelID: th.ChannelID(),
				ThreadID:  th.ThreadID(),
			}
			clusters, err := x.clusterAlerts(ctx, threadData, user, alerts, similarityThreshold, int(topN))
			if err != nil {
				return err
			}

			for i, cluster := range clusters {
				meta, err := service.GenerateAlertListMeta(ctx, cluster, x.llmClient)
				if err != nil {
					return err
				}
				if meta != nil {
					clusters[i].Title = meta.Title
					clusters[i].Description = meta.Description
				}

				if err := x.repository.PutAlertList(ctx, cluster); err != nil {
					return err
				}
			}

			if err := th.PostAlertClusters(ctx, clusters); err != nil {
				return err
			}

			return nil
		},
	}
}
*/
