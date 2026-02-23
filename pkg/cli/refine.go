package cli

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

func cmdRefine() *cli.Command {
	var (
		llmCfg       config.LLMCfg
		firestoreCfg config.Firestore
		slackCfg     config.Slack
	)

	flags := joinFlags(
		llmCfg.Flags(),
		firestoreCfg.Flags(),
		slackCfg.Flags(),
	)

	return &cli.Command{
		Name:  "refine",
		Usage: "Review open tickets and consolidate unbound alerts",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			return runRefine(ctx, &llmCfg, &firestoreCfg, &slackCfg)
		},
	}
}

func runRefine(ctx context.Context, llmCfg *config.LLMCfg, firestoreCfg *config.Firestore, slackCfg *config.Slack) error {
	logger := logging.From(ctx)

	// Setup LLM client (required)
	llmClient, err := llmCfg.Configure(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to create LLM client")
	}

	var ucOptions []usecase.Option
	ucOptions = append(ucOptions, usecase.WithLLMClient(llmClient))

	// Setup Firestore (optional)
	if firestoreCfg.IsConfigured() {
		repo, err := firestoreCfg.Configure(ctx)
		if err != nil {
			return goerr.Wrap(err, "failed to create Firestore client")
		}
		defer safe.Close(ctx, repo)
		ucOptions = append(ucOptions, usecase.WithRepository(repo))
	}

	// Setup Slack (optional)
	if slackCfg.IsConfigured() {
		slackService, err := slackCfg.Configure()
		if err != nil {
			return goerr.Wrap(err, "failed to create Slack service")
		}
		ucOptions = append(ucOptions, usecase.WithSlackService(slackService))
	}

	uc := usecase.New(ucOptions...)

	logger.Info("starting refine")
	if err := uc.Refine(ctx); err != nil {
		return goerr.Wrap(err, "failed to run refine")
	}

	logger.Info("refine completed")
	return nil
}
