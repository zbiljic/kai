package llm

import (
	"strings"
	"testing"
)

func TestMapToString(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]string
		expected []string
	}{
		{
			name:     "Empty map",
			input:    map[string]string{},
			expected: []string{},
		},
		{
			name: "Single entry",
			input: map[string]string{
				"feat": "A new feature",
			},
			expected: []string{`- "feat": "A new feature"`},
		},
		{
			name: "Multiple entries",
			input: map[string]string{
				"feat": "A new feature",
				"fix":  "A bug fix",
			},
			expected: []string{
				`- "feat": "A new feature"`,
				`- "fix": "A bug fix"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapToString(tc.input)

			for _, expected := range tc.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, but got %q", expected, result)
				}
			}

			// Check that we have the right number of lines
			if len(tc.expected) > 0 {
				lines := strings.Split(strings.TrimSpace(result), "\n")
				if len(lines) != len(tc.expected) {
					t.Errorf("Expected %d lines, but got %d lines", len(tc.expected), len(lines))
				}
			}
		})
	}
}

func TestMapToTable(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]string
		expected []string
	}{
		{
			name:  "Empty map",
			input: map[string]string{},
			expected: []string{
				"| Type | Description |",
				"| ---- | ----------- |",
			},
		},
		{
			name: "Single entry",
			input: map[string]string{
				"feat": "A new feature",
			},
			expected: []string{
				"| Type | Description |",
				"| ---- | ----------- |",
				"| feat | A new feature |",
			},
		},
		{
			name: "Multiple entries",
			input: map[string]string{
				"feat": "A new feature",
				"fix":  "A bug fix",
			},
			expected: []string{
				"| Type | Description |",
				"| ---- | ----------- |",
				"| feat | A new feature |",
				"| fix | A bug fix |",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapToTable(tc.input)

			for _, expected := range tc.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, but got %q", expected, result)
				}
			}

			// Check that we have the right number of lines for the table
			// Header (2 lines) + data rows
			expectedLines := 2 + len(tc.input)
			lines := strings.Split(strings.TrimSpace(result), "\n")
			if len(lines) != expectedLines {
				t.Errorf("Expected %d lines, but got %d lines", expectedLines, len(lines))
			}
		})
	}
}
