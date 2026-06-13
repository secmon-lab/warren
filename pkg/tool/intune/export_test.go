package intune

import (
	"context"

	extintune "github.com/gollem-dev/tools/intune"
)

// ConfigureWithOpts appends test-only extintune options (token endpoint and/or
// base URL pointing at httptest servers) then calls Configure.
func (x *Action) ConfigureWithOpts(tokenEndpoint, baseURL string) error {
	if tokenEndpoint != "" {
		x.opts = append(x.opts, extintune.WithTokenEndpoint(tokenEndpoint))
	}
	if baseURL != "" {
		x.opts = append(x.opts, extintune.WithBaseURL(baseURL))
	}
	return x.Configure(context.Background())
}

// Opts exposes the accumulated external options for testing that flag Action
// callbacks append the expected option.
func (x *Action) Opts() []extintune.Option {
	return x.opts
}
