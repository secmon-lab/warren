package interfaces

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type Repository interface {
	PutAlert(ctx context.Context, alert alert.Alert) error
	GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error)
	GetAlertByThread(ctx context.Context, thread slack.Thread) (*alert.Alert, error)
	PutAlertComment(ctx context.Context, comment alert.AlertComment) error
	GetAlertComments(ctx context.Context, alertID types.AlertID) ([]alert.AlertComment, error)

	// For chat
	GetHistory(ctx context.Context, thread slack.Thread) (*chat.History, error)
	PutHistory(ctx context.Context, history chat.History) error

	// For list management
	GetAlertList(ctx context.Context, listID types.AlertListID) (*alert.List, error)
	PutAlertList(ctx context.Context, list alert.List) error
	GetAlertListByThread(ctx context.Context, thread slack.Thread) (*alert.List, error)
	GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error)

	// For list generation
	GetAlertsByStatus(ctx context.Context, status alert.Status) ([]alert.Alert, error)
	GetAlertsBySpan(ctx context.Context, begin, end time.Time) ([]alert.Alert, error)
	BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) ([]alert.Alert, error)
	BatchUpdateAlertStatus(ctx context.Context, alertIDs []types.AlertID, status alert.Status, reason string) error

	// For policy management
	GetPolicyDiff(ctx context.Context, id model.PolicyDiffID) (*model.PolicyDiff, error)
	PutPolicyDiff(ctx context.Context, diff *model.PolicyDiff) error
}
