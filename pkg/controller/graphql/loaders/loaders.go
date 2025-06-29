package loaders

import (
	"context"
	"net/http"
	"time"

	"github.com/graph-gophers/dataloader/v7"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
	slack_model "github.com/secmon-lab/warren/pkg/domain/model/slack"
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

		for i, id := range keys {
			t, err := repo.GetTicket(ctx, id)
			results[i] = &dataloader.Result[*ticket.Ticket]{
				Data:  t,
				Error: err,
			}
		}

		return results
	}
}

func alertBatchFn(repo interfaces.Repository) func(ctx context.Context, keys []types.AlertID) []*dataloader.Result[*alert.Alert] {
	return func(ctx context.Context, keys []types.AlertID) []*dataloader.Result[*alert.Alert] {
		results := make([]*dataloader.Result[*alert.Alert], len(keys))

		for i, id := range keys {
			a, err := repo.GetAlert(ctx, id)
			results[i] = &dataloader.Result[*alert.Alert]{
				Data:  a,
				Error: err,
			}
		}

		return results
	}
}

func userBatchFn(slackClient interfaces.SlackClient) func(ctx context.Context, keys []string) []*dataloader.Result[*graphql1.User] {
	return func(ctx context.Context, keys []string) []*dataloader.Result[*graphql1.User] {
		results := make([]*dataloader.Result[*graphql1.User], len(keys))

		for i, id := range keys {
			slackUser := slack_model.User{ID: id}
			user := &graphql1.User{
				ID:   id,
				Name: slackUser.ID, // fallback to ID if name not available
			}

			if slackClient != nil {
				if userInfo, err := slackClient.GetUserInfo(id); err == nil {
					user.Name = userInfo.Name
				}
			}

			results[i] = &dataloader.Result[*graphql1.User]{
				Data:  user,
				Error: nil,
			}
		}

		return results
	}
}

// Loaders wrap your data loaders to inject via middleware
type Loaders struct {
	TicketLoader *dataloader.Loader[types.TicketID, *ticket.Ticket]
	AlertLoader  *dataloader.Loader[types.AlertID, *alert.Alert]
	UserLoader   *dataloader.Loader[string, *graphql1.User]
}

// NewLoaders instantiates data loaders for the middleware
func NewLoaders(repo interfaces.Repository, slackClient interfaces.SlackClient) *Loaders {
	return &Loaders{
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

// Middleware injects data loaders into the context
func Middleware(repo interfaces.Repository, slackClient interfaces.SlackClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loaders := NewLoaders(repo, slackClient)
			r = r.WithContext(context.WithValue(r.Context(), loadersKey, loaders))
			next.ServeHTTP(w, r)
		})
	}
}

// For returns the dataloader for a given context
func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

// GetTicket returns single ticket by id efficiently
func GetTicket(ctx context.Context, ticketID types.TicketID) (*ticket.Ticket, error) {
	loaders := For(ctx)
	thunk := loaders.TicketLoader.Load(ctx, ticketID)
	return thunk()
}

// GetAlert returns single alert by id efficiently
func GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error) {
	loaders := For(ctx)
	thunk := loaders.AlertLoader.Load(ctx, alertID)
	return thunk()
}

// GetUser returns single user by id efficiently
func GetUser(ctx context.Context, userID string) (*graphql1.User, error) {
	loaders := For(ctx)
	thunk := loaders.UserLoader.Load(ctx, userID)
	return thunk()
}
