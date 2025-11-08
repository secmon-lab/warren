package authctx

import (
	"github.com/m-mizutani/goerr/v2"
)

// NewSubjectFromIAP creates Subject from IAP JWT claims
func NewSubjectFromIAP(claims map[string]interface{}) (Subject, error) {
	email, ok := claims["email"].(string)
	if !ok || email == "" {
		return Subject{}, goerr.New("email not found in IAP claims")
	}

	// IAP uses "sub" as the user identifier
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return Subject{}, goerr.New("sub not found in IAP claims")
	}

	return Subject{
		Type:   SubjectTypeIAP,
		UserID: sub,
		Email:  email,
	}, nil
}

// NewSubjectFromGoogleID creates Subject from Google ID token claims
func NewSubjectFromGoogleID(claims map[string]interface{}) (Subject, error) {
	email, ok := claims["email"].(string)
	if !ok || email == "" {
		return Subject{}, goerr.New("email not found in Google ID token claims")
	}

	// Google ID token uses "sub" as the user identifier
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return Subject{}, goerr.New("sub not found in Google ID token claims")
	}

	return Subject{
		Type:   SubjectTypeGoogleID,
		UserID: sub,
		Email:  email,
	}, nil
}

// NewSubjectFromSlackUser creates Subject from Slack user information
// Returns error if userID is empty
func NewSubjectFromSlackUser(userID string) (Subject, error) {
	if userID == "" {
		return Subject{}, goerr.New("Slack user ID is required")
	}

	return Subject{
		Type:   SubjectTypeSlack,
		UserID: userID,
		Email:  "", // Slack user ID doesn't include email by default
	}, nil
}
