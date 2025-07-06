package gitdiff

import (
	"strings"
	"testing"
)

func TestParseDiff(t *testing.T) {
	diffOutput := `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -10,7 +10,8 @@ func main() {
 	fmt.Println("Hello")
+	fmt.Println("World")
 	return 0
 }
@@ -20,3 +21,4 @@ func helper() {
 	return true
+	return false
 }`

	hunks, err := ParseDiff(diffOutput)
	if err != nil {
		t.Fatalf("Failed to parse diff: %s", err)
	}

	if len(hunks) != 2 {
		t.Errorf("Expected 2 hunks, got %d", len(hunks))
	}

	// Check first hunk
	if hunks[0].FilePath != "test.go" {
		t.Errorf("Expected file path 'test.go', got '%s'", hunks[0].FilePath)
	}

	if hunks[0].StartLine != 10 {
		t.Errorf("Expected start line 10, got %d", hunks[0].StartLine)
	}

	if hunks[0].EndLine != 17 {
		t.Errorf("Expected end line 17, got %d", hunks[0].EndLine)
	}

	// Check second hunk
	if hunks[1].StartLine != 21 {
		t.Errorf("Expected start line 21, got %d", hunks[1].StartLine)
	}

	if hunks[1].EndLine != 24 {
		t.Errorf("Expected end line 24, got %d", hunks[1].EndLine)
	}
}

func TestAnalyzeHunkChangeType(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "addition",
			content:  "@@ -10,7 +10,8 @@\n fmt.Println(\"Hello\")\n+fmt.Println(\"World\")\n return 0",
			expected: "addition",
		},
		{
			name:     "deletion",
			content:  "@@ -10,8 +10,7 @@\n fmt.Println(\"Hello\")\n-fmt.Println(\"World\")\n return 0",
			expected: "deletion",
		},
		{
			name:     "modification",
			content:  "@@ -10,8 +10,8 @@\n fmt.Println(\"Hello\")\n-fmt.Println(\"Old\")\n+fmt.Println(\"New\")\n return 0",
			expected: "modification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzeHunkChangeType(tt.content)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestValidateHunkCombination(t *testing.T) {
	// Test non-overlapping hunks
	hunks := []*Hunk{
		{
			ID:           "test.go:1-5",
			FilePath:     "test.go",
			StartLine:    1,
			EndLine:      5,
			Content:      "@@ -1,5 +1,5 @@\n content",
			Context:      "",
			Dependencies: make(map[string]bool),
			Dependents:   make(map[string]bool),
			ChangeType:   "modification",
		},
		{
			ID:           "test.go:10-15",
			FilePath:     "test.go",
			StartLine:    10,
			EndLine:      15,
			Content:      "@@ -10,5 +10,5 @@\n content",
			Context:      "",
			Dependencies: make(map[string]bool),
			Dependents:   make(map[string]bool),
			ChangeType:   "modification",
		},
	}

	err := ValidateHunkCombination(hunks)
	if err != nil {
		t.Errorf("Expected valid combination, got error: %s", err.Error())
	}

	// Test overlapping hunks
	overlappingHunks := []*Hunk{
		{
			ID:           "test.go:1-10",
			FilePath:     "test.go",
			StartLine:    1,
			EndLine:      10,
			Content:      "@@ -1,10 +1,10 @@\n content",
			Context:      "",
			Dependencies: make(map[string]bool),
			Dependents:   make(map[string]bool),
			ChangeType:   "modification",
		},
		{
			ID:           "test.go:5-15",
			FilePath:     "test.go",
			StartLine:    5,
			EndLine:      15,
			Content:      "@@ -5,10 +5,10 @@\n content",
			Context:      "",
			Dependencies: make(map[string]bool),
			Dependents:   make(map[string]bool),
			ChangeType:   "modification",
		},
	}

	err = ValidateHunkCombination(overlappingHunks)
	if err == nil {
		t.Error("Expected invalid combination for overlapping hunks")
	}
	if !strings.Contains(err.Error(), "overlapping hunks") {
		t.Errorf("Expected overlap error message, got: %s", err.Error())
	}
}

func TestCreateHunkPatch(t *testing.T) {
	baseDiff := `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -10,7 +10,8 @@ func main() {
 	fmt.Println("Hello")
+	fmt.Println("World")
 	return 0
 }`

	hunks, err := ParseDiff(baseDiff)
	if err != nil {
		t.Fatalf("Failed to parse diff: %s", err)
	}

	if len(hunks) != 1 {
		t.Fatalf("Expected 1 hunk, got %d", len(hunks))
	}

	patch := CreateHunkPatch(hunks, baseDiff)
	if !strings.Contains(patch, "diff --git") {
		t.Error("Generated patch should contain diff header")
	}
	if !strings.Contains(patch, "@@") {
		t.Error("Generated patch should contain hunk header")
	}
	// The patch should contain the original content, not just the addition line
	if !strings.Contains(patch, "fmt.Println(\"World\")") {
		t.Error("Generated patch should contain the addition")
	}
}

func TestValidatePatchFormat(t *testing.T) {
	validPatch := `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -10,7 +10,8 @@ func main() {
 	fmt.Println("Hello")
+	fmt.Println("World")
 	return 0
 }`

	if !ValidatePatchFormat(validPatch) {
		t.Error("Valid patch should pass validation")
	}

	invalidPatch := `This is not a valid patch`
	if ValidatePatchFormat(invalidPatch) {
		t.Error("Invalid patch should fail validation")
	}

	emptyPatch := ``
	if ValidatePatchFormat(emptyPatch) {
		t.Error("Empty patch should fail validation")
	}
}
