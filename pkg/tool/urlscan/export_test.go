package urlscan

import exturlscan "github.com/gollem-dev/tools/urlscan"

// Opts exposes the accumulated external options for testing that flag Action
// callbacks append the expected option.
func (x *Action) Opts() []exturlscan.Option {
	return x.opts
}
