package base

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func (x *Warren) findNearestTicket(ctx context.Context, args map[string]any) (map[string]any, error) {
	limit, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit")
	}

	duration, err := getArg[int64](args, "duration")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get duration")
	}

	now := clock.Now(ctx)
	nearestTickets, err := x.repo.FindNearestTicketsWithSpan(ctx, x.ticket.Embedding, now.Add(-time.Duration(duration)*24*time.Hour), now, int(limit)+1)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to find nearest tickets")
	}

	var results []*ticket.Ticket
	for _, t := range nearestTickets {
		if t.ID == x.ticket.ID {
			continue
		}
		results = append(results, t)
	}

	if len(results) > int(limit) {
		results = results[:limit]
	}

	return map[string]any{
		"tickets": results,
	}, nil
}
