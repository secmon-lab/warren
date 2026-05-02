package session

import (
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
)

// SessionSource identifies the channel that a Session is bound to. A Session
// is tied to exactly one source for its lifetime.
type SessionSource string

const (
	// SessionSourceSlack indicates the Session represents a Slack thread
	// conversation. Slack Sessions are resolved from (ticket_id, thread) and
	// persist across multiple @warren mentions.
	SessionSourceSlack SessionSource = "slack"

	// SessionSourceWeb indicates the Session was started from the Web UI via
	// a WebSocket connection. A new Web Session is created for each
	// connection / Start-Chat action.
	SessionSourceWeb SessionSource = "web"

	// SessionSourceCLI indicates the Session was started from the `warren
	// chat` CLI command. A new CLI Session is created per invocation; an
	// interactive session may contain multiple Turns within it.
	SessionSourceCLI SessionSource = "cli"
)

// Valid reports whether the SessionSource is one of the known values.
func (s SessionSource) Valid() bool {
	switch s {
	case SessionSourceSlack, SessionSourceWeb, SessionSourceCLI:
		return true
	}
	return false
}

// ChannelRef captures the external channel a Session is bound to.
//
// For Slack Sessions, SlackThread is populated and identifies the specific
// channel+thread pair. For Web and CLI Sessions, ChannelRef is nil.
type ChannelRef struct {
	SlackThread *slack.Thread `firestore:"slack_thread,omitempty" json:"slack_thread,omitempty"`
}
