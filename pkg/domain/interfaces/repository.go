package interfaces

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/model/refine"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// AgentSummary contains summary information about an agent's memories
type AgentSummary struct {
	AgentID        string
	Count          int
	LatestMemoryAt time.Time
}

// LegacyKnowledge represents an old-format knowledge entry for migration.
// Old knowledge was stored in: topics/{topic}/knowledges/{slug}/commits/{commitID}/
type LegacyKnowledge struct {
	Topic     string
	Slug      string
	Name      string
	Content   string
	Author    string
	State     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AgentMemoryListOptions contains filtering, sorting, and pagination options for listing agent memories
type AgentMemoryListOptions struct {
	Offset   int
	Limit    int
	SortBy   string  // "score", "created_at", "last_used_at"
	SortDesc bool    // true for descending, false for ascending
	Keyword  *string // filter by query or claim content
	MinScore *float64
	MaxScore *float64
}

type Repository interface {
	GetTicket(ctx context.Context, ticketID types.TicketID) (*ticket.Ticket, error)
	BatchGetTickets(ctx context.Context, ticketIDs []types.TicketID) ([]*ticket.Ticket, error)
	PutTicket(ctx context.Context, ticket ticket.Ticket) error
	BatchUpdateTicketsStatus(ctx context.Context, ticketIDs []types.TicketID, status types.TicketStatus) error
	GetTicketByThread(ctx context.Context, thread slack.Thread) (*ticket.Ticket, error)
	FindNearestTickets(ctx context.Context, embedding []float32, limit int) ([]*ticket.Ticket, error)
	FindNearestTicketsWithSpan(ctx context.Context, embedding []float32, begin, end time.Time, limit int) ([]*ticket.Ticket, error)
	GetTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, keyword, assigneeID string, offset, limit int) ([]*ticket.Ticket, error)
	CountTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, keyword, assigneeID string) (int, error)
	GetTicketsBySpan(ctx context.Context, begin, end time.Time) ([]*ticket.Ticket, error)
	GetTicketsByStatusAndSpan(ctx context.Context, status types.TicketStatus, begin, end time.Time) ([]*ticket.Ticket, error)

	// For comment management
	PutTicketComment(ctx context.Context, comment ticket.Comment) error
	GetTicketComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error)
	GetTicketCommentsPaginated(ctx context.Context, ticketID types.TicketID, offset, limit int) ([]ticket.Comment, error)
	CountTicketComments(ctx context.Context, ticketID types.TicketID) (int, error)
	GetTicketUnpromptedComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error)
	PutTicketCommentsPrompted(ctx context.Context, ticketID types.TicketID, commentIDs []types.CommentID) error

	BindAlertsToTicket(ctx context.Context, alertIDs []types.AlertID, ticketID types.TicketID) error
	UnbindAlertFromTicket(ctx context.Context, alertID types.AlertID) error

	PutAlert(ctx context.Context, alert alert.Alert) error
	BatchPutAlerts(ctx context.Context, alerts alert.Alerts) error
	GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error)
	GetLatestAlertByThread(ctx context.Context, thread slack.Thread) (*alert.Alert, error)
	GetAlertsByThread(ctx context.Context, thread slack.Thread) (alert.Alerts, error)
	SearchAlerts(ctx context.Context, path, op string, value any, limit int) (alert.Alerts, error)
	GetAlertWithoutTicket(ctx context.Context, offset, limit int) (alert.Alerts, error)
	CountAlertsWithoutTicket(ctx context.Context) (int, error)
	GetDeclinedAlerts(ctx context.Context, offset, limit int) (alert.Alerts, error)
	CountDeclinedAlerts(ctx context.Context) (int, error)
	UpdateAlertStatus(ctx context.Context, alertID types.AlertID, status alert.AlertStatus) error
	GetAlertsBySpan(ctx context.Context, begin, end time.Time) (alert.Alerts, error)
	BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) (alert.Alerts, error)
	FindNearestAlerts(ctx context.Context, embedding []float32, limit int) (alert.Alerts, error)

	GetLatestHistory(ctx context.Context, ticketID types.TicketID) (*ticket.History, error)
	PutHistory(ctx context.Context, ticketID types.TicketID, history *ticket.History) error

	// For list management
	GetAlertList(ctx context.Context, listID types.AlertListID) (*alert.List, error)
	PutAlertList(ctx context.Context, list *alert.List) error
	GetAlertListByThread(ctx context.Context, thread slack.Thread) (*alert.List, error)
	GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error)
	GetAlertListsInThread(ctx context.Context, thread slack.Thread) ([]*alert.List, error)

	GetAlertWithoutEmbedding(ctx context.Context) (alert.Alerts, error)
	GetAlertsWithInvalidEmbedding(ctx context.Context) (alert.Alerts, error)
	GetTicketsWithInvalidEmbedding(ctx context.Context) ([]*ticket.Ticket, error)

	// GetAllAlerts returns all alert records. Used by diagnosis rules for full-scan checks.
	GetAllAlerts(ctx context.Context) (alert.Alerts, error)
	// GetAllTickets returns all ticket records. Used by diagnosis rules for full-scan checks.
	GetAllTickets(ctx context.Context) ([]*ticket.Ticket, error)

	// For authentication management
	PutToken(ctx context.Context, token *auth.Token) error
	GetToken(ctx context.Context, tokenID auth.TokenID) (*auth.Token, error)
	DeleteToken(ctx context.Context, tokenID auth.TokenID) error

	// For activity management
	PutActivity(ctx context.Context, activity *activity.Activity) error
	GetActivities(ctx context.Context, offset, limit int) ([]*activity.Activity, error)
	CountActivities(ctx context.Context) (int, error)

	// For tag management (legacy - deprecated, kept for external compatibility)
	RemoveTagFromAllAlerts(ctx context.Context, name string) error
	RemoveTagFromAllTickets(ctx context.Context, name string) error

	// For new tag management (ID-based)
	GetTagByID(ctx context.Context, tagID string) (*tag.Tag, error)
	GetTagsByIDs(ctx context.Context, tagIDs []string) ([]*tag.Tag, error)
	ListAllTags(ctx context.Context) ([]*tag.Tag, error)
	CreateTagWithID(ctx context.Context, tag *tag.Tag) error
	UpdateTag(ctx context.Context, tag *tag.Tag) error
	DeleteTagByID(ctx context.Context, tagID string) error
	RemoveTagIDFromAllAlerts(ctx context.Context, tagID string) error
	RemoveTagIDFromAllTickets(ctx context.Context, tagID string) error

	// For tag name lookup (compatibility)
	GetTagByName(ctx context.Context, name string) (*tag.Tag, error)
	IsTagNameExists(ctx context.Context, name string) (bool, error)
	GetOrCreateTagByName(ctx context.Context, name, description, color, createdBy string) (*tag.Tag, error)

	// For notice management
	CreateNotice(ctx context.Context, notice *notice.Notice) error
	GetNotice(ctx context.Context, id types.NoticeID) (*notice.Notice, error)
	UpdateNotice(ctx context.Context, notice *notice.Notice) error

	// For agent memory management
	// Note: Agent memories are stored in subcollection: agents/{agentID}/memories/{memoryID}
	SaveAgentMemory(ctx context.Context, mem *memory.AgentMemory) error
	GetAgentMemory(ctx context.Context, agentID string, id types.AgentMemoryID) (*memory.AgentMemory, error)
	SearchMemoriesByEmbedding(ctx context.Context, agentID string, embedding []float32, limit int) ([]*memory.AgentMemory, error)

	// BatchSaveAgentMemories saves multiple agent memories efficiently in a batch
	// Uses batch write operations to minimize round trips to the database
	BatchSaveAgentMemories(ctx context.Context, memories []*memory.AgentMemory) error

	// Memory scoring methods
	// UpdateMemoryScoreBatch updates quality scores and last used timestamps for multiple agent memories
	UpdateMemoryScoreBatch(ctx context.Context, agentID string, updates map[types.AgentMemoryID]struct {
		Score      float64
		LastUsedAt time.Time
	}) error

	// DeleteAgentMemoriesBatch deletes multiple agent memories in a batch
	// Returns the number of successfully deleted memories
	DeleteAgentMemoriesBatch(ctx context.Context, agentID string, memoryIDs []types.AgentMemoryID) (int, error)

	// ListAgentMemories lists all memories for an agent (for pruning)
	// Results are ordered by CreatedAt DESC
	ListAgentMemories(ctx context.Context, agentID string) ([]*memory.AgentMemory, error)

	// ListAgentMemoriesWithOptions lists memories with filtering, sorting, and pagination
	// Returns memories and total count (before pagination)
	ListAgentMemoriesWithOptions(ctx context.Context, agentID string, opts AgentMemoryListOptions) ([]*memory.AgentMemory, int, error)

	// ListAllAgentIDs returns all agent IDs that have memories with their counts and latest memory timestamp
	// Used for the agent summary list in the UI
	ListAllAgentIDs(ctx context.Context) ([]*AgentSummary, error)

	// Refine group management
	PutRefineGroup(ctx context.Context, group *refine.Group) error
	GetRefineGroup(ctx context.Context, groupID types.RefineGroupID) (*refine.Group, error)

	// Session management
	PutSession(ctx context.Context, session *session.Session) error
	GetSession(ctx context.Context, sessionID types.SessionID) (*session.Session, error)
	GetSessionsByTicket(ctx context.Context, ticketID types.TicketID) ([]*session.Session, error)
	DeleteSession(ctx context.Context, sessionID types.SessionID) error

	// Session message management
	PutSessionMessage(ctx context.Context, message *session.Message) error
	GetSessionMessages(ctx context.Context, sessionID types.SessionID) ([]*session.Message, error)

	// Diagnosis management
	// PutDiagnosis saves or updates a diagnosis header record.
	PutDiagnosis(ctx context.Context, d *diagnosis.Diagnosis) error
	// GetDiagnosis retrieves a diagnosis by ID.
	GetDiagnosis(ctx context.Context, id types.DiagnosisID) (*diagnosis.Diagnosis, error)
	// ListDiagnoses returns a paginated list of diagnoses ordered by CreatedAt DESC.
	// Returns the diagnoses, total count, and any error.
	ListDiagnoses(ctx context.Context, offset, limit int) ([]*diagnosis.Diagnosis, int, error)

	// Diagnosis issue management (subcollection: diagnoses/{id}/issues/{issueID})
	// PutDiagnosisIssue saves or updates a single issue.
	PutDiagnosisIssue(ctx context.Context, issue *diagnosis.Issue) error
	// ListDiagnosisIssues returns a paginated list of issues for a diagnosis.
	// status and ruleID are optional server-side filters (nil means no filter).
	// Returns the issues, total matching count, and any error.
	ListDiagnosisIssues(ctx context.Context, diagnosisID types.DiagnosisID, offset, limit int, status *diagnosis.IssueStatus, ruleID *diagnosis.RuleID) ([]*diagnosis.Issue, int, error)
	// GetDiagnosisIssue retrieves a specific issue by diagnosisID and issueID.
	GetDiagnosisIssue(ctx context.Context, diagnosisID types.DiagnosisID, issueID string) (*diagnosis.Issue, error)
	// CountDiagnosisIssues returns the number of issues for a diagnosis.
	// If status is nil, counts all issues; otherwise counts only issues with the given status.
	CountDiagnosisIssues(ctx context.Context, diagnosisID types.DiagnosisID, status *diagnosis.IssueStatus) (int, error)
	// GetDiagnosisIssueCounts returns all status counts for a diagnosis in a single operation.
	GetDiagnosisIssueCounts(ctx context.Context, diagnosisID types.DiagnosisID) (diagnosis.IssueCounts, error)
	// BatchGetDiagnosisIssueCounts returns issue counts for multiple diagnoses.
	BatchGetDiagnosisIssueCounts(ctx context.Context, diagnosisIDs []types.DiagnosisID) (map[types.DiagnosisID]diagnosis.IssueCounts, error)
	// ListPendingDiagnosisIssues returns all pending issues for a diagnosis (no pagination).
	ListPendingDiagnosisIssues(ctx context.Context, diagnosisID types.DiagnosisID) ([]*diagnosis.Issue, error)

	// HITL request management
	PutHITLRequest(ctx context.Context, req *hitl.Request) error
	GetHITLRequest(ctx context.Context, id types.HITLRequestID) (*hitl.Request, error)
	UpdateHITLRequestStatus(ctx context.Context, id types.HITLRequestID, status hitl.Status, respondedBy string, response map[string]any) error

	// WatchHITLRequest watches for changes to a HITL request document.
	// Returns a channel that receives the updated request when status changes.
	// The channel is closed when the context is cancelled or an error occurs.
	// The error channel receives any errors during watching.
	WatchHITLRequest(ctx context.Context, id types.HITLRequestID) (<-chan *hitl.Request, <-chan error)

	// Queued alert management (circuit breaker)
	PutQueuedAlert(ctx context.Context, qa *alert.QueuedAlert) error
	GetQueuedAlert(ctx context.Context, id types.QueuedAlertID) (*alert.QueuedAlert, error)
	ListQueuedAlerts(ctx context.Context, offset, limit int) ([]*alert.QueuedAlert, error)
	DeleteQueuedAlerts(ctx context.Context, ids []types.QueuedAlertID) error
	CountQueuedAlerts(ctx context.Context) (int, error)
	SearchQueuedAlerts(ctx context.Context, keyword string, offset, limit int) ([]*alert.QueuedAlert, int, error)

	// Reprocess job management
	PutReprocessJob(ctx context.Context, job *alert.ReprocessJob) error
	GetReprocessJob(ctx context.Context, id types.ReprocessJobID) (*alert.ReprocessJob, error)

	// Reprocess batch job management
	PutReprocessBatchJob(ctx context.Context, job *alert.ReprocessBatchJob) error
	GetReprocessBatchJob(ctx context.Context, id types.ReprocessBatchJobID) (*alert.ReprocessBatchJob, error)

	// Alert throttle management (sliding window rate limiting)
	// CheckAlertThrottle checks whether throttle slots are available (read-only).
	// Does NOT consume a slot. Used for optimistic early rejection before pipeline.
	CheckAlertThrottle(ctx context.Context, window time.Duration, limit int) (*alert.ThrottleResult, error)

	// AcquireAlertThrottleSlot atomically checks and consumes a throttle slot.
	// Returns the result indicating whether the slot was acquired and whether notification is needed.
	// Used after pipeline completion for each non-discarded alert.
	AcquireAlertThrottleSlot(ctx context.Context, window time.Duration, limit int) (*alert.ThrottleResult, error)

	// Knowledge: topic-based knowledge store with category and tags
	// GetKnowledge retrieves a specific knowledge by ID
	GetKnowledge(ctx context.Context, id types.KnowledgeID) (*knowledge.Knowledge, error)
	// PutKnowledge saves or updates a knowledge entry
	PutKnowledge(ctx context.Context, k *knowledge.Knowledge) error
	// DeleteKnowledge physically deletes a knowledge entry
	DeleteKnowledge(ctx context.Context, id types.KnowledgeID) error
	// ListAllKnowledges retrieves all knowledges (for Web UI listing).
	ListAllKnowledges(ctx context.Context) ([]*knowledge.Knowledge, error)
	// ListKnowledgesByCategoryAndTags retrieves knowledges filtered by category and at least one tag.
	ListKnowledgesByCategoryAndTags(ctx context.Context, category types.KnowledgeCategory, tagIDs []types.KnowledgeTagID) ([]*knowledge.Knowledge, error)

	// Knowledge v2 log management (subcollection: knowledges/{id}/logs/{logID})
	GetKnowledgeLog(ctx context.Context, knowledgeID types.KnowledgeID, logID types.KnowledgeLogID) (*knowledge.KnowledgeLog, error)
	ListKnowledgeLogs(ctx context.Context, knowledgeID types.KnowledgeID) ([]*knowledge.KnowledgeLog, error)
	PutKnowledgeLog(ctx context.Context, log *knowledge.KnowledgeLog) error

	// Legacy knowledge migration
	// ListLegacyKnowledges returns all old-format knowledge entries from the topics collection.
	// Returns empty slice if no legacy data exists.
	ListLegacyKnowledges(ctx context.Context) ([]*LegacyKnowledge, error)

	// Knowledge v2 tag management (collection: knowledge_tags)
	GetKnowledgeTag(ctx context.Context, id types.KnowledgeTagID) (*knowledge.KnowledgeTag, error)
	ListKnowledgeTags(ctx context.Context) ([]*knowledge.KnowledgeTag, error)
	PutKnowledgeTag(ctx context.Context, tag *knowledge.KnowledgeTag) error
	DeleteKnowledgeTag(ctx context.Context, id types.KnowledgeTagID) error
}
