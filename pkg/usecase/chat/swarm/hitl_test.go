package swarm_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository/memory"
	hitlSvc "github.com/secmon-lab/warren/pkg/service/hitl"
	"github.com/secmon-lab/warren/pkg/usecase/chat/swarm"
)

type testPresenter struct {
	presented atomic.Pointer[hitl.Request]
}

func (p *testPresenter) Present(_ context.Context, req *hitl.Request) error {
	p.presented.Store(req)
	return nil
}

func buildToolExecRequest(toolName string, args map[string]any) *gollem.ToolExecRequest {
	return &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{
			Name:      toolName,
			Arguments: args,
		},
	}
}

func TestHITLMiddleware_Approved(t *testing.T) {
	repo := memory.New()
	svc := hitlSvc.New(repo, hitlSvc.WithTimeout(10*time.Second))
	presenter := &testPresenter{}

	mw := swarm.NewHITLMiddleware(swarm.NewHITLConfig(
		map[string]bool{"web_fetch": true},
		svc, presenter, "U12345", types.NewSessionID(), nil,
	))

	var toolExecuted atomic.Bool
	baseHandler := func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		toolExecuted.Store(true)
		return &gollem.ToolExecResponse{Result: map[string]any{"ok": true}}, nil
	}

	handler := mw(baseHandler)
	req := buildToolExecRequest("web_fetch", map[string]any{"url": "https://example.com"})

	// Respond with approval after the request is presented
	go func() {
		for presenter.presented.Load() == nil {
			time.Sleep(50 * time.Millisecond)
		}
		hitlReq := presenter.presented.Load()
		_ = svc.Respond(t.Context(), hitlReq.ID, hitl.StatusApproved, "U67890", map[string]any{"comment": "ok"})
	}()

	resp, err := handler(t.Context(), req)
	gt.NoError(t, err).Required()
	gt.Value(t, resp.Error).Equal(nil)
	gt.Value(t, resp.Result["ok"]).Equal(true)
	gt.Value(t, toolExecuted.Load()).Equal(true)
}

func TestHITLMiddleware_Denied(t *testing.T) {
	repo := memory.New()
	svc := hitlSvc.New(repo, hitlSvc.WithTimeout(10*time.Second))
	presenter := &testPresenter{}

	mw := swarm.NewHITLMiddleware(swarm.NewHITLConfig(
		map[string]bool{"web_fetch": true},
		svc, presenter, "U12345", types.NewSessionID(), nil,
	))

	var toolExecuted atomic.Bool
	baseHandler := func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		toolExecuted.Store(true)
		return &gollem.ToolExecResponse{Result: map[string]any{"ok": true}}, nil
	}

	handler := mw(baseHandler)
	req := buildToolExecRequest("web_fetch", map[string]any{"url": "https://example.com"})

	go func() {
		for presenter.presented.Load() == nil {
			time.Sleep(50 * time.Millisecond)
		}
		hitlReq := presenter.presented.Load()
		_ = svc.Respond(t.Context(), hitlReq.ID, hitl.StatusDenied, "U67890", map[string]any{"comment": "not allowed"})
	}()

	resp, err := handler(t.Context(), req)
	gt.NoError(t, err).Required()
	gt.Value(t, resp.Error).Equal(nil)
	gt.Value(t, resp.Result["error"]).Equal("Tool execution denied by user: not allowed")
	gt.Value(t, toolExecuted.Load()).Equal(false)
}

func TestHITLMiddleware_NonHITLTool_PassesThrough(t *testing.T) {
	repo := memory.New()
	svc := hitlSvc.New(repo)
	presenter := &testPresenter{}

	mw := swarm.NewHITLMiddleware(swarm.NewHITLConfig(
		map[string]bool{"web_fetch": true},
		svc, presenter, "U12345", types.NewSessionID(), nil,
	))

	var toolExecuted atomic.Bool
	baseHandler := func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		toolExecuted.Store(true)
		return &gollem.ToolExecResponse{Result: map[string]any{"ok": true}}, nil
	}

	handler := mw(baseHandler)
	req := buildToolExecRequest("whois_domain", map[string]any{"target": "example.com"})

	resp, err := handler(t.Context(), req)
	gt.NoError(t, err).Required()
	gt.Value(t, resp.Result["ok"]).Equal(true)
	gt.Value(t, toolExecuted.Load()).Equal(true)
}

