package slack_test

import (
	"testing"

	"github.com/secmon-lab/warren/pkg/controller/slack"
	"github.com/secmon-lab/warren/pkg/domain/model"
)

func TestParseArgs(t *testing.T) {
	testCases := map[string]struct {
		input    string
		expected []string
	}{
		"simple": {
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		"with quotes": {
			input:    `hello "world with spaces"`,
			expected: []string{"hello", "world with spaces"},
		},
		"with single quotes": {
			input:    `hello 'world with spaces'`,
			expected: []string{"hello", "world with spaces"},
		},
		"with escaped quotes": {
			input:    `hello \"world\" test`,
			expected: []string{"hello", `"world"`, "test"},
		},
		"with Japanese quotes": {
			input:    `hello "world" test`,
			expected: []string{"hello", "world", "test"},
		},
		"with multiple spaces": {
			input:    "hello   world",
			expected: []string{"hello", "world"},
		},
		"empty": {
			input:    "",
			expected: nil,
		},
		"with backslash": {
			input:    `hello\\world`,
			expected: []string{"hello\\world"},
		},
		"mixed quotes": {
			input:    `"hello" 'world' "test"`,
			expected: []string{"hello", "world", "test"},
		},
		"with backticks": {
			input:    "hello \\`world\\` test",
			expected: []string{"hello", "`world`", "test"},
		},
		"with utf-8 quotes": {
			input:    `ok “this is” “hello world”`,
			expected: []string{"ok", "this is", "hello world"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := slack.ParseArgs(tc.input)
			if len(result) != len(tc.expected) {
				t.Errorf("expected %d args, got %d", len(tc.expected), len(result))
				return
			}
			for i := range result {
				if result[i] != tc.expected[i] {
					t.Errorf("arg %d: expected %q, got %q", i, tc.expected[i], result[i])
				}
			}
		})
	}
}

func TestParseMention(t *testing.T) {
	testCases := map[string]struct {
		input    string
		expected []model.SlackMention
	}{
		"single mention": {
			input: "<@U123> hello world",
			expected: []model.SlackMention{
				{
					UserID: "U123",
					Args:   []string{"hello", "world"},
				},
			},
		},
		"multiple mentions": {
			input: "<@U123> hello <@U456> world",
			expected: []model.SlackMention{
				{
					UserID: "U123",
					Args:   []string{"hello"},
				},
				{
					UserID: "U456",
					Args:   []string{"world"},
				},
			},
		},
		"mention with quoted args": {
			input: `<@U123> "hello world" test`,
			expected: []model.SlackMention{
				{
					UserID: "U123",
					Args:   []string{"hello world", "test"},
				},
			},
		},
		"mention without args": {
			input: "<@U123>",
			expected: []model.SlackMention{
				{
					UserID: "U123",
					Args:   []string{},
				},
			},
		},
		"empty string": {
			input:    "",
			expected: nil,
		},
		"text without mention": {
			input:    "hello world",
			expected: nil,
		},
		"mention with Japanese text": {
			input: "<@U123> こんにちは 世界",
			expected: []model.SlackMention{
				{
					UserID: "U123",
					Args:   []string{"こんにちは", "世界"},
				},
			},
		},
		"mention with mixed quotes": {
			input: `<@U123> "hello world" 'test case' "quoted"`,
			expected: []model.SlackMention{
				{
					UserID: "U123",
					Args:   []string{"hello world", "test case", "quoted"},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := slack.ParseMention(tc.input)
			if len(result) != len(tc.expected) {
				t.Errorf("expected %d mentions, got %d", len(tc.expected), len(result))
				return
			}
			for i := range result {
				if result[i].UserID != tc.expected[i].UserID {
					t.Errorf("mention %d: expected UserID %q, got %q", i, tc.expected[i].UserID, result[i].UserID)
				}
				if len(result[i].Args) != len(tc.expected[i].Args) {
					t.Errorf("mention %d: expected %d args, got %d", i, len(tc.expected[i].Args), len(result[i].Args))
					continue
				}
				for j := range result[i].Args {
					if result[i].Args[j] != tc.expected[i].Args[j] {
						t.Errorf("mention %d arg %d: expected %q, got %q", i, j, tc.expected[i].Args[j], result[i].Args[j])
					}
				}
			}
		})
	}
}
