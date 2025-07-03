package absorb

import (
	"strconv"
	"strings"
	"testing"
)

// LineRange represents a range of lines in a file that have been modified
type LineRange struct {
	Start int
	End   int
}

func TestParseModifiedLineRanges(t *testing.T) {
	tests := []struct {
		name           string
		diffOutput     string
		expectedRanges []LineRange
	}{
		{
			name: "Single line modification",
			diffOutput: `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -10,1 +10,1 @@ func test() {
-	old line
+	new line`,
			expectedRanges: []LineRange{{Start: 10, End: 10}},
		},
		{
			name: "Multiple line addition",
			diffOutput: `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -5,0 +5,3 @@ func test() {
+	line 1
+	line 2
+	line 3`,
			expectedRanges: []LineRange{{Start: 5, End: 7}},
		},
		{
			name: "Multiple separate ranges",
			diffOutput: `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -5,1 +5,2 @@ func test() {
-	old line
+	new line 1
+	new line 2
@@ -20,1 +21,1 @@ func another() {
-	another old line
+	another new line`,
			expectedRanges: []LineRange{
				{Start: 5, End: 6},
				{Start: 21, End: 21},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ranges := parseModifiedLineRangesFromDiff(tt.diffOutput)

			if len(ranges) != len(tt.expectedRanges) {
				t.Errorf("expected %d ranges, got %d", len(tt.expectedRanges), len(ranges))
				return
			}

			for i, expected := range tt.expectedRanges {
				if ranges[i].Start != expected.Start || ranges[i].End != expected.End {
					t.Errorf("range %d: expected {%d, %d}, got {%d, %d}",
						i, expected.Start, expected.End, ranges[i].Start, ranges[i].End)
				}
			}
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1234567890abcdef", true},
		{"1234567890ABCDEF", true},
		{"1234567890abcdefghijk", false},
		{"", false},
		{"123xyz", false},
		{"a1b2c3d4e5f67890123456789012345678901234", true}, // 40 chars
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isHexString(tt.input)
			if result != tt.expected {
				t.Errorf("isHexString(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function to parse diff output directly (for testing)
func parseModifiedLineRangesFromDiff(diffOutput string) []LineRange {
	ranges := []LineRange{}
	lines := strings.Split(diffOutput, "\n")

	for _, line := range lines {
		// Look for lines like "@@ -10,3 +10,4 @@" which indicate line ranges
		if strings.HasPrefix(line, "@@") {
			// Parse the line range from the diff header
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				// Extract the new file range (e.g., "+10,4" means starting at line 10, 4 lines)
				newRange := parts[2]
				if strings.HasPrefix(newRange, "+") {
					newRange = newRange[1:] // Remove the '+' prefix
					rangeParts := strings.Split(newRange, ",")
					if len(rangeParts) >= 1 {
						start, err := strconv.Atoi(rangeParts[0])
						if err != nil {
							continue
						}

						count := 1
						if len(rangeParts) > 1 {
							if c, err := strconv.Atoi(rangeParts[1]); err == nil {
								count = c
							}
						}

						if count > 0 {
							ranges = append(ranges, LineRange{
								Start: start,
								End:   start + count - 1,
							})
						}
					}
				}
			}
		}
	}

	return ranges
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, char := range s {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}
