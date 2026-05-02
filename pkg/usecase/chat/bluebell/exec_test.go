package bluebell_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	llmconfig "github.com/secmon-lab/warren/pkg/cli/config/llm"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/repository"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/usecase/chat/bluebell"
)

// trackingLLM is a mock LLMClient that counts Generate invocations and
// records the user prompts it received. Multiple instances can be used
// to verify that the planner's llm_id selection routes tasks to the right
// client and not to another.
type trackingLLM struct {
	id           string
	calls        atomic.Int32
	receivedMu   sync.Mutex
	receivedTask []string

	// plannerJSON, if non-empty, is returned for the first call to
	// Generate (used when this client doubles as the planner).
	plannerJSON string
	// genericResp is returned for subsequent calls (task execution).
	genericResp string
	// genErr, if non-nil, is returned instead of a successful response.
	genErr error
}

func (t *trackingLLM) record(prompt string) {
	t.receivedMu.Lock()
	defer t.receivedMu.Unlock()
	t.receivedTask = append(t.receivedTask, prompt)
}

func (t *trackingLLM) totalCalls() int {
	return int(t.calls.Load())
}

func (t *trackingLLM) toClient() *gollem_mock.LLMClientMock {
	return &gollem_mock.LLMClientMock{
		NewSessionFunc: func(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(_ context.Context, input []gollem.Input, _ ...gollem.GenerateOption) (*gollem.Response, error) {
				t.calls.Add(1)
				prompt := ""
				for _, inp := range input {
					if txt, ok := inp.(gollem.Text); ok {
						prompt = string(txt)
						t.record(prompt)
					}
				}
				if t.genErr != nil {
					return nil, t.genErr
				}
				// Differentiate by prompt content rather than call ordering,
				// because bluebell calls selector → planner → (task agents) → replan → final.
				switch {
				case strings.Contains(prompt, "intent resolver"):
					return &gollem.Response{Texts: []string{`{"prompt_id":"default","intent":"investigate"}`}}, nil
				case strings.Contains(prompt, "Create an execution plan") && t.plannerJSON != "":
					return &gollem.Response{Texts: []string{t.plannerJSON}}, nil
				case strings.Contains(prompt, "Previous task phases have completed"):
					return &gollem.Response{Texts: []string{`{"tasks": []}`}}, nil
				}
				return &gollem.Response{Texts: []string{t.genericResp}}, nil
			}
			return ssn, nil
		},
		GenerateEmbeddingFunc: func(_ context.Context, dim int, _ []string) ([][]float64, error) {
			out := make([][]float64, 1)
			out[0] = make([]float64, dim)
			return out, nil
		},
	}
}

// buildRegistry constructs a registry where the main LLM is the planner and
// taskLLMs are exposed under [agent].task. The main is also added to task to
// keep semantics simple.
func buildRegistry(main *trackingLLM, taskLLMs map[string]*trackingLLM, embedding gollem.LLMClient) *llmconfig.Registry {
	entries := map[string]*llmconfig.LLMEntry{
		main.id: llmconfig.NewLLMEntryForTest(main.id, "primary planner", "claude", "main-model", main.toClient()),
	}
	taskIDs := []string{main.id}
	for id, tl := range taskLLMs {
		entries[id] = llmconfig.NewLLMEntryForTest(id, "task llm "+id, "gemini", "task-model-"+id, tl.toClient())
		taskIDs = append(taskIDs, id)
	}
	return llmconfig.NewRegistryForTest(main.id, taskIDs, entries, embedding)
}

