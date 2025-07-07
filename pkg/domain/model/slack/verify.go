package slack

import (
	"context"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/slack-go/slack"
)

type PayloadVerifier func(ctx context.Context, header http.Header, payload []byte) error

func NewPayloadVerifier(signingSecret string) PayloadVerifier {
	return func(ctx context.Context, header http.Header, payload []byte) error {
		eb := goerr.NewBuilder(goerr.V("body", string(payload)), goerr.V("header", header))
		verifier, err := slack.NewSecretsVerifier(header, signingSecret)
		if err != nil {
			return eb.Wrap(err, "failed to create secrets verifier")
		}

		if _, err := verifier.Write(payload); err != nil {
			return eb.Wrap(err, "failed to write request body to verifier")
		}

		if err := verifier.Ensure(); err != nil {
			return eb.Wrap(err, "invalid slack signature")
		}

		return nil
	}
}
