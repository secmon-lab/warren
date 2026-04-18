package firestore

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// chat-session-redesign additions on *Firestore. See session.go for the
// pre-existing methods. Methods here use the same top-level collections
// (`sessions`, `turns`) to avoid sub-collection scan limitations.

// CreateSession writes sess only if no Firestore document exists at its ID.
// Returns interfaces.ErrSessionAlreadyExists on collision so the caller
// (typically SessionResolver) can fall back to Get.
func (r *Firestore) CreateSession(ctx context.Context, sess *session.Session) error {
	_, err := r.db.Collection(collectionSessions).Doc(sess.ID.String()).Create(ctx, sess)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return interfaces.ErrSessionAlreadyExists
		}
		return r.eb.Wrap(err, "failed to create session",
			goerr.V("session_id", sess.ID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) UpdateSessionLastActive(ctx context.Context, sessionID types.SessionID, t time.Time) error {
	_, err := r.db.Collection(collectionSessions).Doc(sessionID.String()).Update(ctx, []firestore.Update{
		{Path: "last_active_at", Value: t},
	})
	if err != nil {
		return r.eb.Wrap(err, "failed to update last_active_at",
			goerr.V("session_id", sessionID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) PromoteSessionToTicket(ctx context.Context, sessionID types.SessionID, ticketID types.TicketID) error {
	_, err := r.db.Collection(collectionSessions).Doc(sessionID.String()).Update(ctx, []firestore.Update{
		{Path: "ticket_id", Value: ticketID.String()},
		{Path: "ticket_id_ptr", Value: ticketID.String()},
	})
	if err != nil {
		return r.eb.Wrap(err, "failed to promote session to ticket",
			goerr.V("session_id", sessionID),
			goerr.V("ticket_id", ticketID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

// AcquireSessionLock performs a Firestore transaction that checks the
// current Lock sub-field and overwrites it only when absent or expired.
func (r *Firestore) AcquireSessionLock(ctx context.Context, sessionID types.SessionID, holderID string, ttl time.Duration) (bool, error) {
	ref := r.db.Collection(collectionSessions).Doc(sessionID.String())
	acquired := false
	err := r.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var sess session.Session
		if err := doc.DataTo(&sess); err != nil {
			return err
		}
		now := time.Now()
		if sess.Lock != nil && !sess.Lock.IsExpired(now) {
			acquired = false
			return nil
		}
		lock := session.SessionLock{
			HolderID:   holderID,
			AcquiredAt: now,
			ExpiresAt:  now.Add(ttl),
		}
		if err := tx.Update(ref, []firestore.Update{{Path: "lock", Value: lock}}); err != nil {
			return err
		}
		acquired = true
		return nil
	})
	if err != nil {
		return false, r.eb.Wrap(err, "failed to acquire session lock",
			goerr.V("session_id", sessionID),
			goerr.T(errutil.TagDatabase))
	}
	return acquired, nil
}

func (r *Firestore) RefreshSessionLock(ctx context.Context, sessionID types.SessionID, holderID string, ttl time.Duration) error {
	ref := r.db.Collection(collectionSessions).Doc(sessionID.String())
	return r.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var sess session.Session
		if err := doc.DataTo(&sess); err != nil {
			return err
		}
		if sess.Lock == nil || sess.Lock.HolderID != holderID {
			return interfaces.ErrLockNotHeld
		}
		newExpires := time.Now().Add(ttl)
		return tx.Update(ref, []firestore.Update{{Path: "lock.expires_at", Value: newExpires}})
	})
}

func (r *Firestore) ReleaseSessionLock(ctx context.Context, sessionID types.SessionID, holderID string) error {
	ref := r.db.Collection(collectionSessions).Doc(sessionID.String())
	return r.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var sess session.Session
		if err := doc.DataTo(&sess); err != nil {
			return err
		}
		if sess.Lock == nil || sess.Lock.HolderID != holderID {
			return interfaces.ErrLockNotHeld
		}
		return tx.Update(ref, []firestore.Update{{Path: "lock", Value: nil}})
	})
}

// --- Turn CRUD (top-level collection `turns`) ---

func (r *Firestore) PutTurn(ctx context.Context, turn *session.Turn) error {
	_, err := r.db.Collection(collectionTurns).Doc(turn.ID.String()).Set(ctx, turn)
	if err != nil {
		return r.eb.Wrap(err, "failed to put turn",
			goerr.V("turn_id", turn.ID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) GetTurn(ctx context.Context, turnID types.TurnID) (*session.Turn, error) {
	doc, err := r.db.Collection(collectionTurns).Doc(turnID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, r.eb.Wrap(err, "failed to get turn",
			goerr.V("turn_id", turnID),
			goerr.T(errutil.TagDatabase))
	}
	var t session.Turn
	if err := doc.DataTo(&t); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to turn",
			goerr.T(errutil.TagInternal))
	}
	return &t, nil
}

func (r *Firestore) GetTurnsBySession(ctx context.Context, sessionID types.SessionID) ([]*session.Turn, error) {
	query := r.db.Collection(collectionTurns).
		Where("session_id", "==", sessionID.String()).
		OrderBy("started_at", firestore.Asc)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var turns []*session.Turn
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate turns",
				goerr.V("session_id", sessionID),
				goerr.T(errutil.TagDatabase))
		}
		var t session.Turn
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to turn",
				goerr.T(errutil.TagInternal))
		}
		turns = append(turns, &t)
	}
	return turns, nil
}

