package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func createOrGetSession(ctx context.Context, repo interfaces.Repository, slackMsg *slack.Message) (*session.Session, error) {
	ssn, err := repo.GetSessionByThread(ctx, slackMsg.Thread())
	if err != nil {
		return nil, err
	}

	if ssn != nil {
		return ssn, nil
	}

	return createSession(ctx, repo, slackMsg)
}

func createSession(ctx context.Context, repo interfaces.Repository, slackMsg *slack.Message) (*session.Session, error) {
	var alertIDs []types.AlertID

	alert, err := repo.GetAlertByThread(ctx, slackMsg.Thread())
	if err != nil {
		return nil, err
	}
	if alert != nil {
		alertIDs = []types.AlertID{alert.ID}
	}

	alertList, err := repo.GetAlertListByThread(ctx, slackMsg.Thread())
	if err != nil {
		return nil, err
	}
	if alertList != nil {
		alertIDs = append(alertIDs, alertList.AlertIDs...)
	}

	ssn := session.New(ctx, slackMsg, alertIDs)
	if err := repo.PutSession(ctx, *ssn); err != nil {
		return nil, err
	}

	return ssn, nil
}
