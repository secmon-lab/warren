package config_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
)

func TestLoadTestFiles(t *testing.T) {
	data, err := config.LoadTestFiles("testdata/test")
	gt.NoError(t, err)

	gt.Map(t, data.Data).
		HaveKey("schema1").
		At("schema1", func(t testing.TB, v map[string]any) {
			gt.Map(t, v).HaveKeyValue("nest/data.json", map[string]any{"msg": "schema1 test"})
			gt.Map(t, v).HaveKeyValue("nest/nest2/data.json", map[string]any{"msg": "nest2 test"})
		}).
		At("schema2", func(t testing.TB, v map[string]any) {
			gt.Map(t, v).HaveKeyValue("data.json", map[string]any{"msg": "schema2 test"})
		})
}