func (r *Firestore) UpdateTurnStatus(ctx context.Context, turnID types.TurnID, ts session.TurnStatus, endedAt *time.Time) error {
	updates := []firestore.Update{
		{Path: "status", Value: string(ts)},
	}
	if endedAt != nil {
		updates = append(updates, firestore.Update{Path: "ended_at", Value: *endedAt})
	}
	_, err := r.db.Collection(collectionTurns).Doc(turnID.String()).Update(ctx, updates)
	if err != nil {
		return r.eb.Wrap(err, "failed to update turn status",
			goerr.V("turn_id", turnID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) UpdateTurnIntent(ctx context.Context, turnID types.TurnID, intent string) error {
	_, err := r.db.Collection(collectionTurns).Doc(turnID.String()).Update(ctx, []firestore.Update{
		{Path: "intent", Value: intent},
	})
	if err != nil {
		return r.eb.Wrap(err, "failed to update turn intent",
			goerr.V("turn_id", turnID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

// --- Session message queries ---

// GetMessagesByTurn scans every Session's message subcollection for entries
// with turn_id == turnID. Messages are stored under
// sessions/{sessionID}/messages/{messageID}; looking up by turn_id therefore
// requires a collection group query.
func (r *Firestore) GetMessagesByTurn(ctx context.Context, turnID types.TurnID) ([]*session.Message, error) {
	query := r.db.CollectionGroup("messages").
		Where("turn_id", "==", turnID.String()).
		OrderBy("created_at", firestore.Asc)

	return queryMessages(ctx, r, query)
}

func (r *Firestore) SearchSessionMessages(ctx context.Context, ticketID types.TicketID, query string, limit int) ([]*session.Message, error) {
	// 1) Resolve Session IDs for the Ticket.
	sess, err := r.GetSessionsByTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	// 2) For each Session, scan its messages subcollection and filter by
	// substring match. Firestore lacks native full-text search; a later
	// iteration may swap in vector search.
	needle := strings.ToLower(query)
	var out []*session.Message
	for _, s := range sess {
		msgs, err := r.GetSessionMessages(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range msgs {
			if needle == "" || strings.Contains(strings.ToLower(m.Content), needle) {
				out = append(out, m)
			}
		}
	}
	// Sort newest first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *Firestore) GetTicketSessionMessages(ctx context.Context, ticketID types.TicketID, source *session.SessionSource, msgType *session.MessageType, limit, offset int) ([]*session.Message, error) {
	sess, err := r.GetSessionsByTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	var out []*session.Message
	for _, s := range sess {
		if source != nil && s.Source != *source {
			continue
		}
		msgs, err := r.GetSessionMessages(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range msgs {
			if msgType != nil && m.Type != *msgType {
				continue
			}
			out = append(out, m)
		}
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(out) {
		return []*session.Message{}, nil
	}
	out = out[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func queryMessages(ctx context.Context, r *Firestore, q firestore.Query) ([]*session.Message, error) {
	iter := q.Documents(ctx)
	defer iter.Stop()

	var out []*session.Message
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate session messages",
				goerr.T(errutil.TagDatabase))
		}
		var m session.Message
		if err := doc.DataTo(&m); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to message",
				goerr.T(errutil.TagInternal))
		}
		out = append(out, &m)
	}
	return out, nil
}
