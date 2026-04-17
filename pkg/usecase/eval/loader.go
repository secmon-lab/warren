package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/eval"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"gopkg.in/yaml.v3"
)

// LoadScenario loads a scenario from the given directory.
func LoadScenario(scenarioDir string) (*eval.Scenario, error) {
	scenarioPath := filepath.Join(scenarioDir, "scenario.yaml")

	data, err := os.ReadFile(filepath.Clean(scenarioPath))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read scenario file",
			goerr.V("path", scenarioPath),
			goerr.T(errutil.TagValidation),
		)
	}

	var scenario eval.Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return nil, goerr.Wrap(err, "failed to parse scenario YAML",
			goerr.V("path", scenarioPath),
			goerr.T(errutil.TagValidation),
		)
	}

	return &scenario, nil
}

// ValidateScenarioDir performs comprehensive validation of a scenario directory.
// It loads the scenario, validates the model, and validates response files.
// Returns a list of validation errors (empty = valid).
func ValidateScenarioDir(scenarioDir string) []string {
	// Load and parse
	scenarioPath := filepath.Join(scenarioDir, "scenario.yaml")
	data, err := os.ReadFile(filepath.Clean(scenarioPath))
	if err != nil {
		return []string{fmt.Sprintf("cannot read scenario.yaml: %v", err)}
	}

	var scenario eval.Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return []string{fmt.Sprintf("invalid YAML in scenario.yaml: %v", err)}
	}

	// Model validation
	errors := scenario.Validate()

	// Policy dir existence check
	if scenario.Config.PolicyDir != "" {
		policyPath := filepath.Join(scenarioDir, scenario.Config.PolicyDir)
		if info, statErr := os.Stat(policyPath); statErr != nil || !info.IsDir() {
			errors = append(errors, fmt.Sprintf("config.policy_dir %q does not exist or is not a directory", scenario.Config.PolicyDir))
		}
	}

	// Response files validation (I/O layer)
	responsesDir := ResponsesDir(scenarioDir)
	if info, statErr := os.Stat(responsesDir); statErr == nil && info.IsDir() {
		errors = append(errors, validateResponseFiles(responsesDir)...)
	}

	return errors
}

func validateResponseFiles(responsesDir string) []string {
	var errors []string

	entries, err := os.ReadDir(responsesDir)
	if err != nil {
		return []string{fmt.Sprintf("cannot read responses/: %v", err)}
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(responsesDir, entry.Name())
		data, err := os.ReadFile(filepath.Clean(filePath))
		if err != nil {
			errors = append(errors, fmt.Sprintf("responses/%s: cannot read file: %v", entry.Name(), err))
			continue
		}

		var rf eval.ResponseFile
		if err := json.Unmarshal(data, &rf); err != nil {
			errors = append(errors, fmt.Sprintf("responses/%s: invalid JSON: %v", entry.Name(), err))
			continue
		}

		errors = append(errors, rf.Validate(entry.Name())...)
	}

	return errors
}

// ResponsesDir returns the path to the responses directory for a scenario.
func ResponsesDir(scenarioDir string) string {
	return filepath.Join(scenarioDir, "responses")
}

