package eval_test

import (
	"slices"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/eval"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestScenario_Validate_Valid(t *testing.T) {
	s := &eval.Scenario{
		Name:           "test",
		Alert:          eval.AlertConfig{Schema: types.AlertSchema("gcp_scc"), Data: map[string]any{"key": "val"}},
		World:          eval.WorldConfig{Description: "test world"},
		InitialMessage: "investigate",
		Config:         eval.ScenarioConfig{PolicyDir: "policy"},
	}
	errors := s.Validate()
	gt.V(t, len(errors)).Equal(0)
}

func TestScenario_Validate_MissingFields(t *testing.T) {
	s := &eval.Scenario{}
	errors := s.Validate()
	gt.V(t, slices.Contains(errors, "name is required")).Equal(true)
	gt.V(t, slices.Contains(errors, "alert.schema is required")).Equal(true)
	gt.V(t, slices.Contains(errors, "alert.data is required")).Equal(true)
	gt.V(t, slices.Contains(errors, "initial_message is required")).Equal(true)
	gt.V(t, slices.Contains(errors, "world.description is required (mock agent needs scenario context)")).Equal(true)
	gt.V(t, slices.Contains(errors, "config.policy_dir is required (pipeline needs Rego policies)")).Equal(true)
}

func TestExpectations_Validate_TrajectoryConflict(t *testing.T) {
	exp := &eval.Expectations{
		Trajectory: &eval.TrajectoryExpectation{
			MustCall:    []string{"virustotal", "bigquery"},
			MustNotCall: []string{"virustotal"},
		},
	}
	errors := exp.Validate()
	gt.V(t, slices.Contains(errors, `tool "virustotal" appears in both must_call and must_not_call`)).Equal(true)
}

func TestExpectations_Validate_OrderedCallsSubset(t *testing.T) {
	exp := &eval.Expectations{
		Trajectory: &eval.TrajectoryExpectation{
			MustCall:     []string{"virustotal"},
			OrderedCalls: []string{"virustotal", "bigquery"},
		},
	}
	errors := exp.Validate()
	gt.V(t, slices.Contains(errors, `ordered_calls tool "bigquery" is not in must_call`)).Equal(true)
}

func TestResponseFile_Validate_Valid(t *testing.T) {
	rf := &eval.ResponseFile{
		ToolName: "virustotal",
		Args:     map[string]any{"ip": "1.2.3.4"},
		Response: map[string]any{"malicious": true},
	}
	errors := rf.Validate("virustotal__a3f2b1c9.json")
	gt.V(t, len(errors)).Equal(0)
}

func TestResponseFile_Validate_MissingToolName(t *testing.T) {
	rf := &eval.ResponseFile{
		Response: map[string]any{"ok": true},
	}
	errors := rf.Validate("bad__12345678.json")
	gt.V(t, slices.Contains(errors, "responses/bad__12345678.json: tool_name is required")).Equal(true)
}

func TestResponseFile_Validate_FilenameMismatch(t *testing.T) {
	rf := &eval.ResponseFile{
		ToolName: "virustotal",
		Response: map[string]any{"ok": true},
	}
	errors := rf.Validate("wrong__12345678.json")
	gt.V(t, slices.Contains(errors, `responses/wrong__12345678.json: filename should start with "virustotal__" (got tool_name="virustotal")`)).Equal(true)
}
