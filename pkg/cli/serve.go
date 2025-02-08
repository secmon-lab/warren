package cli

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/action/bigquery"
	"github.com/secmon-lab/warren/pkg/action/otx"
	"github.com/secmon-lab/warren/pkg/action/urlscan"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/server"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdServe() *cli.Command {
	var (
		addr         string
		policyCfg    config.Policy
		sentryCfg    config.Sentry
		slackCfg     config.Slack
		geminiCfg    config.GeminiCfg
		firestoreCfg config.Firestore
	)

	actions := []interfaces.Action{
		&urlscan.Action{},
		&otx.Action{},
		&bigquery.Action{},
	}
	flags := joinFlags(
		[]cli.Flag{
			&cli.StringFlag{
				Name:        "addr",
				Aliases:     []string{"a"},
				Sources:     cli.EnvVars("WARREN_ADDR"),
				Usage:       "Listen address (default: 127.0.0.1:8080)",
				Value:       "127.0.0.1:8080",
				Destination: &addr,
			},
		},
		policyCfg.Flags(),
		sentryCfg.Flags(),
		slackCfg.Flags(),
		geminiCfg.Flags(),
		firestoreCfg.Flags(),
	)

	for _, action := range actions {
		flags = append(flags, action.Flags()...)
	}

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "Run server",
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logging.Default().Info("starting server",
				"addr", addr,
				"policy", policyCfg,
				"sentry", sentryCfg,
				"slack", slackCfg,
				"gemini", geminiCfg,
				"firestore", firestoreCfg,
			)

			policyClient, err := policyCfg.Configure()
			if err != nil {
				return err
			}

			geminiModel, err := geminiCfg.Configure(ctx)
			if err != nil {
				return err
			}

			if err := sentryCfg.Configure(); err != nil {
				return err
			}

			slackSvc := slackCfg.Configure()

			firestore, err := firestoreCfg.Configure(ctx)
			if err != nil {
				return err
			}

			var enabledActions []interfaces.Action
			var enabledActionNames []string
			for _, action := range actions {
				if err := action.Configure(ctx); err != nil {
					if !errors.Is(err, model.ErrActionUnavailable) {
						return goerr.Wrap(err, "action is not available", goerr.V("action", action.Spec().Name))
					}
					continue
				}

				enabledActions = append(enabledActions, action)
				enabledActionNames = append(enabledActionNames, action.Spec().Name)
			}
			actionSvc := service.NewActionService(enabledActions)
			logging.Default().Info("enabled actions", "actions", enabledActionNames)

			uc := usecase.New(
				func() interfaces.GenAIChatSession {
					return geminiModel.StartChat()
				},
				usecase.WithPolicyClient(policyClient),
				usecase.WithSlackService(slackSvc),
				usecase.WithRepository(firestore),
				usecase.WithActionService(actionSvc),
			)

			httpServer := http.Server{
				Addr:              addr,
				Handler:           server.New(uc),
				ReadTimeout:       30 * time.Second,
				ReadHeaderTimeout: 10 * time.Second,
			}

			errCh := make(chan error, 1)
			go func() {
				defer close(errCh)
				if err := httpServer.ListenAndServe(); err != nil {
					errCh <- err
				}
			}()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			select {
			case err := <-errCh:
				return err
			case <-sigCh:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return httpServer.Shutdown(ctx)
			}
		},
	}
}
