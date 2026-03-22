package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const collectionHITLRequests = "hitl_requests"

func (r *Firestore) PutHITLRequest(ctx context.Context, req *hitl.Request) error {
	_, err := r.db.Collection(collectionHITLRequests).Doc(req.ID.String()).Set(ctx, req)
	if err != nil {
		return r.eb.Wrap(err, "failed to put HITL request",
			goerr.V("id", req.ID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) GetHITLRequest(ctx context.Context, id types.HITLRequestID) (*hitl.Request, error) {
	doc, err := r.db.Collection(collectionHITLRequests).Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, r.eb.Wrap(goerr.New("HITL request not found"),
				"not found",
				goerr.T(errutil.TagNotFound),
				goerr.V("id", id))
		}
		return nil, r.eb.Wrap(err, "failed to get HITL request",
			goerr.V("id", id),
			goerr.T(errutil.TagDatabase))
	}

	var req hitl.Request
	if err := doc.DataTo(&req); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to HITL request",
			goerr.V("id", id),
			goerr.T(errutil.TagInternal))
	}
	return &req, nil
}

func (r *Firestore) UpdateHITLRequestStatus(ctx context.Context, id types.HITLRequestID, st hitl.Status, respondedBy string, response map[string]any) error {
	ref := r.db.Collection(collectionHITLRequests).Doc(id.String())
	_, err := ref.Update(ctx, []firestore.Update{
		{Path: "Status", Value: st},
		{Path: "RespondedBy", Value: respondedBy},
		{Path: "Response", Value: response},
		{Path: "RespondedAt", Value: time.Now()},
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return r.eb.Wrap(goerr.New("HITL request not found"),
				"not found",
				goerr.T(errutil.TagNotFound),
				goerr.V("id", id))
		}
		return r.eb.Wrap(err, "failed to update HITL request status",
			goerr.V("id", id),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) WatchHITLRequest(ctx context.Context, id types.HITLRequestID) (<-chan *hitl.Request, <-chan error) {
	ch := make(chan *hitl.Request, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(ch)
		defer close(errCh)

		const maxRetries = 3
		retries := 0

		ref := r.db.Collection(collectionHITLRequests).Doc(id.String())
		snapIter := ref.Snapshots(ctx)
		defer snapIter.Stop()

		for {
			snap, err := snapIter.Next()
			if err != nil {
				if ctx.Err() != nil {
					return
				}

				retries++
				if retries > maxRetries {
					errCh <- goerr.Wrap(err, "snapshot iterator failed after retries",
						goerr.V("id", id),
						goerr.V("retries", maxRetries),
						goerr.T(errutil.TagDatabase))
					return
				}

				// Retry: stop current iterator and create a new one
				snapIter.Stop()
				snapIter = ref.Snapshots(ctx)
				continue
			}

			// Reset retry counter on successful read
			retries = 0

			if !snap.Exists() {
				continue
			}

			var req hitl.Request
			if err := snap.DataTo(&req); err != nil {
				errCh <- goerr.Wrap(err, "failed to convert snapshot to HITL request",
					goerr.V("id", id),
					goerr.T(errutil.TagInternal))
				return
			}

			if req.Status != hitl.StatusPending {
				ch <- &req
				return
			}
		}
	}()

	return ch, errCh
}
