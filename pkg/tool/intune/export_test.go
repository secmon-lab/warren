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
