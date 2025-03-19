package usecase

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/service/list"
	"github.com/secmon-lab/warren/pkg/service/source"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
	"github.com/urfave/cli/v3"
)

func (x *UseCases) RunCommand(ctx context.Context, args []string, alerts []model.Alert, th interfaces.SlackThreadService, user *model.SlackUser) error {
	ctx = thread.WithReplyFunc(ctx, th.Reply)

	var buf bytes.Buffer
	cmd := cli.Command{
		Name:  "warren",
		Usage: "Slack bot for security monitoring",
		Commands: []*cli.Command{
			x.cmdList(alerts, th, user),
			x.cmdIgnore(alerts, th),
			x.cmdShow(alerts, th),
			x.cmdBlock(alerts, th),
			x.cmdResolve(alerts, th),
			x.cmdClustering(alerts, th, user),
		},
		Writer: &buf,
	}

	err := cmd.Run(ctx, args)
	if err != nil {
		thread.Reply(ctx, "💥 Failed to run command: "+err.Error())
		logging.From(ctx).Error("Failed to run command", "error", err)
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

func (x *UseCases) cmdList(alerts []model.Alert, th interfaces.SlackThreadService, user *model.SlackUser) *cli.Command {
	var (
		duration   time.Duration
		spanFrom   string
		spanTo     string
		unresolved bool
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

func sourceFromTarget(ctx context.Context, target string, th interfaces.SlackThreadService, alerts []model.Alert) source.Source {
	switch target {
	case "last":
		return source.LatestAlertList(model.SlackThread{
			ChannelID: th.ChannelID(),
			ThreadID:  th.ThreadID(),
		})

	case "thread":
		if len(alerts) == 0 {
			th.Reply(ctx, "💥 No alert found. Please run the command in the alert thread.")
			return nil
		}
		return source.Static(alerts)

	case "":
		if len(alerts) > 0 {
			return source.Static(alerts)
		}
		return source.LatestAlertList(model.SlackThread{
			ChannelID: th.ChannelID(),
			ThreadID:  th.ThreadID(),
		})

	default:
		return source.AlertListID(model.AlertListID(target))
	}
}

func (x *UseCases) cmdIgnore(alerts []model.Alert, th interfaces.SlackThreadService) *cli.Command {
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
			src := sourceFromTarget(ctx, targetAlerts, th, alerts)
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

func (x *UseCases) cmdShow(alerts []model.Alert, th interfaces.SlackThreadService) *cli.Command {
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

func (x *UseCases) cmdBlock(alerts []model.Alert, th interfaces.SlackThreadService) *cli.Command {
	var targetAlerts string

	return &cli.Command{
		Name:      "block",
		Aliases:   []string{"b"},
		Usage:     "Change status of alerts to blocked",
		UsageText: "@warren block [-t last|thread|${list_id}]",
		Action: func(ctx context.Context, c *cli.Command) error {
			src := sourceFromTarget(ctx, targetAlerts, th, alerts)
			if src == nil {
				return nil
			}

			alerts, err := src(ctx, x.repository)
			if err != nil {
				return err
			}

			var baseAlerts []*model.Alert
			alertIDs := make([]model.AlertID, len(alerts))
			for i, a := range alerts {
				alertIDs[i] = a.ID

				if a.ParentID == "" {
					baseAlerts = append(baseAlerts, &a)
				}
			}

			if err := x.repository.BatchUpdateAlertStatus(ctx, alertIDs, model.AlertStatusBlocked); err != nil {
				return err
			}
			thread.Reply(ctx, fmt.Sprintf("🚫 Blocked %d alerts", len(alertIDs)))

			for _, a := range baseAlerts {
				if err := th.UpdateAlert(ctx, *a); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func (x *UseCases) cmdResolve(alerts []model.Alert, th interfaces.SlackThreadService) *cli.Command {
	var (
		targetAlerts string
		conclusion   model.AlertConclusion
		reason       string
	)

	return &cli.Command{
		Name:        "resolve",
		Aliases:     []string{"r"},
		Usage:       "Change status of alerts to resolved",
		UsageText:   "@warren resolve [-t last|thread|${list_id}]",
		Description: "Change status of alerts to resolved",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "conclusion",
				Aliases:     []string{"c"},
				Usage:       "Conclusion of alerts",
				Destination: (*string)(&conclusion),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "reason",
				Aliases:     []string{"m"},
				Usage:       "Reason of resolved alerts",
				Destination: &reason,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := conclusion.Validate(); err != nil {
				return err
			}

			src := sourceFromTarget(ctx, targetAlerts, th, alerts)
			if src == nil {
				return nil
			}

			alerts, err := src(ctx, x.repository)
			if err != nil {
				return err
			}

			var baseAlerts []*model.Alert
			alertIDs := make([]model.AlertID, len(alerts))
			for i, a := range alerts {
				alertIDs[i] = a.ID

				if a.ParentID == "" {
					baseAlerts = append(baseAlerts, &a)
				}
			}

			if err := x.repository.BatchUpdateAlertStatus(ctx, alertIDs, model.AlertStatusResolved); err != nil {
				return err
			}

			if err := x.repository.BatchUpdateAlertConclusion(ctx, alertIDs, conclusion, reason); err != nil {
				return err
			}

			thread.Reply(ctx, fmt.Sprintf(`✅ Resolved %d alerts as *%s* because of "%s"`, len(alertIDs), conclusion.String(), reason))

			for _, a := range baseAlerts {
				if err := th.UpdateAlert(ctx, *a); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func (x *UseCases) cmdClustering(alerts []model.Alert, th interfaces.SlackThreadService, user *model.SlackUser) *cli.Command {
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

			threadData := model.SlackThread{
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

func (x *UseCases) clusterAlerts(ctx context.Context, th model.SlackThread, user *model.SlackUser, alerts []model.Alert, similarityThreshold float64, topN int) ([]model.AlertList, error) {
	clusters := newAlertCluster(ctx, th, user, alerts, similarityThreshold)

	sort.Slice(clusters.clusters, func(i, j int) bool {
		return len(clusters.clusters[i].Alerts) > len(clusters.clusters[j].Alerts)
	})

	if topN > 0 && topN < len(clusters.clusters) {
		clusters.clusters = clusters.clusters[:topN]
	}

	return clusters.clusters, nil
}

type alertCluster struct {
	clusters []model.AlertList
}

func newAlertCluster(ctx context.Context, th model.SlackThread, user *model.SlackUser, alerts []model.Alert, similarityThreshold float64) *alertCluster {
	// Initialize clusters
	clusters := make([]model.AlertList, 0)

	// Process each alert
	for _, alert := range alerts {
		if len(alert.Embedding) == 0 {
			continue // Skip alerts without embeddings
		}

		// Try to find matching cluster
		matched := false
		for j := range clusters {
			// Compare with first alert in cluster as representative
			if service.CosineSimilarity(alert.Embedding, clusters[j].Alerts[0].Embedding) >= similarityThreshold {
				clusters[j].Alerts = append(clusters[j].Alerts, alert)
				clusters[j].AlertIDs = append(clusters[j].AlertIDs, alert.ID)
				matched = true
				break
			}
		}

		// Create new cluster if no match found
		if !matched {
			clusters = append(clusters, model.NewAlertList(ctx, th, user, []model.Alert{alert}))
		}
	}

	return &alertCluster{
		clusters: clusters,
	}

}
