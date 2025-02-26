package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/lang"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
	"github.com/urfave/cli/v3"
)

func cmdRun() *cli.Command {
	var language model.Lang
	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Run alert investigation on local",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "lang",
				Aliases:     []string{"l"},
				Usage:       "Language of GenAI output [en, ja]",
				Sources:     cli.EnvVars("WARREN_LANG"),
				Destination: (*string)(&language),
				Value:       "en",
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if err := language.Validate(); err != nil {
				return nil, goerr.Wrap(err, "invalid language")
			}
			return lang.With(ctx, language), nil
		},
		Commands: []*cli.Command{
			cmdInspect(),
			cmdGroup(),
		},
	}
}

func cmdGroup() *cli.Command {
	var (
		firestoreCfg config.Firestore
		geminiCfg    config.GeminiCfg
	)

	flags := joinFlags(
		firestoreCfg.Flags(),
		geminiCfg.Flags(),
	)

	return &cli.Command{
		Name:    "group",
		Aliases: []string{"g"},
		Usage:   "Group alerts",
		Flags:   flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			logger := logging.From(ctx)
			logger.Info("group mode", "firestore", firestoreCfg, "gemini", geminiCfg)

			console := service.NewConsole(os.Stdout)
			firestore, err := firestoreCfg.Configure(ctx)
			if err != nil {
				return err
			}

			geminiModel, err := geminiCfg.Configure(ctx)
			if err != nil {
				return err
			}

			uc := usecase.New(
				func() interfaces.GenAIChatSession {
					return geminiModel.StartChat()
				},
				usecase.WithRepository(firestore),
				usecase.WithSlackService(service.NewConsole(os.Stdout)),
			)

			th := console.NewThread(model.SlackThread{})
			ctx = thread.WithReplyFunc(ctx, th.Reply)

			err = uc.GroupUnclosedAlerts(ctx, th)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func cmdInspect() *cli.Command {
	var (
		alertPath   string
		alertSchema string
		policyCfg   config.Policy
		geminiCfg   config.GeminiCfg
	)

	return &cli.Command{
		Name:    "inspect",
		Aliases: []string{"i"},
		Usage:   "Inspect alert",
		Flags: joinFlags(
			[]cli.Flag{
				&cli.StringFlag{
					Name:        "alert",
					Aliases:     []string{"a"},
					Usage:       "alert file path",
					Destination: &alertPath,
					Required:    true,
					Sources:     cli.EnvVars("WARREN_ALERT_PATH"),
				},
				&cli.StringFlag{
					Name:        "schema",
					Aliases:     []string{"s"},
					Usage:       "alert schema",
					Destination: &alertSchema,
					Required:    true,
					Sources:     cli.EnvVars("WARREN_ALERT_SCHEMA"),
				},
			},
			policyCfg.Flags(),
			geminiCfg.Flags(),
			actions.Flags(),
		),
		Action: func(ctx context.Context, c *cli.Command) error {
			logger := logging.From(ctx)
			logger.Info("run mode",
				"alert", alertPath,
				"schema", alertSchema,
				"policy", policyCfg,
				"gemini", geminiCfg,
			)

			policyClient, err := policyCfg.Configure()
			if err != nil {
				return err
			}

			geminiModel, err := geminiCfg.Configure(ctx)
			if err != nil {
				return err
			}

			fd, err := os.Open(filepath.Clean(alertPath))
			if err != nil {
				return err
			}
			var alertData any
			if err := json.NewDecoder(fd).Decode(&alertData); err != nil {
				return err
			}

			enabledActions, err := actions.Configure(ctx)
			if err != nil {
				return err
			}
			actionSvc := service.NewActionService(enabledActions)
			logging.Default().Info("enabled actions", "actions", actions)

			uc := usecase.New(
				func() interfaces.GenAIChatSession {
					return geminiModel.StartChat()
				},
				usecase.WithPolicyService(policy.New(repository.NewMemory(), policyClient, &model.TestDataSet{})),
				usecase.WithSlackService(service.NewConsole(os.Stdout)),
				usecase.WithActionService(actionSvc),
			)

			alerts, err := uc.HandleAlert(ctx, alertSchema, alertData, policyClient)
			if err != nil {
				return err
			}

			for _, alert := range alerts {
				if err := uc.RunWorkflow(ctx, *alert); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
