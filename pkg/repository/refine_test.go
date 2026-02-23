package repository_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/refine"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestRefineGroup(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		t.Run("PutAndGetRefineGroup", func(t *testing.T) {
			groupID := types.NewRefineGroupID()
			alertID1 := types.NewAlertID()
			alertID2 := types.NewAlertID()
			primaryAlertID := alertID1

			group := &refine.Group{
				ID:             groupID,
				PrimaryAlertID: primaryAlertID,
				AlertIDs:       []types.AlertID{alertID1, alertID2},
				Reason:         "same source IP detected",
				CreatedAt:      time.Now().Truncate(time.Millisecond),
				Status:         refine.GroupStatusPending,
			}

			gt.NoError(t, repo.PutRefineGroup(ctx, group))

			got, err := repo.GetRefineGroup(ctx, groupID)
			gt.NoError(t, err)
			gt.V(t, got.ID).Equal(groupID)
			gt.V(t, got.PrimaryAlertID).Equal(primaryAlertID)
			gt.V(t, len(got.AlertIDs)).Equal(2)
			gt.V(t, got.AlertIDs[0]).Equal(alertID1)
			gt.V(t, got.AlertIDs[1]).Equal(alertID2)
			gt.V(t, got.Reason).Equal("same source IP detected")
			gt.V(t, got.Status).Equal(refine.GroupStatusPending)
		})

		t.Run("UpdateRefineGroupStatus", func(t *testing.T) {
			groupID := types.NewRefineGroupID()
			group := &refine.Group{
				ID:             groupID,
				PrimaryAlertID: types.NewAlertID(),
				AlertIDs:       []types.AlertID{types.NewAlertID()},
				Reason:         "test reason",
				CreatedAt:      time.Now().Truncate(time.Millisecond),
				Status:         refine.GroupStatusPending,
			}

			gt.NoError(t, repo.PutRefineGroup(ctx, group))

			// Update status
			group.Status = refine.GroupStatusAccepted
			gt.NoError(t, repo.PutRefineGroup(ctx, group))

			got, err := repo.GetRefineGroup(ctx, groupID)
			gt.NoError(t, err)
			gt.V(t, got.Status).Equal(refine.GroupStatusAccepted)
		})

		t.Run("GetRefineGroupNotFound", func(t *testing.T) {
			_, err := repo.GetRefineGroup(ctx, types.NewRefineGroupID())
			gt.Error(t, err)
		})
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}
