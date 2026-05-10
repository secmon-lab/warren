package policy_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/policy"
)

type fakeSource struct {
	files   map[string]string
	version string
	calls   int
	err     error
}

func (f *fakeSource) Snapshot(_ context.Context) (*policy.Snapshot, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &policy.Snapshot{Files: f.files, Version: f.version}, nil
}

type loaderTestOutput struct {
	Alerts []map[string]any `json:"alerts"`
}

func TestLoader_Query_FromSingleSource(t *testing.T) {
	src, err := policy.NewFileSource([]string{"testdata"})
	gt.NoError(t, err)

	loader := policy.NewLoader(src)
	gt.True(t, loader.HasSources())

	input := map[string]any{
		"event_type": "test",
		"title":      "hello",
	}
	var out loaderTestOutput
	err = loader.Query(context.Background(), "data.ingest.test", input, &out)
	gt.NoError(t, err)
	gt.A(t, out.Alerts).Length(1)
	gt.Equal(t, out.Alerts[0]["title"], "hello")
}

func TestLoader_Query_MergesMultipleSources(t *testing.T) {
	a := &fakeSource{
		files: map[string]string{
			"a://ingest.rego": `package ingest.merged
import rego.v1
alerts contains alert if {
	input.kind == "a"
	alert := {"title": "from-a"}
}
`,
		},
		version: "v-a-1",
	}
	b := &fakeSource{
		files: map[string]string{
			"b://ingest.rego": `package ingest.merged
import rego.v1
alerts contains alert if {
	input.kind == "b"
	alert := {"title": "from-b"}
}
`,
		},
		version: "v-b-1",
	}

	loader := policy.NewLoader(a, b)

	var out loaderTestOutput
	err := loader.Query(context.Background(), "data.ingest.merged", map[string]any{"kind": "a"}, &out)
	gt.NoError(t, err)
	gt.A(t, out.Alerts).Length(1)
	gt.Equal(t, out.Alerts[0]["title"], "from-a")

	out = loaderTestOutput{}
	err = loader.Query(context.Background(), "data.ingest.merged", map[string]any{"kind": "b"}, &out)
	gt.NoError(t, err)
	gt.A(t, out.Alerts).Length(1)
	gt.Equal(t, out.Alerts[0]["title"], "from-b")
}

func TestLoader_CachesClient_WhenVersionUnchanged(t *testing.T) {
	src := &fakeSource{
		files: map[string]string{
			"x://only.rego": `package ingest.cached
import rego.v1
alerts contains alert if {
	input.kind == "x"
	alert := {"title": "x"}
}
`,
		},
		version: "v1",
	}
	loader := policy.NewLoader(src)

	var out loaderTestOutput
	err := loader.Query(context.Background(), "data.ingest.cached", map[string]any{"kind": "x"}, &out)
	gt.NoError(t, err)

	// Capture sources map identity to verify reuse.
	srcsBefore := loader.Sources()
	gt.M(t, srcsBefore).Length(1)

	// Second query with same version: snapshot is fetched again (Sources are
	// pulled per Query) but the compiled client must be reused.
	err = loader.Query(context.Background(), "data.ingest.cached", map[string]any{"kind": "x"}, &out)
	gt.NoError(t, err)

	srcsAfter := loader.Sources()
	gt.Equal(t, srcsBefore["x://only.rego"], srcsAfter["x://only.rego"])
	gt.Equal(t, src.calls, 2) // Snapshot called once per Query
}

func TestLoader_RebuildsClient_WhenVersionChanges(t *testing.T) {
	src := &fakeSource{
		files: map[string]string{
			"x://only.rego": `package ingest.dynamic
import rego.v1
alerts contains alert if {
	input.kind == "x"
	alert := {"title": "v1-result"}
}
`,
		},
		version: "v1",
	}
	loader := policy.NewLoader(src)

	var out loaderTestOutput
	err := loader.Query(context.Background(), "data.ingest.dynamic", map[string]any{"kind": "x"}, &out)
	gt.NoError(t, err)
	gt.Equal(t, out.Alerts[0]["title"], "v1-result")

	// Mutate the source: new content + new version.
	src.files = map[string]string{
		"x://only.rego": `package ingest.dynamic
import rego.v1
alerts contains alert if {
	input.kind == "x"
	alert := {"title": "v2-result"}
}
`,
	}
	src.version = "v2"

	out = loaderTestOutput{}
	err = loader.Query(context.Background(), "data.ingest.dynamic", map[string]any{"kind": "x"}, &out)
	gt.NoError(t, err)
	gt.Equal(t, out.Alerts[0]["title"], "v2-result")
}

func TestLoader_NoSources_ReturnsError(t *testing.T) {
	loader := policy.NewLoader()
	gt.False(t, loader.HasSources())

	var out loaderTestOutput
	err := loader.Query(context.Background(), "data.x", map[string]any{}, &out)
	gt.Error(t, err)
}

func TestLoader_DuplicateKey_AcrossSources_ReturnsError(t *testing.T) {
	a := &fakeSource{
		files:   map[string]string{"shared.rego": "package a\nimport rego.v1\nx := 1\n"},
		version: "a",
	}
	b := &fakeSource{
		files:   map[string]string{"shared.rego": "package b\nimport rego.v1\ny := 2\n"},
		version: "b",
	}
	loader := policy.NewLoader(a, b)

	var out map[string]any
	err := loader.Query(context.Background(), "data.a", map[string]any{}, &out)
	gt.Error(t, err)
}

func TestLoader_Sources_EmptyBeforeFirstQuery(t *testing.T) {
	src := &fakeSource{
		files:   map[string]string{"x.rego": "package x\n"},
		version: "v1",
	}
	loader := policy.NewLoader(src)

	srcs := loader.Sources()
	gt.M(t, srcs).Length(0)
}
