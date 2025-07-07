package policy_test

import (
	"testing"

	"github.com/secmon-lab/warren/pkg/domain/model/policy"
)

func TestDiffPolicy(t *testing.T) {
	cases := []struct {
		name     string
		old      map[string]string
		new      map[string]string
		expected map[string]string
	}{
		{
			name: "no changes",
			old: map[string]string{
				"test.rego": "package test\n\ndefault allow = false",
			},
			new: map[string]string{
				"test.rego": "package test\n\ndefault allow = false",
			},
			expected: nil,
		},
		{
			name: "add new file",
			old:  map[string]string{},
			new: map[string]string{
				"test.rego": "package test\n\ndefault allow = false",
			},
			expected: map[string]string{
				"test.rego": "+ package test\n+ \n+ default allow = false\n",
			},
		},
		{
			name: "delete file",
			old: map[string]string{
				"test.rego": "package test\n\ndefault allow = false",
			},
			new: map[string]string{},
			expected: map[string]string{
				"test.rego": "- package test\n- \n- default allow = false\n",
			},
		},
		{
			name: "modify file",
			old: map[string]string{
				"test.rego": "package test\n\ndefault allow = false",
			},
			new: map[string]string{
				"test.rego": "package test\n\ndefault allow = true",
			},
			expected: map[string]string{
				"test.rego": "  package test\n  \n- default allow = false\n+ default allow = true\n",
			},
		},
		{
			name: "add line in middle",
			old: map[string]string{
				"test.rego": "package test\n\ndefault allow = false",
			},
			new: map[string]string{
				"test.rego": "package test\n\nallow = true\ndefault allow = false",
			},
			expected: map[string]string{
				"test.rego": "  package test\n  \n+ allow = true\n  default allow = false\n",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := policy.DiffPolicy(tc.old, tc.new)
			if len(result) != len(tc.expected) {
				t.Errorf("expected %d diffs, got %d", len(tc.expected), len(result))
			}
			for file, diff := range tc.expected {
				if result[file] != diff {
					t.Errorf("for file %s\nexpected:\n%s\ngot:\n%s", file, diff, result[file])
				}
			}
		})
	}

}
