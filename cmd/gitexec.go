package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/zbiljic/gitexec"
)

var (
	filesToExclude = []string{
		"*.lock*", // yarn.lock, Cargo.lock, Gemfile.lock, Pipfile.lock, etc.
		"go*.sum",
		"package-lock.json",
		"pnpm-lock.yaml",
	}

	excludeFromDiff = slice.FlatMap(filesToExclude, func(i int, s string) []string {
		return []string{":(exclude)" + s}
	})
)

func gitWorkingTreeDir(path string) (string, error) {
	out, err := gitexec.RevParse(&gitexec.RevParseOptions{
		CmdDir:       path,
		ShowToplevel: true,
	})
	if err != nil {
		return string(out), err
	}

	return strings.TrimSpace(string(out)), nil
}

func gitDiffStaged(path string) ([]string, string, error) {
	out, err := gitexec.Diff(&gitexec.DiffOptions{
		CmdDir:   path,
		Cached:   true,
		Minimal:  true,
		NameOnly: true,
		Path:     excludeFromDiff,
	})
	if err != nil {
		return []string{}, "", err
	}

	outString := strings.TrimSpace(string(out))
	if outString == "" {
		return []string{}, "", nil
	}

	files := strings.Split(outString, "\n")

	out, err = gitexec.Diff(&gitexec.DiffOptions{
		CmdDir:  path,
		Cached:  true,
		Minimal: true,
		Path:    excludeFromDiff,
	})
	if err != nil {
		return []string{}, "", err
	}

	diff := strings.TrimSpace(string(out))

	return files, diff, nil
}

func gitCommit(path, message string) error {
	_, err := gitexec.Commit(&gitexec.CommitOptions{
		CmdDir:  path,
		Message: message,
	})
	if err != nil {
		return err
	}

	return nil
}

func gitAddAll(path string) error {
	_, err := gitexec.Add(&gitexec.AddOptions{
		CmdDir: path,
		All:    true,
	})
	if err != nil {
		return err
	}

	return nil
}

// gitUnstagedFiles returns a list of unstaged files.
func gitUnstagedFiles(workDir string) ([]string, error) {
	opts := &gitexec.StatusOptions{
		CmdDir:    workDir,
		Porcelain: true,
	}

	output, err := gitexec.Status(opts)
	if err != nil {
		return nil, err
	}

	var unstagedFiles []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		// In porcelain format, unstaged changes are indicated by changes in the working tree (second character)
		// If the second character is not a space, the file has unstaged changes
		if len(line) > 2 && line[1] != ' ' {
			// Extract the file path (skip the status prefix and any spaces)
			path := strings.TrimSpace(line[3:])
			unstagedFiles = append(unstagedFiles, path)
		}
	}

	return unstagedFiles, nil
}

// gitStagedFiles returns a list of staged files.
func gitStagedFiles(workDir string) ([]string, error) {
	opts := &gitexec.StatusOptions{
		CmdDir:    workDir,
		Porcelain: true,
	}

	output, err := gitexec.Status(opts)
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return nil, nil
	}

	var stagedFiles []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse the status line (format: XY PATH)
		// X is the status in the index/staging area
		if len(line) > 3 {
			statusChar := line[0]
			// Staged files have one of these statuses in the first column:
			// A: addition, M: modification, D: deletion, R: rename, C: copy
			if statusChar == 'A' || statusChar == 'M' || statusChar == 'D' ||
				statusChar == 'R' || statusChar == 'C' {
				// Extract the file path (skip the status prefix and any spaces)
				path := strings.TrimSpace(line[3:])
				stagedFiles = append(stagedFiles, path)
			}
		}
	}

	return stagedFiles, nil
}

// gitPreviousCommitMessages returns previous commit messages for the specified files.
func gitPreviousCommitMessages(workDir string, files []string, maxCommits int) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}

	opts := &gitexec.LogOptions{
		CmdDir:   workDir,
		MaxCount: maxCommits,
		Format:   "%s", // subject only
		Path:     files,
	}

	output, err := gitexec.Log(opts)
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return nil, nil
	}

	// Split by newlines and filter out empty messages
	messages := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string

	for _, msg := range messages {
		if msg != "" {
			result = append(result, msg)
		}
	}

	return result, nil
}

