package types

import (
	"strings"

	"github.com/m-mizutani/goerr/v2"
)

// KnowledgeSlug uniquely identifies a knowledge within a topic
type KnowledgeSlug string

func (s KnowledgeSlug) String() string {
	return string(s)
}

func (s KnowledgeSlug) Validate() error {
	if s == "" {
		return goerr.New("knowledge slug is required")
	}

	// Check for Firestore forbidden characters
	// Firestore document IDs cannot contain: / \ . (period at start/end) __
	slug := string(s)

	if strings.Contains(slug, "/") {
		return goerr.New("knowledge slug contains forbidden character '/'",
			goerr.V("slug", slug),
			goerr.V("forbidden_char", "/"),
			goerr.V("reason", "Firestore document IDs cannot contain forward slashes"))
	}

	if strings.Contains(slug, "\\") {
		return goerr.New("knowledge slug contains forbidden character '\\'",
			goerr.V("slug", slug),
			goerr.V("forbidden_char", "\\"),
			goerr.V("reason", "Firestore document IDs cannot contain backslashes"))
	}

	if strings.HasPrefix(slug, ".") {
		return goerr.New("knowledge slug cannot start with '.'",
			goerr.V("slug", slug),
			goerr.V("reason", "Firestore document IDs cannot start with a period"))
	}

	if strings.HasSuffix(slug, ".") {
		return goerr.New("knowledge slug cannot end with '.'",
			goerr.V("slug", slug),
			goerr.V("reason", "Firestore document IDs cannot end with a period"))
	}

	if strings.Contains(slug, "__") {
		return goerr.New("knowledge slug contains forbidden sequence '__'",
			goerr.V("slug", slug),
			goerr.V("forbidden_sequence", "__"),
			goerr.V("reason", "Firestore document IDs cannot contain double underscores"))
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

	// Check for Firestore forbidden characters
	// Firestore document IDs cannot contain: / \ . (period at start/end) __
	topic := string(t)

	if strings.Contains(topic, "/") {
		return goerr.New("knowledge topic contains forbidden character '/'",
			goerr.V("topic", topic),
			goerr.V("forbidden_char", "/"),
			goerr.V("reason", "Firestore document IDs cannot contain forward slashes"))
	}

	if strings.Contains(topic, "\\") {
		return goerr.New("knowledge topic contains forbidden character '\\'",
			goerr.V("topic", topic),
			goerr.V("forbidden_char", "\\"),
			goerr.V("reason", "Firestore document IDs cannot contain backslashes"))
	}

	if strings.HasPrefix(topic, ".") {
		return goerr.New("knowledge topic cannot start with '.'",
			goerr.V("topic", topic),
			goerr.V("reason", "Firestore document IDs cannot start with a period"))
	}

	if strings.HasSuffix(topic, ".") {
		return goerr.New("knowledge topic cannot end with '.'",
			goerr.V("topic", topic),
			goerr.V("reason", "Firestore document IDs cannot end with a period"))
	}

	if strings.Contains(topic, "__") {
		return goerr.New("knowledge topic contains forbidden sequence '__'",
			goerr.V("topic", topic),
			goerr.V("forbidden_sequence", "__"),
			goerr.V("reason", "Firestore document IDs cannot contain double underscores"))
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
