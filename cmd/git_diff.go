package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zbiljic/kai/pkg/llm"
)

// parseDiffIntoHunks parses git diff output into structured hunks
func parseDiffIntoHunks(diff string) ([]llm.Hunk, error) {
	var hunks []llm.Hunk

	// Regex patterns for parsing diff
	fileHeaderRegex := regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)

	lines := strings.Split(diff, "\n")
	var currentFile string
	var currentHunk *llm.Hunk
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

			currentHunk = &llm.Hunk{
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
func createHunkMap(hunks []llm.Hunk) map[string]*llm.Hunk {
	hunkMap := make(map[string]*llm.Hunk)
	for i := range hunks {
		hunkMap[hunks[i].ID] = &hunks[i]
	}
	return hunkMap
}

// validateHunkReferences validates that all referenced hunk IDs exist in the hunk map
func validateHunkReferences(hunkIDs []string, hunkMap map[string]*llm.Hunk) error {
	for _, hunkID := range hunkIDs {
		if _, exists := hunkMap[hunkID]; !exists {
			return fmt.Errorf("hunk %s not found in parsed hunks", hunkID)
		}
	}
	return nil
}
