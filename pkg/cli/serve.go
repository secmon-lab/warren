package cli

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/secmon-lab/warren/pkg/cli/config"
	server "github.com/secmon-lab/warren/pkg/controller/http"
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
		testDataCfg  config.TestData
		embeddingCfg config.EmbeddingCfg
		githubAppCfg config.GitHubAppCfg
	)

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
		testDataCfg.Flags(),
		actions.Flags(),
		embeddingCfg.Flags(),
		githubAppCfg.Flags(),
	)

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
				"embedding", embeddingCfg,
				"firestore", firestoreCfg,
				"testdata", testDataCfg,
			)

			policyClient, err := policyCfg.Configure()
			if err != nil {
				return err
			}

			geminiModel, err := geminiCfg.Configure(ctx)
			if err != nil {
				return err
			}

			embeddingClient := embeddingCfg.Configure()

			if err := sentryCfg.Configure(); err != nil {
				return err
			}

			slackSvc, err := slackCfg.Configure()
			if err != nil {
				return err
			}

			firestore, err := firestoreCfg.Configure(ctx)
			if err != nil {
				return err
			}

			testDataSet, err := testDataCfg.Configure()
			if err != nil {
				return err
			}

			/*
				enabledActions, err := actions.Configure(ctx)
				if err != nil {
					return err
				}
				logging.Default().Info("enabled actions", "actions", actions)
			*/
			ucOptions := []usecase.Option{
				usecase.WithLLMClient(geminiModel),
				usecase.WithEmbeddingClient(embeddingClient),
				usecase.WithPolicyClient(policyClient),
				usecase.WithRepository(firestore),
				usecase.WithSlackService(slackSvc),
				usecase.WithTestDataSet(testDataSet),
			}

			githubApp, err := githubAppCfg.Configure(ctx)
			if err != nil {
				return err
			}
			ucOptions = append(ucOptions, usecase.WithGitHubApp(githubApp))

			uc := usecase.New(ucOptions...)

			httpServer := http.Server{
				Addr: addr,
				Handler: server.New(uc,
					server.WithSlackVerifier(slackCfg.Verifier()),
				),
				ReadTimeout:       30 * time.Second,
				ReadHeaderTimeout: 10 * time.Second,
				BaseContext: func(l net.Listener) context.Context {
					return ctx
				},
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
