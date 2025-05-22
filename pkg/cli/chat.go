package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func cmdChat() *cli.Command {
	var (
		alertID     types.AlertID
		alertListID types.AlertListID
		firestoreDB config.Firestore
		geminiCfg   config.GeminiCfg
		policyCfg   config.Policy
		storageCfg  config.Storage
		query       string
	)

	flags := joinFlags(
		[]cli.Flag{
			&cli.StringFlag{
				Name:        "alert-id",
				Aliases:     []string{"a"},
				Usage:       "Alert ID",
				Destination: (*string)(&alertID),
			},
			&cli.StringFlag{
				Name:        "alert-list-id",
				Aliases:     []string{"l"},
				Usage:       "Alert List ID",
				Destination: (*string)(&alertListID),
			},
			&cli.StringFlag{
				Name:        "query",
				Aliases:     []string{"q"},
				Usage:       "Query",
				Destination: (*string)(&query),
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
		Usage:   "Chat with the security analyst",
		Flags:   flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			return goerr.New("not implemented for now")
			/*
					logger := logging.From(ctx)

					repo, err := firestoreDB.Configure(ctx)
					if err != nil {
						return goerr.Wrap(err, "failed to configure firestore")
					}

					llmClient, err := geminiCfg.Configure(ctx)
					if err != nil {
						return goerr.Wrap(err, "failed to configure gemini")
					}

					if (alertID == "") == (alertListID == "") {
						return goerr.New("either alert-id or alert-list-id must be provided")
					}

					var alertIDs []types.AlertID
					var alertRecord *alert.Alert
					var alertList *alert.List
					if alertID != "" {
						alertRecord, err = repo.GetAlert(ctx, alertID)
						if err != nil {
							return goerr.Wrap(err, "failed to get alert")
						}
						alertIDs = []types.AlertID{alertID}
					}
					if alertListID != "" {
						list, err := repo.GetAlertList(ctx, alertListID)
						if err != nil {
							return goerr.Wrap(err, "failed to get alert list")
						}
						alertList = list
						alertIDs = list.AlertIDs
					}
					if len(alertIDs) == 0 {
						return goerr.New("no alert provided")
					}

					policyClient, err := policyCfg.Configure()
					if err != nil {
						return goerr.Wrap(err, "failed to configure policy")
					}

					ssn := session.New(ctx, nil, alertIDs)

					tools = append(tools, base.New(repo, alertIDs, policyClient.Sources(), ssn.ID))
					var toolNames []string
					for _, tool := range tools {
						specs, err := tool.Specs(ctx)
						if err != nil {
							return goerr.Wrap(err, "failed to get tool specs")
						}
						for _, spec := range specs {
							toolNames = append(toolNames, spec.Name)
						}
					}
					logger.Info("Enabled tools", "tools", toolNames)
					logger.Debug("Enabled tool config", "config", tools)

					fmt.Printf("\n")
					if alertRecord != nil {
						fmt.Printf("🔔 Alert Information:\n")
						fmt.Printf("  📝 Title: %s\n", alertRecord.Title)
						fmt.Printf("  📋 Description: %s\n", alertRecord.Description)
						fmt.Printf("  🔍 Attributes:\n")
						for _, attr := range alertRecord.Attributes {
							fmt.Printf("    - %s: %v\n", attr.Key, attr.Value)
						}
					}
					if alertList != nil {
						fmt.Printf("📋 Alert List Information:\n")
						fmt.Printf("  📝 Title: %s\n", alertList.Title)
						fmt.Printf("  📋 Description: %s\n", alertList.Description)
						fmt.Printf("  🔢 Number of Alerts: %d\n", len(alertList.AlertIDs))
					}
					fmt.Printf("\n")

					alerts, err := repo.BatchGetAlerts(ctx, alertIDs)
					if err != nil {
						return goerr.Wrap(err, "failed to get alerts")
					}

					systemPrompt, err := prompt.BuildSessionInitPrompt(ctx, alerts)
					if err != nil {
						return goerr.Wrap(err, "failed to build system prompt")
					}

					toolSets, err := tools.ToolSets(ctx)
					if err != nil {
						return goerr.Wrap(err, "failed to get tool sets")
					}

					agent := gollem.New(llmClient,
						gollem.WithToolSets(toolSets...),
						gollem.WithResponseMode(gollem.ResponseModeStreaming),
						gollem.WithSystemPrompt(systemPrompt),
						gollem.WithMessageHook(func(ctx context.Context, msg string) error {
							fmt.Print(msg)
							return nil
						}),
						gollem.WithToolRequestHook(func(ctx context.Context, tool gollem.FunctionCall) error {
							fmt.Printf("\n⚡ Execute Tool: %s\n", tool.Name)
							for k, v := range tool.Arguments {
								fmt.Printf("  ▶️ %s: %v\n", k, v)
							}
							fmt.Printf("\n")
							return nil
						}),
					)

					ctx = msg.With(ctx, notify, newTrace)

					if query != "" {
						if _, err = agent.Prompt(ctx, query); err != nil {
							return goerr.Wrap(err, "failed to chat")
						}
						return nil
					}

					var history *gollem.History
					for {
						msg, err := recvInput()
						if err != nil {
							if err == io.EOF {
								break
							}
							return goerr.Wrap(err, "failed to read line")
						}

						if msg == "exit" {
							break
						}

						msg = "# Main instruction\n\n" + msg
						history, err = agent.Prompt(ctx, msg, gollem.WithHistory(history))
						if err != nil {
							return goerr.Wrap(err, "failed to chat")
						}
						fmt.Printf("\n")
					}
				return nil
			*/
		},
	}
}

func recvInput() (string, error) {
	fmt.Printf("\033[2K> ")

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", goerr.Wrap(err, "failed to set raw mode")
	}
	defer func() {
		if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
			fmt.Fprintf(os.Stderr, "failed to restore terminal: %v\n", err)
		}
	}()

	t := term.NewTerminal(os.Stdin, "")
	msg, err := t.ReadLine()
	if err != nil {
		if err == io.EOF {
			return "", err
		}
		return "", goerr.Wrap(err, "failed to read line")
	}

	return msg, nil
}

func notify(ctx context.Context, msg string) {
	fmt.Printf("\033[1m>>> %s\033[0m\n", msg)
}

func newTrace(ctx context.Context, msg string) func(ctx context.Context, msg string) {
	fmt.Printf("<< %s >>\n", msg)
	return func(ctx context.Context, msg string) {
		fmt.Printf("%s\n", msg)
	}
}
