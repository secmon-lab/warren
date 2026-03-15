package slack

import "time"

// HistoryMessage is a lightweight message model for embedding Slack conversation
// history into LLM system prompts. Unlike the full Message struct which includes
// parser functionality, this focuses on displayable content only.
type HistoryMessage struct {
	UserID    string
	UserName  string
	Text      string
	Timestamp time.Time
	IsBot     bool
	IsThread  bool
}
