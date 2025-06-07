package cmd

import (
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/zbiljic/kai/pkg/gitexec"
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
		ShowTopLevel: true,
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

	// Split by newlines and deduplicate
	messages := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Deduplicate messages
	seen := make(map[string]struct{})
	var deduped []string

	for _, msg := range messages {
		if msg == "" {
			continue
		}

		if _, exists := seen[msg]; !exists {
			seen[msg] = struct{}{}
			deduped = append(deduped, msg)
		}
	}

	return deduped, nil
}
