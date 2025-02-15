package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdRun() *cli.Command {
	var (
		alertPath   string
		alertSchema string
		lang        string
		policyCfg   config.Policy
		geminiCfg   config.GeminiCfg
	)

	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Run alert investigation on local",
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
				&cli.StringFlag{
					Name:        "lang",
					Aliases:     []string{"l"},
					Usage:       "Language of the text [en, ja]",
					Destination: &lang,
					Value:       "en",
					Sources:     cli.EnvVars("WARREN_LANG"),
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
				"lang", lang,
			)

			if err := prompt.SetDefaultLang(lang); err != nil {
				return err
			}

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
				usecase.WithPolicyClient(policyClient),
				usecase.WithSlackService(service.NewConsole(os.Stdout)),
				usecase.WithActionService(actionSvc),
			)

			alerts, err := uc.HandleAlert(ctx, alertSchema, alertData)
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
