package types

import "github.com/google/uuid"

type MessageID string

func NewMessageID() MessageID {
	return MessageID(uuid.New().String())
}

// DeterministicMessageID returns a stable UUID derived from the given
// external identifier (e.g. a Slack message ts). The caller is
// responsible for ensuring externalKey is unique within the scope it
// intends to dedupe — typically (sessionID + slack_ts) — so that
// retried Slack event deliveries or duplicate `message` +
// `message_changed` deliveries collapse onto a single SessionMessage
// row rather than appending duplicates.
//
// Uses UUIDv5 with a fixed namespace so the same externalKey always
// yields the same MessageID across processes and restarts.
func DeterministicMessageID(externalKey string) MessageID {
	// A random-but-fixed namespace UUID: generated once, hard-coded
	// here so every warren instance produces the same IDs for the
	// same externalKey.
	ns := uuid.MustParse("7f9c8a2e-5b33-4d9a-9e2f-1a6a9b4c0d2f")
	return MessageID(uuid.NewSHA1(ns, []byte(externalKey)).String())
}

func (x MessageID) String() string {
	return string(x)
}
