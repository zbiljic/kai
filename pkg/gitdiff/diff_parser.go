package gitdiff

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Hunk represents an individual hunk (change block) from a git diff.
type Hunk struct {
	ID           string          `json:"id"`
	FilePath     string          `json:"file_path"`
	StartLine    int             `json:"start_line"`
	EndLine      int             `json:"end_line"`
	Content      string          `json:"content"`
	Context      string          `json:"context"`
	Dependencies map[string]bool `json:"dependencies"` // Hunk IDs this depends on
	Dependents   map[string]bool `json:"dependents"`   // Hunk IDs that depend on this
	ChangeType   string          `json:"change_type"`  // addition, deletion, modification
	IsNewFile    bool            `json:"is_new_file"`  // true if the file is newly added
}

// ParseDiff parses git diff output into individual hunks.
func ParseDiff(diffOutput string) ([]*Hunk, error) {
	contextLines := 3

	hunks := []*Hunk{}
	currentFile := ""
	isNewFile := false

	lines := strings.Split(diffOutput, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Check for file header using switch statement
		switch {
		case strings.HasPrefix(line, "diff --git"):
			// Extract file path from the diff header
			// A diff header line can look like:
			//   diff --git a/path/to/file b/path/to/file   (default)
			//   diff --git c/path/to/file w/path/to/file   (mnemonic prefixes)
			//   diff --git path/to/file path/to/file       (no prefixes, e.g. diff.noprefix)
			// We capture the two pathname fields and later strip any leading single-letter
			// prefix ("a/", "b/", "c/", "i/", "w/", etc.) if present.
			//
			// Git diff prefixes:
			// a/ - Original file (before changes)
			// b/ - Modified file (after changes) - default
			// c/ - Index/staging area version
			// w/ - Working directory version
			// i/ - Index/staging area version (alternative to c/)
			// o/ - Original version (alternative to a/)
			//
			re := regexp.MustCompile(`^diff --git\s+([^\s]+)\s+([^\s]+)$`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 3 {
				currentFile = stripGitDiffPrefix(matches[2]) // Use the 'b/' path (after changes)
				isNewFile = false                            // reset for this file
			}
		case strings.HasPrefix(line, "new file mode"):
			// Mark that the upcoming hunks belong to a newly added file
			isNewFile = true
		case strings.HasPrefix(line, "@@") && currentFile != "":
			// Check for hunk header
			re := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 5 {
				newStart, err := strconv.Atoi(matches[3])
				if err != nil {
					return nil, fmt.Errorf("failed to parse hunk start line: %w", err)
				}

				newCount := 1
				if matches[4] != "" {
					newCount, err = strconv.Atoi(matches[4])
					if err != nil {
						return nil, fmt.Errorf("failed to parse hunk count: %w", err)
					}
				}

				// Collect hunk content
				hunkContentLines := []string{line} // Include the @@ line
				i++

				// Read until next file header, hunk header, or end
			readLoop:
				for i < len(lines) {
					nextLine := lines[i]
					switch {
					case strings.HasPrefix(nextLine, "diff --git") || strings.HasPrefix(nextLine, "@@"):
						break readLoop
					case strings.HasPrefix(nextLine, "\\") && strings.Contains(nextLine, "No newline"):
						// NOTE: Only include "No newline at end of file" markers if they're legitimate
						// Validate that the previous line is actual content, not just part of malformed patch
						if len(hunkContentLines) > 1 {
							prevLine := hunkContentLines[len(hunkContentLines)-1]
							// Check if previous line is actual file content (starts with +, -, or space)
							if len(prevLine) > 0 && (prevLine[0] == '+' || prevLine[0] == '-' || prevLine[0] == ' ') {
								hunkContentLines = append(hunkContentLines, nextLine)
								i++
								break readLoop
							} else {
								debugLog("Skipping suspicious 'No newline' marker after non-content line: %s\n", prevLine)
								i++
								break readLoop
							}
						} else {
							debugLog("Skipping 'No newline' marker in hunk with insufficient content\n")
							i++
							break readLoop
						}
					default:
						hunkContentLines = append(hunkContentLines, nextLine)
						i++
					}
				}

				// Create hunk
				hunkContent := strings.Join(hunkContentLines, "\n")

				// Calculate line range for the hunk ID
				// Use the new file line numbers for the range
				endLine := newStart + int(math.Max(0, float64(newCount-1)))

				hunkID := fmt.Sprintf("%s:%d-%d", currentFile, newStart, endLine)

				// Get context around the hunk
				context := getHunkContext(currentFile, newStart, endLine, contextLines)

				// Analyze change type
				changeType := analyzeHunkChangeType(hunkContent)

				hunk := &Hunk{
					ID:           hunkID,
					FilePath:     currentFile,
					StartLine:    newStart,
					EndLine:      endLine,
					Content:      hunkContent,
					Context:      context,
					Dependencies: make(map[string]bool),
					Dependents:   make(map[string]bool),
					ChangeType:   changeType,
					IsNewFile:    isNewFile,
				}

				hunks = append(hunks, hunk)
				continue // Don't increment i, we already did it in the while loop
			}
		}

		i++
	}

	// Analyze dependencies between hunks
	analyzeHunkDependencies(hunks)

	return hunks, nil
}

// getHunkContext extracts surrounding code context for better AI understanding.
func getHunkContext(filePath string, startLine, endLine, contextLines int) string {
	file, err := os.Open(filePath)
	if err != nil {
		// If we can't read the file, return minimal context
		return fmt.Sprintf("File: %s (lines %d-%d)", filePath, startLine, endLine)
	}
	defer file.Close()

	var fileLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fileLines = append(fileLines, scanner.Text())
	}

	// Calculate context boundaries
	contextStart := int(math.Max(0, float64(startLine-contextLines-1))) // -1 for 0-based indexing
	contextEnd := int(math.Min(float64(len(fileLines)), float64(endLine+contextLines)))

	// Extract context lines
	var numberedLines []string
	for i := contextStart; i < contextEnd; i++ {
		lineNum := i + 1
		prefix := "    "
		if startLine <= lineNum && lineNum <= endLine {
			prefix = ">>> "
		}
		numberedLines = append(numberedLines, fmt.Sprintf("%s%4d: %s", prefix, lineNum, fileLines[i]))
	}

	return strings.Join(numberedLines, "\n")
}

