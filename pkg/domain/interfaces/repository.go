package interfaces

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
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
	GetTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, offset, limit int) ([]*ticket.Ticket, error)
	CountTicketsByStatus(ctx context.Context, statuses []types.TicketStatus) (int, error)
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

	// Session management
	PutSession(ctx context.Context, session *session.Session) error
	GetSession(ctx context.Context, sessionID types.SessionID) (*session.Session, error)
	GetSessionsByTicket(ctx context.Context, ticketID types.TicketID) ([]*session.Session, error)
	DeleteSession(ctx context.Context, sessionID types.SessionID) error

	// Session message management
	PutSessionMessage(ctx context.Context, message *session.Message) error
	GetSessionMessages(ctx context.Context, sessionID types.SessionID) ([]*session.Message, error)

	// Knowledge memory management
	// Knowledges are stored with topic/slug as composite key
	// Each knowledge maintains version history via commit IDs

	// GetKnowledges retrieves all non-archived latest knowledges for a topic
	GetKnowledges(ctx context.Context, topic types.KnowledgeTopic) ([]*knowledge.Knowledge, error)

	// GetKnowledge retrieves a specific knowledge by topic and slug (latest version)
	GetKnowledge(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug) (*knowledge.Knowledge, error)

	// GetKnowledgeByCommit retrieves a specific version by topic, slug, and commit ID
	GetKnowledgeByCommit(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug, commitID string) (*knowledge.Knowledge, error)

	// ListKnowledgeSlugs returns all non-archived slugs and names for a topic
	ListKnowledgeSlugs(ctx context.Context, topic types.KnowledgeTopic) ([]*knowledge.SlugInfo, error)

	// PutKnowledge saves a new version of a knowledge (append-only)
	// CommitID must be generated by GenerateCommitID before calling
	PutKnowledge(ctx context.Context, k *knowledge.Knowledge) error

	// ArchiveKnowledge marks all versions of a slug as archived
	ArchiveKnowledge(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug) error

	// CalculateKnowledgeSize returns total content size for non-archived knowledges in a topic
	CalculateKnowledgeSize(ctx context.Context, topic types.KnowledgeTopic) (int, error)

	// ListKnowledgeTopics returns all topics with their knowledge counts (non-archived only)
	ListKnowledgeTopics(ctx context.Context) ([]*knowledge.TopicSummary, error)
}
