package websocket

import (
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
)

// hitlViewFromRequest converts a domain hitl.Request into the wire
// HITLView shape used by hitl_request_pending / hitl_request_resolved
// envelopes. messageID binds the prompt to an existing progress
// Message so the frontend can render approval UI inline.
func hitlViewFromRequest(req *hitl.Request, messageID string) *HITLView {
	if req == nil {
		return nil
	}
	return &HITLView{
		ID:        string(req.ID),
		SessionID: string(req.SessionID),
		Type:      string(req.Type),
		Status:    string(req.Status),
		UserID:    req.UserID,
		Payload:   req.Payload,
		Response:  req.Response,
		MessageID: messageID,
	}
}
