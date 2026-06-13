package shodan

import extshodan "github.com/gollem-dev/tools/shodan"

// Opts exposes the accumulated external options for testing that flag Action
// callbacks append the expected option.
func (x *Action) Opts() []extshodan.Option {
	return x.opts
}
