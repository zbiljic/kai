package commit

import "testing"

func TestMessageToString(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name: "omits empty scope",
			message: Message{
				Type:          "refactor",
				Scope:         "",
				CommitMessage: "simplify configuration",
			},
			expected: "refactor: simplify configuration",
		},
		{
			name: "omits whitespace-only scope",
			message: Message{
				Type:          "refactor",
				Scope:         "   ",
				CommitMessage: "simplify configuration",
			},
			expected: "refactor: simplify configuration",
		},
		{
			name: "keeps comma-separated scope",
			message: Message{
				Type:          "refactor",
				Scope:         "api,client",
				CommitMessage: "simplify configuration",
			},
			expected: "refactor(api,client): simplify configuration",
		},
		{
			name: "scope suffix marks breaking change",
			message: Message{
				Type:          "refactor",
				Scope:         "api!",
				CommitMessage: "simplify configuration",
			},
			expected: "refactor(api)!: simplify configuration",
		},
		{
			name: "scope suffix without scope omits scope",
			message: Message{
				Type:          "refactor",
				Scope:         "!",
				CommitMessage: "simplify configuration",
			},
			expected: "refactor!: simplify configuration",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.message.ToString()
			if result != test.expected {
				t.Errorf("ToString() = %q; want %q", result, test.expected)
			}
		})
	}
}
