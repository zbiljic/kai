package gitdiff

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ApplyHunks applies specific hunks to the git staging area using
// dependency-aware grouping.
func ApplyHunks(hunkIDs []string, hunksByID map[string]*Hunk, baseDiff string) error {
	if len(hunkIDs) == 0 {
		return nil
	}

	// Get the hunks to apply
	hunksToApply := []*Hunk{}
	for _, hunkID := range hunkIDs {
		hunk, exists := hunksByID[hunkID]
		if !exists {
			return fmt.Errorf("hunk ID not found: %s", hunkID)
		}
		hunksToApply = append(hunksToApply, hunk)
	}

	// Validate that hunks can be applied together
	err := ValidateHunkCombination(hunksToApply)
	if err != nil {
		return fmt.Errorf("invalid hunk combination: %w", err)
	}

	// Use dependency-aware application for better handling of complex changes
	return applyHunksWithDependencies(hunksToApply, baseDiff)
}

// applyHunksWithDependencies applies hunks using dependency-aware grouping for
// better handling of complex changes.
func applyHunksWithDependencies(hunks []*Hunk, baseDiff string) error {
	// Create dependency groups
	dependencyGroups := CreateDependencyGroups(hunks)

	debugLog("Dependency analysis: %d groups identified\n", len(dependencyGroups))
	for i, group := range dependencyGroups {
		debugLog("  Group %d: %d hunks\n", i+1, len(group))
		for _, hunk := range group {
			deps := len(hunk.Dependencies)
			dependents := len(hunk.Dependents)
			debugLog("    - %s (%s, deps: %d, dependents: %d)\n", hunk.ID, hunk.ChangeType, deps, dependents)
		}
	}

	// Apply groups in order
	for i, group := range dependencyGroups {
		debugLog("Applying group %d/%d (%d hunks)...\n", i+1, len(dependencyGroups), len(group))

		// NOTE: Save staging state before attempting group application
		// This prevents corrupt patch failures from leaving the repository in a broken state
		groupStagingState := saveStagingState()

		var success bool
		if len(group) == 1 {
			// Single hunk - apply individually for better error isolation
			success = applyHunksSequentially(group, baseDiff)
		} else {
			// Multiple interdependent hunks - try atomic application first
			success = applyDependencyGroupAtomically(group, baseDiff)

			if !success {
				debugLog("  Atomic application failed, trying sequential with smart ordering...\n")
				// Fallback to sequential application with dependency ordering
				success = applyDependencyGroupSequentially(group, baseDiff)
			}
		}

		if !success {
			debugLog("Failed to apply group %d, restoring staging state...\n", i+1)
			// NOTE: Restore staging state to prevent broken repository state
			restoreStagingState(groupStagingState)
			return fmt.Errorf("failed to apply group %d", i+1)
		}

		debugLog("✓ Group %d applied successfully\n", i+1)
	}

	return nil
}

// applyDependencyGroupAtomically applies a dependency group using git's native
// patch application with proper line number calculation.
func applyDependencyGroupAtomically(hunks []*Hunk, baseDiff string) bool {
	// Generate valid patch with corrected line numbers
	patchContent := createValidGitPatch(hunks, baseDiff)

	if strings.TrimSpace(patchContent) == "" {
		debugLog("No valid patch content generated\n")
		return false
	}

	// Apply using git's native mechanism
	return applyPatchWithGit(patchContent)
}

// applyDependencyGroupSequentially applies hunks in a dependency group
// sequentially using git native mechanisms.
func applyDependencyGroupSequentially(hunks []*Hunk, baseDiff string) bool {
	// Order hunks by dependencies (topological sort)
	orderedHunks := topologicalSortHunks(hunks)

	if len(orderedHunks) == 0 {
		// Fallback to simple ordering if topological sort fails
		sort.Slice(hunks, func(i, j int) bool {
			if hunks[i].FilePath != hunks[j].FilePath {
				return hunks[i].FilePath < hunks[j].FilePath
			}
			return hunks[i].StartLine < hunks[j].StartLine
		})
		orderedHunks = hunks
	}

	// Apply hunks in dependency order using git native mechanisms
	for i, hunk := range orderedHunks {
		success := relocateAndApplyHunk(hunk, baseDiff)
		if !success {
			debugLog("Failed to apply hunk %s (%d/%d) via git apply\n", hunk.ID, i+1, len(orderedHunks))
			return false
		}
	}

	return true
}

