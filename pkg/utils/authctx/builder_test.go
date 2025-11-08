package authctx_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/authctx"
)

func TestNewSubjectFromIAP_Success(t *testing.T) {
	claims := map[string]interface{}{
		"sub":   "iap-user-123",
		"email": "user@example.com",
		"aud":   "/projects/123/apps/myapp",
	}

	subject, err := authctx.NewSubjectFromIAP(claims)
	gt.NoError(t, err)
	gt.Equal(t, subject.Type, authctx.SubjectTypeIAP)
	gt.Equal(t, subject.UserID, "iap-user-123")
	gt.Equal(t, subject.Email, "user@example.com")
}

func TestNewSubjectFromIAP_MissingEmail(t *testing.T) {
	claims := map[string]interface{}{
		"sub": "iap-user-123",
	}

	_, err := authctx.NewSubjectFromIAP(claims)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("email not found")
}

func TestNewSubjectFromIAP_MissingSub(t *testing.T) {
	claims := map[string]interface{}{
		"email": "user@example.com",
	}

	_, err := authctx.NewSubjectFromIAP(claims)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("sub not found")
}

func TestNewSubjectFromIAP_EmptyEmail(t *testing.T) {
	claims := map[string]interface{}{
		"sub":   "iap-user-123",
		"email": "",
	}

	_, err := authctx.NewSubjectFromIAP(claims)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("email not found")
}

func TestNewSubjectFromGoogleID_Success(t *testing.T) {
	claims := map[string]interface{}{
		"sub":   "google-user-456",
		"email": "google@example.com",
		"aud":   "my-client-id.apps.googleusercontent.com",
	}

	subject, err := authctx.NewSubjectFromGoogleID(claims)
	gt.NoError(t, err)
	gt.Equal(t, subject.Type, authctx.SubjectTypeGoogleID)
	gt.Equal(t, subject.UserID, "google-user-456")
	gt.Equal(t, subject.Email, "google@example.com")
}

func TestNewSubjectFromGoogleID_MissingEmail(t *testing.T) {
	claims := map[string]interface{}{
		"sub": "google-user-456",
	}

	_, err := authctx.NewSubjectFromGoogleID(claims)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("email not found")
}

func TestNewSubjectFromGoogleID_MissingSub(t *testing.T) {
	claims := map[string]interface{}{
		"email": "google@example.com",
	}

	_, err := authctx.NewSubjectFromGoogleID(claims)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("sub not found")
}

func TestNewSubjectFromSlackUser(t *testing.T) {
	subject, err := authctx.NewSubjectFromSlackUser("U12345")
	gt.NoError(t, err)
	gt.Equal(t, subject.Type, authctx.SubjectTypeSlack)
	gt.Equal(t, subject.UserID, "U12345")
	gt.Equal(t, subject.Email, "")
}

func TestNewSubjectFromSlackUser_EmptyUserID(t *testing.T) {
	_, err := authctx.NewSubjectFromSlackUser("")
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("Slack user ID is required")
}
