package cli

/*
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

			ctx = thread.WithReplyFunc(ctx, func(ctx context.Context, msg string) {
				fmt.Println(msg)
			})
			newPolicyDiff, err := uc.GenerateIgnorePolicy(ctx, source.Static(alert.Alerts{alert}), "")
			if err != nil {
				return err
			}

			logger.Info("New policy diff", "newPolicyDiff", newPolicyDiff)

			return nil
		},
	}
}
*/
