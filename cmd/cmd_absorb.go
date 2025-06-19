package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/orochaa/go-clack/prompts"
	"github.com/orochaa/go-clack/third_party/picocolors"
	"github.com/spf13/cobra"

	"github.com/zbiljic/kai/pkg/promptsx"
)

var absorbCmd = &cobra.Command{
	Use:   "absorb",
	Short: "Automatically absorb staged changes into appropriate commits",
	Long: `Automatically stages staged changes as fixup commits targeting the appropriate commits.

This command analyzes your staged changes and creates fixup commits targeting the
original commits that introduced the changes. This is useful for making small
fixes to existing commits during code review.

After running this command, you can run 'git rebase -i --autosquash' to automatically
fold the fixup commits into their target commits.

If --and-rebase is specified, the rebase will be run automatically.`,
	RunE: runAbsorbE,
}

var absorbFlags = absorbOptions{
	AndRebase: false,
	DryRun:    false,
	All:       false,
}

func absorbAddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&absorbFlags.AndRebase, "and-rebase", "r", false, "Automatically run 'git rebase --autosquash' after creating fixups")
	cmd.Flags().BoolVarP(&absorbFlags.DryRun, "dry-run", "n", false, "Don't make any actual changes")
	cmd.Flags().BoolVarP(&absorbFlags.All, "all", "a", false, "Automatically stage all changes in tracked files")
}

func init() {
	absorbAddFlags(absorbCmd)

	rootCmd.AddCommand(absorbCmd)
}

type absorbOptions struct {
	AndRebase bool
	DryRun    bool
	All       bool
}

func absorbSetup(cmd *cobra.Command) (string, error) {
	workDir, err := gitWorkingTreeDir(getWd())
	if err != nil {
		return "", errors.New("The current directory must be a Git repository") //nolint:staticcheck
	}
	return workDir, nil
}

func absorbDetectAndStageFiles(workDir string, all bool) ([]string, error) {
	// Check for staged files first
	files, err := gitStagedFiles(workDir)
	if err != nil {
		return nil, err
	}

	// If no files are staged, automatically set All flag to true
	if len(files) == 0 {
		// except in dry-run mode, where we get unstaged files
		if absorbFlags.DryRun {
			files, err = gitUnstagedFiles(workDir)
			if err != nil {
				return nil, fmt.Errorf("failed to get unstaged files: %w", err)
			}
		} else {
			all = true
		}
	}

	if all {
		if absorbFlags.DryRun {
			if len(files) > 0 {
				promptsx.InfoWithLastLine(fmt.Sprintf(
					"Would stage %d file(s):\n     %s",
					len(files),
					strings.Join(files, "\n     "),
				))
			}
		} else {
			prompts.Info("Staging all changes in tracked files...")
			if err := gitAddAll(workDir); err != nil {
				return nil, fmt.Errorf("failed to stage all changes: %w", err)
			}

			// Get updated list of staged files after adding all
			files, err = gitStagedFiles(workDir)
			if err != nil {
				return nil, fmt.Errorf("failed to get staged files: %w", err)
			}

			if len(files) == 0 {
				return nil, errors.New("no changes detected to stage")
			}

			prompts.Info(fmt.Sprintf("Staged %d file(s)", len(files)))
		}
	}

	if len(files) == 0 {
		return nil, errors.New("no staged changes found. Use 'git add' to stage changes first")
	}

	return files, nil
}

func runAbsorbE(cmd *cobra.Command, args []string) error {
	workDir, err := absorbSetup(cmd)
	if err != nil {
		return err
	}

	if absorbFlags.DryRun {
		msg := "Running in dry-run mode. No changes will be made."
		if absorbFlags.All {
			msg += " (--all flag is set)"
		}
		promptsx.Note(msg)
	}

	// Get staged files, potentially staging all changes if --all is set
	stagedFiles, err := absorbDetectAndStageFiles(workDir, absorbFlags.All)
	if err != nil {
		if absorbFlags.DryRun {
			// In dry-run mode, just show a note and continue
			promptsx.Note(err.Error())
			return nil
		}
		return fmt.Errorf("error detecting staged files: %w", err)
	}

	// For each staged file, find the appropriate commit to fixup
	fixupCommits := make(map[string][]string) // commit -> []files

	for _, file := range stagedFiles {
		// Use git log to find the last commit that touched this file
		commitHash, err := gitLastCommitForFile(workDir, file)
		if err != nil {
			return fmt.Errorf("failed to get last commit for %s: %w", file, err)
		}

		// Skip if we couldn't find a commit or if it's the working copy
		if commitHash == "" || commitHash == "0000000000000000000000000000000000000000" {
			continue
		}

		// Add file to this commit's list
		fixupCommits[commitHash] = append(fixupCommits[commitHash], file)
	}

	if len(fixupCommits) == 0 {
		promptsx.Note("No appropriate commits found to fixup. All changes appear to be new.")
		return nil
	}

	// Create fixup commits for each target commit
	for commitHash, files := range fixupCommits {
		// Show what would be done in dry-run mode
		if absorbFlags.DryRun {
			promptsx.InfoWithLastLine(fmt.Sprintf(
				"Would create fixup! commit for %s with %d file(s):\n     %s",
				commitHash[:7],
				len(files),
				strings.Join(files, "\n     "),
			))
			continue
		}

		// Unstage all files first
		if err := gitUnstageAll(workDir); err != nil {
			return fmt.Errorf("failed to unstage files: %w", err)
		}

		// Stage only the files for this commit
		if err := gitStageFiles(workDir, files); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}

		// Create fixup commit
		prompts.Info(fmt.Sprintf("Creating fixup! commit for %s (%d files)", commitHash[:7], len(files)))
		if err := gitCreateFixupCommit(workDir, commitHash); err != nil {
			return fmt.Errorf("failed to create fixup commit: %w", err)
		}
	}

	// If --and-rebase was specified, run the rebase
	if absorbFlags.AndRebase {
		// Find the parent of the oldest commit that needs to be fixed up
		baseCommit, err := gitFindOldestFixupParent(workDir, fixupCommits)
		if err != nil {
			return fmt.Errorf("failed to find base commit for rebase: %w", err)
		}

		rebaseCmdString := gitDebugRebaseAutosquash(workDir, baseCommit)

		if absorbFlags.DryRun {
			promptsx.InfoNoSplitLines("Would run: " + rebaseCmdString)
		} else {
			prompts.Info("Running 'git rebase --autosquash'...")

			if baseCommit == "" {
				// Fall back to the current branch if no fixup commits found
				currentBranch, err := gitCurrentBranch(workDir)
				if err != nil {
					return fmt.Errorf("failed to get current branch: %w", err)
				}
				baseCommit = currentBranch
			}

			if err := gitRebaseAutosquash(workDir, baseCommit); err != nil {
				promptsx.ErrorNoSplitLines("Rebase failed: " + rebaseCmdString)
				return fmt.Errorf("rebase failed: %w", err)
			}

			prompts.Outro(fmt.Sprintf("%s Rebase completed successfully", picocolors.Green("✔")))
		}
	} else if !absorbFlags.DryRun {
		promptsx.Note("Run 'git rebase --autosquash -i' to apply the fixup commits")
	}

	return nil
}
