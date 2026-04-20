package graphql

import (
	"encoding/json"

	graphql1 "github.com/secmon-lab/warren/pkg/domain/model/graphql"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
)

// hitlRequestToGraphQL converts a domain hitl.Request into the GraphQL
// HITLRequest shape. Payload and Response are serialized as JSON
// strings because the schema uses opaque String fields there — the
// frontend parses the same JSON shape the hitl_request_pending
// envelope ships.
func hitlRequestToGraphQL(r *hitl.Request) *graphql1.HITLRequest {
	if r == nil {
		return nil
	}
	out := &graphql1.HITLRequest{
		ID:        string(r.ID),
		SessionID: string(r.SessionID),
		Type:      string(r.Type),
		Status:    string(r.Status),
		CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
	}
	if r.UserID != "" {
		uid := r.UserID
		out.UserID = &uid
	}
	if len(r.Payload) > 0 {
		if b, err := json.Marshal(r.Payload); err == nil {
			s := string(b)
			out.Payload = &s
		}
	}
	if len(r.Response) > 0 {
		if b, err := json.Marshal(r.Response); err == nil {
			s := string(b)
			out.Response = &s
		}
	}
	if !r.RespondedAt.IsZero() {
		ts := r.RespondedAt.Format("2006-01-02T15:04:05.000Z")
		out.RespondedAt = &ts
	}
	return out
}
