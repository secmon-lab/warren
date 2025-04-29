package session_test

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/storage"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	ssn_model "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/action"
	ssn_svc "github.com/secmon-lab/warren/pkg/service/session"
)

func TestHistory(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	actionService, err := action.New(ctx, []interfaces.Action{})
	gt.NoError(t, err)

	ssn := ssn_model.New(ctx, &slack.User{}, &slack.Thread{}, []types.AlertID{})
	clients := interfaces.NewClients(
		interfaces.WithRepository(repo),
		interfaces.WithStorageClient(storage.NewMock()),
	)

	svc := ssn_svc.New(clients, actionService, ssn)

	// Prepare test data
	testHistory := ssn_model.NewHistory(ctx, ssn.ID, []*genai.Content{
		{
			Role: "user",
			Parts: []genai.Part{
				genai.Text("test message"),
			},
		},
	})

	// Test Put operation
	t.Run("put history", func(t *testing.T) {
		err := svc.PutHistory(ctx, testHistory)
		gt.NoError(t, err)
	})

	// Test Get operation
	t.Run("get history", func(t *testing.T) {
		history, err := svc.GetHistory(ctx, ssn.ID)
		gt.NoError(t, err).Required()
		gt.NotNil(t, history)
		gt.Value(t, history.ID).Equal(testHistory.ID)
		gt.Value(t, history.SessionID).Equal(testHistory.SessionID)
		gt.Value(t, history.Contents).Equal(testHistory.Contents)
	})

	// Test with non-existent ID
	t.Run("get non-existent history", func(t *testing.T) {
		nonExistentID := types.NewSessionID()
		hist, err := svc.GetHistory(ctx, nonExistentID)
		gt.NoError(t, err)
		gt.Nil(t, hist)
	})
}

func TestHistoryWithStorage(t *testing.T) {
	ctx := t.Context()
	if os.Getenv("TEST_STORAGE_BUCKET") == "" {
		t.Skip("TEST_STORAGE_BUCKET is not set")
	}

	storageClient, err := storage.New(ctx,
		os.Getenv("TEST_STORAGE_BUCKET"),
		os.Getenv("TEST_STORAGE_PREFIX"),
	)
	gt.NoError(t, err)

	repo := repository.NewMemory()
	actionService, err := action.New(ctx, []interfaces.Action{})
	gt.NoError(t, err)

	ssn := ssn_model.New(ctx, &slack.User{}, &slack.Thread{}, []types.AlertID{})

	testHistory := ssn_model.NewHistory(ctx, ssn.ID, []*genai.Content{
		{
			Role: "user",
			Parts: []genai.Part{
				genai.Text("test message"),
			},
		},
	})

	clients := interfaces.NewClients(
		interfaces.WithRepository(repo),
		interfaces.WithStorageClient(storageClient),
	)

	svc := ssn_svc.New(clients, actionService, ssn)

	// Test Put operation
	t.Run("put history", func(t *testing.T) {
		err := svc.PutHistory(ctx, testHistory)
		gt.NoError(t, err)
	})

	// Test Get operation
	t.Run("get history", func(t *testing.T) {
		history, err := svc.GetHistory(ctx, ssn.ID)
		gt.NoError(t, err).Required()
		gt.NotNil(t, history)
		gt.Value(t, history.ID).Equal(testHistory.ID)
		gt.Value(t, history.SessionID).Equal(testHistory.SessionID)
		gt.Value(t, history.Contents).Equal(testHistory.Contents)
	})
}
