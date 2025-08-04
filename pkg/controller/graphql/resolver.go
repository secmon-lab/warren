package graphql

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/mrkdwn"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver serves as dependency injection point for the application.
type Resolver struct {
	repo         interfaces.Repository
	slackService *slack.Service
	mrkdwnConv   *mrkdwn.Converter
	uc           *usecase.UseCases
}

// NewResolver creates a new resolver instance.
func NewResolver(repo interfaces.Repository, slackService *slack.Service, uc *usecase.UseCases) *Resolver {
	var mrkdwnConv *mrkdwn.Converter
	if slackService != nil {
		mrkdwnConv = mrkdwn.NewConverter(slackService)
	}

	return &Resolver{
		repo:         repo,
		slackService: slackService,
		mrkdwnConv:   mrkdwnConv,
		uc:           uc,
	}
}

// createTagGetter creates a common tag getter function for resolving tag names
func (r *Resolver) createTagGetter() func(context.Context, []types.TagID) ([]*tag.Tag, error) {
	return func(ctx context.Context, tagIDs []types.TagID) ([]*tag.Tag, error) {
		tagService := r.uc.GetTagService()
		if tagService == nil {
			return nil, goerr.New("tag service not available")
		}
		return tagService.GetTagsByIDs(ctx, tagIDs)
	}
}
