package base

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (x *Base) listPolicies(ctx context.Context, _ map[string]any) (*action.Result, error) {
	var rows []any
	for name := range x.policies {
		rows = append(rows, name)
	}

	result := &action.Result{
		Name: "base.policy.list",
		Data: map[string]any{
			"policies": rows,
		},
	}

	logging.From(ctx).Debug("list policies", "policies", rows)

	return result, nil
}

func (x *Base) getPolicy(ctx context.Context, args map[string]any) (*action.Result, error) {
	errResp := func(msg string) (*action.Result, error) {
		return &action.Result{
			Name: "base.policy.get",
			Data: map[string]any{
				"policy": "",
			},
		}, goerr.New(msg)
	}

	argName, ok := args["name"]
	if !ok {
		return errResp("Policy name is required")
	}

	name, ok := argName.(string)
	if !ok {
		return errResp("Policy name is not a string")
	}

	policy, ok := x.policies[name]
	if !ok {
		return errResp("Policy not found")
	}

	result := &action.Result{
		Name: "base.policy.get",
		Data: map[string]any{
			"policy": policy,
		},
	}

	logging.From(ctx).Debug("get policy", "policy", policy)

	return result, nil
}
