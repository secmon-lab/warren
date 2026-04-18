package session

import "github.com/secmon-lab/warren/pkg/domain/types"

// Author identifies a human author of a user-type Message.
//
// For AI-produced Messages (trace/plan/response/warning) the Message.Author
// field is nil. For user Messages, Author is required and contains the
// minimum information needed to render attribution across all channels.
type Author struct {
	UserID      types.UserID `firestore:"user_id" json:"user_id"`
	DisplayName string       `firestore:"display_name" json:"display_name"`
	// SlackUserID is set when the author is a Slack workspace member.
	SlackUserID *string `firestore:"slack_user_id,omitempty" json:"slack_user_id,omitempty"`
	Email       *string `firestore:"email,omitempty" json:"email,omitempty"`
}