// createHunkPatch creates a patch file containing only the specified hunks using ABSOLUTE MINIMAL modification.
func createHunkPatch(hunks []*Hunk, baseDiff string) string {
	if len(hunks) == 0 {
		return ""
	}

	// Use the absolutely minimal patch creation logic
	return createAbsolutelyMinimalPatch(hunks, baseDiff)
}

// createAbsolutelyMinimalPatch creates a patch with ABSOLUTELY MINIMAL
// modification that NEVER modifies original content.
func createAbsolutelyMinimalPatch(hunks []*Hunk, baseDiff string) string {
	if len(hunks) == 0 {
		return ""
	}

	// Parse the base diff to extract original raw content
	originalHunksMap := extractOriginalHunksRaw(baseDiff)
	originalHeaders := extractOriginalHeaders(baseDiff)

	// Group hunks by file
	hunksByFile := make(map[string][]*Hunk)
	for _, hunk := range hunks {
		hunksByFile[hunk.FilePath] = append(hunksByFile[hunk.FilePath], hunk)
	}

	// Build patch by direct reassembly
	var patchParts []string

	for filePath, fileHunks := range hunksByFile {
		// Add original file header
		if headers, exists := originalHeaders[filePath]; exists {
			patchParts = append(patchParts, headers...)
		} else {
			// Determine if this is a newly-added file (all hunks flagged as new file)
			isNewFile := true
			for _, h := range fileHunks {
				if !h.IsNewFile {
					isNewFile = false
					break
				}
			}

			// Construct a minimal but valid fallback header
			patchParts = append(patchParts, fmt.Sprintf("diff --git a/%s b/%s", filePath, filePath))

			if isNewFile {
				// Provide mandatory markers for new files so git create them properly
				patchParts = append(patchParts,
					"new file mode 100644",   // default mode
					"index 0000000..0000000", // placeholder SHAs (git will ignore)
					"--- /dev/null",
					fmt.Sprintf("+++ b/%s", filePath),
				)
			} else {
				// Existing file fallback header
				patchParts = append(patchParts,
					fmt.Sprintf("--- a/%s", filePath),
					fmt.Sprintf("+++ b/%s", filePath),
				)
			}
		}

		// Add hunks in original order, using EXACT original content
		sort.Slice(fileHunks, func(i, j int) bool {
			return fileHunks[i].StartLine < fileHunks[j].StartLine
		})

		for _, hunk := range fileHunks {
			// Use the EXACT original hunk content without ANY modifications
			if originalContent, exists := originalHunksMap[hunk.ID]; exists {
				patchParts = append(patchParts, originalContent)
			} else {
				// Fallback: use hunk content as-is (this should rarely happen)
				patchParts = append(patchParts, hunk.Content)
			}
		}
	}

	// Assemble final patch with minimal processing
	result := strings.Join(patchParts, "\n")

	// Only add final newline if the patch doesn't already end with one
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result
}

