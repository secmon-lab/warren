package graphql

import (
	"context"

	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

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
	}
}

// memoryToGraphQL converts domain AgentMemory to GraphQL AgentMemory
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
