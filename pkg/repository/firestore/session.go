package firestore

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *Firestore) PutSession(ctx context.Context, sess *session.Session) error {
	// sessions/{sessionID} に保存（top-level）
	_, err := r.db.Collection(collectionSessions).Doc(sess.ID.String()).Set(ctx, sess)
	if err != nil {
		return r.eb.Wrap(err, "failed to put session",
			goerr.V("session_id", sess.ID),
			goerr.V("ticket_id", sess.TicketID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) GetSession(ctx context.Context, sessionID types.SessionID) (*session.Session, error) {
	doc, err := r.db.Collection(collectionSessions).Doc(sessionID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, r.eb.Wrap(err, "failed to get session",
			goerr.V("session_id", sessionID),
			goerr.T(errutil.TagDatabase))
	}

	var sess session.Session
	if err := doc.DataTo(&sess); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to session",
			goerr.V("session_id", sessionID),
			goerr.T(errutil.TagInternal))
	}
	return &sess, nil
}

func (r *Firestore) GetSessionsByTicket(ctx context.Context, ticketID types.TicketID) ([]*session.Session, error) {
	query := r.db.Collection(collectionSessions).
		Where("ticket_id", "==", ticketID.String())

	iter := query.Documents(ctx)
	defer iter.Stop()

	var sessions []*session.Session
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to query sessions by ticket",
				goerr.V("ticket_id", ticketID),
				goerr.T(errutil.TagDatabase))
		}

		var sess session.Session
		if err := doc.DataTo(&sess); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to session",
				goerr.V("ticket_id", ticketID),
				goerr.T(errutil.TagInternal))
		}
		sessions = append(sessions, &sess)
	}

	return sessions, nil
}

func (r *Firestore) DeleteSession(ctx context.Context, sessionID types.SessionID) error {
	_, err := r.db.Collection(collectionSessions).Doc(sessionID.String()).Delete(ctx)
	if err != nil {
		// NotFound is not an error - deleting non-existent document is idempotent
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return r.eb.Wrap(err, "failed to delete session",
			goerr.V("session_id", sessionID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) PutSessionMessage(ctx context.Context, message *session.Message) error {
	// Store in subcollection: sessions/{sessionID}/messages/{messageID}
	_, err := r.db.Collection(collectionSessions).
		Doc(message.SessionID.String()).
		Collection("messages").
		Doc(message.ID.String()).
		Set(ctx, message)
	if err != nil {
		return r.eb.Wrap(err, "failed to put session message",
			goerr.V("session_id", message.SessionID),
			goerr.V("message_id", message.ID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) GetSessionMessages(ctx context.Context, sessionID types.SessionID) ([]*session.Message, error) {
	// Query messages subcollection ordered by created_at
	query := r.db.Collection(collectionSessions).
		Doc(sessionID.String()).
		Collection("messages").
		OrderBy("created_at", firestore.Asc)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var messages []*session.Message
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to query session messages",
				goerr.V("session_id", sessionID),
				goerr.T(errutil.TagDatabase))
		}

		var msg session.Message
		if err := doc.DataTo(&msg); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to message",
				goerr.V("session_id", sessionID),
				goerr.T(errutil.TagInternal))
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}