// extractOriginalHunksRaw extracts the original hunk content exactly as it appears in the base diff.
func extractOriginalHunksRaw(baseDiff string) map[string]string {
	hunksMap := make(map[string]string)
	lines := strings.Split(baseDiff, "\n")
	i := 0
	currentFile := ""

	for i < len(lines) {
		line := lines[i]

		// Track current file
		if strings.HasPrefix(line, "diff --git") {
			re := regexp.MustCompile(`^diff --git a/(.*) b/(.*)$`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 3 {
				currentFile = stripGitDiffPrefix(matches[2])
			}
		} else if strings.HasPrefix(line, "@@") && currentFile != "" {
			// Found a hunk header
			re := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 5 {
				newStart, _ := strconv.Atoi(matches[3])
				newCount := 1
				if matches[4] != "" {
					newCount, _ = strconv.Atoi(matches[4])
				}

				// Calculate end line for hunk ID
				endLine := newStart + int(math.Max(0, float64(newCount-1)))
				hunkID := fmt.Sprintf("%s:%d-%d", currentFile, newStart, endLine)

				// Extract the complete hunk content
				i++
				hunkLines := []string{line} // Start with the @@ line

				// Read until next hunk, file, or end
				for i < len(lines) {
					nextLine := lines[i]
					if strings.HasPrefix(nextLine, "diff --git") || strings.HasPrefix(nextLine, "@@") {
						break
					}

					hunkLines = append(hunkLines, nextLine)
					i++
				}

				// Store the EXACT original content
				hunksMap[hunkID] = strings.Join(hunkLines, "\n")
				continue
			}
		}

		i++
	}

	return hunksMap
}

// extractOriginalHeaders extracts original file headers from the base diff.
func extractOriginalHeaders(baseDiff string) map[string][]string {
	headers := make(map[string][]string)
	lines := strings.Split(baseDiff, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]
		if strings.HasPrefix(line, "diff --git") {
			// Extract file path
			re := regexp.MustCompile(`^diff --git a/(.*) b/(.*)$`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 3 {
				// Normalize by stripping any single-letter git diff prefix (e.g., "b/")
				filePath := stripGitDiffPrefix(matches[2])
				headerLines := []string{line}
				i++

				// Collect header lines until first @@
				for i < len(lines) && !strings.HasPrefix(lines[i], "@@") {
					if strings.HasPrefix(lines[i], "diff --git") {
						break
					}
					headerLines = append(headerLines, lines[i])
					i++
				}

				headers[filePath] = headerLines
				continue
			}
		}
		i++
	}

	return headers
}

// analyzeHunkChangeType analyzes the type of change in a hunk to help with dependency detection.
func analyzeHunkChangeType(hunkContent string) string {
	lines := strings.Split(hunkContent, "\n")
	// Count different types of changes
	additions := 0
	deletions := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}
	// Determine change type based on addition/deletion ratio
	switch {
	case deletions == 0 && additions > 0:
		return "addition"
	case additions == 0 && deletions > 0:
		return "deletion"
	default:
		return "modification"
	}
}

