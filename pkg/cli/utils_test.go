package cli_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	cliv3 "github.com/urfave/cli/v3"
)

// fakeHITLTool is a minimal interfaces.Tool that also implements the
// RequiresHITL() interface used by the HITLToolNames helper.
type fakeHITLTool struct {
	id    string
	specs []gollem.ToolSpec
	hitl  bool
}

func (f *fakeHITLTool) ID() string          { return f.id }
func (f *fakeHITLTool) Description() string { return f.id }
func (f *fakeHITLTool) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return f.specs, nil
}
func (f *fakeHITLTool) Run(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *fakeHITLTool) Prompt(_ context.Context) (string, error) { return "", nil }
func (f *fakeHITLTool) Flags() []cliv3.Flag                      { return nil }
func (f *fakeHITLTool) Configure(_ context.Context) error        { return nil }
func (f *fakeHITLTool) LogValue() slog.Value                     { return slog.GroupValue() }
func (f *fakeHITLTool) Helper() *cliv3.Command                   { return nil }
func (f *fakeHITLTool) RequiresHITL() bool                       { return f.hitl }

var _ interfaces.Tool = &fakeHITLTool{}

// fakeNoHITLTool is a minimal interfaces.Tool that does NOT implement
// RequiresHITL(). The HITLToolNames helper must silently skip such tools.
type fakeNoHITLTool struct {
	id    string
	specs []gollem.ToolSpec
}

func (f *fakeNoHITLTool) ID() string          { return f.id }
func (f *fakeNoHITLTool) Description() string { return f.id }
func (f *fakeNoHITLTool) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return f.specs, nil
}
func (f *fakeNoHITLTool) Run(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *fakeNoHITLTool) Prompt(_ context.Context) (string, error) { return "", nil }
func (f *fakeNoHITLTool) Flags() []cliv3.Flag                      { return nil }
func (f *fakeNoHITLTool) Configure(_ context.Context) error        { return nil }
func (f *fakeNoHITLTool) LogValue() slog.Value                     { return slog.GroupValue() }
func (f *fakeNoHITLTool) Helper() *cliv3.Command                   { return nil }

var _ interfaces.Tool = &fakeNoHITLTool{}

func TestHITLToolNames_IncludesToolWhenRequiresHITLTrue(t *testing.T) {
	hitlTool := &fakeHITLTool{
		id:    "fetcher",
		specs: []gollem.ToolSpec{{Name: "web_fetch"}},
		hitl:  true,
	}
	plainTool := &fakeHITLTool{
		id:    "lookup",
		specs: []gollem.ToolSpec{{Name: "lookup_thing"}},
		hitl:  false,
	}

	names, err := cli.HITLToolNamesForTest(t.Context(), hitlTool, plainTool)
	gt.NoError(t, err).Required()
	gt.Array(t, names).Length(1)
	gt.Value(t, names[0]).Equal("web_fetch")
}

func TestHITLToolNames_ExcludesToolWhenRequiresHITLFalse(t *testing.T) {
	tool := &fakeHITLTool{
		id:    "fetcher",
		specs: []gollem.ToolSpec{{Name: "web_fetch"}},
		hitl:  false,
	}
	names, err := cli.HITLToolNamesForTest(t.Context(), tool)
	gt.NoError(t, err).Required()
	gt.Array(t, names).Length(0)
}

func TestHITLToolNames_ToolWithoutInterfaceIsSkipped(t *testing.T) {
	tool := &fakeNoHITLTool{
		id:    "lookup",
		specs: []gollem.ToolSpec{{Name: "lookup_thing"}},
	}
	names, err := cli.HITLToolNamesForTest(t.Context(), tool)
	gt.NoError(t, err).Required()
	gt.Array(t, names).Length(0)
}

func TestHITLToolNames_AggregatesMultipleSpecs(t *testing.T) {
	tool := &fakeHITLTool{
		id: "multi",
		specs: []gollem.ToolSpec{
			{Name: "fn_one"},
			{Name: "fn_two"},
		},
		hitl: true,
	}
	names, err := cli.HITLToolNamesForTest(t.Context(), tool)
	gt.NoError(t, err).Required()
	gt.Array(t, names).Length(2)
	gt.Value(t, names[0]).Equal("fn_one")
	gt.Value(t, names[1]).Equal("fn_two")
}