// topologicalSortHunks sorts hunks based on their dependencies using
// topological sort.
func topologicalSortHunks(hunks []*Hunk) []*Hunk {
	// Build hunk map for quick lookups
	hunkMap := make(map[string]*Hunk)
	hunkIDs := make(map[string]bool)
	for _, hunk := range hunks {
		hunkMap[hunk.ID] = hunk
		hunkIDs[hunk.ID] = true
	}

	// Calculate in-degrees (number of dependencies within this group)
	inDegree := make(map[string]int)
	for _, hunk := range hunks {
		// Only count dependencies that are within this group
		localDeps := 0
		for depID := range hunk.Dependencies {
			if hunkIDs[depID] {
				localDeps++
			}
		}
		inDegree[hunk.ID] = localDeps
	}

	// Start with hunks that have no dependencies within the group
	var queue []string
	for hunkID := range hunkIDs {
		if inDegree[hunkID] == 0 {
			queue = append(queue, hunkID)
		}
	}

	var result []*Hunk

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]
		result = append(result, hunkMap[currentID])

		// Reduce in-degree for dependents
		currentHunk := hunkMap[currentID]
		for dependentID := range currentHunk.Dependents {
			if hunkIDs[dependentID] {
				inDegree[dependentID]--
				if inDegree[dependentID] == 0 {
					queue = append(queue, dependentID)
				}
			}
		}
	}

	// Check for cycles
	if len(result) != len(hunks) {
		debugLog("Warning: Cyclic dependencies detected, using fallback ordering\n")
		return nil
	}

	return result
}

// applyHunksSequentially applies hunks one by one using git native mechanisms
// for better reliability.
func applyHunksSequentially(hunks []*Hunk, baseDiff string) bool {
	// Sort hunks by file and line number for consistent application order
	sort.Slice(hunks, func(i, j int) bool {
		if hunks[i].FilePath != hunks[j].FilePath {
			return hunks[i].FilePath < hunks[j].FilePath
		}
		return hunks[i].StartLine < hunks[j].StartLine
	})

	for i, hunk := range hunks {
		// Use git native patch application
		success := relocateAndApplyHunk(hunk, baseDiff)
		if !success {
			debugLog("Failed to apply hunk %s (%d/%d) via git apply\n", hunk.ID, i+1, len(hunks))
			return false
		}
	}

	return true
}

// extractFilesFromPatch extracts the list of files affected by a patch.
func extractFilesFromPatch(patchContent string) map[string]bool {
	files := make(map[string]bool)
	lines := strings.Split(patchContent, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			// Extract file path from diff header
			// Format: diff --git a/path/to/file b/path/to/file
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				// Remove any single-letter prefix (a/, b/, c/, w/, i/, o/, etc.)
				filePath := parts[3]
				filePath = stripGitDiffPrefix(filePath)
				files[filePath] = true
			}
		} else if strings.HasPrefix(line, "+++") {
			// Alternative: extract from +++ header
			// Format: +++ b/path/to/file
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[1] != "/dev/null" {
				filePath := parts[1]
				filePath = stripGitDiffPrefix(filePath)
				files[filePath] = true
			}
		}
	}

	return files
}

// stripGitDiffPrefix removes any single-letter prefix from Git diff paths
// Handles: a/, b/, c/, w/, i/, o/, etc.
func stripGitDiffPrefix(path string) string {
	if len(path) > 2 && path[1] == '/' {
		return path[2:]
	}
	return path
}

// syncFilesFromStaging syncs specific files from staging area to working
// directory.
func syncFilesFromStaging(filePaths map[string]bool) bool {
	if len(filePaths) == 0 {
		// If no files specified, sync all
		cmd := exec.Command("git", "checkout-index", "-f", "-a")
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run() == nil
	}

	// Sync each file individually
	allSuccess := true
	for filePath := range filePaths {
		// Use git checkout-index to sync specific file
		cmd := exec.Command("git", "checkout-index", "-f", "--", filePath)
		cmd.Stdout = nil
		cmd.Stderr = nil

		if cmd.Run() != nil {
			debugLog("Failed to sync %s\n", filePath)
			allSuccess = false
		}
	}

	return allSuccess
}