// analyzeHunkDependencies analyzes dependencies between hunks to enable intelligent grouping.
func analyzeHunkDependencies(hunks []*Hunk) {
	// Build maps for quick lookups
	hunksByFile := make(map[string][]*Hunk)
	for _, hunk := range hunks {
		hunksByFile[hunk.FilePath] = append(hunksByFile[hunk.FilePath], hunk)
	}

	// Analyze dependencies
	for _, hunk := range hunks {
		// 1. Line number dependencies (hunks that affect each other's line numbers)
		for _, otherHunk := range hunksByFile[hunk.FilePath] {
			if otherHunk.ID != hunk.ID {
				// Check if this hunk's line numbers depend on the other hunk
				if hunksHaveLineDependencies(hunk, otherHunk) {
					if otherHunk.StartLine < hunk.StartLine {
						// This hunk depends on the earlier hunk
						hunk.Dependencies[otherHunk.ID] = true
						otherHunk.Dependents[hunk.ID] = true
					}
				}
			}
		}
		// 2. Same file proximity dependencies (changes in the same file that are close together)
		for _, otherHunk := range hunksByFile[hunk.FilePath] {
			if otherHunk.ID != hunk.ID {
				// If hunks are very close (within 10 lines), they might be related
				lineDistance := int(math.Abs(float64(hunk.StartLine - otherHunk.StartLine)))
				if lineDistance <= 10 {
					// Create weak dependencies for same-file proximity
					if hunk.StartLine > otherHunk.StartLine {
						hunk.Dependencies[otherHunk.ID] = true
						otherHunk.Dependents[hunk.ID] = true
					}
				}
			}
		}
	}
}

// hunksHaveLineDependencies checks if two hunks have line number dependencies.
func hunksHaveLineDependencies(hunk1, hunk2 *Hunk) bool {
	// Only check hunks in the same file
	if hunk1.FilePath != hunk2.FilePath {
		return false
	}

	// Parse hunk headers to understand line number changes
	adds2, dels2 := countHunkChanges(hunk2)

	// Calculate net line change (positive = file grows, negative = file shrinks)
	netChange2 := adds2 - dels2

	// If hunk2 changes the file size and comes before hunk1,
	// then hunk1's line numbers are affected by hunk2
	if hunk2.StartLine < hunk1.StartLine && netChange2 != 0 {
		return true
	}

	// Check for overlapping line ranges that would affect each other
	range1Start := hunk1.StartLine
	range1End := hunk1.EndLine
	range2Start := hunk2.StartLine
	range2End := hunk2.EndLine

	// If ranges overlap or are very close, they likely depend on each other
	if (range1Start <= range2End && range1End >= range2Start) ||
		int(math.Abs(float64(range1Start-range2End))) <= 3 || int(math.Abs(float64(range2Start-range1End))) <= 3 {
		return true
	}

	return false
}

// countHunkChanges counts additions and deletions in a hunk.
func countHunkChanges(hunk *Hunk) (int, int) {
	lines := strings.Split(hunk.Content, "\n")
	additions := 0
	deletions := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}

	return additions, deletions
}

