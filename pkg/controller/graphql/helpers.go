package graphql

import (
	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
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
