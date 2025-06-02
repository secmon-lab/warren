package base

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (x *Warren) listPolicies(ctx context.Context, _ map[string]any) (map[string]any, error) {
	var rows []any
	for name := range x.policies {
		rows = append(rows, name)
	}

	result := map[string]any{
		"policies": rows,
	}

	logging.From(ctx).Debug("list policies", "policies", rows)

	return result, nil
}

func (x *Warren) getPolicy(ctx context.Context, args map[string]any) (map[string]any, error) {
	errResp := func(msg string) (map[string]any, error) {
		return map[string]any{
			"policy": "",
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

	result := map[string]any{
		"policy": policy,
	}

	logging.From(ctx).Debug("get policy", "policy", policy)

	return result, nil
}
