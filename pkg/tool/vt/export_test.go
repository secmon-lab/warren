package vt

import extvt "github.com/gollem-dev/tools/vt"

// Opts exposes the accumulated external options for testing that flag Action
// callbacks append the expected option.
func (x *Action) Opts() []extvt.Option {
	return x.opts
}