func TestExecuteTask_RoutesByLLMID_MultipleEntries(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	// Three tasks routed to three different LLMs.
	planJSON := `{
		"message": "running",
		"tasks": [
			{"id": "t1", "title": "fast task", "description": "FAST_DESC", "tools": [], "llm_id": "fast"},
			{"id": "t2", "title": "smart task", "description": "SMART_DESC", "tools": [], "llm_id": "smart"},
			{"id": "t3", "title": "secondary task", "description": "SECONDARY_DESC", "tools": [], "llm_id": "fast2"}
		]
	}`

	mainLLM := &trackingLLM{id: "main", plannerJSON: planJSON, genericResp: `{"tasks": []}`}
	fastLLM := &trackingLLM{id: "fast", genericResp: "fast result"}
	smartLLM := &trackingLLM{id: "smart", genericResp: "smart result"}
	fast2LLM := &trackingLLM{id: "fast2", genericResp: "fast2 result"}

	embedding := mainLLM.toClient()

	reg := buildRegistry(mainLLM, map[string]*trackingLLM{
		"fast":  fastLLM,
		"smart": smartLLM,
		"fast2": fast2LLM,
	}, embedding)

	chatUC, err := bluebell.New(repo, reg, bluebell.WithKnowledgeService(knowledgeSvc))
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "test routing",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)

	// Each task LLM was called exactly once.
	gt.Equal(t, fastLLM.totalCalls(), 1)
	gt.Equal(t, smartLLM.totalCalls(), 1)
	gt.Equal(t, fast2LLM.totalCalls(), 1)

	// Each task LLM received the description matching its assigned task — no crosstalk.
	containsAny := func(strs []string, needle string) bool {
		for _, s := range strs {
			if strings.Contains(s, needle) {
				return true
			}
		}
		return false
	}
	fastLLM.receivedMu.Lock()
	gt.True(t, containsAny(fastLLM.receivedTask, "FAST_DESC"))
	gt.True(t, !containsAny(fastLLM.receivedTask, "SMART_DESC"))
	gt.True(t, !containsAny(fastLLM.receivedTask, "SECONDARY_DESC"))
	fastLLM.receivedMu.Unlock()

	smartLLM.receivedMu.Lock()
	gt.True(t, containsAny(smartLLM.receivedTask, "SMART_DESC"))
	gt.True(t, !containsAny(smartLLM.receivedTask, "FAST_DESC"))
	smartLLM.receivedMu.Unlock()

	fast2LLM.receivedMu.Lock()
	gt.True(t, containsAny(fast2LLM.receivedTask, "SECONDARY_DESC"))
	fast2LLM.receivedMu.Unlock()
}

func TestExecuteTask_SameLLMID_MultipleTasks(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	planJSON := `{
		"message": "many",
		"tasks": [
			{"id": "a", "title": "A", "description": "A_DESC", "tools": [], "llm_id": "fast"},
			{"id": "b", "title": "B", "description": "B_DESC", "tools": [], "llm_id": "fast"},
			{"id": "c", "title": "C", "description": "C_DESC", "tools": [], "llm_id": "fast"}
		]
	}`
	mainLLM := &trackingLLM{id: "main", plannerJSON: planJSON, genericResp: `{"tasks": []}`}
	fastLLM := &trackingLLM{id: "fast", genericResp: "ok"}
	smartLLM := &trackingLLM{id: "smart", genericResp: "ok"}

	reg := buildRegistry(mainLLM, map[string]*trackingLLM{
		"fast":  fastLLM,
		"smart": smartLLM,
	}, mainLLM.toClient())

	chatUC, err := bluebell.New(repo, reg, bluebell.WithKnowledgeService(knowledgeSvc))
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "x",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)

	gt.Equal(t, fastLLM.totalCalls(), 3)
	gt.Equal(t, smartLLM.totalCalls(), 0)
}

func TestExecuteTask_RejectsUnknownLLMID(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	// Planner emits an llm_id that doesn't exist anywhere in [[llm]].
	planJSON := `{
		"message": "ghost",
		"tasks": [
			{"id": "g1", "title": "Ghost", "description": "GHOST", "tools": [], "llm_id": "nonexistent-id"}
		]
	}`
	mainLLM := &trackingLLM{id: "main", plannerJSON: planJSON, genericResp: `{"tasks": []}`}
	fastLLM := &trackingLLM{id: "fast", genericResp: "should not be called"}

	reg := buildRegistry(mainLLM, map[string]*trackingLLM{"fast": fastLLM}, mainLLM.toClient())

	chatUC, err := bluebell.New(repo, reg, bluebell.WithKnowledgeService(knowledgeSvc))
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "x",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	// Execute itself does not error — task failures bubble up to replan,
	// which sees an empty next phase and proceeds to final response.
	gt.NoError(t, err)

	// No task LLM should have been invoked since resolve failed before agent creation.
	gt.Equal(t, fastLLM.totalCalls(), 0)
}

