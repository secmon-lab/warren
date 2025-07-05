package config_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
)

func TestLoadTestFiles(t *testing.T) {
	data, err := config.LoadTestFiles("testdata/test")
	gt.NoError(t, err)

	gt.Map(t, data.Data).
		HasKey("schema1").
		At("schema1", func(t testing.TB, v map[string]any) {
			gt.Map(t, v).HasKeyValue("nest/data.json", map[string]any{"msg": "schema1 test"})
			gt.Map(t, v).HasKeyValue("nest/nest2/data.json", map[string]any{"msg": "nest2 test"})

			// Test JSONL file with multiple objects
			gt.Map(t, v).HasKeyValue("multi_objects.jsonl", map[string]any{"msg": "first object", "id": float64(1)})
			gt.Map(t, v).HasKeyValue("multi_objects_obj2.jsonl", map[string]any{"msg": "second object", "id": float64(2)})
			gt.Map(t, v).HasKeyValue("multi_objects_obj3.jsonl", map[string]any{"msg": "third object", "id": float64(3)})
		}).
		At("schema2", func(t testing.TB, v map[string]any) {
			gt.Map(t, v).HasKeyValue("data.json", map[string]any{"msg": "schema2 test"})

			// Test JSONL file with multiple objects
			gt.Map(t, v).HasKeyValue("multi_objects.jsonl", map[string]any{"msg": "json first", "type": "json"})
			gt.Map(t, v).HasKeyValue("multi_objects_obj2.jsonl", map[string]any{"msg": "json second", "type": "json"})
		})
}

func TestLoadTestFiles_EmptyFile(t *testing.T) {
	// Test that files with no JSON objects return an error
	data, err := config.LoadTestFiles("testdata/empty")
	gt.Error(t, err)
	gt.Value(t, data).Equal((*policy.TestData)(nil))
}

func TestLoadTestFiles_InvalidJSON(t *testing.T) {
	// Test that files with invalid JSON return an error
	data, err := config.LoadTestFiles("testdata/invalid")
	gt.Error(t, err)
	gt.Value(t, data).Equal((*policy.TestData)(nil))
}
