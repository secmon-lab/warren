package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// createSlackWarnFunc creates a WarnFunc that posts warning messages to a Slack thread
func createSlackWarnFunc(threadSvc interfaces.SlackThreadService) msg.WarnFunc {
	return func(ctx context.Context, message string) {
		if err := threadSvc.PostComment(ctx, message); err != nil {
			logging.From(ctx).Error("failed to post warning message to slack", "error", err)
		}
	}
}
