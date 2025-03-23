package config

import (
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/urfave/cli/v3"
)

type Policy struct {
	filePaths []string
}

func (x *Policy) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "policy",
			Usage:       "Policy file/dir path",
			Aliases:     []string{"p"},
			Destination: &x.filePaths,
			Category:    "Policy",
			Sources:     cli.EnvVars("WARREN_POLICY"),
		},
	}
}

func (x Policy) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("file_paths", x.filePaths),
	)
}

func (x *Policy) Configure() (*opaq.Client, policy.Contents, error) {
	client, err := opaq.New(opaq.Files(x.filePaths...))
	if err != nil {
		return nil, nil, goerr.Wrap(err, "failed to create opaq client", goerr.V("file_paths", x.filePaths))
	}

	contents, err := policy.NewContents(x.filePaths...)
	if err != nil {
		return nil, nil, goerr.Wrap(err, "failed to create policy contents", goerr.V("file_paths", x.filePaths))
	}

	return client, contents, nil
}
