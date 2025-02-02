package config

import (
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opac"
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

func (x *Policy) Configure() (*opac.Client, error) {
	client, err := opac.New(opac.Files(x.filePaths...))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create opac client", goerr.V("file_paths", x.filePaths))
	}

	return client, nil
}
