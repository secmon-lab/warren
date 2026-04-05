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
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"

	"github.com/secmon-lab/warren/pkg/adapter/storage"
	traceAdapter "github.com/secmon-lab/warren/pkg/adapter/trace"
	"github.com/secmon-lab/warren/pkg/agents"
	"github.com/secmon-lab/warren/pkg/cli/config"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	websocket_controller "github.com/secmon-lab/warren/pkg/controller/websocket"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/circuitbreaker"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/service/prompt"
	"github.com/secmon-lab/warren/pkg/service/tag"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/usecase/chat/aster"
	"github.com/secmon-lab/warren/pkg/usecase/chat/bluebell"
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
		addr                 string
		enableGraphQL        bool
		enableGraphiQL       bool
		noAuthorization      bool
		disableHTTPLogger    bool
		disableLLM           bool
		strictAlert          bool
		wsAllowedOrigins     []string
		webUICfg             config.WebUI
		policyCfg            config.Policy
		genaiCfg             config.GenAI
		sentryCfg            config.Sentry
		slackCfg             config.Slack
		llmCfg               config.LLMCfg
		firestoreCfg         config.Firestore
		storageCfg           config.Storage
		mcpCfg               config.MCPConfig
		asyncCfg             config.AsyncAlertHook
		traceCfg             config.Trace
		userSystemPromptCfg  config.UserSystemPrompt
		strategyPromptsCfg config.StrategySystemPrompts
		cbCfg                config.CircuitBreaker
		chatStrategy         string
		budgetStrategy       string
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
				Name:        "disable-http-logger",
				Usage:       "Disable HTTP access logging (both access log and detailed access log)",
				Category:    "Logging",
				Sources:     cli.EnvVars("WARREN_DISABLE_HTTP_LOGGER"),
				Destination: &disableHTTPLogger,
			},
			&cli.BoolFlag{
				Name:        "disable-llm",
				Usage:       "Disable LLM initialization and use no-op clients (for E2E testing)",
				Category:    "LLM",
				Sources:     cli.EnvVars("WARREN_DISABLE_LLM"),
				Destination: &disableLLM,
			},
			&cli.BoolFlag{
				Name:        "strict-alert",
				Usage:       "Reject alerts without corresponding policy package",
				Category:    "Policy",
				Sources:     cli.EnvVars("WARREN_STRICT_ALERT"),
				Destination: &strictAlert,
				Value:       false,
			},
			&cli.StringSliceFlag{
				Name:        "ws-allowed-origins",
				Usage:       "Additional allowed origins for WebSocket connections (e.g., http://localhost:5173)",
				Category:    "WebSocket",
				Sources:     cli.EnvVars("WARREN_WS_ALLOWED_ORIGINS"),
				Destination: &wsAllowedOrigins,
			},
			&cli.StringFlag{
				Name:        "chat-strategy",
				Usage:       "Chat execution strategy (default: 'aster')",
				Category:    "Chat",
				Sources:     cli.EnvVars("WARREN_CHAT_STRATEGY"),
				Value:       "aster",
				Destination: &chatStrategy,
			},
			&cli.StringFlag{
				Name:        "budget-strategy",
				Usage:       "Budget strategy for task execution: 'none' (unlimited) or 'default' (action budget)",
				Category:    "Chat",
				Sources:     cli.EnvVars("WARREN_BUDGET_STRATEGY"),
				Value:       "none",
				Destination: &budgetStrategy,
			},
		},
		webUICfg.Flags(),
		policyCfg.Flags(),
		genaiCfg.Flags(),
		sentryCfg.Flags(),
		slackCfg.Flags(),
		llmCfg.Flags(),
		firestoreCfg.Flags(),
		tools.Flags(),
		storageCfg.Flags(),
		mcpCfg.Flags(),
		asyncCfg.Flags(),
		agents.AllFlags(),
		traceCfg.Flags(),
		userSystemPromptCfg.Flags(),
		strategyPromptsCfg.Flags(),
		cbCfg.Flags(),
	)

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "Run server",
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Parse async configuration
			if err := asyncCfg.Parse(cmd); err != nil {
				return err
			}

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
				"genai", genaiCfg,
				"sentry", sentryCfg,
				"slack", slackCfg,
				"llm", llmCfg,
				"firestore", firestoreCfg,
				"storage", storageCfg,
				"mcp", mcpCfg,
				"async", asyncCfg,
			)

			policyClient, err := policyCfg.Configure()
			if err != nil {
				return err
			}

			// Validate strict-alert with policy configuration
			if !policyCfg.HasPolicies() && strictAlert {
				return goerr.New("--strict-alert requires at least one policy file to be specified")
			}

			// Configure LLM client
			var llmClient gollem.LLMClient
			if disableLLM {
				logging.From(ctx).Warn("⚠️  LLM is disabled, using no-op client",
					"recommendation", "Only use --disable-llm for E2E testing")
				llmClient = config.NewNoopLLMClient()
			} else {
				var err error
				llmClient, err = llmCfg.Configure(ctx)
				if err != nil {
					return err
				}
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

			// Configure repository with fallback
			var repo interfaces.Repository
			if !firestoreCfg.IsConfigured() {
				logging.From(ctx).Warn("⚠️  Using in-memory repository (Firestore not configured)",
					"recommendation", "For production, configure Firestore with --firestore-project-id")
				repo = repository.NewMemory()
			} else {
				firestore, err := firestoreCfg.Configure(ctx)
				if err != nil {
					return err
				}
				repo = firestore
			}

			// Configure storage with fallback
			var storageClient interfaces.StorageClient
			if !storageCfg.IsConfigured() {
				logging.From(ctx).Warn("⚠️  Using in-memory storage (Cloud Storage not configured)",
					"recommendation", "For production, configure Cloud Storage with --storage-bucket")
				storageClient = storage.NewMemoryClient()
			} else {
				client, err := storageCfg.Configure(ctx)
				if err != nil {
					return err
				}
				storageClient = client
			}

			// Create embedding client
			var embeddingAdapter interfaces.EmbeddingClient
			if disableLLM {
				embeddingAdapter = config.NewNoopEmbeddingClient()
			} else {
				var err error
				embeddingAdapter, err = llmCfg.ConfigureEmbeddingClient(ctx)
				if err != nil {
					return err
				}
			}

			// Inject dependencies into tools that support them
			tools.InjectDependencies(repo, embeddingAdapter)

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
				logging.From(ctx).Info("MCP tool sets configured",
					"servers", mcpCfg.GetServerNames(),
					"count", len(mcpToolSets))
			}

			// Create WebSocket hub and handler
			wsHub := websocket_controller.NewHub(ctx)
			go wsHub.Run() // Start the hub in a goroutine

			// Create tag service
			tagService := tag.New(repo)

			// Initialize all configured agents and merge into tool sets
			agentToolSets, err := agents.ConfigureAll(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure agents")
			}
			toolSets = append(toolSets, agentToolSets...)

			// Configure user system prompt
			userSystemPrompt, err := userSystemPromptCfg.Configure()
			if err != nil {
				return goerr.Wrap(err, "failed to configure user system prompt")
			}

			// Configure trace repository if trace bucket is set
			var safeTraceRepo trace.Repository
			if traceCfg.IsConfigured() {
				traceRepo, err := traceCfg.Configure(ctx)
				if err != nil {
					return goerr.Wrap(err, "failed to configure trace repository")
				}
				safeTraceRepo = traceAdapter.NewSafe(traceRepo, logging.From(ctx))
				logging.From(ctx).Info("Trace recording enabled", "trace", traceCfg.LogValue())
			}

			// Create knowledge service for reflection and GraphQL
			var knowledgeOpts []svcknowledge.ServiceOption
			if safeTraceRepo != nil {
				knowledgeOpts = append(knowledgeOpts, svcknowledge.WithTraceRepository(safeTraceRepo))
			}
			knowledgeSvc := svcknowledge.New(repo, embeddingAdapter, knowledgeOpts...)

			ucOptions := []usecase.Option{
				usecase.WithLLMClient(llmClient),
				usecase.WithPolicyClient(policyClient),
				usecase.WithRepository(repo),
				usecase.WithStorageClient(storageClient),
				usecase.WithTools(toolSets),
				usecase.WithStrictAlert(strictAlert),
				usecase.WithNoAuthorization(noAuthorization),
				usecase.WithTagService(tagService),
				usecase.WithUserSystemPrompt(userSystemPrompt),
				usecase.WithKnowledgeService(knowledgeSvc),
			}

			if safeTraceRepo != nil {
				ucOptions = append(ucOptions, usecase.WithTraceRepository(safeTraceRepo))
			}

			if cbCfg.Enabled {
				cbSvc := circuitbreaker.New(repo, cbCfg.ToConfig())
				ucOptions = append(ucOptions, usecase.WithCircuitBreaker(cbSvc))
				logging.From(ctx).Info("Circuit breaker enabled",
					"window", cbCfg.Window,
					"limit", cbCfg.Limit)
			}

			// Add storage prefix if configured
			if storageCfg.IsConfigured() && storageCfg.Prefix() != "" {
				ucOptions = append(ucOptions, usecase.WithStoragePrefix(storageCfg.Prefix()))
			}

			// Add GenAI configuration if configured
			if genaiCfg.IsConfigured() {
				promptService, err := prompt.New(genaiCfg.GetPromptDir())
				if err != nil {
					return goerr.Wrap(err, "failed to create prompt service")
				}
				ucOptions = append(ucOptions, usecase.WithPromptService(promptService))
			}

			// Add Slack service if available
			if slackSvc != nil {
				ucOptions = append(ucOptions, usecase.WithSlackService(slackSvc))
			}

			// Add frontend URL if configured
			if webUICfg.GetFrontendURL() != "" {
				ucOptions = append(ucOptions, usecase.WithFrontendURL(webUICfg.GetFrontendURL()))
			}

			// Configure chat strategy
			switch chatStrategy {
			case "aster":
				// Merge all tool sets: configured tools + MCP tools
				allTools := make([]interfaces.ToolSet, 0, len(toolSets)+len(mcpToolSets))
				allTools = append(allTools, toolSets...)
				for _, mts := range mcpToolSets {
					allTools = append(allTools, interfaces.WrapToolSet(mts, "mcp", "MCP external tool"))
				}

				asterOpts := []aster.Option{
					aster.WithTools(allTools),
					aster.WithStorageClient(storageClient),
					aster.WithNoAuthorization(noAuthorization),
					aster.WithUserSystemPrompt(userSystemPrompt),
				}
				if slackSvc != nil {
					asterOpts = append(asterOpts, aster.WithSlackService(slackSvc))
				}
				if webUICfg.GetFrontendURL() != "" {
					asterOpts = append(asterOpts, aster.WithFrontendURL(webUICfg.GetFrontendURL()))
				}
				if storageCfg.IsConfigured() && storageCfg.Prefix() != "" {
					asterOpts = append(asterOpts, aster.WithStoragePrefix(storageCfg.Prefix()))
				}
				if safeTraceRepo != nil {
					asterOpts = append(asterOpts, aster.WithTraceRepository(safeTraceRepo))
				}

				asterOpts = append(asterOpts, aster.WithKnowledgeService(knowledgeSvc))

				// Configure budget strategy
				switch budgetStrategy {
				case "default":
					asterOpts = append(asterOpts, aster.WithBudgetStrategy(aster.NewDefaultBudgetStrategy()))
					logging.From(ctx).Info("Budget strategy: default (action budget enabled)")
				case "none":
					// budgetStrategy remains nil (budget disabled)
				default:
					return goerr.New("unknown budget strategy", goerr.V("strategy", budgetStrategy))
				}

				// Configure HITL tools that require human approval
				asterOpts = append(asterOpts, aster.WithHITLTools([]string{"web_fetch"}))

				asterChat := aster.New(repo, llmClient, policyClient, asterOpts...)
				ucOptions = append(ucOptions, usecase.WithChatUseCase(asterChat))
				logging.From(ctx).Info("Chat strategy: aster")

			case "bluebell":
				if knowledgeSvc == nil {
					return goerr.New("bluebell strategy requires knowledge service to be configured")
				}

				// Load user-defined prompt entries
				promptEntries, err := strategyPromptsCfg.Configure()
				if err != nil {
					return goerr.Wrap(err, "failed to configure user system prompts")
				}

				allTools := make([]interfaces.ToolSet, 0, len(toolSets)+len(mcpToolSets))
				allTools = append(allTools, toolSets...)
				for _, mts := range mcpToolSets {
					allTools = append(allTools, interfaces.WrapToolSet(mts, "mcp", "MCP external tool"))
				}

				bluebellOpts := []bluebell.Option{
					bluebell.WithTools(allTools),
					bluebell.WithStorageClient(storageClient),
					bluebell.WithNoAuthorization(noAuthorization),
					bluebell.WithUserSystemPrompt(userSystemPrompt),
					bluebell.WithKnowledgeService(knowledgeSvc),
					bluebell.WithPromptEntries(promptEntries),
				}
				if slackSvc != nil {
					bluebellOpts = append(bluebellOpts, bluebell.WithSlackService(slackSvc))
				}
				if webUICfg.GetFrontendURL() != "" {
					bluebellOpts = append(bluebellOpts, bluebell.WithFrontendURL(webUICfg.GetFrontendURL()))
				}
				if storageCfg.IsConfigured() && storageCfg.Prefix() != "" {
					bluebellOpts = append(bluebellOpts, bluebell.WithStoragePrefix(storageCfg.Prefix()))
				}
				if safeTraceRepo != nil {
					bluebellOpts = append(bluebellOpts, bluebell.WithTraceRepository(safeTraceRepo))
				}

				switch budgetStrategy {
				case "default":
					bluebellOpts = append(bluebellOpts, bluebell.WithBudgetStrategy(bluebell.NewDefaultBudgetStrategy()))
					logging.From(ctx).Info("Budget strategy: default (action budget enabled)")
				case "none":
					// budgetStrategy remains nil (budget disabled)
				default:
					return goerr.New("unknown budget strategy", goerr.V("strategy", budgetStrategy))
				}

				bluebellOpts = append(bluebellOpts, bluebell.WithHITLTools([]string{"web_fetch"}))

				bluebellChat, err := bluebell.New(repo, llmClient, policyClient, bluebellOpts...)
				if err != nil {
					return goerr.Wrap(err, "failed to create bluebell chat strategy")
				}
				ucOptions = append(ucOptions, usecase.WithChatUseCase(bluebellChat))
				logging.From(ctx).Info("Chat strategy: bluebell",
					"prompt_entries", len(promptEntries))

			default:
				return goerr.New("unknown chat strategy", goerr.V("strategy", chatStrategy))
			}

			uc := usecase.New(ucOptions...)

			// Build HTTP server options
			serverOptions := []server.Options{
				server.WithPolicy(policyClient),
				server.WithKnowledgeService(knowledgeSvc),
				server.WithDisableHTTPLogger(disableHTTPLogger),
			}

			if disableHTTPLogger {
				logging.From(ctx).Warn("HTTP access logging is DISABLED",
					"flag", "--disable-http-logger")
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
				serverOptions = append(serverOptions, server.WithGraphQLRepo(repo))
			}

			// Add GraphiQL option when GraphiQL is enabled
			if enableGraphiQL {
				serverOptions = append(serverOptions, server.WithGraphiQL(true))
				if !enableGraphQL {
					logging.From(ctx).Warn("GraphiQL is enabled but GraphQL is not enabled. GraphiQL will not work.")
				}
			}

			// Add AuthUseCase if authentication options are provided
			authUC, err := webUICfg.Configure(ctx, repo, slackSvc)
			if err != nil {
				return err
			}
			if authUC != nil {
				serverOptions = append(serverOptions, server.WithAuthUseCase(authUC))
			} else {
				// Authentication is required for WebUI
				return goerr.New("WebUI requires authentication configuration. Please set either --slack-client-id/--slack-client-secret or --no-authentication flag")
			}

			// Create and add WebSocket handler with frontend URL for origin checking
			wsHandler := websocket_controller.NewHandler(wsHub, repo, uc)
			if webUICfg.GetFrontendURL() != "" {
				wsHandler = wsHandler.WithFrontendURL(webUICfg.GetFrontendURL())
			}

			// Add explicitly configured allowed origins for WebSocket
			additionalOrigins := append([]string{}, wsAllowedOrigins...)

			// If frontend URL is 127.0.0.1, also allow localhost (and vice versa)
			frontendURL := webUICfg.GetFrontendURL()
			if strings.Contains(frontendURL, "://127.0.0.1:") {
				localhostURL := strings.Replace(frontendURL, "://127.0.0.1:", "://localhost:", 1)
				additionalOrigins = append(additionalOrigins, localhostURL)
			} else if strings.Contains(frontendURL, "://localhost:") {
				loopbackURL := strings.Replace(frontendURL, "://localhost:", "://127.0.0.1:", 1)
				additionalOrigins = append(additionalOrigins, loopbackURL)
			}

			if len(additionalOrigins) > 0 {
				wsHandler = wsHandler.WithAllowedOrigins(additionalOrigins)
				logging.From(ctx).Info("WebSocket: Configured additional allowed origins",
					"origins", additionalOrigins)
			}

			serverOptions = append(serverOptions, server.WithWebSocketHandler(wsHandler))

			// Add async configuration to server options
			serverOptions = append(serverOptions, server.WithAsyncAlertHook(&server.AsyncAlertHookConfig{
				Raw:    asyncCfg.Raw,
				PubSub: asyncCfg.PubSub,
				SNS:    asyncCfg.SNS,
			}))

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
