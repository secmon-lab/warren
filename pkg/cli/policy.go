package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/service/source"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
	"github.com/urfave/cli/v3"
)

func cmdPolicy() *cli.Command {
	return &cli.Command{
		Name:    "policy",
		Aliases: []string{"p"},
		Usage:   "Manage policies",
		Commands: []*cli.Command{
			cmdPolicyIgnore(),
		},
	}
}

func cmdPolicyIgnore() *cli.Command {
	var (
		testDataCfg   config.TestData
		policyCfg     config.Policy
		geminiCfg     config.GeminiCfg
		alertDataPath string
		schema        string
		outputDir     string
	)

	flags := joinFlags(
		testDataCfg.Flags(),
		policyCfg.Flags(),
		geminiCfg.Flags(),
		[]cli.Flag{
			&cli.StringFlag{
				Name:        "alert-data",
				Aliases:     []string{"d"},
				Usage:       "Raw alert data file path",
				Destination: &alertDataPath,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "schema",
				Aliases:     []string{"s"},
				Usage:       "Schema",
				Destination: &schema,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "output-dir",
				Aliases:     []string{"o"},
				Usage:       "Output directory",
				Destination: &outputDir,
			},
		},
	)

	return &cli.Command{
		Name:  "ignore",
		Usage: "Create or update ignore policy",
		Flags: flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logger := logging.From(ctx)
			logger.Info("Creating or updating ignore policy",
				"alertDataPath", alertDataPath,
				"schema", schema,
				"policyCfg", policyCfg,
				"testDataCfg", testDataCfg,
			)

			policyClient, err := policyCfg.Configure()
			if err != nil {
				return err
			}

			testDataSet, err := testDataCfg.Configure()
			if err != nil {
				return err
			}

			geminiModel, err := geminiCfg.Configure(ctx)
			if err != nil {
				return err
			}

			policyService := policy.New(repository.NewMemory(), policyClient, testDataSet)

			data, err := os.ReadFile(filepath.Clean(alertDataPath))
			if err != nil {
				return goerr.Wrap(err, "failed to read alert file", goerr.V("alertFile", alertDataPath))
			}

			var alertData any
			if err := json.Unmarshal(data, &alertData); err != nil {
				return goerr.Wrap(err, "failed to unmarshal alert", goerr.V("alertFile", alertDataPath))
			}

			uc := usecase.New(
				usecase.WithLLMClient(geminiModel),
				usecase.WithPolicyService(policyService),
			)

			alert := model.NewAlert(ctx, schema, model.PolicyAlert{
				Data: alertData,
			})
			ctx = thread.WithReplyFunc(ctx, func(ctx context.Context, msg string) {
				fmt.Println(msg)
			})
			newPolicyDiff, err := uc.GenerateIgnorePolicy(ctx, source.Static([]model.Alert{alert}), "")
			if err != nil {
				return err
			}

			logger.Info("New policy diff", "newPolicyDiff", newPolicyDiff)

			// TODO: Revise output of policy diff
			/*
				if outputDir == "" {
					tmpDir, err := os.MkdirTemp("", "warren-policy")
					if err != nil {
						return goerr.Wrap(err, "failed to create temporary directory")
					}
					outputDir = tmpDir
				}

				svc := policy.New(repository.NewMemory(), policyClient, newPolicyDiff.NewTestDataSet)

				if err := svc.Save(ctx, outputDir); err != nil {
					return goerr.Wrap(err, "failed to save policy")
				}
			*/

			return nil
		},
	}
}