// ValidateHunkCombination validates that a combination of hunks can
// be applied together.
func ValidateHunkCombination(hunks []*Hunk) error {
	if len(hunks) == 0 {
		return nil
	}

	// Group by file and check for overlaps
	hunksByFile := make(map[string][]*Hunk)
	for _, hunk := range hunks {
		hunksByFile[hunk.FilePath] = append(hunksByFile[hunk.FilePath], hunk)
	}

	for filePath, fileHunks := range hunksByFile {
		// Sort hunks by start line
		sort.Slice(fileHunks, func(i, j int) bool {
			return fileHunks[i].StartLine < fileHunks[j].StartLine
		})

		// Check for overlapping hunks
		for i := 0; i < len(fileHunks)-1; i++ {
			currentHunk := fileHunks[i]
			nextHunk := fileHunks[i+1]

			if currentHunk.EndLine >= nextHunk.StartLine {
				return fmt.Errorf("overlapping hunks in %s: %s and %s", filePath, currentHunk.ID, nextHunk.ID)
			}
		}
	}

	return nil
}

// CreateDependencyGroups groups hunks based on their dependencies for atomic
// application.
func CreateDependencyGroups(hunks []*Hunk) [][]*Hunk {
	// Start with all hunks ungrouped
	ungrouped := make(map[string]bool)
	hunkMap := make(map[string]*Hunk)

	for _, hunk := range hunks {
		ungrouped[hunk.ID] = true
		hunkMap[hunk.ID] = hunk
	}

	var groups [][]*Hunk

	for len(ungrouped) > 0 {
		// Start a new group with a hunk that has no ungrouped dependencies
		var groupSeeds []string
		for hunkID := range ungrouped {
			hunk := hunkMap[hunkID]
			ungroupedDeps := 0
			for depID := range hunk.Dependencies {
				if ungrouped[depID] {
					ungroupedDeps++
				}
			}
			if ungroupedDeps == 0 {
				groupSeeds = append(groupSeeds, hunkID)
			}
		}

		if len(groupSeeds) == 0 {
			// If no seeds found, we have circular dependencies - break by picking the first one
			for hunkID := range ungrouped {
				groupSeeds = []string{hunkID}
				break
			}
		}

		// Build a group starting from a seed
		currentGroup := make(map[string]bool)
		toProcess := []string{groupSeeds[0]}

		for len(toProcess) > 0 {
			currentID := toProcess[0]
			toProcess = toProcess[1:]

			if ungrouped[currentID] && !currentGroup[currentID] {
				currentGroup[currentID] = true
				hunk := hunkMap[currentID]

				// Add all dependents that are still ungrouped
				for dependentID := range hunk.Dependents {
					if ungrouped[dependentID] && !currentGroup[dependentID] {
						toProcess = append(toProcess, dependentID)
					}
				}

				// Add dependencies that are still ungrouped
				for depID := range hunk.Dependencies {
					if ungrouped[depID] && !currentGroup[depID] {
						toProcess = append(toProcess, depID)
					}
				}
			}
		}

		// Convert group to list of hunks
		var groupHunks []*Hunk
		for hunkID := range currentGroup {
			groupHunks = append(groupHunks, hunkMap[hunkID])
		}
		groups = append(groups, groupHunks)

		// Remove grouped hunks from ungrouped
		for hunkID := range currentGroup {
			delete(ungrouped, hunkID)
		}
	}

	return groups
}

// CreateHunkPatch creates a patch file containing only the specified hunks.
// This is the public interface for creating patches from hunks.
func CreateHunkPatch(hunks []*Hunk, baseDiff string) string {
	return createHunkPatch(hunks, baseDiff)
}

// ValidatePatchFormat validates that a patch has proper git patch format.
func ValidatePatchFormat(patchContent string) bool {
	if strings.TrimSpace(patchContent) == "" {
		return false
	}

	lines := strings.Split(patchContent, "\n")

	// Must start with diff header
	hasDiffHeader := false
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			hasDiffHeader = true
			break
		}
	}
	if !hasDiffHeader {
		return false
	}

	// Must have proper hunk headers
	hunkCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			hunkCount++
		}
	}
	if hunkCount == 0 {
		return false
	}

	// Check for malformed hunk headers
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// Hunk header should match pattern: @@ -old_start,old_count +new_start,new_count @@
			if strings.Count(line, "@@") < 2 {
				return false
			}
			if !strings.Contains(line, "-") || !strings.Contains(line, "+") {
				return false
			}
		}
	}

	return true
}