func TestHITLMiddleware_NoPresenter_BlocksExecution(t *testing.T) {
	repo := memory.New()
	svc := hitlSvc.New(repo)

	mw := swarm.NewHITLMiddleware(swarm.NewHITLConfig(
		map[string]bool{"web_fetch": true},
		svc, nil, "U12345", types.NewSessionID(), nil,
	))

	var toolExecuted atomic.Bool
	baseHandler := func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		toolExecuted.Store(true)
		return &gollem.ToolExecResponse{Result: map[string]any{"ok": true}}, nil
	}

	handler := mw(baseHandler)
	req := buildToolExecRequest("web_fetch", map[string]any{"url": "https://example.com"})

	resp, err := handler(t.Context(), req)
	gt.NoError(t, err).Required()
	gt.Value(t, resp.Error != nil).Equal(true)
	gt.Value(t, toolExecuted.Load()).Equal(false)
}

func TestHITLMiddleware_Timeout(t *testing.T) {
	repo := memory.New()
	svc := hitlSvc.New(repo, hitlSvc.WithTimeout(500*time.Millisecond))
	presenter := &testPresenter{}

	mw := swarm.NewHITLMiddleware(swarm.NewHITLConfig(
		map[string]bool{"web_fetch": true},
		svc, presenter, "U12345", types.NewSessionID(), nil,
	))

	var toolExecuted atomic.Bool
	baseHandler := func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		toolExecuted.Store(true)
		return &gollem.ToolExecResponse{Result: map[string]any{"ok": true}}, nil
	}

	handler := mw(baseHandler)
	req := buildToolExecRequest("web_fetch", map[string]any{"url": "https://example.com"})

	// Don't respond — should timeout
	resp, err := handler(t.Context(), req)
	gt.NoError(t, err).Required()
	gt.Value(t, resp.Error != nil).Equal(true)
	gt.Value(t, toolExecuted.Load()).Equal(false)
}

func TestHandleQuestion_AnswerFlow(t *testing.T) {
	repo := memory.New()
	presenter := &testPresenter{}
	sessionID := types.NewSessionID()

	q := &swarm.Question{
		Question: "Is 10.0.0.5 an internal IP?",
		Options:  []string{"Yes, VPN GW", "No", "None of the above"},
		Reason:   "Cannot determine from tool output",
	}

	svc := hitlSvc.New(repo, hitlSvc.WithTimeout(10*time.Second))

	// Respond after the presenter receives the request
	go func() {
		for presenter.presented.Load() == nil {
			time.Sleep(50 * time.Millisecond)
		}
		hitlReq := presenter.presented.Load()
		_ = svc.Respond(t.Context(), hitlReq.ID, hitl.StatusApproved, "U67890", map[string]any{
			"answer":  "Yes, VPN GW",
			"comment": "Tokyo DC",
		})
	}()

	result, err := swarm.ExecHandleQuestion(t.Context(), repo, presenter, q, sessionID, "U12345")
	gt.NoError(t, err).Required()
	gt.Value(t, result.Question).Equal("Is 10.0.0.5 an internal IP?")
	gt.Array(t, result.Options).Length(3).Required()
	gt.Value(t, result.Answer).Equal("Yes, VPN GW")
	gt.Value(t, result.Comment).Equal("Tokyo DC")
}

func TestHandleQuestion_NilPresenter_Fails(t *testing.T) {
	repo := memory.New()
	sessionID := types.NewSessionID()

	q := &swarm.Question{
		Question: "Is this allowed?",
		Options:  []string{"Yes", "No"},
		Reason:   "Policy unclear",
	}

	_, err := swarm.ExecHandleQuestion(t.Context(), repo, nil, q, sessionID, "U12345")
	gt.Error(t, err)
}