// applyPatchWithGit applies a patch using git's native mechanism with improved
// file-specific sync.
func applyPatchWithGit(patchContent string) bool {
	// Extract affected files from patch content
	affectedFiles := extractFilesFromPatch(patchContent)

	// Save current staging state for rollback
	stagingState := saveStagingState()

	// NOTE: Save working directory state for affected files BEFORE any patch operations
	workingDirState := saveWorkingDirState(affectedFiles)

	// NOTE: Also save the current staging state before applying patches
	originalStagingState := saveStagingState()

	// Create temporary patch file with enhanced validation
	tmpFile, err := os.CreateTemp("", "patch-*.patch")
	if err != nil {
		debugLog("Error creating temporary patch file: %v\n", err)
		return false
	}
	defer os.Remove(tmpFile.Name())

	// Write patch content
	_, err = tmpFile.WriteString(patchContent)
	if err != nil {
		debugLog("Error writing patch content: %v\n", err)
		return false
	}

	// NOTE: Ensure content is written to disk before git reads it
	err = tmpFile.Sync()
	if err != nil {
		debugLog("Error syncing patch file: %v\n", err)
		return false
	}
	tmpFile.Close()

	// NOTE: Validate patch file is readable before attempting to apply
	verificationContent, err := os.ReadFile(tmpFile.Name())
	if err != nil || string(verificationContent) != patchContent {
		debugLog("Patch file validation failed\n")
		// Restore states and return failure
		restoreWorkingDirState(workingDirState, affectedFiles)
		restoreStagingState(originalStagingState)
		return false
	}

	// Apply patch using git apply --index to update both staging area and working directory
	cmd := exec.Command("git", "apply", "--index", "--whitespace=nowarn", tmpFile.Name())
	cmd.Stdout = nil
	cmd.Stderr = nil

	if cmd.Run() == nil {
		debugLog("✓ Patch applied successfully via git apply --index\n")
		// NOTE: Verify working directory matches expected state after successful apply
		if verifyWorkingDirIntegrity(affectedFiles, patchContent) {
			return true
		} else {
			debugLog("Warning: Working directory integrity check failed after successful patch apply\n")
			// Don't fail here, but log the issue
			return true
		}
	} else {
		debugLog("Git apply --index failed\n")
		// NOTE: Immediately restore BOTH working directory and staging states
		debugLog("Restoring working directory and staging states after --index failure...\n")
		restoreWorkingDirState(workingDirState, affectedFiles)
		restoreStagingState(originalStagingState)

		// If --index fails, fallback to --cached and then sync working directory
		cmdCached := exec.Command("git", "apply", "--cached", "--whitespace=nowarn", tmpFile.Name())
		cmdCached.Stdout = nil
		cmdCached.Stderr = nil

		if cmdCached.Run() == nil {
			debugLog("✓ Patch applied to staging area, syncing working directory...\n")

			// Sync only the affected files from staging to working directory
			syncSuccess := syncFilesFromStaging(affectedFiles)

			if syncSuccess {
				debugLog("✓ Working directory synchronized for affected files\n")
				return true
			} else {
				debugLog("Failed to sync working directory\n")
				// NOTE: Restore both staging and working directory states
				restoreStagingState(stagingState)
				restoreWorkingDirState(workingDirState, affectedFiles)
				return false
			}
		} else {
			debugLog("Git apply --cached also failed\n")
			// NOTE: Restore both staging and working directory states
			restoreStagingState(stagingState)
			restoreWorkingDirState(workingDirState, affectedFiles)
			return false
		}
	}
}

// saveStagingState saves current staging state for rollback.
func saveStagingState() string {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(output)
}

// restoreStagingState restores staging state from saved state.
func restoreStagingState(savedState string) bool {
	// Reset staging area
	cmd := exec.Command("git", "reset", "HEAD")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if cmd.Run() != nil {
		return false
	}

	// If there was staged content, reapply it
	if strings.TrimSpace(savedState) != "" {
		tmpFile, err := os.CreateTemp("", "staging-*.patch")
		if err != nil {
			return false
		}
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(savedState)
		if err != nil {
			return false
		}
		tmpFile.Close()

		cmd := exec.Command("git", "apply", "--cached", tmpFile.Name())
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run() == nil
	}

	return true
}

