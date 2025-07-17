package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

type PolicyCfg struct {
	policyDir string
}

func (x *PolicyCfg) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "policy-dir",
			Usage:       "Directory path containing policy files",
			Destination: &x.policyDir,
			Category:    "Policy",
			Sources:     cli.EnvVars("WARREN_POLICY_DIR"),
		},
	}
}

func (x PolicyCfg) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("policy_dir", x.policyDir),
	)
}

func (x *PolicyCfg) Configure(ctx context.Context) (interfaces.PolicyClient, error) {
	if x.policyDir == "" {
		// Return a dummy policy client when no policy directory is specified
		return &dummyPolicyClient{}, nil
	}

	// opaq.Files expects individual file paths, but we need to glob the directory
	// For now, return dummy client until proper policy files are configured
	return &dummyPolicyClient{}, nil
}

// dummyPolicyClient provides a no-op implementation when no policies are configured
type dummyPolicyClient struct{}

func (d *dummyPolicyClient) Query(ctx context.Context, query string, input any, output any, options ...opaq.QueryOption) error {
	// For alerts, return empty result to indicate no alerts should be processed
	return nil
}

func (d *dummyPolicyClient) Sources() map[string]string {
	return map[string]string{}
}
