// Package toolset adapts a fixed list of individual gollem.Tool values into the
// gollem.ToolSet (Specs/Run) contract.
//
// gollem's type-safe constructor gollem.NewTool builds one gollem.Tool per
// handler, but Warren groups tools as a ToolSet: the planner selects tools by
// ToolSet ID and hands the whole group to a task agent. This adapter bridges the
// two so Warren can author type-safe tools with gollem.NewTool while still
// exposing the ToolSet grouping that interfaces.ToolSet requires, removing the
// hand-written Specs() literals and args[...] type assertions each tool used to
// carry.
package toolset

import (
	"context"

	"github.com/gollem-dev/gollem"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// Set is a gollem.ToolSet backed by a fixed list of gollem.Tool values. Run
// dispatches by tool name; Specs returns each tool's spec in registration order
// so the wire-level ordering stays stable across calls.
type Set struct {
	order []string
	tools map[string]gollem.Tool
}

var _ gollem.ToolSet = (*Set)(nil)

// New builds a Set from the given tools. The tool list is fixed at construction
// time (static registration), so a duplicate tool name is a programming error
// and panics rather than silently shadowing a sibling — mirroring the
// MustNewTool convention used to build the tools passed in here.
func New(tools ...gollem.Tool) *Set {
	s := &Set{
		order: make([]string, 0, len(tools)),
		tools: make(map[string]gollem.Tool, len(tools)),
	}
	for _, t := range tools {
		name := t.Spec().Name
		if _, ok := s.tools[name]; ok {
			panic(goerr.New("duplicate tool name in toolset", goerr.T(errutil.TagInvalidState), goerr.V("name", name)))
		}
		s.order = append(s.order, name)
		s.tools[name] = t
	}
	return s
}

// Specs implements gollem.ToolSet.
func (s *Set) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	specs := make([]gollem.ToolSpec, 0, len(s.order))
	for _, name := range s.order {
		specs = append(specs, s.tools[name].Spec())
	}
	return specs, nil
}

// Run implements gollem.ToolSet.
func (s *Set) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	t, ok := s.tools[name]
	if !ok {
		return nil, goerr.New("unknown tool", goerr.T(errutil.TagInvalidRequest), goerr.V("name", name))
	}
	return t.Run(ctx, args)
}
