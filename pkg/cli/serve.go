package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	websocket_controller "github.com/secmon-lab/warren/pkg/controller/websocket"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/chat"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

// generateFrontendURL generates a frontend URL from the server address
func generateFrontendURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// This could be an address without a port, or just a port like ":8080".
		if strings.HasPrefix(addr, ":") {
			// Port only format (e.g., ":8080")
			return fmt.Sprintf("http://localhost%s", addr)
		}
		// For other malformed addresses, just prepend http://
		return fmt.Sprintf("http://%s", addr)
	}

	// If host is empty (e.g. from ":8080"), "0.0.0.0", or "::" (unspecified IPv6), replace with localhost.
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	return fmt.Sprintf("http://%s", net.JoinHostPort(host, port))
}

func cmdServe() *cli.Command {
	var (
		addr            string
		enableGraphQL   bool
		enableGraphiQL  bool
		noAuthorization bool
		strictAlert     bool
		webUICfg        config.WebUI
		policyCfg       config.Policy
		sentryCfg       config.Sentry
		slackCfg        config.Slack
		llmCfg          config.LLMCfg
		firestoreCfg    config.Firestore
		storageCfg      config.Storage
		mcpCfg          config.MCPConfig
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
				Value:       true,
				Destination: &enableGraphQL,
			},
			&cli.BoolFlag{
				Name:        "enable-graphiql",
				Usage:       "Enable GraphiQL playground (requires --enable-graphql)",
				Category:    "GraphQL",
				Sources:     cli.EnvVars("WARREN_ENABLE_GRAPHIQL"),
				Destination: &enableGraphiQL,
			},
			&cli.BoolFlag{
				Name:        "no-authorization",
				Aliases:     []string{"no-authz"},
				Usage:       "Disable policy-based authorization checks (development only)",
				Category:    "Security",
				Sources:     cli.EnvVars("WARREN_NO_AUTHORIZATION"),
				Destination: &noAuthorization,
			},
			&cli.BoolFlag{
				Name:        "strict-alert",
				Usage:       "Reject alerts without corresponding policy package",
				Category:    "Policy",
				Sources:     cli.EnvVars("WARREN_STRICT_ALERT"),
				Destination: &strictAlert,
				Value:       false,
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
			// Auto-generate frontend URL if not set
			if webUICfg.GetFrontendURL() == "" {
				generatedURL := generateFrontendURL(addr)
				webUICfg.SetFrontendURL(generatedURL)
				logging.Default().Warn("⚠️  Frontend URL is automatically set",
					"auto-generated-url", generatedURL,
					"recommendation", "For production use, please explicitly set --frontend-url")
			}

			logging.Default().Info("starting server",
				"addr", addr,
				"enableGraphQL", enableGraphQL,
				"enableGraphiQL", enableGraphiQL,
				"noAuthorization", noAuthorization,
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

			// Validate strict-alert with policy configuration
			if !policyCfg.HasPolicies() && strictAlert {
				return goerr.New("--strict-alert requires at least one policy file to be specified")
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
			var slackNotifier interfaces.SlackNotifier
			if slackSvc != nil {
				slackNotifier = slackSvc // slack.Service implements interfaces.SlackNotifier directly
			} else {
				slackNotifier = usecase.NewDiscardSlackNotifier()
			}

			// Create WebSocket hub and handler
			wsHub := websocket_controller.NewHub(ctx)
			go wsHub.Run() // Start the hub in a goroutine

			// Create ChatNotifier based on configuration
			var chatNotifier interfaces.ChatNotifier
			if slackSvc != nil {
				// Create a MultiNotifier that sends to both Slack and WebSocket
				slackChatNotifier := chat.NewSlackNotifier()
				multiNotifier := chat.NewMultiNotifier(
					slackChatNotifier,
					chat.NewWebSocketNotifier(wsHub),
				)
				chatNotifier = multiNotifier
			} else {
				// If no Slack, use WebSocket only
				chatNotifier = chat.NewWebSocketNotifier(wsHub)
			}

			ucOptions := []usecase.Option{
				usecase.WithLLMClient(llmClient),
				usecase.WithPolicyClient(policyClient),
				usecase.WithRepository(firestore),
				usecase.WithSlackNotifier(slackNotifier),
				usecase.WithStorageClient(storageClient),
				usecase.WithTools(toolSets),
				usecase.WithStrictAlert(strictAlert),
				usecase.WithChatNotifier(chatNotifier),
			}

			uc := usecase.New(ucOptions...)

			// Build HTTP server options
			serverOptions := []server.Options{
				server.WithPolicy(policyClient),
			}

			// Add no-authorization option if specified
			if noAuthorization {
				logging.From(ctx).Warn("⚠️  SECURITY WARNING: Authorization checks are DISABLED",
					"flag", "--no-authorization",
					"recommendation", "This should only be used in development environments")
				serverOptions = append(serverOptions, server.WithNoAuthorization(true))
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
			authUC, err := webUICfg.Configure(ctx, firestore, slackSvc)
			if err != nil {
				return err
			}
			if authUC != nil {
				serverOptions = append(serverOptions, server.WithAuthUseCase(authUC))
			} else {
				// Authentication is required for WebUI
				return goerr.New("WebUI requires authentication configuration. Please set either --slack-client-id/--slack-client-secret or --no-authentication flag")
			}

			// Create and add WebSocket handler
			wsHandler := websocket_controller.NewHandler(wsHub, firestore, uc)
			serverOptions = append(serverOptions, server.WithWebSocketHandler(wsHandler))

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
				// Close WebSocket hub
				if err := wsHub.Close(); err != nil {
					logging.From(ctx).Error("failed to close WebSocket hub", "error", err)
				}

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return httpServer.Shutdown(ctx)
			}
		},
	}
}
