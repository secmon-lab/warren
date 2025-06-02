package base

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
)

func (x *Warren) findNearestTicket(ctx context.Context, args map[string]any) (map[string]any, error) {
	limit, err := getArg[int64](args, "limit")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get limit")
	}

	nearestTickets, err := x.repo.FindNearestTickets(ctx, x.ticket.Embedding, int(limit))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to find nearest tickets")
	}

	return map[string]any{
		"tickets": nearestTickets,
	}, nil
}
