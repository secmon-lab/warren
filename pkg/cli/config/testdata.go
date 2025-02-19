package config

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
)

type TestData struct {
	detectDataPath string
	ignoreDataPath string
}

func (x *TestData) DetectDataPath() string {
	return x.detectDataPath
}

func (x *TestData) IgnoreDataPath() string {
	return x.ignoreDataPath
}

func (x *TestData) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "test-detect-data-path",
			Usage:       "Path to the detect data file",
			Destination: &x.detectDataPath,
			Sources:     cli.EnvVars("WARREN_TEST_DETECT_DATA_PATH"),
		},
		&cli.StringFlag{
			Name:        "test-ignore-data-path",
			Usage:       "Path to the ignore data file",
			Destination: &x.ignoreDataPath,
			Sources:     cli.EnvVars("WARREN_TEST_IGNORE_DATA_PATH"),
		},
	}
}

func (x *TestData) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("test-detect-data-path", x.detectDataPath),
		slog.String("test-ignore-data-path", x.ignoreDataPath),
	)
}

func loadFiles(basePath string) (map[string]any, error) {
	result := make(map[string]any)
	if basePath == "" {
		return result, nil
	}

	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return goerr.Wrap(err, "failed to read file", goerr.V("path", path))
		}

		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return goerr.Wrap(err, "failed to unmarshal json", goerr.V("path", path))
		}

		result[path] = v
		return nil
	})

	if err != nil {
		return nil, goerr.Wrap(err, "failed to walk directory")
	}

	return result, nil

}

func (x *TestData) Configure() (map[string]any, map[string]any, error) {
	detectData, err := loadFiles(x.detectDataPath)
	if err != nil {
		return nil, nil, err
	}

	ignoreData, err := loadFiles(x.ignoreDataPath)
	if err != nil {
		return nil, nil, err
	}

	return detectData, ignoreData, nil
}
