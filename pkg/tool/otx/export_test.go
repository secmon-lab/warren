package otx

import extotx "github.com/gollem-dev/tools/otx"

// Opts exposes the accumulated external options for testing that flag Action
// callbacks append the expected option.
func (x *Action) Opts() []extotx.Option {
	return x.opts
}
