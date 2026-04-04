package chat

import (
	"context"
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

var (
	// ErrAgentAuthPolicyNotDefined is returned when agent authorization policy is not defined
	ErrAgentAuthPolicyNotDefined = errors.New("agent authorization policy not defined")

	// ErrAgentAuthDenied is returned when agent authorization is denied
	ErrAgentAuthDenied = errors.New("agent request not authorized")
)

// AuthorizeAgentRequest checks policy-based authorization for agent execution.
func AuthorizeAgentRequest(ctx context.Context, policyClient interfaces.PolicyClient, noAuthz bool, message string) error {
	logger := logging.From(ctx)

	if noAuthz {
		return nil
	}

	authCtx := auth.BuildAgentContext(ctx, message)

	var result struct {
		Allow bool `json:"allow"`
	}

	query := "data.auth.agent"
	err := policyClient.Query(ctx, query, authCtx, &result, opaq.WithPrintHook(func(ctx context.Context, loc opaq.PrintLocation, msg string) error {
		logger.Debug("[rego] "+msg, "loc", loc)
		return nil
	}))
	if err != nil {
		if errors.Is(err, opaq.ErrNoEvalResult) {
			logger.Warn("agent authorization policy not defined, denying by default")
			return goerr.Wrap(ErrAgentAuthPolicyNotDefined, "agent authorization policy not defined")
		}
		return goerr.Wrap(err, "failed to evaluate agent authorization policy")
	}

	logger.Debug("agent authorization result", "input", authCtx, "output", result)

	if !result.Allow {
		logger.Warn("agent authorization failed", "message", message)
		return goerr.Wrap(ErrAgentAuthDenied, "agent request denied by policy", goerr.V("message", message))
	}

	return nil
}
