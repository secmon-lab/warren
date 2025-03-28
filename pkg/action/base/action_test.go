package base_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/action/base"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestBase(t *testing.T) {
	ctx := context.Background()
	alerts := alert.Alerts{
		{
			ID:          types.NewAlertID(),
			Schema:      types.AlertSchema("test"),
			Title:       "Test Alert 1",
			Description: "Test Description 1",
			Status:      types.AlertStatusNew,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          types.NewAlertID(),
			Schema:      types.AlertSchema("test"),
			Title:       "Test Alert 2",
			Description: "Test Description 2",
			Status:      types.AlertStatusNew,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          types.NewAlertID(),
			Schema:      types.AlertSchema("test"),
			Title:       "Test Alert 3",
			Description: "Test Description 3",
			Status:      types.AlertStatusNew,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	baseAction := base.New(&interfaces.RepositoryMock{}, alerts)

	t.Run("get_alerts without pagination", func(t *testing.T) {
		result, err := baseAction.Execute(ctx, "base.get_alerts", map[string]any{})
		gt.NoError(t, err)
		gt.Value(t, result.Type).Equal(action.ResultTypeJSON)
		gt.Value(t, result.Message).Equal("Retrieved alerts")
		gt.Array(t, result.Rows).Length(3)
	})

	t.Run("get_alerts with limit", func(t *testing.T) {
		result, err := baseAction.Execute(ctx, "base.get_alerts", map[string]any{
			"limit": float64(2),
		})
		gt.NoError(t, err)
		gt.Array(t, result.Rows).Length(2)
	})

	t.Run("get_alerts with offset", func(t *testing.T) {
		result, err := baseAction.Execute(ctx, "base.get_alerts", map[string]any{
			"offset": float64(1),
		})
		gt.NoError(t, err)
		gt.Array(t, result.Rows).Length(2)
	})

	t.Run("get_alerts with offset beyond length", func(t *testing.T) {
		result, err := baseAction.Execute(ctx, "base.get_alerts", map[string]any{
			"offset": float64(10),
		})
		gt.NoError(t, err)
		gt.Array(t, result.Rows).Length(0)
	})

	t.Run("exit with conclusion", func(t *testing.T) {
		conclusion := "Test conclusion"
		result, err := baseAction.Execute(ctx, "base.exit", map[string]any{
			"conclusion": conclusion,
		})
		gt.NoError(t, err)
		gt.Value(t, result.Type).Equal(action.ResultTypeText)
		gt.Value(t, result.Message).Equal(conclusion)
	})

	t.Run("exit without conclusion", func(t *testing.T) {
		_, err := baseAction.Execute(ctx, "base.exit", map[string]any{})
		gt.Error(t, err)
	})

	t.Run("unknown function", func(t *testing.T) {
		_, err := baseAction.Execute(ctx, "unknown.function", map[string]any{})
		gt.Error(t, err)
	})
}
