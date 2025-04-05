package proc_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/proc"
)

func TestParseTime(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "RFC3339 format",
			input:    "2024-01-01T10:00:00+09:00",
			expected: "10:00:00",
			wantErr:  false,
		},
		{
			name:     "time only",
			input:    "10:00",
			expected: "10:00:00",
			wantErr:  false,
		},
		{
			name:     "date only",
			input:    "2/3",
			expected: "00:00:00",
			wantErr:  false,
		},
		{
			name:     "date and time",
			input:    "02-03T00:00",
			expected: "00:00:00",
			wantErr:  false,
		},
		{
			name:     "today",
			input:    "today",
			expected: "00:00:00",
			wantErr:  false,
		},
		{
			name:     "yesterday",
			input:    "yesterday",
			expected: "00:00:00",
			wantErr:  false,
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := proc.ParseTime(tt.input)
			if tt.wantErr {
				gt.Error(t, err)
				return
			}
			gt.NoError(t, err)
			gt.Equal(t, got.Format("15:04:05"), tt.expected)
		})
	}
}
