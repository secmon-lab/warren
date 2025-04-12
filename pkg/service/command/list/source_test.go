package list_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command/list"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func TestParseTime(t *testing.T) {
	// Create a fixed time for testing
	jst := time.FixedZone("JST", 9*60*60)
	fixedTime := time.Date(2024, 3, 15, 12, 0, 0, 0, jst)
	ctx := clock.With(context.Background(), func() time.Time { return fixedTime })
	ctx = clock.WithTimezone(ctx, jst)

	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "time only format",
			input:    "15:30",
			expected: time.Date(fixedTime.Year(), fixedTime.Month(), fixedTime.Day(), 15, 30, 0, 0, jst),
		},
		{
			name:     "date only format",
			input:    "2024-03-15",
			expected: time.Date(2024, 3, 15, 0, 0, 0, 0, jst),
		},
		{
			name:     "date and time format",
			input:    "2024-03-15 15:30",
			expected: time.Date(2024, 3, 15, 15, 30, 0, 0, jst),
		},
		{
			name:     "ISO format with timezone",
			input:    "2024-03-15T15:30:00Z",
			expected: time.Date(2024, 3, 15, 15, 30, 0, 0, time.UTC),
		},
		{
			name:     "12-hour format AM",
			input:    "4:00 AM",
			expected: time.Date(fixedTime.Year(), fixedTime.Month(), fixedTime.Day(), 4, 0, 0, 0, jst),
		},
		{
			name:     "12-hour format PM",
			input:    "10:00 PM",
			expected: time.Date(fixedTime.Year(), fixedTime.Month(), fixedTime.Day(), 22, 0, 0, 0, jst),
		},
		{
			name:     "12-hour format am",
			input:    "4:00am",
			expected: time.Date(fixedTime.Year(), fixedTime.Month(), fixedTime.Day(), 4, 0, 0, 0, jst),
		},
		{
			name:     "12-hour format pm",
			input:    "10:00pm",
			expected: time.Date(fixedTime.Year(), fixedTime.Month(), fixedTime.Day(), 22, 0, 0, 0, jst),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := list.ParseTime(ctx, tt.input)
			gt.NoError(t, err).Required()
			gt.Value(t, result).Equal(tt.expected)
		})
	}
}

func TestParseArgsToSource(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	fixedTime := time.Date(2024, 3, 15, 12, 0, 0, 0, jst)
	ctx := clock.With(context.Background(), func() time.Time { return fixedTime })
	ctx = clock.WithTimezone(ctx, jst)

	tests := []struct {
		name     string
		args     []string
		expected func(*interfaces.RepositoryMock)
	}{
		{
			name: "empty args returns last 24 hours",
			args: []string{},
			expected: func(m *interfaces.RepositoryMock) {
				gt.Array(t, m.GetAlertsBySpanCalls()).Length(1)
				gt.Value(t, m.GetAlertsBySpanCalls()[0].Begin).Equal(fixedTime.Add(-24 * time.Hour))
				gt.Value(t, m.GetAlertsBySpanCalls()[0].End).Equal(fixedTime)
			},
		},
		{
			name: "unresolved command",
			args: []string{"unresolved"},
			expected: func(m *interfaces.RepositoryMock) {
				gt.Array(t, m.GetAlertsWithoutStatusCalls()).Length(1)
				gt.Value(t, m.GetAlertsWithoutStatusCalls()[0].Status).Equal(types.AlertStatusResolved)
			},
		},
		{
			name: "status command with multiple statuses",
			args: []string{"status", "resolved", "acked"},
			expected: func(m *interfaces.RepositoryMock) {
				gt.Array(t, m.GetAlertsByStatusCalls()).Length(1)
				gt.Value(t, m.GetAlertsByStatusCalls()[0].Status).Equal([]types.AlertStatus{types.AlertStatusResolved, types.AlertStatusAcknowledged})
			},
		},
		{
			name: "between command with date range",
			args: []string{"between", "2024-03-15", "2024-03-16"},
			expected: func(m *interfaces.RepositoryMock) {
				gt.Array(t, m.GetAlertsBySpanCalls()).Length(1)
				gt.Value(t, m.GetAlertsBySpanCalls()[0].Begin).Equal(time.Date(2024, 3, 15, 0, 0, 0, 0, jst))
				gt.Value(t, m.GetAlertsBySpanCalls()[0].End).Equal(time.Date(2024, 3, 16, 0, 0, 0, 0, jst))
			},
		},
		{
			name: "after command with date",
			args: []string{"after", "2024-03-15"},
			expected: func(m *interfaces.RepositoryMock) {
				gt.Array(t, m.GetAlertsBySpanCalls()).Length(1)
				gt.Value(t, m.GetAlertsBySpanCalls()[0].Begin).Equal(time.Date(2024, 3, 15, 0, 0, 0, 0, jst))
				gt.Value(t, m.GetAlertsBySpanCalls()[0].End).Equal(fixedTime)
			},
		},
		{
			name: "since command with duration",
			args: []string{"since", "1h"},
			expected: func(m *interfaces.RepositoryMock) {
				gt.Array(t, m.GetAlertsBySpanCalls()).Length(1)
				gt.Value(t, m.GetAlertsBySpanCalls()[0].Begin).Equal(fixedTime.Add(-1 * time.Hour))
				gt.Value(t, m.GetAlertsBySpanCalls()[0].End).Equal(fixedTime)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &interfaces.RepositoryMock{}
			src, err := list.ParseArgsToSource(ctx, tt.args)
			gt.NoError(t, err).Required()
			gt.Value(t, src).NotNil()

			// Execute the source to trigger repository calls
			_, err = src(ctx, mock)
			gt.NoError(t, err).Required()

			// Verify the expected repository calls
			tt.expected(mock)
		})
	}
}

func TestParseArgsToSourceErrors(t *testing.T) {
	type testCase struct {
		args  []string
		error string
	}
	runTest := func(tc testCase) func(*testing.T) {
		return func(t *testing.T) {
			result, err := list.ParseArgsToSource(t.Context(), tc.args)
			gt.Error(t, err).Contains(tc.error)
			gt.Value(t, result).Nil()
		}
	}

	t.Run("invalid from command format", runTest(testCase{
		args:  []string{"from", "2024-03-15"},
		error: "unknown command",
	}))

	t.Run("invalid from command format", runTest(testCase{
		args:  []string{"from", "2024-03-15"},
		error: "unknown command",
	}))

	t.Run("invalid between command format", runTest(testCase{
		args:  []string{"between", "2024-03-15"},
		error: "invalid date range format",
	}))

	t.Run("invalid after command format", runTest(testCase{
		args:  []string{"after"},
		error: "invalid date format",
	}))

	t.Run("invalid since command format", runTest(testCase{
		args:  []string{"since"},
		error: "invalid duration format",
	}))

	t.Run("unknown command", runTest(testCase{
		args:  []string{"unknown"},
		error: "unknown command",
	}))
}
