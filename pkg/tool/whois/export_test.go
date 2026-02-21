package whois

import "context"

// SetQueryFunc sets a mock query function for testing.
func (x *Action) SetQueryFunc(fn func(ctx context.Context, target string) (string, error)) {
	x.queryFn = fn
}
