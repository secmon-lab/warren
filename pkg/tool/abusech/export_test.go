package abusech

import extabusech "github.com/gollem-dev/tools/abusech"

// Opts exposes the accumulated external options for testing that flag Action
// callbacks append the expected option.
func (x *Action) Opts() []extabusech.Option {
	return x.opts
}
