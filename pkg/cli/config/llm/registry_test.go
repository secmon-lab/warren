package llm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config/llm"
)

func mockClient(name string) gollem.LLMClient {
	return &gollem_mock.LLMClientMock{
		NewSessionFunc: func(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
			panic("unused in registry tests: " + name)
		},
		GenerateEmbeddingFunc: func(_ context.Context, _ int, _ []string) ([][]float64, error) {
			panic("unused in registry tests: " + name)
		},
	}
}

func TestRegistry_MainAndResolve(t *testing.T) {
	primary := llm.NewLLMEntryForTest("primary", "main desc", "claude", "claude-x", mockClient("p"))
	fast := llm.NewLLMEntryForTest("fast", "fast desc", "gemini", "gemini-flash", mockClient("f"))
	embedding := mockClient("e")

	reg := llm.NewRegistryForTest(
		"primary",
		[]string{"primary", "fast"},
		map[string]*llm.LLMEntry{"primary": primary, "fast": fast},
		embedding,
	)

	gt.Equal(t, reg.Main(), primary)
	gt.Equal(t, reg.Embedding(), embedding)

	got, err := reg.Resolve("fast")
	gt.NoError(t, err)
	gt.Equal(t, got, fast)
}

func TestRegistry_Resolve_RejectsEmpty(t *testing.T) {
	reg := llm.NewRegistryForTest("p", []string{"p"},
		map[string]*llm.LLMEntry{"p": llm.NewLLMEntryForTest("p", "d", "claude", "m", mockClient("p"))},
		mockClient("e"))

	_, err := reg.Resolve("")
	gt.Error(t, err)
	gt.True(t, errors.Is(err, llm.ErrEmptyLLMID))
}

func TestRegistry_Resolve_RejectsUnknown(t *testing.T) {
	reg := llm.NewRegistryForTest("p", []string{"p"},
		map[string]*llm.LLMEntry{"p": llm.NewLLMEntryForTest("p", "d", "claude", "m", mockClient("p"))},
		mockClient("e"))

	_, err := reg.Resolve("ghost")
	gt.Error(t, err)
	gt.True(t, errors.Is(err, llm.ErrUnknownLLMID))
}

func TestRegistry_Resolve_RejectsLLMNotInTaskList(t *testing.T) {
	primary := llm.NewLLMEntryForTest("primary", "d", "claude", "m", mockClient("p"))
	hidden := llm.NewLLMEntryForTest("hidden", "d", "claude", "m", mockClient("h"))

	reg := llm.NewRegistryForTest(
		"primary",
		[]string{"primary"}, // hidden defined but not in task list
		map[string]*llm.LLMEntry{"primary": primary, "hidden": hidden},
		mockClient("e"),
	)

	_, err := reg.Resolve("hidden")
	gt.Error(t, err)
	gt.True(t, errors.Is(err, llm.ErrLLMNotInTaskList))
}

func TestRegistry_Catalog_OrderAndFilter(t *testing.T) {
	a := llm.NewLLMEntryForTest("a", "desc-a", "claude", "model-a", mockClient("a"))
	b := llm.NewLLMEntryForTest("b", "desc-b", "gemini", "model-b", mockClient("b"))
	c := llm.NewLLMEntryForTest("c", "desc-c", "gemini", "model-c", mockClient("c"))

	reg := llm.NewRegistryForTest(
		"a",
		[]string{"b", "a"}, // c is defined but not in task list → catalog excludes
		map[string]*llm.LLMEntry{"a": a, "b": b, "c": c},
		mockClient("e"),
	)

	cat := reg.Catalog()
	gt.A(t, cat).Length(2)
	gt.Equal(t, cat[0].ID, "b")
	gt.Equal(t, cat[0].Description, "desc-b")
	gt.Equal(t, cat[0].Provider, "gemini")
	gt.Equal(t, cat[0].Model, "model-b")
	gt.Equal(t, cat[1].ID, "a")
}

func TestRegistry_TaskIDs_Copy(t *testing.T) {
	reg := llm.NewRegistryForTest(
		"a",
		[]string{"a", "b"},
		map[string]*llm.LLMEntry{
			"a": llm.NewLLMEntryForTest("a", "d", "claude", "m", mockClient("a")),
			"b": llm.NewLLMEntryForTest("b", "d", "gemini", "m", mockClient("b")),
		},
		mockClient("e"),
	)
	ids := reg.TaskIDs()
	gt.A(t, ids).Length(2)
	gt.Equal(t, ids[0], "a")
	gt.Equal(t, ids[1], "b")

	ids[0] = "MUTATED"
	again := reg.TaskIDs()
	gt.Equal(t, again[0], "a")
}
