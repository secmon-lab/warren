package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/action/base"
	"github.com/secmon-lab/warren/pkg/cli/config"
	session_model "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/action"
	"github.com/secmon-lab/warren/pkg/service/session"
	"github.com/secmon-lab/warren/pkg/utils/msg"
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
		},
		firestoreDB.Flags(),
		geminiCfg.Flags(),
		policyCfg.Flags(),
		actions.Flags(),
	)

	return &cli.Command{
		Name:    "chat",
		Aliases: []string{"c"},
		Usage:   "Chat with the security analyst",
		Flags:   flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			repo, err := firestoreDB.Configure(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure firestore")
			}

			geminiClient, err := geminiCfg.Configure(ctx)
			if err != nil {
				return goerr.Wrap(err, "failed to configure gemini")
			}

			if (alertID == "") == (alertListID == "") {
				return goerr.New("either alert-id or alert-list-id must be provided")
			}

			var alertIDs []types.AlertID
			if alertID != "" {
				alert, err := repo.GetAlert(ctx, alertID)
				if err != nil {
					return goerr.Wrap(err, "failed to get alert")
				}

				fmt.Printf("🔔 Alert Information:\n")
				fmt.Printf("  📝 Title: %s\n", alert.Title)
				fmt.Printf("  📋 Description: %s\n", alert.Description)
				fmt.Printf("  🔍 Attributes:\n")
				for _, attr := range alert.Attributes {
					fmt.Printf("    - %s: %v\n", attr.Key, attr.Value)
				}

				alertIDs = []types.AlertID{alertID}
			} else if alertListID != "" {
				list, err := repo.GetAlertList(ctx, alertListID)
				if err != nil {
					return goerr.Wrap(err, "failed to get alert list")
				}

				fmt.Printf("📋 Alert List Information:\n")
				fmt.Printf("  📝 Title: %s\n", list.Title)
				fmt.Printf("  📋 Description: %s\n", list.Description)
				fmt.Printf("  🔢 Number of Alerts: %d\n", len(list.AlertIDs))

				alertIDs = list.AlertIDs
			}
			fmt.Printf("\n")

			policyClient, err := policyCfg.Configure()
			if err != nil {
				return goerr.Wrap(err, "failed to configure policy")
			}

			ssn := session_model.New(ctx, nil, nil, alertIDs)

			actions = append(actions, base.New(repo, alertIDs, policyClient.Sources(), ssn.ID))
			actionSvc, err := action.New(ctx, actions)
			if err != nil {
				return goerr.Wrap(err, "failed to configure action")
			}

			ssnSvc := session.New(repo, geminiClient, actionSvc, ssn)

			ctx = msg.With(ctx, notify, newTrace)

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

				if err := ssnSvc.Chat(ctx, msg); err != nil {
					return goerr.Wrap(err, "failed to chat")
				}
			}

			return nil
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
