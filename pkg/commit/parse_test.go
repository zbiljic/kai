package commit

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected Message
	}{
		{
			input: "fix: correct minor typos",
			expected: Message{
				Type:          "fix",
				Scope:         "",
				CommitMessage: "correct minor typos",
			},
		},
		{
			input: "feat(parser): add new parsing functions",
			expected: Message{
				Type:          "feat",
				Scope:         "parser",
				CommitMessage: "add new parsing functions",
			},
		},
		{
			input: "refactor(core)!: extract methods",
			expected: Message{
				Type:          "refactor",
				Scope:         "core",
				Breaking:      true,
				CommitMessage: "extract methods",
			},
		},
		{
			input: "chore: update dependencies",
			expected: Message{
				Type:          "chore",
				Scope:         "",
				CommitMessage: "update dependencies",
			},
		},
		{
			input: "docs(readme): update instructions",
			expected: Message{
				Type:          "docs",
				Scope:         "readme",
				CommitMessage: "update instructions",
			},
		},
		{
			input: "style!: remove unused imports",
			expected: Message{
				Type:          "style",
				Scope:         "",
				Breaking:      true,
				CommitMessage: "remove unused imports",
			},
		},
		{
			input: "wrong format message",
			expected: Message{
				CommitMessage: "wrong format message",
			},
		},
	}

	for _, test := range tests {
		result := ParseMessage(test.input)
		if result != test.expected {
			t.Errorf("Parse(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}
