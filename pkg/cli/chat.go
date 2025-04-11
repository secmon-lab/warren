package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/m-mizutani/goerr/v2"
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

			actionSvc, err := action.New(ctx, actions)
			if err != nil {
				return goerr.Wrap(err, "failed to configure action")
			}

			if (alertID == "") == (alertListID == "") {
				return goerr.New("either alert-id or alert-list-id must be provided")
			}

			var alertIDs []types.AlertID
			if alertID != "" {
				alertIDs = []types.AlertID{alertID}
			} else if alertListID != "" {
				resp, err := repo.GetAlertList(ctx, alertListID)
				if err != nil {
					return goerr.Wrap(err, "failed to get alert list")
				}
				alertIDs = resp.AlertIDs
			}

			ssn := session_model.New(ctx, nil, nil, alertIDs)

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
	defer term.Restore(int(os.Stdin.Fd()), oldState)

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
