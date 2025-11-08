package authctx_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/authctx"
)

func TestGetSubjects_Empty(t *testing.T) {
	ctx := context.Background()
	subjects := authctx.GetSubjects(ctx)
	gt.Equal(t, len(subjects), 0)
	gt.True(t, subjects != nil)
}

func TestWithSubject_Single(t *testing.T) {
	ctx := context.Background()
	subject := authctx.Subject{
		Type:   authctx.SubjectTypeSlack,
		UserID: "U12345",
		Email:  "",
	}

	ctx = authctx.WithSubject(ctx, subject)
	subjects := authctx.GetSubjects(ctx)

	gt.Equal(t, len(subjects), 1)
	gt.Equal(t, subjects[0].Type, authctx.SubjectTypeSlack)
	gt.Equal(t, subjects[0].UserID, "U12345")
	gt.Equal(t, subjects[0].Email, "")
}

func TestWithSubject_Multiple(t *testing.T) {
	ctx := context.Background()

	iapSubject := authctx.Subject{
		Type:   authctx.SubjectTypeIAP,
		UserID: "iap-user-123",
		Email:  "user@example.com",
	}

	slackSubject := authctx.Subject{
		Type:   authctx.SubjectTypeSlack,
		UserID: "U12345",
		Email:  "",
	}

	ctx = authctx.WithSubject(ctx, iapSubject)
	ctx = authctx.WithSubject(ctx, slackSubject)

	subjects := authctx.GetSubjects(ctx)

	gt.Equal(t, len(subjects), 2)
	gt.Equal(t, subjects[0].Type, authctx.SubjectTypeIAP)
	gt.Equal(t, subjects[0].UserID, "iap-user-123")
	gt.Equal(t, subjects[0].Email, "user@example.com")
	gt.Equal(t, subjects[1].Type, authctx.SubjectTypeSlack)
	gt.Equal(t, subjects[1].UserID, "U12345")
}

func TestWithSubject_ThreeSubjects(t *testing.T) {
	ctx := context.Background()

	iapSubject := authctx.Subject{
		Type:   authctx.SubjectTypeIAP,
		UserID: "iap-user",
		Email:  "iap@example.com",
	}

	googleSubject := authctx.Subject{
		Type:   authctx.SubjectTypeGoogleID,
		UserID: "google-user",
		Email:  "google@example.com",
	}

	slackSubject := authctx.Subject{
		Type:   authctx.SubjectTypeSlack,
		UserID: "U12345",
		Email:  "",
	}

	ctx = authctx.WithSubject(ctx, iapSubject)
	ctx = authctx.WithSubject(ctx, googleSubject)
	ctx = authctx.WithSubject(ctx, slackSubject)

	subjects := authctx.GetSubjects(ctx)

	gt.Equal(t, len(subjects), 3)
	gt.Equal(t, subjects[0].Type, authctx.SubjectTypeIAP)
	gt.Equal(t, subjects[1].Type, authctx.SubjectTypeGoogleID)
	gt.Equal(t, subjects[2].Type, authctx.SubjectTypeSlack)
}

// TestGetSubjects_Immutability verifies that modifying the returned slice
// does not affect the context's internal state
func TestGetSubjects_Immutability(t *testing.T) {
	ctx := context.Background()

	subject1 := authctx.Subject{
		Type:   authctx.SubjectTypeIAP,
		UserID: "user-1",
		Email:  "user1@example.com",
	}

	ctx = authctx.WithSubject(ctx, subject1)

	// Get subjects and modify the returned slice
	subjects1 := authctx.GetSubjects(ctx)
	subjects1 = append(subjects1, authctx.Subject{
		Type:   authctx.SubjectTypeSlack,
		UserID: "U99999",
		Email:  "",
	})

	// Verify the local slice was modified
	gt.Equal(t, len(subjects1), 2)

	// Get subjects again - should not include the appended subject (verifies immutability)
	subjects2 := authctx.GetSubjects(ctx)
	gt.Equal(t, len(subjects2), 1)
	gt.Equal(t, subjects2[0].UserID, "user-1")
}

// TestWithSubject_Immutability verifies that adding a new subject
// does not modify subjects retrieved from the original context
func TestWithSubject_Immutability(t *testing.T) {
	ctx := context.Background()

	subject1 := authctx.Subject{
		Type:   authctx.SubjectTypeIAP,
		UserID: "user-1",
		Email:  "user1@example.com",
	}

	ctx1 := authctx.WithSubject(ctx, subject1)

	// Get subjects from first context
	subjects1 := authctx.GetSubjects(ctx1)
	gt.Equal(t, len(subjects1), 1)

	// Add another subject to create ctx2
	subject2 := authctx.Subject{
		Type:   authctx.SubjectTypeSlack,
		UserID: "U12345",
		Email:  "",
	}
	ctx2 := authctx.WithSubject(ctx1, subject2)

	// Get subjects from both contexts
	subjects1Again := authctx.GetSubjects(ctx1)
	subjects2 := authctx.GetSubjects(ctx2)

	// ctx1 should still have only 1 subject
	gt.Equal(t, len(subjects1Again), 1)
	gt.Equal(t, subjects1Again[0].UserID, "user-1")

	// ctx2 should have 2 subjects
	gt.Equal(t, len(subjects2), 2)
	gt.Equal(t, subjects2[0].UserID, "user-1")
	gt.Equal(t, subjects2[1].UserID, "U12345")
}
