package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Hunk represents a chunk of code changes from a git diff
type Hunk struct {
	ID         string `json:"id"`
	FilePath   string `json:"file_path"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	Content    string `json:"content"`
	Context    string `json:"context"`
	ChangeType string `json:"change_type"`
}

// CommitPlan represents the AI-generated commit reorganization plan
type CommitPlan struct {
	Commits []PlannedCommit `json:"commits"`
}

// PlannedCommit represents a single commit in the reorganization plan
type PlannedCommit struct {
	Message   string   `json:"message"`
	HunkIDs   []string `json:"hunk_ids"`
	Rationale string   `json:"rationale"`
}

// parseDiffIntoHunks parses git diff output into structured hunks
func parseDiffIntoHunks(diff string) ([]Hunk, error) {
	var hunks []Hunk

	// Regex patterns for parsing diff
	fileHeaderRegex := regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)

	lines := strings.Split(diff, "\n")
	var currentFile string
	var currentHunk *Hunk
	hunkCounter := 0

	for _, line := range lines {
		// Check for file header
		if matches := fileHeaderRegex.FindStringSubmatch(line); matches != nil {
			currentFile = matches[2] // Use the new file path
			continue
		}

		// Check for hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Finalize previous hunk if exists
			if currentHunk != nil {
				hunks = append(hunks, *currentHunk)
			}

			hunkCounter++
			startLine, _ := strconv.Atoi(matches[3])
			endLine := startLine
			if matches[4] != "" {
				count, _ := strconv.Atoi(matches[4])
				endLine = startLine + count - 1
			}

			currentHunk = &Hunk{
				ID:        fmt.Sprintf("%s:%d-%d", currentFile, startLine, endLine),
				FilePath:  currentFile,
				StartLine: startLine,
				EndLine:   endLine,
				Content:   "",
				Context:   strings.TrimSpace(matches[5]),
			}
			continue
		}

		// Add content to current hunk
		if currentHunk != nil && (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ")) {
			if currentHunk.Content != "" {
				currentHunk.Content += "\n"
			}
			currentHunk.Content += line

			// Determine change type
			if strings.HasPrefix(line, "+") {
				switch currentHunk.ChangeType {
				case "":
					currentHunk.ChangeType = "addition"
				case "deletion":
					currentHunk.ChangeType = "modification"
				}
			} else if strings.HasPrefix(line, "-") {
				switch currentHunk.ChangeType {
				case "":
					currentHunk.ChangeType = "deletion"
				case "addition":
					currentHunk.ChangeType = "modification"
				}
			}
		}
	}

	// Add the last hunk if exists
	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks, nil
}

// createHunkMap creates a lookup map from hunk ID to hunk pointer
func createHunkMap(hunks []Hunk) map[string]*Hunk {
	hunkMap := make(map[string]*Hunk)
	for i := range hunks {
		hunkMap[hunks[i].ID] = &hunks[i]
	}
	return hunkMap
}

// validateHunkReferences validates that all referenced hunk IDs exist in the hunk map
func validateHunkReferences(hunkIDs []string, hunkMap map[string]*Hunk) error {
	for _, hunkID := range hunkIDs {
		if _, exists := hunkMap[hunkID]; !exists {
			return fmt.Errorf("hunk %s not found in parsed hunks", hunkID)
		}
	}
	return nil
}

// extractJSONFromResponse extracts JSON content from AI responses that may be wrapped in markdown
func extractJSONFromResponse(response string) string {
	// Remove leading/trailing whitespace
	response = strings.TrimSpace(response)

	// Check if response is wrapped in markdown code block
	if strings.HasPrefix(response, "```") {
		// Find the first ``` and the closing ```
		lines := strings.Split(response, "\n")
		var jsonLines []string
		inCodeBlock := false

		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				if inCodeBlock {
					// End of code block, stop collecting
					break
				} else {
					// Start of code block, start collecting from next line
					inCodeBlock = true
					continue
				}
			}

			if inCodeBlock {
				jsonLines = append(jsonLines, line)
			}
		}

		if len(jsonLines) > 0 {
			return strings.Join(jsonLines, "\n")
		}
	}

	// If no markdown wrapper found, return original response
	return response
}
