package eval_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/m-mizutani/gt"
	evalModel "github.com/secmon-lab/warren/pkg/domain/model/eval"
	"github.com/secmon-lab/warren/pkg/usecase/eval"
)

const validScenarioYAML = `
name: test_scenario
description: "Test scenario"
alert:
  schema: "gcp_scc"
  data:
    finding:
      category: "test"
world:
  description: "Test world"
  tool_hints:
    bigquery: "test hint"
initial_message: "Investigate"
config:
  policy_dir: "policy"
expectations:
  outcome:
    finding_must_contain:
      - "test"
    severity: "high"
  trajectory:
    must_call:
      - virustotal
      - bigquery
    must_not_call:
      - intune
    ordered_calls:
      - virustotal
      - bigquery
  efficiency:
    max_total_calls: 20
    max_duplicate_calls: 3
`

func TestLoadScenario_Valid(t *testing.T) {
	dir := t.TempDir()
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(validScenarioYAML), 0o640))

	scenario, err := eval.LoadScenario(dir)
	gt.NoError(t, err)

	gt.V(t, scenario.Name).Equal("test_scenario")
	gt.V(t, scenario.Description).Equal("Test scenario")
	gt.V(t, string(scenario.Alert.Schema)).Equal("gcp_scc")
	gt.V(t, scenario.World.Description).Equal("Test world")
	gt.V(t, scenario.World.ToolHints["bigquery"]).Equal("test hint")
	gt.V(t, scenario.InitialMessage).Equal("Investigate")
	gt.V(t, scenario.Expectations).NotNil()
	gt.V(t, scenario.Expectations.Outcome.Severity).Equal("high")
	gt.V(t, scenario.Expectations.Trajectory.MustCall).Equal([]string{"virustotal", "bigquery"})
	gt.V(t, scenario.Expectations.Efficiency.MaxTotalCalls).Equal(20)
}

func TestLoadScenario_MissingFile(t *testing.T) {
	_, err := eval.LoadScenario("/nonexistent/path")
	gt.V(t, err).NotNil()
}

func writeValidScenario(t *testing.T, dir string) {
	t.Helper()
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(validScenarioYAML), 0o640))
	gt.NoError(t, os.MkdirAll(filepath.Join(dir, "policy"), 0o750))
}

func TestValidateScenario_Valid(t *testing.T) {
	dir := t.TempDir()
	writeValidScenario(t, dir)

	errors := eval.ValidateScenarioDir(dir)
	gt.V(t, len(errors)).Equal(0)
}

func TestValidateScenario_MissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	yaml := `
description: "No name or schema"
`
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(yaml), 0o640))

	errors := eval.ValidateScenarioDir(dir)
	gt.V(t, len(errors) >= 3).Equal(true) // name, alert.schema, alert.data, initial_message, world.description
}

func TestValidateScenario_TrajectoryConflict(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: test
alert:
  schema: "test"
  data: {key: val}
world:
  description: "test"
initial_message: "test"
expectations:
  trajectory:
    must_call:
      - virustotal
    must_not_call:
      - virustotal
`
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(yaml), 0o640))

	errors := eval.ValidateScenarioDir(dir)
	gt.V(t, slices.Contains(errors, `tool "virustotal" appears in both must_call and must_not_call`)).Equal(true)
}

func TestValidateScenario_OrderedCallsNotInMustCall(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: test
alert:
  schema: "test"
  data: {key: val}
world:
  description: "test"
initial_message: "test"
expectations:
  trajectory:
    must_call:
      - virustotal
    ordered_calls:
      - bigquery
`
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(yaml), 0o640))

	errors := eval.ValidateScenarioDir(dir)
	gt.V(t, slices.Contains(errors, `ordered_calls tool "bigquery" is not in must_call`)).Equal(true)
}

func TestValidateScenario_InvalidResponseFile(t *testing.T) {
	dir := t.TempDir()
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(validScenarioYAML), 0o640))
	responsesDir := filepath.Join(dir, "responses")
	gt.NoError(t, os.MkdirAll(responsesDir, 0o750))

	// Write invalid JSON
	gt.NoError(t, os.WriteFile(filepath.Join(responsesDir, "bad__12345678.json"), []byte("{invalid"), 0o640))

	errors := eval.ValidateScenarioDir(dir)
	gt.V(t, len(errors) > 0).Equal(true)
}

func TestValidateScenario_ResponseFileMissingToolName(t *testing.T) {
	dir := t.TempDir()
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(validScenarioYAML), 0o640))
	responsesDir := filepath.Join(dir, "responses")
	gt.NoError(t, os.MkdirAll(responsesDir, 0o750))

	rf := evalModel.ResponseFile{
		ToolName: "",
		Args:     map[string]any{"ip": "1.2.3.4"},
		Response: map[string]any{"ok": true},
	}
	data, _ := json.Marshal(rf)
	gt.NoError(t, os.WriteFile(filepath.Join(responsesDir, "bad__12345678.json"), data, 0o640))

	errors := eval.ValidateScenarioDir(dir)
	gt.V(t, slices.Contains(errors, "responses/bad__12345678.json: tool_name is required")).Equal(true)
}

func TestValidateScenario_ResponseFileNameMismatch(t *testing.T) {
	dir := t.TempDir()
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(validScenarioYAML), 0o640))
	responsesDir := filepath.Join(dir, "responses")
	gt.NoError(t, os.MkdirAll(responsesDir, 0o750))

	rf := evalModel.ResponseFile{
		ToolName: "virustotal",
		Args:     map[string]any{"ip": "1.2.3.4"},
		Response: map[string]any{"malicious": true},
	}
	data, _ := json.Marshal(rf)
	gt.NoError(t, os.WriteFile(filepath.Join(responsesDir, "wrong__12345678.json"), data, 0o640))

	errors := eval.ValidateScenarioDir(dir)
	gt.V(t, slices.Contains(errors, `responses/wrong__12345678.json: filename should start with "virustotal__" (got tool_name="virustotal")`)).Equal(true)
}
