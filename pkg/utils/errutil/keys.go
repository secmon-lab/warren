package errutil

import (
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

var (
	// IDs
	AlertIDKey     = goerr.NewTypedKey[types.AlertID]("alert_id")
	AlertListIDKey = goerr.NewTypedKey[types.AlertListID]("alert_list_id")
	TicketIDKey    = goerr.NewTypedKey[types.TicketID]("ticket_id")
	CommentIDKey   = goerr.NewTypedKey[types.CommentID]("comment_id")
	ActivityIDKey  = goerr.NewTypedKey[types.ActivityID]("activity_id")
	ClusterIDKey   = goerr.NewTypedKey[string]("cluster_id")
	UserIDKey      = goerr.NewTypedKey[string]("user_id")
	RequestIDKey   = goerr.NewTypedKey[string]("request_id")
	SessionIDKey   = goerr.NewTypedKey[string]("session_id")

	// Field names
	FieldKey        = goerr.NewTypedKey[string]("field")
	ParameterKey    = goerr.NewTypedKey[string]("parameter")
	FunctionNameKey = goerr.NewTypedKey[string]("function_name")

	// Values
	SeverityKey   = goerr.NewTypedKey[string]("severity")
	ReasonKey     = goerr.NewTypedKey[string]("reason")
	StatusKey     = goerr.NewTypedKey[string]("status")
	OperationKey  = goerr.NewTypedKey[string]("operation")
	RepositoryKey = goerr.NewTypedKey[string]("repository")
	CollectionKey = goerr.NewTypedKey[string]("collection")
	QueryKey      = goerr.NewTypedKey[string]("query")
	LimitKey      = goerr.NewTypedKey[int]("limit")
	OffsetKey     = goerr.NewTypedKey[int]("offset")
	CountKey      = goerr.NewTypedKey[int]("count")
	DurationKey   = goerr.NewTypedKey[time.Duration]("duration")

	// External services
	ServiceKey      = goerr.NewTypedKey[string]("service")
	EndpointKey     = goerr.NewTypedKey[string]("endpoint")
	HTTPStatusKey   = goerr.NewTypedKey[int]("http_status")
	ErrorMessageKey = goerr.NewTypedKey[string]("error_message")
	URLKey          = goerr.NewTypedKey[string]("url")

	// File and path
	FilePathKey = goerr.NewTypedKey[string]("file_path")
	LineKey     = goerr.NewTypedKey[int]("line")

	// Slack specific
	ChannelIDKey = goerr.NewTypedKey[string]("channel_id")
	MessageTSKey = goerr.NewTypedKey[string]("message_ts")
	SlackUserKey = goerr.NewTypedKey[string]("slack_user")
	ActionIDKey  = goerr.NewTypedKey[string]("action_id")
)