// saveWorkingDirState saves current working directory state for affected files
// with enhanced reliability.
func saveWorkingDirState(affectedFiles map[string]bool) map[string]string {
	fileStates := make(map[string]string)
	for filePath := range affectedFiles {
		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			// NOTE: Use binary mode first to handle any file type
			binaryContent, err := os.ReadFile(filePath)
			if err != nil {
				debugLog("Warning: Could not save state for %s: %v\n", filePath, err)
				continue
			}

			// Convert binary content to string (both UTF-8 and latin-1 use the same conversion)
			fileStates[filePath] = string(binaryContent)

			// NOTE: Verify the file was read correctly by checking length
			if len(fileStates[filePath]) == 0 {
				info, _ := os.Stat(filePath)
				if info != nil && info.Size() > 0 {
					debugLog("Warning: File %s appears non-empty but read as empty\n", filePath)
				}
			}
		} else {
			// Mark non-existent files as such
			fileStates[filePath] = ""
		}
	}

	debugLog("Successfully saved working directory state for %d files\n", len(fileStates))
	return fileStates
}

// restoreWorkingDirState restores working directory state for affected files
// with enhanced reliability.
func restoreWorkingDirState(
	savedStates map[string]string,
	affectedFiles map[string]bool,
) bool {
	if len(savedStates) == 0 {
		debugLog("No saved states to restore\n")
		return true
	}

	restoredCount := 0
	failedCount := 0

	for filePath := range affectedFiles {
		savedContent, exists := savedStates[filePath]
		if !exists {
			debugLog("Warning: No saved state found for %s\n", filePath)
			failedCount++
			continue
		}

		if savedContent == "" {
			// File didn't exist, remove it if it exists now
			if _, err := os.Stat(filePath); err == nil {
				err := os.Remove(filePath)
				if err != nil {
					debugLog("Error removing file that shouldn't exist: %s\n", filePath)
					failedCount++
				} else {
					debugLog("Removed file that shouldn't exist: %s\n", filePath)
					restoredCount++
				}
			}
		} else {
			// NOTE: Create directory if it doesn't exist
			dirPath := filepath.Dir(filePath)
			if dirPath != "." {
				err := os.MkdirAll(dirPath, 0o755)
				if err != nil {
					debugLog("Error creating directory %s: %v\n", dirPath, err)
					failedCount++
					continue
				}
			}

			// NOTE: Use same encoding strategy as saving
			err := os.WriteFile(filePath, []byte(savedContent), 0o644)
			if err != nil {
				debugLog("Error: Could not restore %s: %v\n", filePath, err)
				failedCount++
				continue
			}

			// NOTE: Verify the file was written correctly
			writtenContent, err := os.ReadFile(filePath)
			if err == nil && len(writtenContent) != len(savedContent) {
				debugLog("Warning: File %s restoration size mismatch: expected %d, got %d\n", filePath, len(savedContent), len(writtenContent))
			}

			debugLog("Restored working directory state for: %s\n", filePath)
			restoredCount++
		}
	}

	debugLog("Working directory restoration: %d restored, %d failed out of %d files\n", restoredCount, failedCount, len(affectedFiles))
	return failedCount == 0
}

// verifyWorkingDirIntegrity verifies that working directory files are in a
// valid state after patch application.
func verifyWorkingDirIntegrity(affectedFiles map[string]bool, patchContent string) bool {
	issuesFound := 0
	totalFiles := len(affectedFiles)

	for filePath := range affectedFiles {
		issues := verifyFileIntegrity(filePath, patchContent)
		issuesFound += issues
	}

	if issuesFound > 0 {
		debugLog("Working directory integrity check: %d potential issues found out of %d files\n", issuesFound, totalFiles)
		return false
	}
	debugLog("Working directory integrity check: All %d files appear valid\n", totalFiles)
	return true
}

// verifyFileIntegrity checks a single file for integrity issues.
func verifyFileIntegrity(filePath, patchContent string) int {
	info, err := os.Stat(filePath)
	if err != nil {
		debugLog("Warning: Could not verify integrity of %s: %v\n", filePath, err)
		return 1
	}

	if info.Size() == 0 && !strings.Contains(strings.ToLower(patchContent), "delete") {
		debugLog("Warning: File %s is unexpectedly empty\n", filePath)
		return 1
	}

	return 0
}

