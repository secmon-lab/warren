package memory

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/refine"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

func (r *Memory) PutRefineGroup(ctx context.Context, group *refine.Group) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if group.ID == types.EmptyRefineGroupID {
		return r.eb.New("refine group ID is empty")
	}

	groupCopy := *group
	groupCopy.AlertIDs = make([]types.AlertID, len(group.AlertIDs))
	copy(groupCopy.AlertIDs, group.AlertIDs)
	r.refineGroups[group.ID] = &groupCopy

	return nil
}

func (r *Memory) GetRefineGroup(ctx context.Context, groupID types.RefineGroupID) (*refine.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	group, exists := r.refineGroups[groupID]
	if !exists {
		return nil, r.eb.New("refine group not found",
			goerr.V("group_id", groupID),
			goerr.T(errutil.TagNotFound),
		)
	}

	groupCopy := *group
	groupCopy.AlertIDs = make([]types.AlertID, len(group.AlertIDs))
	copy(groupCopy.AlertIDs, group.AlertIDs)
	return &groupCopy, nil
}
