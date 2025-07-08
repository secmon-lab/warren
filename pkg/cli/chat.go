package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/tool/base"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/dryrun"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdChat() *cli.Command {
	var (
		ticketID    types.TicketID
		firestoreDB config.Firestore
		geminiCfg   config.GeminiCfg
		policyCfg   config.Policy
		storageCfg  config.Storage

		query    string
		noDryRun bool // Value for --no-dry-run flag
		dryRun   bool // Actual dry-run setting
	)

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
			&cli.BoolFlag{
				Name:        "no-dry-run",
				Usage:       "Disable dry-run mode (dry-run is enabled by default)",
				Destination: &noDryRun,
			},
		},
		firestoreDB.Flags(),
		geminiCfg.Flags(),
		policyCfg.Flags(),
		storageCfg.Flags(),
		tools.Flags(),
	)

	return &cli.Command{
		Name:    "chat",
		Aliases: []string{"c"},
		Usage:   "Chat with the security analyst about a specific ticket",
		Flags:   flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			logger := logging.From(ctx)

			// Set dry-run mode: enabled by default, disabled if --no-dry-run is specified
			dryRun = !noDryRun

			// Add dry-run information to context (enabled by default)
			ctx = dryrun.With(ctx, dryRun)
			if dryRun {
				logger.Info("Dry-run mode enabled - database modifications will be skipped")
			} else {
				logger.Info("Dry-run mode disabled - database modifications will be executed")
			}

			// Configure repository
			repo, err := firestoreDB.Configure(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure firestore")
			}

			// Configure LLM client
			llmClient, err := geminiCfg.Configure(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure gemini")
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

			// Configure tools
			baseAction := base.New(repo, policyClient, ticket.ID)
			toolSets := append(tools, baseAction)

			// Get tool sets
			allToolSets, err := toolSets.ToolSets(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to get tool sets")
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

			if dryRun {
				fmt.Printf("🔒 Dry-run mode: Database modifications will be simulated\n\n")
			}

			// Create usecase
			uc := usecase.New(
				usecase.WithRepository(repo),
				usecase.WithLLMClient(llmClient),
				usecase.WithPolicyClient(policyClient),
				usecase.WithStorageClient(storageClient),
				usecase.WithTools(allToolSets),
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
