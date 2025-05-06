package cli

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-mizutani/gollem"
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
		embeddingCfg config.EmbeddingCfg
		storageCfg   config.Storage
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
		tools.Flags(),
		embeddingCfg.Flags(),
		storageCfg.Flags(),
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
				"storage", storageCfg,
			)

			policyClient, err := policyCfg.Configure()
			if err != nil {
				return err
			}

			geminiModel, err := geminiCfg.Configure(ctx)
			if err != nil {
				return err
			}

			embeddingClient, err := embeddingCfg.Configure()
			if err != nil {
				return err
			}

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

			storageClient, err := storageCfg.Configure(ctx)
			if err != nil {
				return err
			}

			toolSets, err := tools.ToolSets(ctx)
			if err != nil {
				return err
			}

			agent := gollem.New(geminiModel,
				gollem.WithToolSets(toolSets...),
			)

			ucOptions := []usecase.Option{
				usecase.WithLLMClient(geminiModel),
				usecase.WithEmbeddingClient(embeddingClient),
				usecase.WithPolicyClient(policyClient),
				usecase.WithRepository(firestore),
				usecase.WithSlackService(slackSvc),
				usecase.WithStorageClient(storageClient),
				usecase.WithAgent(agent),
			}

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