func TestExecuteTask_RejectsLLMNotInTaskList(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	// Define an extra LLM "hidden" that is in [[llm]] but NOT in [agent].task.
	hiddenLLM := &trackingLLM{id: "hidden", genericResp: "hidden result"}
	mainLLM := &trackingLLM{
		id: "main",
		plannerJSON: `{
			"message": "x",
			"tasks": [
				{"id": "h1", "title": "Hidden", "description": "H", "tools": [], "llm_id": "hidden"}
			]
		}`,
		genericResp: `{"tasks": []}`,
	}

	entries := map[string]*llmconfig.LLMEntry{
		"main":   llmconfig.NewLLMEntryForTest("main", "primary", "claude", "m", mainLLM.toClient()),
		"hidden": llmconfig.NewLLMEntryForTest("hidden", "hidden", "gemini", "h", hiddenLLM.toClient()),
	}
	// task list excludes "hidden"
	reg := llmconfig.NewRegistryForTest("main", []string{"main"}, entries, mainLLM.toClient())

	chatUC, err := bluebell.New(repo, reg, bluebell.WithKnowledgeService(knowledgeSvc))
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "x",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)

	// The hidden LLM was defined but not in the allow-list — Resolve must reject.
	gt.Equal(t, hiddenLLM.totalCalls(), 0)
}

func TestExecuteTask_RejectsEmptyLLMID(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	// Schema declares llm_id required, but if the planner returns it empty
	// (or the JSON omits the field), Resolve must reject with ErrEmptyLLMID.
	planJSON := `{
		"message": "x",
		"tasks": [
			{"id": "e1", "title": "Empty", "description": "E", "tools": [], "llm_id": ""}
		]
	}`
	mainLLM := &trackingLLM{id: "main", plannerJSON: planJSON, genericResp: `{"tasks": []}`}
	fastLLM := &trackingLLM{id: "fast", genericResp: "should not be called"}

	reg := buildRegistry(mainLLM, map[string]*trackingLLM{"fast": fastLLM}, mainLLM.toClient())

	chatUC, err := bluebell.New(repo, reg, bluebell.WithKnowledgeService(knowledgeSvc))
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "x",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)
	gt.Equal(t, fastLLM.totalCalls(), 0)
}

func TestExecuteTask_LLMCallError_PropagatesToResult(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	// fast LLM is selected by planner but Generate returns an error → task
	// fails; bluebell continues to replan/final without aborting Execute.
	planJSON := `{
		"message": "x",
		"tasks": [
			{"id": "x1", "title": "Will fail", "description": "FAIL", "tools": [], "llm_id": "fast"}
		]
	}`
	mainLLM := &trackingLLM{id: "main", plannerJSON: planJSON, genericResp: `{"tasks": []}`}
	fastLLM := &trackingLLM{id: "fast", genErr: errors.New("transient upstream error")}

	reg := buildRegistry(mainLLM, map[string]*trackingLLM{"fast": fastLLM}, mainLLM.toClient())

	chatUC, err := bluebell.New(repo, reg, bluebell.WithKnowledgeService(knowledgeSvc))
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "x",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	// Execute swallows individual task failures (they go to replan).
	gt.NoError(t, err)
	// fast LLM was still attempted (and failed).
	gt.True(t, fastLLM.totalCalls() >= 1)
}

