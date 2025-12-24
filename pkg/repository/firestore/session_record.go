package firestore

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"google.golang.org/api/iterator"
)

func (r *Firestore) PutSessionRecord(ctx context.Context, record *session.SessionRecord) error {
	// sessions/{sessionID}/records/{recordID} に保存（subcollection）
	_, err := r.db.Collection(collectionSessions).
		Doc(record.SessionID.String()).
		Collection(collectionSessionRecords).
		Doc(record.ID.String()).
		Set(ctx, record)
	if err != nil {
		return r.eb.Wrap(err, "failed to put session record",
			goerr.V("session_id", record.SessionID),
			goerr.V("record_id", record.ID),
			goerr.T(errs.TagDatabase))
	}
	return nil
}

func (r *Firestore) GetSessionRecords(ctx context.Context, sessionID types.SessionID) ([]*session.SessionRecord, error) {
	// sessions/{sessionID}/records から全てのレコードを取得
	query := r.db.Collection(collectionSessions).
		Doc(sessionID.String()).
		Collection(collectionSessionRecords).
		OrderBy("created_at", firestore.Asc)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var records []*session.SessionRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to query session records",
				goerr.V("session_id", sessionID),
				goerr.T(errs.TagDatabase))
		}

		var record session.SessionRecord
		if err := doc.DataTo(&record); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to session record",
				goerr.V("session_id", sessionID),
				goerr.T(errs.TagInternal))
		}
		records = append(records, &record)
	}

	return records, nil
}
