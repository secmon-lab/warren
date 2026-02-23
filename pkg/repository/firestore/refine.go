package firestore

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/refine"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *Firestore) PutRefineGroup(ctx context.Context, group *refine.Group) error {
	if group.ID == types.EmptyRefineGroupID {
		return r.eb.New("refine group ID is empty")
	}

	docRef := r.db.Collection(collectionRefineGroups).Doc(string(group.ID))
	if _, err := docRef.Set(ctx, group); err != nil {
		return r.eb.Wrap(err, "failed to put refine group",
			goerr.V("group_id", group.ID),
			goerr.T(errutil.TagDatabase),
		)
	}

	return nil
}

func (r *Firestore) GetRefineGroup(ctx context.Context, groupID types.RefineGroupID) (*refine.Group, error) {
	if groupID == types.EmptyRefineGroupID {
		return nil, r.eb.New("refine group ID is empty")
	}

	docRef := r.db.Collection(collectionRefineGroups).Doc(string(groupID))
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, r.eb.New("refine group not found",
				goerr.V("group_id", groupID),
				goerr.T(errutil.TagNotFound),
			)
		}
		return nil, r.eb.Wrap(err, "failed to get refine group",
			goerr.V("group_id", groupID),
			goerr.T(errutil.TagDatabase),
		)
	}

	var group refine.Group
	if err := doc.DataTo(&group); err != nil {
		return nil, r.eb.Wrap(err, "failed to decode refine group data",
			goerr.V("group_id", groupID),
			goerr.T(errutil.TagDatabase),
		)
	}

	return &group, nil
}
