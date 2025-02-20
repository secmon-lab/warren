package config

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
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
			Name:        "test-detect-data",
			Usage:       "Path to the alert data JSON file should be detected. File path under the path will be used as [schema]/[filename].json",
			Category:    "Test",
			Destination: &x.detectDataPath,
			Sources:     cli.EnvVars("WARREN_TEST_DETECT_DATA"),
		},
		&cli.StringFlag{
			Name:        "test-ignore-data",
			Usage:       "Path to the alert data JSON file should be ignored. File path under the path will be used as [schema]/[filename].json",
			Category:    "Test",
			Destination: &x.ignoreDataPath,
			Sources:     cli.EnvVars("WARREN_TEST_IGNORE_DATA"),
		},
	}
}

func (x TestData) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("test-detect-data", x.detectDataPath),
		slog.String("test-ignore-data", x.ignoreDataPath),
	)
}

func loadTestFiles(basePath string) (model.TestData, error) {
	result := make(model.TestData)
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

		// Get relative path from basePath
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return goerr.Wrap(err, "failed to get relative path", goerr.V("path", path))
		}

		// Get first directory as key
		parts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
		if len(parts) == 0 || parts[0] == "." {
			return nil
		}
		firstDir := parts[0]

		// Get remaining path
		remainPath := relPath[len(firstDir)+1:]
		if remainPath == "" {
			remainPath = filepath.Base(relPath)
		}

		// Initialize map for first directory if not exists
		if _, ok := result[firstDir]; !ok {
			result[firstDir] = make(map[string]any)
		}

		// Store value with remaining path as key
		result[firstDir][remainPath] = v
		return nil
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to walk directory")
	}

	return result, nil
}

func (x *TestData) Configure() (*model.TestDataSet, error) {
	detectData, err := loadTestFiles(x.detectDataPath)
	if err != nil {
		return nil, err
	}

	ignoreData, err := loadTestFiles(x.ignoreDataPath)
	if err != nil {
		return nil, err
	}

	return &model.TestDataSet{
		Detect: detectData,
		Ignore: ignoreData,
	}, nil
}
