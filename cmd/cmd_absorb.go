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

By default, the command searches up to 20 commits back in history. You can adjust
this limit using the --max-history flag.

After running this command, you can run 'git rebase -i --autosquash' to automatically
fold the fixup commits into their target commits.

If --and-rebase is specified, the rebase will be run automatically.`,
	RunE: runAbsorbE,
}

var absorbFlags = absorbOptions{
	AndRebase:  false,
	DryRun:     false,
	Backup:     false,
	All:        false,
	MaxHistory: 20,
}

func absorbAddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&absorbFlags.AndRebase, "and-rebase", "r", false, "Automatically run 'git rebase --autosquash' after creating fixups")
	cmd.Flags().BoolVarP(&absorbFlags.DryRun, "dry-run", "n", false, "Don't make any actual changes")
	cmd.Flags().BoolVarP(&absorbFlags.Backup, "backup", "b", false, "Create a backup branch before rebasing")
	cmd.Flags().BoolVarP(&absorbFlags.All, "all", "a", false, "Automatically stage all changes in tracked files")
	cmd.Flags().IntVar(&absorbFlags.MaxHistory, "max-history", 20, "Maximum number of commits to look back in history")
}

func init() {
	absorbAddFlags(absorbCmd)

	rootCmd.AddCommand(absorbCmd)
}

type absorbOptions struct {
	AndRebase  bool
	DryRun     bool
	Backup     bool
	All        bool
	MaxHistory int
}

func absorbSetup(cmd *cobra.Command) (string, error) {
	return setupGitWorkDir()
}

func absorbDetectAndStageFiles(workDir string, all bool) ([]string, error) {
	files, err := gitStagedFiles(workDir)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
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
			promptsx.Note(err.Error())
			return nil
		}
		return fmt.Errorf("error detecting staged files: %w", err)
	}

	// Find and process fixup commits
	fixupCommits, err := absorbFindFixupCommits(workDir, stagedFiles)
	if err != nil {
		return err
	}

	if len(fixupCommits) == 0 {
		promptsx.Note("No appropriate commits found to fixup. All changes appear to be new.")
		return nil
	}

	// Create fixup commits
	if err := absorbCreateFixupCommits(workDir, fixupCommits); err != nil {
		return err
	}

	// Handle rebase if requested
	if absorbFlags.AndRebase {
		return absorbHandleRebase(workDir, fixupCommits)
	}

	if !absorbFlags.DryRun {
		promptsx.Note("Run 'git rebase --autosquash -i' to apply the fixup commits")
	}

	return nil
}

func absorbFindFixupCommits(workDir string, stagedFiles []string) (map[string][]string, error) {
	fixupCommits := make(map[string][]string) // commit -> []files

	for _, file := range stagedFiles {
		// Use line-based analysis to find the best commit to target for fixup
		commitHash, err := gitFindBestCommitForFile(workDir, file, absorbFlags.MaxHistory)
		if err != nil {
			return nil, fmt.Errorf("failed to find best commit for %s: %w", file, err)
		}

		if commitHash == "" || commitHash == "0000000000000000000000000000000000000000" {
			continue
		}

		fixupCommits[commitHash] = append(fixupCommits[commitHash], file)
	}

	return fixupCommits, nil
}

func absorbCreateFixupCommits(workDir string, fixupCommits map[string][]string) error {
	for commitHash, files := range fixupCommits {
		if absorbFlags.DryRun {
			promptsx.InfoWithLastLine(fmt.Sprintf(
				"Would create fixup! commit for %s with %d file(s):\n     %s",
				commitHash[:7],
				len(files),
				strings.Join(files, "\n     "),
			))
			continue
		}

		if err := gitUnstageAll(workDir); err != nil {
			return fmt.Errorf("failed to unstage files: %w", err)
		}

		// Stage only the files for this commit
		if err := gitStageFiles(workDir, files); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}

		prompts.Info(fmt.Sprintf("Creating fixup! commit for %s (%d files)", commitHash[:7], len(files)))
		if err := gitCreateFixupCommit(workDir, commitHash); err != nil {
			return fmt.Errorf("failed to create fixup commit: %w", err)
		}
	}

	return nil
}

func absorbHandleRebase(workDir string, fixupCommits map[string][]string) error {
	baseCommit, err := gitFindOldestFixupParent(workDir, fixupCommits)
	if err != nil {
		return fmt.Errorf("failed to find base commit for rebase: %w", err)
	}

	rebaseCmdString := gitDebugRebaseAutosquash(workDir, baseCommit)

	if absorbFlags.DryRun {
		promptsx.InfoNoSplitLines("Would run: " + rebaseCmdString)
		return nil
	}

	// Create backup branch if requested
	backupBranch, err := absorbCreateBackupIfNeeded(workDir)
	if err != nil {
		return err
	}

	// Run the rebase
	if err := absorbExecuteRebase(workDir, baseCommit, backupBranch); err != nil {
		return err
	}

	// Show completion message
	if backupBranch != "" {
		prompts.Outro(fmt.Sprintf("%s Rebase completed successfully. Backup branch: %s",
			picocolors.Green("✔"), backupBranch))
	} else {
		prompts.Outro(fmt.Sprintf("%s Rebase completed successfully", picocolors.Green("✔")))
	}

	return nil
}

func absorbCreateBackupIfNeeded(workDir string) (string, error) {
	backupBranch, isNewBranch, err := createBackupBranchIfNeeded(workDir, absorbFlags.Backup)
	if err != nil {
		return "", err
	}

	if backupBranch != "" {
		if isNewBranch {
			prompts.Info(fmt.Sprintf("Created backup branch: %s", backupBranch))
		} else {
			prompts.Info(fmt.Sprintf("Using existing backup branch: %s", backupBranch))
		}
	}

	return backupBranch, nil
}

func absorbExecuteRebase(workDir, baseCommit, backupBranch string) error {
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
		rebaseCmdString := gitDebugRebaseAutosquash(workDir, baseCommit)
		promptsx.ErrorNoSplitLines("Rebase failed: " + rebaseCmdString)

		// Provide instructions to restore from backup if one was created
		if backupBranch != "" {
			errMsg := fmt.Sprintf("%s Rebase failed. To restore your original branch, run:\n    git checkout -f %s",
				picocolors.Red("✖"), backupBranch)
			return fmt.Errorf("%s\n%s", err, errMsg)
		}

		return fmt.Errorf("rebase failed: %w", err)
	}

	return nil
}