// relocateAndApplyHunk applies a hunk using git's native patch application
// instead of direct file modification.
func relocateAndApplyHunk(hunk *Hunk, baseDiff string) bool {
	// Generate valid patch for single hunk
	patchContent := createValidGitPatch([]*Hunk{hunk}, baseDiff)

	if strings.TrimSpace(patchContent) == "" {
		debugLog("Could not generate valid patch for hunk %s\n", hunk.ID)
		return false
	}

	// Apply using git's native mechanism
	return applyPatchWithGit(patchContent)
}

// createValidGitPatch creates a valid git patch using ABSOLUTELY MINIMAL
// modification approach.
func createValidGitPatch(hunks []*Hunk, baseDiff string) string {
	return createAbsolutelyMinimalPatch(hunks, baseDiff)
}

// parseHunkContent parses hunk content to extract additions, deletions,
// and context lines.
func parseHunkContent(hunk *Hunk) ([]string, []string, []string) {
	var additions []string
	var deletions []string
	var contextLines []string

	lines := strings.Split(hunk.Content, "\n")
	for i := 1; i < len(lines); i++ { // Skip header
		line := lines[i]
		// NOTE: Don't filter out empty lines - they're significant in git diffs
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			additions = append(additions, line[1:]) // Remove + prefix
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			deletions = append(deletions, line[1:]) // Remove - prefix
		case strings.HasPrefix(line, " "):
			contextLines = append(contextLines, line[1:]) // Remove space prefix
		case line == "":
			// Empty lines are context lines (preserve file structure)
			contextLines = append(contextLines, "")
		}
	}

	return additions, deletions, contextLines
}

// ApplyHunksWithFallback applies hunks using the hunk-based approach only.
func ApplyHunksWithFallback(hunkIDs []string, hunksByID map[string]*Hunk, baseDiff string) error {
	return ApplyHunks(hunkIDs, hunksByID, baseDiff)
}

// ResetStagingArea resets the staging area to match HEAD.
func ResetStagingArea() bool {
	cmd := exec.Command("git", "reset", "HEAD")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// PreviewHunkApplication generates a preview of what would be applied when
// staging these hunks.
func PreviewHunkApplication(hunkIDs []string, hunksByID map[string]*Hunk) string {
	if len(hunkIDs) == 0 {
		return "No hunks selected."
	}

	// Group hunks by file
	filesAffected := make(map[string][]*Hunk)
	for _, hunkID := range hunkIDs {
		if hunk, exists := hunksByID[hunkID]; exists {
			filesAffected[hunk.FilePath] = append(filesAffected[hunk.FilePath], hunk)
		}
	}

	// Generate preview
	var previewLines []string
	for filePath, hunks := range filesAffected {
		previewLines = append(previewLines, fmt.Sprintf("File: %s", filePath))
		sort.Slice(hunks, func(i, j int) bool {
			return hunks[i].StartLine < hunks[j].StartLine
		})
		for _, hunk := range hunks {
			lineRange := fmt.Sprintf("lines %d-%d", hunk.StartLine, hunk.EndLine)
			previewLines = append(previewLines, fmt.Sprintf("  - %s (%s)", hunk.ID, lineRange))
		}
		previewLines = append(previewLines, "")
	}

	return strings.Join(previewLines, "\n")
}

// StagingStatus represents the current staging status
type StagingStatus struct {
	Staged   []string `json:"staged"`
	Modified []string `json:"modified"`
}

// GetStagingStatus gets the current staging status.
func GetStagingStatus() StagingStatus {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return StagingStatus{Staged: []string{}, Modified: []string{}}
	}

	var staged []string
	var modified []string

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) >= 2 {
			status := line[:2]
			filePath := line[3:]

			if status[0] != ' ' && status[0] != '?' { // Staged changes
				staged = append(staged, filePath)
			}
			if status[1] != ' ' && status[1] != '?' { // Modified changes
				modified = append(modified, filePath)
			}
		}
	}

	return StagingStatus{Staged: staged, Modified: modified}
}
