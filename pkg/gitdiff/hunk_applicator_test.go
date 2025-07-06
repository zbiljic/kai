package gitdiff

import (
	"strings"
	"testing"
)

func TestPreviewHunkApplication(t *testing.T) {
	hunksByID := map[string]*Hunk{
		"test.go:1-5": {
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
		"test.go:10-15": {
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
		"other.go:1-3": {
			ID:           "other.go:1-3",
			FilePath:     "other.go",
			StartLine:    1,
			EndLine:      3,
			Content:      "@@ -1,3 +1,3 @@\n content",
			Context:      "",
			Dependencies: make(map[string]bool),
			Dependents:   make(map[string]bool),
			ChangeType:   "modification",
		},
	}

	hunkIDs := []string{"test.go:1-5", "test.go:10-15", "other.go:1-3"}

	preview := PreviewHunkApplication(hunkIDs, hunksByID)

	// Check that preview contains expected content
	if !strings.Contains(preview, "File: test.go") {
		t.Error("Preview should contain test.go file")
	}
	if !strings.Contains(preview, "File: other.go") {
		t.Error("Preview should contain other.go file")
	}
	if !strings.Contains(preview, "test.go:1-5") {
		t.Error("Preview should contain first hunk")
	}
	if !strings.Contains(preview, "test.go:10-15") {
		t.Error("Preview should contain second hunk")
	}
	if !strings.Contains(preview, "other.go:1-3") {
		t.Error("Preview should contain third hunk")
	}
}

func TestPreviewHunkApplicationEmpty(t *testing.T) {
	hunksByID := map[string]*Hunk{}
	hunkIDs := []string{}

	preview := PreviewHunkApplication(hunkIDs, hunksByID)

	if preview != "No hunks selected." {
		t.Errorf("Expected 'No hunks selected.', got '%s'", preview)
	}
}

func TestExtractFilesFromPatch(t *testing.T) {
	patchContent := `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -10,7 +10,8 @@ func main() {
 	fmt.Println("Hello")
+	fmt.Println("World")
 	return 0
 }
diff --git a/other.go b/other.go
index 1234567..abcdefg 100644
--- a/other.go
+++ b/other.go
@@ -1,3 +1,4 @@
 func helper() {
 	return true
+	return false
 }`

	files := extractFilesFromPatch(patchContent)

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	if !files["test.go"] {
		t.Error("Expected test.go to be in affected files")
	}

	if !files["other.go"] {
		t.Error("Expected other.go to be in affected files")
	}
}

func TestParseHunkContent(t *testing.T) {
	hunk := &Hunk{
		ID:        "test.go:1-5",
		FilePath:  "test.go",
		StartLine: 1,
		EndLine:   5,
		Content: `@@ -1,5 +1,6 @@
 func main() {
 	fmt.Println("Hello")
+	fmt.Println("World")
 	return 0
 }`,
		Context:      "",
		Dependencies: make(map[string]bool),
		Dependents:   make(map[string]bool),
		ChangeType:   "modification",
	}

	additions, deletions, contextLines := parseHunkContent(hunk)

	if len(additions) != 1 {
		t.Errorf("Expected 1 addition, got %d", len(additions))
	}
	if additions[0] != `	fmt.Println("World")` {
		t.Errorf("Expected addition to be 'fmt.Println(\"World\")', got '%s'", additions[0])
	}

	if len(deletions) != 0 {
		t.Errorf("Expected 0 deletions, got %d", len(deletions))
	}

	if len(contextLines) != 4 {
		t.Errorf("Expected 4 context lines, got %d", len(contextLines))
	}
}

func TestTopologicalSortHunks(t *testing.T) {
	// Create hunks with dependencies
	hunk1 := &Hunk{
		ID:           "test.go:1-5",
		FilePath:     "test.go",
		StartLine:    1,
		EndLine:      5,
		Content:      "@@ -1,5 +1,5 @@\n content",
		Context:      "",
		Dependencies: make(map[string]bool),
		Dependents:   make(map[string]bool),
		ChangeType:   "modification",
	}
	hunk2 := &Hunk{
		ID:           "test.go:10-15",
		FilePath:     "test.go",
		StartLine:    10,
		EndLine:      15,
		Content:      "@@ -10,5 +10,5 @@\n content",
		Context:      "",
		Dependencies: make(map[string]bool),
		Dependents:   make(map[string]bool),
		ChangeType:   "modification",
	}
	hunk3 := &Hunk{
		ID:           "test.go:20-25",
		FilePath:     "test.go",
		StartLine:    20,
		EndLine:      25,
		Content:      "@@ -20,5 +20,5 @@\n content",
		Context:      "",
		Dependencies: make(map[string]bool),
		Dependents:   make(map[string]bool),
		ChangeType:   "modification",
	}

	// Set up dependencies: hunk2 depends on hunk1, hunk3 depends on hunk2
	hunk2.Dependencies[hunk1.ID] = true
	hunk1.Dependents[hunk2.ID] = true
	hunk3.Dependencies[hunk2.ID] = true
	hunk2.Dependents[hunk3.ID] = true

	hunks := []*Hunk{hunk1, hunk2, hunk3}

	ordered := topologicalSortHunks(hunks)

	if len(ordered) != 3 {
		t.Errorf("Expected 3 hunks in result, got %d", len(ordered))
	}

	// Check that dependencies come before dependents
	hunk1Index := -1
	hunk2Index := -1
	hunk3Index := -1

	for i, hunk := range ordered {
		switch hunk.ID {
		case hunk1.ID:
			hunk1Index = i
		case hunk2.ID:
			hunk2Index = i
		case hunk3.ID:
			hunk3Index = i
		}
	}

	if hunk1Index == -1 || hunk2Index == -1 || hunk3Index == -1 {
		t.Error("All hunks should be present in result")
	}

	if hunk1Index >= hunk2Index {
		t.Error("hunk1 should come before hunk2")
	}

	if hunk2Index >= hunk3Index {
		t.Error("hunk2 should come before hunk3")
	}
}

func TestTopologicalSortHunksWithCycle(t *testing.T) {
	// Create hunks with circular dependencies
	hunk1 := &Hunk{
		ID:           "test.go:1-5",
		FilePath:     "test.go",
		StartLine:    1,
		EndLine:      5,
		Content:      "@@ -1,5 +1,5 @@\n content",
		Context:      "",
		Dependencies: make(map[string]bool),
		Dependents:   make(map[string]bool),
		ChangeType:   "modification",
	}
	hunk2 := &Hunk{
		ID:           "test.go:10-15",
		FilePath:     "test.go",
		StartLine:    10,
		EndLine:      15,
		Content:      "@@ -10,5 +10,5 @@\n content",
		Context:      "",
		Dependencies: make(map[string]bool),
		Dependents:   make(map[string]bool),
		ChangeType:   "modification",
	}

	// Set up circular dependency
	hunk1.Dependencies[hunk2.ID] = true
	hunk2.Dependents[hunk1.ID] = true
	hunk2.Dependencies[hunk1.ID] = true
	hunk1.Dependents[hunk2.ID] = true

	hunks := []*Hunk{hunk1, hunk2}

	ordered := topologicalSortHunks(hunks)

	// Should return nil for circular dependencies
	if ordered != nil {
		t.Error("Expected nil result for circular dependencies")
	}
}

func TestApplyHunksValidation(t *testing.T) {
	hunksByID := map[string]*Hunk{
		"test.go:1-5": {
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
	}

	// Test with non-existent hunk ID
	err := ApplyHunks([]string{"non-existent"}, hunksByID, "")
	if err == nil {
		t.Error("Expected error for non-existent hunk ID")
	}

	if !strings.Contains(err.Error(), "hunk ID not found") {
		t.Errorf("Expected 'hunk ID not found' error, got: %s", err.Error())
	}
}

func TestApplyHunksEmpty(t *testing.T) {
	hunksByID := map[string]*Hunk{}

	err := ApplyHunks([]string{}, hunksByID, "")
	if err != nil {
		t.Errorf("Expected no error for empty hunk list, got: %v", err)
	}
}
