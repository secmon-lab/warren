package slack

import extslack "github.com/gollem-dev/tools/slack"

// SetOAuthToken sets the OAuth token directly (test-only).
func (x *Action) SetOAuthToken(token string) {
	x.oauthToken = token
}

// SetTestURL points the underlying toolset at a stub server (test-only).
func (x *Action) SetTestURL(url string) {
	x.opts = append(x.opts, extslack.WithBaseURL(url))
}
