package firestore

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Notice management methods

func (r *Firestore) CreateNotice(ctx context.Context, notice *notice.Notice) error {
	if notice.ID == types.EmptyNoticeID {
		return r.eb.New("notice ID is empty")
	}

	docRef := r.db.Collection(collectionNotices).Doc(string(notice.ID))

	// Check if document already exists
	_, err := docRef.Get(ctx)
	if status.Code(err) != codes.NotFound {
		if err == nil {
			return r.eb.New("notice already exists", goerr.V("notice_id", notice.ID))
		}
		return r.eb.Wrap(err, "failed to check notice existence", goerr.V("notice_id", notice.ID))
	}

	// Create the notice document
	_, err = docRef.Set(ctx, notice)
	if err != nil {
		return r.eb.Wrap(err, "failed to create notice", goerr.V("notice_id", notice.ID))
	}

	return nil
}

func (r *Firestore) GetNotice(ctx context.Context, id types.NoticeID) (*notice.Notice, error) {
	if id == types.EmptyNoticeID {
		return nil, r.eb.New("notice ID is empty")
	}

	docRef := r.db.Collection(collectionNotices).Doc(string(id))
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, r.eb.New("notice not found", goerr.V("notice_id", id))
		}
		return nil, r.eb.Wrap(err, "failed to get notice", goerr.V("notice_id", id))
	}

	var notice notice.Notice
	if err := doc.DataTo(&notice); err != nil {
		return nil, r.eb.Wrap(err, "failed to decode notice data", goerr.V("notice_id", id))
	}

	return &notice, nil
}

func (r *Firestore) UpdateNotice(ctx context.Context, notice *notice.Notice) error {
	if notice.ID == types.EmptyNoticeID {
		return r.eb.New("notice ID is empty")
	}

	docRef := r.db.Collection(collectionNotices).Doc(string(notice.ID))

	// Check if document exists
	_, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return r.eb.New("notice not found", goerr.V("notice_id", notice.ID))
		}
		return r.eb.Wrap(err, "failed to check notice existence", goerr.V("notice_id", notice.ID))
	}

	// Update the notice document
	_, err = docRef.Set(ctx, notice)
	if err != nil {
		return r.eb.Wrap(err, "failed to update notice", goerr.V("notice_id", notice.ID))
	}

	return nil
}

func (r *Firestore) PutNotice(ctx context.Context, notice *notice.Notice) error {
	// Check if notice exists
	existing, err := r.GetNotice(ctx, notice.ID)
	if err != nil {
		// If it's a "not found" error, create it
		if status.Code(err) == codes.NotFound {
			return r.CreateNotice(ctx, notice)
		}
		return err
	}
	// If it exists, update it
	if existing != nil {
		return r.UpdateNotice(ctx, notice)
	}
	// Otherwise create it
	return r.CreateNotice(ctx, notice)
}

func (r *Firestore) GetNotices(ctx context.Context) ([]*notice.Notice, error) {
	iter := r.db.Collection(collectionNotices).Documents(ctx)
	defer iter.Stop()

	var notices []*notice.Notice
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate notices")
		}

		var n notice.Notice
		if err := doc.DataTo(&n); err != nil {
			return nil, r.eb.Wrap(err, "failed to decode notice data", goerr.V("doc_id", doc.Ref.ID))
		}
		notices = append(notices, &n)
	}

	return notices, nil
}
