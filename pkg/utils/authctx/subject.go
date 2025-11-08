package authctx

// SubjectType represents the type of authentication source
type SubjectType string

const (
	SubjectTypeIAP      SubjectType = "iap"
	SubjectTypeGoogleID SubjectType = "google_id"
	SubjectTypeSlack    SubjectType = "slack"
)

// Subject represents an authenticated subject (user or system)
type Subject struct {
	Type   SubjectType `json:"type"`
	UserID string      `json:"user_id"`
	Email  string      `json:"email,omitempty"`
}
