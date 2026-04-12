package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"

	gollemTrace "github.com/m-mizutani/gollem/trace"
	traceAdapter "github.com/secmon-lab/warren/pkg/adapter/trace"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/model/eval"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
	evalUC "github.com/secmon-lab/warren/pkg/usecase/eval"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

func cmdEval() *cli.Command {
	return &cli.Command{
		Name:    "eval",
		Aliases: []string{"e"},
		Usage:   "Evaluate agent behavior with scenarios",
		Commands: []*cli.Command{
			cmdEvalRun(),
			cmdEvalValidate(),
		},
	}
}

func cmdEvalValidate() *cli.Command {
	var scenarioDir string

	return &cli.Command{
		Name:    "validate",
		Aliases: []string{"v"},
		Usage:   "Validate a scenario directory without running it",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "scenario",
				Aliases:     []string{"s"},
				Usage:       "Path to scenario directory",
				Required:    true,
				Destination: &scenarioDir,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			errors := evalUC.ValidateScenarioDir(scenarioDir)
			if len(errors) == 0 {
				fmt.Println("✅ Scenario is valid")
				return nil
			}

			fmt.Printf("❌ Scenario validation failed (%d errors):\n", len(errors))
			for _, e := range errors {
				fmt.Printf("  - %s\n", e)
			}
			return goerr.New("scenario validation failed",
				goerr.V("scenario_dir", scenarioDir),
				goerr.V("error_count", len(errors)),
			)
		},
	}
}

func cmdEvalRun() *cli.Command {
	var (
		scenarioDir  string
		mode         string
		outputPath   string
		outputFormat string

		policyCfg           config.Policy
		llmCfg              config.LLMCfg
		storageCfg          config.Storage
		userSystemPromptCfg config.UserSystemPrompt
	)

	flags := joinFlags(
		[]cli.Flag{
			&cli.StringFlag{
				Name:        "scenario",
				Aliases:     []string{"s"},
				Usage:       "Path to scenario directory",
				Required:    true,
				Destination: &scenarioDir,
			},
			&cli.StringFlag{
				Name:        "mode",
				Aliases:     []string{"m"},
				Usage:       "Execution mode: cli (default) or slack",
				Value:       "cli",
				Destination: &mode,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "Output file path (default: stdout)",
				Destination: &outputPath,
			},
			&cli.StringFlag{
				Name:        "output-format",
				Aliases:     []string{"f"},
				Usage:       "Output format: markdown (default) or json",
				Value:       "markdown",
				Destination: &outputFormat,
			},
		},
		policyCfg.Flags(),
		llmCfg.Flags(),
		tools.Flags(),
		storageCfg.Flags(),
		userSystemPromptCfg.Flags(),
	)

	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Run an evaluation scenario",
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logger := logging.From(ctx)

			// Validate scenario first
			validationErrors := evalUC.ValidateScenarioDir(scenarioDir)
			if len(validationErrors) > 0 {
				fmt.Printf("❌ Scenario validation failed (%d errors):\n", len(validationErrors))
				for _, e := range validationErrors {
					fmt.Printf("  - %s\n", e)
				}
				return goerr.New("scenario validation failed, cannot run eval")
			}

			logger.Info("eval: starting",
				"scenario", scenarioDir,
				"output", outputPath,
				"format", outputFormat,
			)

			// Load scenario
			scenario, err := evalUC.LoadScenario(scenarioDir)
			if err != nil {
				return err
			}
			logger.Info("eval: scenario loaded", "name", scenario.Name)

			// Configure policy
			policyClient, err := policyCfg.Configure()
			if err != nil {
				return goerr.Wrap(err, "failed to configure policy")
			}

			// Configure LLM (used for both Warren agent and mock/eval)
			llmClient, err := llmCfg.Configure(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure LLM")
			}

			// Configure embedding
			embeddingClient, err := llmCfg.ConfigureEmbeddingClient(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure embedding client")
			}

			// Use memory repository (no Firestore for eval)
			repo := repository.NewMemory()

			// Configure tools
			tools.InjectDependencies(repo, embeddingClient)
			toolSets, err := tools.ToolSets(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure tools")
			}

			// Configure user system prompt
			userSystemPrompt, err := userSystemPromptCfg.Configure()
			if err != nil {
				return goerr.Wrap(err, "failed to configure user system prompt")
			}

			// Configure gollem trace repository — save to scenario's traces/ dir
			tracesDir := filepath.Join(scenarioDir, "traces")
			fileTraceRepo := gollemTrace.NewFileRepository(tracesDir)
			safeTraceRepo := traceAdapter.NewSafe(fileTraceRepo, logging.From(ctx))
			logger.Info("eval: trace recording enabled", "traces_dir", tracesDir)

			// Build usecase options (tools will be wrapped by Runner)
			ucOpts := []usecase.Option{
				usecase.WithLLMClient(llmClient),
				usecase.WithPolicyClient(policyClient),
				usecase.WithRepository(repo),
				usecase.WithNoAuthorization(true),
				usecase.WithUserSystemPrompt(userSystemPrompt),
				usecase.WithTraceRepository(safeTraceRepo),
			}

			// Create and run eval
			runner, err := evalUC.NewRunner(
				ctx,
				scenario,
				scenarioDir,
				llmClient,
				ucOpts,
				toolSets,
				evalUC.WithLLMClientForEval(llmClient),
			)
			if err != nil {
				return goerr.Wrap(err, "failed to create eval runner")
			}

			var evalResult *eval.EvalResult
			switch mode {
			case "cli":
				evalResult, err = runner.RunCLI(ctx)
				if err != nil {
					return goerr.Wrap(err, "eval CLI run failed")
				}
			case "slack":
				_, slackErr := runner.RunSlack(ctx)
				return slackErr
			default:
				return goerr.New("unknown eval mode", goerr.V("mode", mode))
			}

			// Determine report format
			format := eval.ReportFormatMarkdown
			if outputFormat == "json" {
				format = eval.ReportFormatJSON
			}

			// Generate report
			report, err := evalUC.GenerateReport(ctx, evalResult, format, llmClient)
			if err != nil {
				return goerr.Wrap(err, "failed to generate report")
			}

			// Write output
			var w io.Writer = os.Stdout
			if outputPath != "" {
				f, err := os.Create(outputPath) // #nosec G304 -- path from CLI flag
				if err != nil {
					return goerr.Wrap(err, "failed to create output file",
						goerr.V("path", outputPath))
				}
				defer safe.Close(ctx, f)
				w = f
			}

			if _, err := fmt.Fprint(w, report.Content); err != nil {
				return goerr.Wrap(err, "failed to write report")
			}

			logger.Info("eval: completed",
				"scenario", scenario.Name,
				"tool_calls", len(evalResult.Trace.ToolCalls),
			)

			return nil
		},
	}
}
