package intune

import (
	"context"
	"net/http"

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

// ConfigureWithHTTPClient injects a test HTTP client and a graph base URL, then
// calls Configure WITHOUT overriding the token endpoint. This lets a test
// observe the default token endpoint (which embeds the tenant ID) and the
// credentials carried in the token request.
func (x *Action) ConfigureWithHTTPClient(client *http.Client, baseURL string) error {
	x.opts = append(x.opts, extintune.WithHTTPClient(client), extintune.WithBaseURL(baseURL))
	return x.Configure(context.Background())
}
