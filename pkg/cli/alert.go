package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/notifier"
	"github.com/secmon-lab/warren/pkg/service/prompt"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

func cmdAlert() *cli.Command {
	var (
		policyCfg config.Policy
		genaiCfg  config.GenAI
		llmCfg    config.LLMCfg
		schema    string
		inputFile string
	)

	flags := joinFlags(
		[]cli.Flag{
			&cli.StringFlag{
				Name:        "schema",
				Aliases:     []string{"s"},
				Usage:       "Alert schema ID",
				Required:    true,
				Destination: &schema,
			},
			&cli.StringFlag{
				Name:        "input",
				Aliases:     []string{"i"},
				Usage:       "Input file path (default: stdin)",
				Destination: &inputFile,
			},
		},
		policyCfg.Flags(),
		genaiCfg.Flags(),
		llmCfg.Flags(),
	)

	return &cli.Command{
		Name:  "alert",
		Usage: "Process alert through policy pipeline (dry-run mode)",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			return runAlertPipeline(ctx, &policyCfg, &genaiCfg, &llmCfg, schema, inputFile)
		},
	}
}

func runAlertPipeline(ctx context.Context, policyCfg *config.Policy, genaiCfg *config.GenAI, llmCfg *config.LLMCfg, schemaStr string, inputFile string) error {
	// Read alert data from file or stdin
	alertData, err := readAlertData(inputFile)
	if err != nil {
		return goerr.Wrap(err, "failed to read alert data")
	}

	schema := types.AlertSchema(schemaStr)

	// Setup policy client
	policyClient, err := policyCfg.Configure()
	if err != nil {
		return goerr.Wrap(err, "failed to create policy client")
	}

	// Setup LLM client
	llmClient, err := llmCfg.Configure(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to create LLM client")
	}

	// Setup prompt service (if configured)
	var promptService interfaces.PromptService
	if genaiCfg.IsConfigured() {
		promptService, err = prompt.New(genaiCfg.GetPromptDir())
		if err != nil {
			return goerr.Wrap(err, "failed to create prompt service",
				goerr.V("prompt_dir", genaiCfg.GetPromptDir()))
		}
	}

	// Create use case
	uc := usecase.New(
		usecase.WithPolicyClient(policyClient),
		usecase.WithLLMClient(llmClient),
		usecase.WithPromptService(promptService),
	)

	// Console notifier for CLI mode - outputs to stderr for progress, stdout for results
	eventNotifier := notifier.NewConsoleNotifier()

	// Process pipeline
	results, err := uc.ProcessAlertPipeline(ctx, schema, alertData, eventNotifier)
	if err != nil {
		return goerr.Wrap(err, "failed to process alert pipeline")
	}

	// Display results to stdout
	return displayPipelineResult(results)
}

func readAlertData(inputFile string) (any, error) {
	var reader io.Reader

	if inputFile != "" {
		// Read from file
		// #nosec G304 -- This is a CLI tool that intentionally reads user-specified files
		file, err := os.Open(inputFile)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to open input file")
		}
		defer safe.Close(context.Background(), file)
		reader = file
	} else {
		// Read from stdin
		reader = os.Stdin
	}

	// Parse JSON
	var alertData any
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&alertData); err != nil {
		return nil, goerr.Wrap(err, "failed to decode alert data")
	}

	return alertData, nil
}

func displayPipelineResult(results []*usecase.AlertPipelineResult) error {
	if len(results) == 0 {
		fmt.Println("No alerts generated")
		return nil
	}

	for i, r := range results {
		if i > 0 {
			fmt.Println("\n" + "═══════════════════════════════════════════════════════════════════════════")
			fmt.Println()
		}

		// Alert section
		fmt.Println("📋 ALERT")
		fmt.Println("─────────────────────────────────────────────────────────────────────────")
		fmt.Printf("ID:          %s\n", r.Alert.ID)
		fmt.Printf("Schema:      %s\n", r.Alert.Schema)
		fmt.Printf("Title:       %s\n", r.Alert.Title)
		fmt.Printf("Description: %s\n", r.Alert.Description)
		fmt.Printf("Channel:     %s\n", r.Alert.Channel)

		if len(r.Alert.Attributes) > 0 {
			fmt.Println("\nAttributes:")
			for _, attr := range r.Alert.Attributes {
				autoFlag := ""
				if attr.Auto {
					autoFlag = " (auto)"
				}
				fmt.Printf("  • %s: %s%s\n", attr.Key, attr.Value, autoFlag)
			}
		}

		// Commit result section
		fmt.Println("\n✅ COMMIT RESULT")
		fmt.Println("─────────────────────────────────────────────────────────────────────────")
		fmt.Printf("Publish:     %s\n", r.CommitResult.Publish)
		fmt.Printf("Title:       %s\n", r.CommitResult.Title)
		if r.CommitResult.Description != "" {
			fmt.Printf("Description: %s\n", r.CommitResult.Description)
		}
		if r.CommitResult.Channel != "" {
			fmt.Printf("Channel:     %s\n", r.CommitResult.Channel)
		}

		if len(r.CommitResult.Attr) > 0 {
			fmt.Println("\nAdditional Attributes:")
			for _, attr := range r.CommitResult.Attr {
				autoFlag := ""
				if attr.Auto {
					autoFlag = " (auto)"
				}
				fmt.Printf("  • %s: %s%s\n", attr.Key, attr.Value, autoFlag)
			}
		}
	}

	return nil
}
