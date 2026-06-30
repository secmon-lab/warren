package toolset_test

import (
	"context"
	"testing"

	"github.com/gollem-dev/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/toolset"
)

type echoInput struct {
	Text string `json:"text" required:"true" description:"text to echo back"`
}

type addInput struct {
	A int64 `json:"a" description:"first addend"`
	B int64 `json:"b" description:"second addend"`
}

func newEchoTool() gollem.Tool {
	return gollem.MustNewTool("echo", "Echo the given text",
		func(_ context.Context, in echoInput) (map[string]any, error) {
			return map[string]any{"echo": in.Text}, nil
		})
}

func newAddTool() gollem.Tool {
	return gollem.MustNewTool("add", "Add two integers",
		func(_ context.Context, in addInput) (map[string]any, error) {
			return map[string]any{"sum": in.A + in.B}, nil
		})
}

func TestSet_SpecsKeepsRegistrationOrder(t *testing.T) {
	ctx := context.Background()
	set := toolset.New(newEchoTool(), newAddTool())

	specs, err := set.Specs(ctx)
	gt.NoError(t, err)
	gt.A(t, specs).Length(2)
	gt.Equal(t, specs[0].Name, "echo")
	gt.Equal(t, specs[1].Name, "add")

	// The schema is inferred from the In struct: a required field stays required.
	gt.True(t, specs[0].Parameters["text"].Required)
}

func TestSet_RunDispatchesByName(t *testing.T) {
	ctx := context.Background()
	set := toolset.New(newEchoTool(), newAddTool())

	out, err := set.Run(ctx, "echo", map[string]any{"text": "hello"})
	gt.NoError(t, err)
	gt.Value(t, out["echo"]).Equal("hello")
}

// TestSet_RunDecodesIntegers guards the int64 path: JSON numbers arrive as
// float64 over the wire, and NewTool must decode them into int64 fields without
// the manual float64 special-casing the old hand-written tools carried.
func TestSet_RunDecodesIntegers(t *testing.T) {
	ctx := context.Background()
	set := toolset.New(newAddTool())

	out, err := set.Run(ctx, "add", map[string]any{"a": float64(2), "b": float64(40)})
	gt.NoError(t, err)
	gt.Value(t, out["sum"]).Equal(int64(42))
}

func TestSet_RunUnknownToolErrors(t *testing.T) {
	ctx := context.Background()
	set := toolset.New(newEchoTool())

	_, err := set.Run(ctx, "missing", map[string]any{})
	gt.Error(t, err)
}

func TestSet_DuplicateNamePanics(t *testing.T) {
	defer func() {
		r := recover()
		gt.NotNil(t, r)
	}()

	toolset.New(newEchoTool(), newEchoTool())
	t.Fatal("expected panic on duplicate tool name")
}
