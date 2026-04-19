package graphql

import (
	"context"
	"regexp"

	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// slackUserIDRegexp matches a bare Slack user ID (e.g. "U08A3TTRENS",
// "W12345678"). When the persisted Author.DisplayName looks like a raw
// Slack ID we re-resolve it via the Slack profile API at read time so
// the Conversation UI shows a human-readable name. This auto-heals
// pre-fix rows whose DisplayName was written as the user ID.
var slackUserIDRegexp = regexp.MustCompile(`^[UW][A-Z0-9]{7,}$`)

// toGraphQLSessionMessage converts a domain session.Message to its
// GraphQL representation including chat-session-redesign fields (TurnID,
// TicketID, Author).
func toGraphQLSessionMessage(m *session.Message) *graphql1.SessionMessage {
	out := &graphql1.SessionMessage{
		ID:        string(m.ID),
		SessionID: string(m.SessionID),
		Type:      string(m.Type),
		Content:   m.Content,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if m.TurnID != nil {
		turn := string(*m.TurnID)
		out.TurnID = &turn
	}
	if m.TicketID != nil {
		tid := string(*m.TicketID)
		out.TicketID = &tid
	}
	if m.Author != nil {
		out.Author = &graphql1.MessageAuthor{
			UserID:      string(m.Author.UserID),
			DisplayName: m.Author.DisplayName,
			SlackUserID: m.Author.SlackUserID,
			Email:       m.Author.Email,
		}
	}
	if len(m.Revisions) > 0 {
		out.Revisions = make([]*graphql1.MessageRevision, len(m.Revisions))
		for i, rev := range m.Revisions {
			out.Revisions[i] = &graphql1.MessageRevision{
				Content:   rev.Content,
				CreatedAt: rev.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		}
	}
	return out
}

// resolveAuthorDisplayName patches out.Author.DisplayName when it was
// persisted as a bare Slack user ID (the pre-fix behavior for
// pre-resolved profiles). Non-Slack authors and already-resolved rows
// are returned unchanged. Lookup errors fall through silently — the
// raw ID is a usable fallback.
func resolveAuthorDisplayName(ctx context.Context, svc *slackService.Service, out *graphql1.SessionMessage) {
	if svc == nil || out == nil || out.Author == nil {
		return
	}
	a := out.Author
	// Only attempt Slack lookup when the author is a Slack member —
	// Web/CLI authors have their own DisplayName and their UserIDs do
	// not match the Slack ID format anyway.
	if a.SlackUserID == nil || *a.SlackUserID == "" {
		return
	}
	if a.DisplayName != "" && !slackUserIDRegexp.MatchString(a.DisplayName) {
		return
	}
	name, err := svc.GetUserProfile(ctx, *a.SlackUserID)
	if err != nil {
		logging.From(ctx).Debug("failed to resolve slack display name for session message",
			"error", err, "slack_user_id", *a.SlackUserID)
		return
	}
	if name != "" {
		a.DisplayName = name
	}
}

// toGraphQLSession converts a domain session.Session to a GraphQL Session
func toGraphQLSession(s *session.Session) *graphql1.Session {
	var userID, query, slackURL, intent *string
	if s.UserID != "" {
		uid := string(s.UserID)
		userID = &uid
	}
	if s.Query != "" {
		query = &s.Query
	}
	if s.SlackURL != "" {
		slackURL = &s.SlackURL
	}
	if s.Intent != "" {
		intent = &s.Intent
	}

	return &graphql1.Session{
		ID:        string(s.ID),
		TicketID:  string(s.TicketID),
		Status:    s.Status.String(),
		UserID:    userID,
		Query:     query,
		SlackURL:  slackURL,
		Intent:    intent,
		CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Source:    string(s.Source),
	}
}

// authFromContext extracts the user ID from the context for knowledge operations.
// Falls back to system user.
func authFromContext(_ context.Context) types.UserID {
	// TODO: extract actual user ID from auth context when available
	return types.SystemUserID
}

// knowledgeToGraphQL converts a domain Knowledge to GraphQL Knowledge.
func knowledgeToGraphQL(ctx context.Context, r *Resolver, k *knowledge.Knowledge) *graphql1.Knowledge {
	tags := make([]*graphql1.KnowledgeTag, 0, len(k.Tags))
	for _, tagID := range k.Tags {
		if r.knowledgeSvc != nil {
			tag, err := r.repo.GetKnowledgeTag(ctx, tagID)
			if err == nil && tag != nil {
				tags = append(tags, knowledgeTagToGraphQL(tag))
			}
		}
	}

	return &graphql1.Knowledge{
		ID:        k.ID.String(),
		Category:  string(k.Category),
		Title:     k.Title,
		Claim:     k.Claim,
		Tags:      tags,
		AuthorID:  k.Author.String(),
		CreatedAt: k.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: k.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// knowledgeTagToGraphQL converts a domain KnowledgeTag to GraphQL KnowledgeTag.
func knowledgeTagToGraphQL(t *knowledge.KnowledgeTag) *graphql1.KnowledgeTag {
	return &graphql1.KnowledgeTag{
		ID:          t.ID.String(),
		Name:        t.Name,
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// knowledgeLogToGraphQL converts a domain KnowledgeLog to GraphQL KnowledgeLog.
func knowledgeLogToGraphQL(l *knowledge.KnowledgeLog) *graphql1.KnowledgeLog {
	var ticketID *string
	if l.TicketID != "" {
		tid := l.TicketID.String()
		ticketID = &tid
	}
	return &graphql1.KnowledgeLog{
		ID:          l.ID.String(),
		KnowledgeID: l.KnowledgeID.String(),
		Title:       l.Title,
		Claim:       l.Claim,
		AuthorID:    l.Author.String(),
		TicketID:    ticketID,
		Message:     l.Message,
		CreatedAt:   l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
