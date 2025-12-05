package types

import "github.com/m-mizutani/goerr/v2"

// KnowledgeSlug uniquely identifies a knowledge within a topic
type KnowledgeSlug string

func (s KnowledgeSlug) String() string {
	return string(s)
}

func (s KnowledgeSlug) Validate() error {
	if s == "" {
		return goerr.New("knowledge slug is required")
	}
	return nil
}

// KnowledgeTopic is the namespace for knowledge
type KnowledgeTopic string

func (t KnowledgeTopic) String() string {
	return string(t)
}

func (t KnowledgeTopic) Validate() error {
	if t == "" {
		return goerr.New("knowledge topic is required")
	}
	return nil
}

// UserID represents a user identifier (Slack User ID or system)
type UserID string

const (
	SystemUserID UserID = "system"
)

func (u UserID) String() string {
	return string(u)
}

func (u UserID) Validate() error {
	if u == "" {
		return goerr.New("user ID is required")
	}
	return nil
}

func (u UserID) IsSystem() bool {
	return u == SystemUserID
}

// KnowledgeState represents the state of knowledge
type KnowledgeState string

const (
	KnowledgeStateActive   KnowledgeState = "active"
	KnowledgeStateArchived KnowledgeState = "archived"
)

func (s KnowledgeState) String() string {
	return string(s)
}

func (s KnowledgeState) Validate() error {
	switch s {
	case KnowledgeStateActive, KnowledgeStateArchived:
		return nil
	}
	return goerr.New("invalid knowledge state", goerr.V("state", s))
}

func (s KnowledgeState) IsActive() bool {
	return s == KnowledgeStateActive
}

func (s KnowledgeState) IsArchived() bool {
	return s == KnowledgeStateArchived
}