// gitLastCommitForFile returns the last commit hash that modified the given file.
func gitLastCommitForFile(workDir, file string) (string, error) {
	opts := &gitexec.LogOptions{
		CmdDir:                               workDir,
		MaxCount:                             1,
		Format:                               "%H",
		DoNotInterpretMoreArgumentsAsOptions: true,
		Path:                                 []string{file},
	}

	output, err := gitexec.Log(opts)
	if err != nil {
		return "", err
	}

	if len(output) == 0 {
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// gitUnstageAll unstages all currently staged files in the given directory.
func gitUnstageAll(workDir string) error {
	_, err := gitexec.Reset(&gitexec.ResetOptions{
		CmdDir:   workDir,
		Quiet:    true,
		Pathspec: []string{"--"},
	})
	if err != nil {
		return err
	}
	return nil
}

// gitStageFiles stages the specified files in the given directory.
func gitStageFiles(workDir string, files []string) error {
	_, err := gitexec.Add(&gitexec.AddOptions{
		CmdDir:                               workDir,
		DoNotInterpretMoreArgumentsAsOptions: true,
		Pathspec:                             files,
	})
	if err != nil {
		return err
	}
	return nil
}

// gitCreateFixupCommit creates a fixup commit for the specified commit hash.
func gitCreateFixupCommit(workDir, commitHash string) error {
	_, err := gitexec.Commit(&gitexec.CommitOptions{
		CmdDir:   workDir,
		Fixup:    commitHash,
		NoEdit:   true,
		NoVerify: true,
		Quiet:    true,
	})
	if err != nil {
		return err
	}
	return nil
}

// gitCurrentBranch returns the name of the current branch.
func gitCurrentBranch(workDir string) (string, error) {
	out, err := gitexec.SymbolicRef(&gitexec.SymbolicRefOptions{
		CmdDir: workDir,
		Short:  true,
		Ref:    "HEAD",
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// gitFindOldestFixupParent finds the parent of the oldest commit that needs to be fixed up.
// This is useful for determining the base commit for rebase operations.
func gitFindOldestFixupParent(workDir string, fixupCommits map[string][]string) (string, error) {
	if len(fixupCommits) == 0 {
		return "", nil
	}

	// Get all commit hashes that need to be fixed up
	commitHashes := make([]string, 0, len(fixupCommits))
	for commitHash := range fixupCommits {
		commitHashes = append(commitHashes, commitHash)
	}

	// Get commit dates for all relevant commits
	commitDates := make(map[string]time.Time)
	for _, commitHash := range commitHashes {
		// Get the commit date as a Unix timestamp using git log
		out, err := gitexec.Log(&gitexec.LogOptions{
			CmdDir:        workDir,
			Format:        "%ct",
			MaxCount:      1,
			NoWalk:        "unsorted",
			RevisionRange: commitHash,
		})
		if err != nil {
			return "", fmt.Errorf("failed to get commit date for %s: %w", commitHash, err)
		}

		// Parse the Unix timestamp
		timestamp, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid timestamp for commit %s: %w", commitHash, err)
		}

		commitDates[commitHash] = time.Unix(timestamp, 0)
	}

	// Find the oldest commit by comparing timestamps
	var oldestCommit string
	for commitHash, commitTime := range commitDates {
		if oldestCommit == "" || commitTime.Before(commitDates[oldestCommit]) {
			oldestCommit = commitHash
		}
	}

	// Get the parent of the oldest commit
	out, err := gitexec.RevParse(&gitexec.RevParseOptions{
		CmdDir: workDir,
		Arg:    []string{oldestCommit + "^"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get parent of commit %s: %w", oldestCommit, err)
	}

	return strings.TrimSpace(string(out)), nil
}

// gitDebugRebaseAutosquash returns the command that would be run by gitRebaseAutosquash.
func gitDebugRebaseAutosquash(workDir, upstream string) string {
	opts := &gitexec.RebaseOptions{
		CmdDir:     workDir,
		Autosquash: true,
		Autostash:  true,
		NoVerify:   true,
		NoGpgSign:  true,
		Quiet:      true,
	}

	if upstream != "" {
		opts.Upstream = upstream
	}

	return gitexec.RebaseCmd(opts).String()
}

// gitRebaseAutosquash runs git rebase with autosquash and other recommended options.
// If upstream is empty, it will use the current branch.
func gitRebaseAutosquash(workDir, upstream string) error {
	opts := &gitexec.RebaseOptions{
		CmdDir:     workDir,
		Autosquash: true,
		Autostash:  true,
		NoVerify:   true,
		NoGpgSign:  true,
		Quiet:      true,
	}

	if upstream != "" {
		opts.Upstream = upstream
	}

	_, err := gitexec.Rebase(opts)
	return err
}

// gitCreateBackupBranch creates a backup branch with the current branch's state.
// The branch name will be in the format: backup/<original_branch_name>-HH-MM-SS
func gitCreateBackupBranch(workDir string) (string, error) {
	// Get current branch name
	currentBranch, err := gitCurrentBranch(workDir)
	if err != nil {
		return "", fmt.Errorf("failed to get current branch name: %w", err)
	}

	// Generate timestamp in HH-MM-SS format
	timeStr := time.Now().Format("15-04-05")
	backupBranchName := fmt.Sprintf("backup/%s-%s", currentBranch, timeStr)

	// Create a new branch pointing to the current HEAD
	_, err = gitexec.Branch(&gitexec.BranchOptions{
		CmdDir:     workDir,
		Branchname: backupBranchName,
		Force:      true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create backup branch: %w", err)
	}

	return backupBranchName, nil
}

// gitGetCommitsBetweenBranches returns commits between current branch and base branch
func gitGetCommitsBetweenBranches(workDir, baseBranch string) (string, error) {
	opts := &gitexec.LogOptions{
		CmdDir: workDir,
		Paths:  fmt.Sprintf("%s..HEAD", baseBranch),
		Format: "%H%n%an%n%s%n%b%n---COMMIT---",
	}

	output, err := gitexec.Log(opts)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// gitGetDiffBetweenBranches returns the diff between current branch and base branch
func gitGetDiffBetweenBranches(workDir, baseBranch string) (string, error) {
	opts := &gitexec.DiffOptions{
		CmdDir:  workDir,
		Minimal: true,
		Commit:  baseBranch,
		Path:    excludeFromDiff,
	}

	output, err := gitexec.Diff(opts)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// gitBranchExists checks if a branch exists
func gitBranchExists(workDir, branchName string) bool {
	opts := &gitexec.BranchOptions{
		CmdDir:  workDir,
		List:    true,
		Pattern: []string{branchName},
	}

	output, err := gitexec.Branch(opts)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// gitRemoteBranchExists checks if a remote branch exists
func gitRemoteBranchExists(workDir, branchName string) bool {
	opts := &gitexec.BranchOptions{
		CmdDir:  workDir,
		Remotes: true,
		List:    true,
		Pattern: []string{"*/" + branchName},
	}

	output, err := gitexec.Branch(opts)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// LineRange represents a range of lines in a file that have been modified
type LineRange struct {
	Start int
	End   int
}

// BlameInfo contains information about who last modified a specific line
type BlameInfo struct {
	CommitHash string
	LineNumber int
}

// CommitScore tracks how many lines a commit is responsible for
type CommitScore struct {
	CommitHash string
	Score      int
	Lines      []int
}

// gitGetModifiedLineRanges parses the diff output to find which lines are being modified
func gitGetModifiedLineRanges(workDir, file string) ([]LineRange, error) {
	out, err := gitexec.Diff(&gitexec.DiffOptions{
		CmdDir:  workDir,
		Cached:  true,
		Unified: 0, // Show no context lines
		Path:    []string{file},
	})
	if err != nil {
		return nil, err
	}

	ranges := []LineRange{}
	lines := strings.Split(string(out), "\n")

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

	return ranges, nil
}

// gitBlameLines uses git blame to find which commits introduced specific lines
func gitBlameLines(workDir, file string, lineRanges []LineRange) ([]BlameInfo, error) {
	var blameInfo []BlameInfo

	for _, lineRange := range lineRanges {
		// Use git blame with line range using gitexec
		rangeStr := fmt.Sprintf("%d,%d", lineRange.Start, lineRange.End)

		out, err := gitexec.Blame(&gitexec.BlameOptions{
			CmdDir:        workDir,
			LinePorcelain: true,
			Lstartend:     rangeStr,
			File:          file,
		})
		if err != nil {
			// If blame fails, skip this range
			continue
		}

		// Parse the porcelain output to extract commit hashes
		lines := strings.Split(string(out), "\n")
		currentLine := lineRange.Start

		for _, line := range lines {
			// In porcelain format, commit hash is on lines that start with 40-character hex
			if len(line) == 40 && isHexString(line) {
				blameInfo = append(blameInfo, BlameInfo{
					CommitHash: line,
					LineNumber: currentLine,
				})
				currentLine++
			}
		}
	}

	return blameInfo, nil
}

// gitFindBestCommitForFile uses line-based analysis to find the most appropriate commit
// to target for fixup based on which commit introduced the most modified lines
func gitFindBestCommitForFile(workDir, file string) (string, error) {
	// Get the line ranges that are being modified
	modifiedRanges, err := gitGetModifiedLineRanges(workDir, file)
	if err != nil {
		return "", fmt.Errorf("failed to get modified line ranges: %w", err)
	}

	if len(modifiedRanges) == 0 {
		// fallback to last commit for the file
		return gitLastCommitForFile(workDir, file)
	}

	// Get blame information for the modified lines
	blameInfo, err := gitBlameLines(workDir, file, modifiedRanges)
	if err != nil {
		return "", fmt.Errorf("failed to get blame information: %w", err)
	}

	if len(blameInfo) == 0 {
		// fallback to last commit for the file
		return gitLastCommitForFile(workDir, file)
	}

	// Score each commit based on how many modified lines it introduced
	commitScores := make(map[string]*CommitScore)

	for _, blame := range blameInfo {
		if score, exists := commitScores[blame.CommitHash]; exists {
			score.Score++
			score.Lines = append(score.Lines, blame.LineNumber)
		} else {
			commitScores[blame.CommitHash] = &CommitScore{
				CommitHash: blame.CommitHash,
				Score:      1,
				Lines:      []int{blame.LineNumber},
			}
		}
	}

	// Find the commit with the highest score
	var bestCommit string
	var bestScore int

	for _, score := range commitScores {
		if score.Score > bestScore {
			bestScore = score.Score
			bestCommit = score.CommitHash
		}
	}

	if bestCommit == "" {
		// fallback to last commit for the file
		return gitLastCommitForFile(workDir, file)
	}

	return bestCommit, nil
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
