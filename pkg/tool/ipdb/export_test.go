package ipdb

import extipdb "github.com/gollem-dev/tools/ipdb"

// Opts exposes the accumulated external options for testing that flag Action
// callbacks append the expected option.
func (x *Action) Opts() []extipdb.Option {
	return x.opts
}
