package aster

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	hitlService "github.com/secmon-lab/warren/pkg/service/hitl"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// hitlConfig holds the configuration for the HITL tool middleware.
type hitlConfig struct {
	requireApproval map[string]bool
	service         *hitlService.Service
	presenter       hitlService.Presenter
	userID          string
	sessionID       types.SessionID
	slackThread     *slackModel.Thread
}

// newHITLMiddleware creates a gollem.ToolMiddleware that intercepts tool calls
// requiring human approval. It uses the HITL service to present the request
// and block until the user responds.
func newHITLMiddleware(cfg hitlConfig) gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			if !cfg.requireApproval[req.Tool.Name] {
				return next(ctx, req)
			}

			// Block execution if no presenter is available.
			// HITL-required tools must not bypass approval just because
			// the transport (Slack, CLI, etc.) is not configured.
			if cfg.presenter == nil {
				return &gollem.ToolExecResponse{
					Error: goerr.New("tool requires human approval but no HITL presenter is available",
						goerr.V("tool", req.Tool.Name)),
				}, nil
			}

			// Build HITL request
			hitlReq := &hitl.Request{
				ID:        types.NewHITLRequestID(),
				SessionID: cfg.sessionID,
				Type:      hitl.RequestTypeToolApproval,
				Payload:   hitl.NewToolApprovalPayload(req.Tool.Name, req.Tool.Arguments),
				Status:    hitl.StatusPending,
				UserID:    cfg.userID,
				CreatedAt: time.Now(),
			}
			if cfg.slackThread != nil {
				hitlReq.SlackThread = *cfg.slackThread
			}

			// Request approval and wait
			result, err := cfg.service.RequestAndWait(ctx, hitlReq, cfg.presenter)
			if err != nil {
				return &gollem.ToolExecResponse{
					Error: err,
				}, nil
			}

			// Handle denied
			if result.Status == hitl.StatusDenied {
				denyMsg := "Tool execution denied by user"
				if comment := result.ResponseComment(); comment != "" {
					denyMsg += ": " + comment
				}

				msg.Trace(ctx, "🚫 `%s` was denied", req.Tool.Name)

				return &gollem.ToolExecResponse{
					Result: map[string]any{
						"error": denyMsg,
					},
				}, nil
			}

			// Approved — continue to actual tool execution
			return next(ctx, req)
		}
	}
}

// buildHITLPresenter creates a Slack HITL presenter for a task.
// Returns nil if Slack is not configured.
func buildHITLPresenter(updatableMsg *slackService.UpdatableBlockMessage, taskTitle, userID string) hitlService.Presenter {
	if updatableMsg == nil {
		return nil
	}
	return slackService.NewHITLPresenter(updatableMsg, taskTitle, userID)
}
