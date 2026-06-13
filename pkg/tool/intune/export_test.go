package intune

import (
	"context"

	extintune "github.com/gollem-dev/tools/intune"
)

// ConfigureWithOpts calls configure with extra extintune options, allowing tests
// to inject a token endpoint and/or base URL pointing at httptest servers.
func (x *Action) ConfigureWithOpts(tokenEndpoint, baseURL string) error {
	var opts []extintune.Option
	if tokenEndpoint != "" {
		opts = append(opts, extintune.WithTokenEndpoint(tokenEndpoint))
	}
	if baseURL != "" {
		opts = append(opts, extintune.WithBaseURL(baseURL))
	}
	return x.configure(context.Background(), opts...)
}
