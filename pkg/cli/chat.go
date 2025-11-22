package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	bqagent "github.com/secmon-lab/warren/pkg/agents/bigquery"
	slackagent "github.com/secmon-lab/warren/pkg/agents/slack"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/dryrun"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/urfave/cli/v3"
)

func cmdChat() *cli.Command {
	var (
		ticketID    types.TicketID
		firestoreDB config.Firestore
		llmCfg      config.LLMCfg
		policyCfg   config.Policy
		storageCfg  config.Storage
		mcpCfg      config.MCPConfig

		query string
	)

	bqAgent := bqagent.New()
	slackAgent := slackagent.New()

	flags := joinFlags(
		[]cli.Flag{
			&cli.StringFlag{
				Name:        "ticket-id",
				Aliases:     []string{"t"},
				Usage:       "Ticket ID to chat with",
				Destination: (*string)(&ticketID),
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "query",
				Aliases:     []string{"q"},
				Usage:       "Query prompt (if not provided, interactive mode will start)",
				Destination: &query,
			},
		},
		firestoreDB.Flags(),
		llmCfg.Flags(),
		policyCfg.Flags(),
		storageCfg.Flags(),
		tools.Flags(),
		mcpCfg.Flags(),
		bqAgent.Flags(),
		slackAgent.Flags(),
	)

	return &cli.Command{
		Name:    "chat",
		Aliases: []string{"c"},
		Usage:   "Chat with the security analyst about a specific ticket",
		Flags:   flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			// Configure repository
			repo, err := firestoreDB.Configure(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure firestore")
			}

			// Configure LLM client (automatically selects Claude if available, otherwise Gemini)
			llmClient, err := llmCfg.Configure(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure LLM client")
			}

			// Configure policy client
			policyClient, err := policyCfg.Configure()
			if err != nil {
				return goerr.Wrap(err, "failed to configure policy")
			}

			// Configure storage client
			storageClient, err := storageCfg.Configure(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure storage")
			}

			// Get the ticket
			ticket, err := repo.GetTicket(ctx, ticketID)
			if err != nil {
				return goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", ticketID))
			}
			if ticket == nil {
				return goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
			}

			// Get alerts bound to the ticket
			alerts, err := repo.BatchGetAlerts(ctx, ticket.AlertIDs)
			if err != nil {
				return goerr.Wrap(err, "failed to get alerts")
			}

			// Get tool sets
			allToolSets, err := tools.ToolSets(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to get tool sets")
			}

			// Add MCP tool sets if configured
			mcpToolSets, err := mcpCfg.CreateMCPToolSets(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to create MCP tool sets")
			}
			if len(mcpToolSets) > 0 {
				allToolSets = append(allToolSets, mcpToolSets...)
				logging.From(ctx).Info("MCP tool sets configured",
					"servers", mcpCfg.GetServerNames(),
					"count", len(mcpToolSets))
			}

			// Create memory service
			memoryService := memory.New(llmClient, repo)

			// Initialize BigQuery Agent
			if enabled, err := bqAgent.Init(ctx, llmClient, memoryService); err != nil {
				return err
			} else if enabled {
				allToolSets = append(allToolSets, bqAgent)
			}

			// Initialize Slack Search Agent
			if enabled, err := slackAgent.Init(ctx, llmClient, memoryService); err != nil {
				return err
			} else if enabled {
				allToolSets = append(allToolSets, slackAgent)
			}

			// Show ticket information
			fmt.Printf("\n🎫 Ticket Information:\n")
			fmt.Printf("  📝 ID: %s\n", ticket.ID)
			fmt.Printf("  📋 Title: %s\n", ticket.Title)
			fmt.Printf("  📄 Description: %s\n", ticket.Description)
			fmt.Printf("  📊 Status: %s\n", ticket.Status.Label())
			if ticket.Finding != nil {
				fmt.Printf("  🔍 Finding: %s (%s)\n", ticket.Finding.Summary, ticket.Finding.Severity)
			}
			fmt.Printf("  🔢 Alerts: %d\n", len(alerts))
			fmt.Printf("\n")

			// Create usecase
			uc := usecase.New(
				usecase.WithRepository(repo),
				usecase.WithLLMClient(llmClient),
				usecase.WithPolicyClient(policyClient),
				usecase.WithStorageClient(storageClient),
				usecase.WithTools(allToolSets),
				usecase.WithMemoryService(memoryService),
			)

			// If query is provided, run once and exit
			if query != "" {
				return runSingleQuery(ctx, uc, ticket, query)
			}

			// Otherwise, start interactive mode
			return runInteractiveMode(ctx, uc, ticket)
		},
	}
}

func runSingleQuery(ctx context.Context, uc *usecase.UseCases, ticket *ticket.Ticket, query string) error {
	logger := logging.From(ctx)
	logger.Info("Running single query", "query", query)

	// Setup message handlers for CLI output
	ctx = setupCLIMessageHandlers(ctx)

	if err := uc.Chat(ctx, ticket, query); err != nil {
		return goerr.Wrap(err, "failed to process query")
	}

	return nil
}

func runInteractiveMode(ctx context.Context, uc *usecase.UseCases, ticket *ticket.Ticket) error {
	logger := logging.From(ctx)
	logger.Info("Starting interactive chat mode")

	fmt.Println("💬 Interactive chat mode started. Type 'exit' or 'quit' to end the session.")
	fmt.Println("📝 Type your questions about the ticket and alerts.")
	fmt.Println("📋 Plan mode enabled: Complex tasks will be automatically broken down into steps.")
	if dryrun.IsDryRun(ctx) {
		fmt.Println("🔒 Dry-run mode: Commands that modify the database will be simulated.")
	}
	fmt.Println()

	// Setup message handlers for CLI output
	ctx = setupCLIMessageHandlers(ctx)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")

		input, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				fmt.Println("\n👋 Session ended.")
				break
			}
			return goerr.Wrap(err, "failed to read input")
		}

		message := strings.TrimSpace(string(input))
		if message == "" {
			continue
		}

		if message == "exit" || message == "quit" {
			fmt.Println("👋 Session ended.")
			break
		}

		if err := uc.Chat(ctx, ticket, message); err != nil {
			fmt.Printf("❌ Error: %s\n", err.Error())
			logger.Error("Chat error", "error", err)
		}

		fmt.Println()
	}

	return nil
}

// setupCLIMessageHandlers sets up proper message and trace handlers for CLI output
func setupCLIMessageHandlers(ctx context.Context) context.Context {
	// Handle regular messages - these are Warren's responses
	notifyFunc := func(ctx context.Context, message string) {
		fmt.Printf("%s\n", message)
	}

	// Handle trace messages - these are context blocks/updates
	traceFunc := func(ctx context.Context, message string) func(context.Context, string) {
		// For CLI, we want to show trace messages differently than regular messages
		// Trace messages are status updates, not final responses
		fmt.Printf("🔄 %s\n", message)

		return func(ctx context.Context, traceMsg string) {
			fmt.Printf("🔄 %s\n", traceMsg)
		}
	}

	return msg.With(ctx, notifyFunc, traceFunc)
}
