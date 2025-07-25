package cli

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdServe() *cli.Command {
	var (
		addr           string
		enableGraphQL  bool
		enableGraphiQL bool
		webUICfg       config.WebUI
		policyCfg      config.Policy
		sentryCfg      config.Sentry
		slackCfg       config.Slack
		llmCfg         config.LLMCfg
		firestoreCfg   config.Firestore
		storageCfg     config.Storage
		mcpCfg         config.MCPConfig
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
			&cli.BoolFlag{
				Name:        "enable-graphql",
				Usage:       "Enable GraphQL endpoint",
				Category:    "GraphQL",
				Sources:     cli.EnvVars("WARREN_ENABLE_GRAPHQL"),
				Destination: &enableGraphQL,
			},
			&cli.BoolFlag{
				Name:        "enable-graphiql",
				Usage:       "Enable GraphiQL playground (requires --enable-graphql)",
				Category:    "GraphQL",
				Sources:     cli.EnvVars("WARREN_ENABLE_GRAPHIQL"),
				Destination: &enableGraphiQL,
			},
		},
		webUICfg.Flags(),
		policyCfg.Flags(),
		sentryCfg.Flags(),
		slackCfg.Flags(),
		llmCfg.Flags(),
		firestoreCfg.Flags(),
		tools.Flags(),
		storageCfg.Flags(),
		mcpCfg.Flags(),
	)

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "Run server",
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logging.Default().Info("starting server",
				"addr", addr,
				"enableGraphQL", enableGraphQL,
				"enableGraphiQL", enableGraphiQL,
				"web-ui", webUICfg,
				"policy", policyCfg,
				"sentry", sentryCfg,
				"slack", slackCfg,
				"llm", llmCfg,
				"firestore", firestoreCfg,
				"storage", storageCfg,
				"mcp", mcpCfg,
			)

			policyClient, err := policyCfg.Configure()
			if err != nil {
				return err
			}

			// Configure LLM client (automatically selects Claude if available, otherwise Gemini)
			llmClient, err := llmCfg.Configure(ctx)
			if err != nil {
				return err
			}

			if err := sentryCfg.Configure(); err != nil {
				return err
			}

			slackSvc, err := slackCfg.ConfigureOptionalWithFrontendURL(webUICfg.GetFrontendURL())
			if err != nil {
				return err
			}
			if slackSvc != nil {
				defer slackSvc.Stop()
			}

			firestore, err := firestoreCfg.Configure(ctx)
			if err != nil {
				return err
			}

			storageClient, err := storageCfg.Configure(ctx)
			if err != nil {
				return err
			}

			// Create embedding client using unified LLM configuration
			embeddingAdapter, err := llmCfg.ConfigureEmbeddingClient(ctx)
			if err != nil {
				return err
			}

			// Inject dependencies into tools that support them
			tools.InjectDependencies(firestore, embeddingAdapter)

			toolSets, err := tools.ToolSets(ctx)
			if err != nil {
				return err
			}

			// Add MCP tool sets if configured
			mcpToolSets, err := mcpCfg.CreateMCPToolSets(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to create MCP tool sets")
			}
			if len(mcpToolSets) > 0 {
				toolSets = append(toolSets, mcpToolSets...)
				logging.From(ctx).Info("MCP tool sets configured",
					"servers", mcpCfg.GetServerNames(),
					"count", len(mcpToolSets))
			}

			// Configure SlackNotifier based on whether Slack service is available
			var slackNotifier usecase.SlackNotifier
			if slackSvc != nil {
				slackNotifier = usecase.NewSlackNotifier(slackSvc)
			} else {
				slackNotifier = usecase.NewDiscardSlackNotifier()
			}

			ucOptions := []usecase.Option{
				usecase.WithLLMClient(llmClient),
				usecase.WithPolicyClient(policyClient),
				usecase.WithRepository(firestore),
				usecase.WithSlackNotifier(slackNotifier),
				usecase.WithStorageClient(storageClient),
				usecase.WithTools(toolSets),
			}

			uc := usecase.New(ucOptions...)

			// Build HTTP server options
			serverOptions := []server.Options{
				server.WithPolicy(policyClient),
			}

			// Add Slack-related options only if Slack is configured
			if slackSvc != nil {
				serverOptions = append(serverOptions, 
					server.WithSlackService(slackSvc),
					server.WithSlackVerifier(slackCfg.Verifier()),
				)
			}

			// Add repository when GraphQL is enabled
			if enableGraphQL {
				serverOptions = append(serverOptions, server.WithGraphQLRepo(firestore))
			}

			// Add GraphiQL option when GraphiQL is enabled
			if enableGraphiQL {
				serverOptions = append(serverOptions, server.WithGraphiQL(true))
				if !enableGraphQL {
					logging.From(ctx).Warn("GraphiQL is enabled but GraphQL is not enabled. GraphiQL will not work.")
				}
			}

			// Add AuthUseCase if authentication options are provided
			authUC, err := webUICfg.Configure(firestore, slackSvc)
			if err != nil {
				return err
			}
			if authUC != nil {
				serverOptions = append(serverOptions, server.WithAuthUseCase(authUC))
			} else {
				logging.From(ctx).Warn("Authentication is not configured, Web UI will not work.")
			}

			httpServer := http.Server{
				Addr:              addr,
				Handler:           server.New(uc, serverOptions...),
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
