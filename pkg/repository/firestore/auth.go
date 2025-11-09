package firestore

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Token related methods
func (r *Firestore) PutToken(ctx context.Context, token *auth.Token) error {
	doc := r.db.Collection(collectionTokens).Doc(token.ID.String())
	_, err := doc.Set(ctx, token)
	if err != nil {
		return goerr.Wrap(err, "failed to put token", goerr.V("token_id", token.ID))
	}
	return nil
}

func (r *Firestore) GetToken(ctx context.Context, tokenID auth.TokenID) (*auth.Token, error) {
	doc, err := r.db.Collection(collectionTokens).Doc(tokenID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("token not found", goerr.V("token_id", tokenID))
		}
		return nil, goerr.Wrap(err, "failed to get token", goerr.V("token_id", tokenID))
	}

	var token auth.Token
	if err := doc.DataTo(&token); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to token", goerr.V("token_id", tokenID))
	}

	token.ID = tokenID // Set the ID manually since it's not stored in the document
	return &token, nil
}

func (r *Firestore) DeleteToken(ctx context.Context, tokenID auth.TokenID) error {
	doc := r.db.Collection(collectionTokens).Doc(tokenID.String())
	_, err := doc.Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete token", goerr.V("token_id", tokenID))
	}
	return nil
}

func (r *Firestore) GetTokens(ctx context.Context) ([]*auth.Token, error) {
	iter := r.db.Collection(collectionTokens).Documents(ctx)
	defer iter.Stop()

	var tokens []*auth.Token
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate tokens")
		}

		var token auth.Token
		if err := doc.DataTo(&token); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to token", goerr.V("doc_id", doc.Ref.ID))
		}

		// Set the ID from the document ID
		token.ID = auth.TokenID(doc.Ref.ID)
		tokens = append(tokens, &token)
	}

	return tokens, nil
}
