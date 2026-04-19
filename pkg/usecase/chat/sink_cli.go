package chat

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/m-mizutani/goerr/v2"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// cliSink routes output to the CLI's stdout via the msg.* notify/trace
// handlers already installed on the context by the CLI bootstrap
// (ChatFromCLI). This deliberately forwards to the *outer* handlers
// captured at ResolveSink time so that aster/bluebell's direct sink
// calls produce the same CLI output they produced before the refactor.
//
// CLI never persists SessionMessages from sink calls — the CLI user
// observes output in real time and there is no Conversation UI to
// replay later. (User input is still persisted by the caller via the
// usecase layer; that is unrelated to sink output.)
type cliSink struct {
	chatCtx *chatModel.ChatContext
}

func newCLISink(chatCtx *chatModel.ChatContext) ChatSink {
	if chatCtx == nil {
		return nil
	}
	return &cliSink{chatCtx: chatCtx}
}

func (s *cliSink) PostComment(ctx context.Context, text string) error {
	msg.Notify(ctx, "%s", text)
	return nil
}

func (s *cliSink) PostContextBlock(ctx context.Context, text string) error {
	msg.Trace(ctx, "%s", text)
	return nil
}

func (s *cliSink) PostSectionBlock(ctx context.Context, text string) error {
	msg.Notify(ctx, "%s", text)
	return nil
}

func (s *cliSink) PostDivider(ctx context.Context) error {
	msg.Trace(ctx, "%s", "---")
	return nil
}

// NewUpdatableMessage on CLI prints each update as a new trace line.
// Terminals cannot update a prior line without cursor control that we
// do not want to rely on here, so the stream form is the least-bad
// representation: the reader sees every revision in order.
func (s *cliSink) NewUpdatableMessage(ctx context.Context, initial string) func(ctx context.Context, text string) {
	msg.Trace(ctx, "%s", initial)
	return func(ctx context.Context, text string) {
		msg.Trace(ctx, "%s", text)
	}
}

// cliProgressHandle streams task progress updates to stdout. HITL is
// default-deny on CLI: PresentHITL returns an error so the middleware
// blocks tool execution. (Interactive terminal HITL is a future
// feature; until then CLI callers can still fall back to running the
// agent with HITL tools disabled.)
type cliProgressHandle struct {
	title atomic.Pointer[string] // never nil once set via newCLIProgressHandle
}

func newCLIProgressHandle(ctx context.Context, initialText string) ProgressHandle {
	h := &cliProgressHandle{}
	initial := initialText
	h.title.Store(&initial)
	msg.Trace(ctx, "%s", initial)
	return h
}

func (h *cliProgressHandle) UpdateText(ctx context.Context, text string) {
	if h == nil {
		return
	}
	t := text
	h.title.Store(&t)
	msg.Trace(ctx, "%s", text)
}

// PresentHITL on CLI rejects the request. The HITL middleware will
// surface the returned error as a tool failure, preventing the tool
// from executing without approval. See the test matrix in
// pkg/usecase/chat/aster/hitl_test.go (TestHITLMiddleware_NoPresenter).
func (h *cliProgressHandle) PresentHITL(_ context.Context, req *hitl.Request, _ /* taskTitle */, _ /* userID */ string) error {
	return goerr.New("HITL approval is not available on the CLI transport",
		goerr.V("request_id", req.ID),
		goerr.V("tool_name", fmt.Sprintf("%v", req.Payload["tool_name"])),
	)
}
