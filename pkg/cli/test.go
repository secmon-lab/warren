package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdTest() *cli.Command {
	var (
		policyCfg   config.Policy
		testDataCfg config.TestData
	)

	flags := joinFlags(
		policyCfg.Flags(),
		testDataCfg.Flags(),
	)

	return &cli.Command{
		Name:    "test",
		Aliases: []string{"t"},
		Usage:   "Run test",
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logger := logging.From(ctx)
			logger.Info("Starting test", "testDataCfg", testDataCfg, "policyCfg", policyCfg)

			policyClient, err := policyCfg.Configure()
			if err != nil {
				return err
			}
			if len(policyClient.Sources()) == 0 {
				return goerr.New("no policy sources")
			}

			testDataSet, err := testDataCfg.Configure()
			if err != nil {
				return err
			}
			logger.Info("Test data", "detect", testDataSet.Detect, "ignore", testDataSet.Ignore)

			var runtimeErrors []error
			errs := policy.Test(ctx, policyClient, testDataSet)
			failed := false
			for _, err := range errs {
				if goerr.HasTag(err, errs.TagTestFailed) {
					values := goerr.Values(err)
					fmt.Printf("\n❌ Test Failed!\n")
					fmt.Printf("  Reason:   %s\n", err.Error())
					fmt.Printf("  File:     %s\n", filepath.Join(values["schema"].(string), values["filename"].(string)))
					fmt.Println("  ----------------------------------------")
					failed = true
				} else {
					runtimeErrors = append(runtimeErrors, err)
				}
			}

			if len(runtimeErrors) > 0 {
				return goerr.Wrap(runtimeErrors[0], "test failed by runtime error")
			}

			if failed {
				fmt.Printf("\n❌ Test failed\n\n")
				return goerr.New("test failed")
			}

			fmt.Printf("\n✅️ Test passed\n\n")
			return nil
		},
	}
}
