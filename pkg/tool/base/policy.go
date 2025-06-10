package base

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (x *Warren) listPolicies(ctx context.Context, _ map[string]any) (map[string]any, error) {
	var rows []any
	for name := range x.policyClient.Sources() {
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

	policy, ok := x.policyClient.Sources()[name]
	if !ok {
		return errResp("Policy not found")
	}

	result := map[string]any{
		"policy": policy,
	}

	logging.From(ctx).Debug("get policy", "policy", policy)

	return result, nil
}

func (x *Warren) execPolicy(ctx context.Context, args map[string]any) (map[string]any, error) {
	schema, err := getArg[string](args, "schema")
	if err != nil {
		return nil, goerr.Wrap(err, "schema is not a string")
	}

	input, err := getArg[string](args, "input")
	if err != nil {
		return nil, goerr.Wrap(err, "input is not a string")
	}

	var inputData any
	if err := json.Unmarshal([]byte(input), &inputData); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal input")
	}

	var result struct {
		Alert []alert.Metadata `json:"alert"`
	}
	if err := x.policyClient.Query(ctx, "data.alert."+string(schema), inputData, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("alert", inputData))
	}

	// Convert []alert.Metadata to []any for compatibility with Vertex AI API
	alerts := make([]any, len(result.Alert))
	for i, alert := range result.Alert {
		alerts[i] = alert
	}

	return map[string]any{
		"alert": alerts,
	}, nil
}