func TestExecuteTask_PrimaryUsedAsTask(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	// Plan picks "main" itself as the task LLM (since main is also in
	// [agent].task by buildRegistry's convention).
	planJSON := `{
		"message": "x",
		"tasks": [
			{"id": "p1", "title": "Use main", "description": "MAIN_TASK", "tools": [], "llm_id": "main"}
		]
	}`
	mainLLM := &trackingLLM{id: "main", plannerJSON: planJSON, genericResp: "main result"}

	reg := buildRegistry(mainLLM, map[string]*trackingLLM{}, mainLLM.toClient())

	chatUC, err := bluebell.New(repo, reg, bluebell.WithKnowledgeService(knowledgeSvc))
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "x",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)

	// main LLM was called for both planner and task agent (so calls >= 2).
	gt.True(t, mainLLM.totalCalls() >= 2)

	// Sanity: task description was sent to main.
	mainLLM.receivedMu.Lock()
	defer mainLLM.receivedMu.Unlock()
	found := false
	for _, p := range mainLLM.receivedTask {
		if strings.Contains(p, "MAIN_TASK") {
			found = true
			break
		}
	}
	gt.True(t, found)
}

func TestExecutePhase_ParallelExecution(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	// All three tasks routed to "fast" should be invoked concurrently.
	// We verify by recording timestamps and checking overlap.
	const N = 3
	planJSON := `{
		"message": "x",
		"tasks": [
			{"id": "p1", "title": "P1", "description": "D1", "tools": [], "llm_id": "fast"},
			{"id": "p2", "title": "P2", "description": "D2", "tools": [], "llm_id": "fast"},
			{"id": "p3", "title": "P3", "description": "D3", "tools": [], "llm_id": "fast"}
		]
	}`
	mainLLM := &trackingLLM{id: "main", plannerJSON: planJSON, genericResp: `{"tasks": []}`}

	// fast LLM uses a barrier: each invocation waits for all N to start.
	barrier := make(chan struct{})
	startedCount := atomic.Int32{}
	allStarted := make(chan struct{})
	fastClient := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(_ context.Context, _ []gollem.Input, _ ...gollem.GenerateOption) (*gollem.Response, error) {
				if startedCount.Add(1) == N {
					close(allStarted)
				}
				select {
				case <-allStarted:
				case <-barrier: // unused outlet
				}
				return &gollem.Response{Texts: []string{"ok"}}, nil
			}
			return ssn, nil
		},
		GenerateEmbeddingFunc: func(_ context.Context, dim int, _ []string) ([][]float64, error) {
			return [][]float64{make([]float64, dim)}, nil
		},
	}

	entries := map[string]*llmconfig.LLMEntry{
		"main": llmconfig.NewLLMEntryForTest("main", "main", "claude", "m", mainLLM.toClient()),
		"fast": llmconfig.NewLLMEntryForTest("fast", "fast", "gemini", "f", fastClient),
	}
	reg := llmconfig.NewRegistryForTest("main", []string{"main", "fast"}, entries, mainLLM.toClient())
	_ = barrier

	chatUC, err := bluebell.New(repo, reg, bluebell.WithKnowledgeService(knowledgeSvc))
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "x",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)

	// If execution were serial, the 1st task could never see startedCount == N.
	// All N tasks must be running concurrently to release the channel.
	// Reaching this assert without hanging proves parallelism.
	gt.Equal(t, int(startedCount.Load()), N)
}

// Sanity: ensure the helper integers tests print correctly when failing.
func TestTrackingLLM_Sanity(t *testing.T) {
	tl := &trackingLLM{id: "x", genericResp: "ok"}
	c := tl.toClient()
	ssn, err := c.NewSession(context.Background())
	gt.NoError(t, err)
	resp, err := ssn.Generate(context.Background(), []gollem.Input{gollem.Text("hello")})
	gt.NoError(t, err)
	gt.Equal(t, resp.Texts[0], "ok")
	gt.Equal(t, tl.totalCalls(), 1)
}

// silence unused import lints
var _ = fmt.Sprintf
var _ = mock.LLMClientMock{}
