package graphql

import (
	"context"
	"net/http"
	"time"

	"github.com/graph-gophers/dataloader/v7"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type ctxKey string

const (
	loadersKey = ctxKey("dataloaders")
)

// Batch functions for graph-gophers/dataloader
func ticketBatchFn(repo interfaces.Repository) func(ctx context.Context, keys []types.TicketID) []*dataloader.Result[*ticket.Ticket] {
	return func(ctx context.Context, keys []types.TicketID) []*dataloader.Result[*ticket.Ticket] {
		results := make([]*dataloader.Result[*ticket.Ticket], len(keys))

		// Use batch get to solve N+1 problem
		tickets, err := repo.BatchGetTickets(ctx, keys)
		if err != nil {
			// If batch get fails, return error for all keys
			for i := range keys {
				results[i] = &dataloader.Result[*ticket.Ticket]{
					Data:  nil,
					Error: err,
				}
			}
			return results
		}

		// Create a map for O(1) lookup
		ticketMap := make(map[types.TicketID]*ticket.Ticket)
		for _, t := range tickets {
			if t != nil {
				ticketMap[t.ID] = t
			}
		}

		// Map results back to the original order
		for i, id := range keys {
			if t, found := ticketMap[id]; found {
				results[i] = &dataloader.Result[*ticket.Ticket]{
					Data:  t,
					Error: nil,
				}
			} else {
				results[i] = &dataloader.Result[*ticket.Ticket]{
					Data:  nil,
					Error: goerr.New("ticket not found", goerr.V("ticket_id", id)),
				}
			}
		}

		return results
	}
}

func alertBatchFn(repo interfaces.Repository) func(ctx context.Context, keys []types.AlertID) []*dataloader.Result[*alert.Alert] {
	return func(ctx context.Context, keys []types.AlertID) []*dataloader.Result[*alert.Alert] {
		results := make([]*dataloader.Result[*alert.Alert], len(keys))

		// Use batch get to solve N+1 problem
		alerts, err := repo.BatchGetAlerts(ctx, keys)
		if err != nil {
			// If batch get fails, return error for all keys
			for i := range keys {
				results[i] = &dataloader.Result[*alert.Alert]{
					Data:  nil,
					Error: err,
				}
			}
			return results
		}

		// Create a map for O(1) lookup
		alertMap := make(map[types.AlertID]*alert.Alert)
		for _, a := range alerts {
			if a != nil {
				alertMap[a.ID] = a
			}
		}

		// Map results back to the original order
		for i, id := range keys {
			if a, found := alertMap[id]; found {
				results[i] = &dataloader.Result[*alert.Alert]{
					Data:  a,
					Error: nil,
				}
			} else {
				results[i] = &dataloader.Result[*alert.Alert]{
					Data:  nil,
					Error: goerr.New("alert not found", goerr.V("alert_id", id)),
				}
			}
		}

		return results
	}
}

func userBatchFn(slackClient interfaces.SlackClient) func(ctx context.Context, keys []string) []*dataloader.Result[*graphql1.User] {
	return func(ctx context.Context, keys []string) []*dataloader.Result[*graphql1.User] {
		results := make([]*dataloader.Result[*graphql1.User], len(keys))

		if slackClient != nil {
			// Use batch API to fetch all users at once
			slackUsers, err := slackClient.GetUsersInfo(keys...)
			if err != nil {
				// Handle the error for debugging
				errs.Handle(ctx, goerr.Wrap(err, "failed to get users info from Slack", goerr.V("userIDs", keys)))
				// If Slack API fails with user_not_found or similar errors, fallback to ID instead of propagating error
				// This prevents the entire query from failing when some users don't exist in Slack
				for i, id := range keys {
					results[i] = &dataloader.Result[*graphql1.User]{
						Data:  &graphql1.User{ID: id, Name: id},
						Error: nil,
					}
				}
				return results
			}

			if slackUsers != nil {
				// Create map for O(1) lookup
				userMap := make(map[string]*graphql1.User)
				for _, slackUser := range *slackUsers {
					icon := slackUser.Profile.Image48
					if icon == "" {
						icon = slackUser.Profile.Image32
					}

					// Prefer RealName over Name, then DisplayName
					name := slackUser.RealName
					if name == "" {
						name = slackUser.Profile.RealName
					}
					if name == "" {
						name = slackUser.Profile.DisplayName
					}
					if name == "" {
						name = slackUser.Name
					}
					if name == "" {
						name = slackUser.ID
					}

					userMap[slackUser.ID] = &graphql1.User{
						ID:   slackUser.ID,
						Name: name,
						Icon: &icon,
					}
				}

				// Build results in the same order as keys
				for i, key := range keys {
					if user, found := userMap[key]; found {
						results[i] = &dataloader.Result[*graphql1.User]{Data: user, Error: nil}
					} else {
						// User not found in Slack response, fallback to ID
						results[i] = &dataloader.Result[*graphql1.User]{
							Data:  &graphql1.User{ID: key, Name: key},
							Error: nil,
						}
					}
				}
				return results
			}
		}

		// Fallback for when SlackClient is nil (not an error condition)
		for i, id := range keys {
			results[i] = &dataloader.Result[*graphql1.User]{
				Data:  &graphql1.User{ID: id, Name: id},
				Error: nil,
			}
		}

		return results
	}
}

// dataLoaders wrap your data loaders to inject via middleware
type dataLoaders struct {
	TicketLoader *dataloader.Loader[types.TicketID, *ticket.Ticket]
	AlertLoader  *dataloader.Loader[types.AlertID, *alert.Alert]
	UserLoader   *dataloader.Loader[string, *graphql1.User]
}

// newDataLoaders instantiates data loaders for the middleware
func newDataLoaders(repo interfaces.Repository, slackClient interfaces.SlackClient) *dataLoaders {
	return &dataLoaders{
		TicketLoader: dataloader.NewBatchedLoader[types.TicketID, *ticket.Ticket](
			ticketBatchFn(repo),
			dataloader.WithWait[types.TicketID, *ticket.Ticket](5*time.Millisecond),
		),
		AlertLoader: dataloader.NewBatchedLoader[types.AlertID, *alert.Alert](
			alertBatchFn(repo),
			dataloader.WithWait[types.AlertID, *alert.Alert](5*time.Millisecond),
		),
		UserLoader: dataloader.NewBatchedLoader[string, *graphql1.User](
			userBatchFn(slackClient),
			dataloader.WithWait[string, *graphql1.User](5*time.Millisecond),
		),
	}
}

// DataLoaderMiddleware injects data loaders into the context
func DataLoaderMiddleware(repo interfaces.Repository, slackClient interfaces.SlackClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loaders := newDataLoaders(repo, slackClient)
			r = r.WithContext(context.WithValue(r.Context(), loadersKey, loaders))
			next.ServeHTTP(w, r)
		})
	}
}

// dataLoadersFor returns the dataloader for a given context
func dataLoadersFor(ctx context.Context) *dataLoaders {
	return ctx.Value(loadersKey).(*dataLoaders)
}

// GetTicket returns single ticket by id efficiently
func GetTicket(ctx context.Context, ticketID types.TicketID) (*ticket.Ticket, error) {
	loaders := dataLoadersFor(ctx)
	thunk := loaders.TicketLoader.Load(ctx, ticketID)
	return thunk()
}

// GetAlert returns single alert by id efficiently
func GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error) {
	loaders := dataLoadersFor(ctx)
	thunk := loaders.AlertLoader.Load(ctx, alertID)
	return thunk()
}

// GetUser returns single user by id efficiently
func GetUser(ctx context.Context, userID string) (*graphql1.User, error) {
	loaders := dataLoadersFor(ctx)
	thunk := loaders.UserLoader.Load(ctx, userID)
	return thunk()
}
