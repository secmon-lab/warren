package interfaces

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
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
	GetHistory(ctx context.Context, sessionID types.SessionID) (session.Histories, error)
	PutHistory(ctx context.Context, sessionID types.SessionID, histories session.Histories) error

	// For list management
	GetAlertList(ctx context.Context, listID types.AlertListID) (*alert.List, error)
	PutAlertList(ctx context.Context, list alert.List) error
	GetAlertListByThread(ctx context.Context, thread slack.Thread) (*alert.List, error)
	GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error)

	// For list generation
	GetAlertsByStatus(ctx context.Context, status ...types.AlertStatus) (alert.Alerts, error)
	GetAlertsWithoutStatus(ctx context.Context, status types.AlertStatus) (alert.Alerts, error)
	GetAlertsBySpan(ctx context.Context, begin, end time.Time) (alert.Alerts, error)
	BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) (alert.Alerts, error)
	BatchUpdateAlertStatus(ctx context.Context, alertIDs []types.AlertID, status types.AlertStatus, reason string) error

	// For policy management
	GetPolicyDiff(ctx context.Context, id types.PolicyDiffID) (*policy.Diff, error)
	PutPolicyDiff(ctx context.Context, diff *policy.Diff) error

	// For session management
	GetSession(ctx context.Context, id types.SessionID) (*session.Session, error)
	GetSessionByThread(ctx context.Context, thread slack.Thread) (*session.Session, error)
	PutSession(ctx context.Context, session session.Session) error
}
