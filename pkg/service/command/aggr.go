package command

import (
	"context"
	"strconv"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// Aggregate runs the aggregate command with the given input.
func (x *Service) Aggregate(ctx context.Context, st *svc.ThreadService, user slack.User, alertList *alert.List, remaining string) error {
	args := strings.Fields(remaining)

	threshold := 0.99
	if len(args) > 0 {
		v, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return goerr.Wrap(err, "failed to parse threshold (0.00-1.00)")
		}
		if v < 0.00 || v > 1.00 {
			return goerr.Wrap(err, "invalid threshold range (0.00-1.00)")
		}
		threshold = v
	}

	topN := 5
	if len(args) > 1 {
		v, err := strconv.Atoi(args[1])
		if err != nil {
			return goerr.Wrap(err, "failed to parse topN (1-10)")
		}
		topN = v
	}

	alerts, err := alertList.GetAlerts(ctx, x.repo)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	msg.Trace(ctx, "Aggregating %d alerts (threshold: %f, topN: %d)", len(alerts), threshold, topN)

	clusters := alert.ClusterAlerts(ctx, alerts, threshold, topN)

	var lists []*alert.List
	for _, cluster := range clusters {
		newList, err := x.CreateList(ctx, *alertList.SlackThread, &user, cluster)
		if err != nil {
			return goerr.Wrap(err, "failed to create alert list")
		}

		lists = append(lists, newList)
	}

	if err := st.PostAlertLists(ctx, lists); err != nil {
		return goerr.Wrap(err, "failed to post alert clusters")
	}

	return nil
}
