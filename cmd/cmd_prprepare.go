package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orochaa/go-clack/prompts"
	"github.com/orochaa/go-clack/third_party/picocolors"
	"github.com/spf13/cobra"

	"github.com/zbiljic/kai/pkg/gitdiff"
	"github.com/zbiljic/kai/pkg/llm"
	"github.com/zbiljic/kai/pkg/promptsx"
)

var prprepareCmd = &cobra.Command{
	Use: "prprepare",
	Aliases: []string{
		"prpare",
		"prp",
	},
	Short:       "Transform messy commit history into clean, logical commits",
	Long:        `Uses AI to analyze your commit history and reorganize it into clean, logical commits that are easier to review`,
	Annotations: map[string]string{"group": "main"},
	Args:        cobra.ArbitraryArgs,
	RunE:        runPrPrepareE,
}

var prprepareFlags = prprepareOptions{
	Provider:    PhindProvider,
	Model:       "",
	BaseBranch:  "main",
	MaxDiffSize: llm.DefaultMaxDiffSize,
	AutoApply:   false,
	DryRun:      false,
	Debug:       false,
}

func prprepareAddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&prprepareFlags.BaseBranch, "base", "b", "main", "Base branch to compare against")
	cmd.Flags().IntVar(&prprepareFlags.MaxDiffSize, "max-diff", 10000, "Maximum size of diff to send to LLM (in characters)")
	cmd.Flags().BoolVar(&prprepareFlags.AutoApply, "auto-apply", false, "Automatically apply the reorganization without confirmation")
	cmd.Flags().BoolVarP(&prprepareFlags.DryRun, "dry-run", "n", false, "Don't make any actual changes, just show what would be done")
	cmd.Flags().BoolVar(&prprepareFlags.Debug, "debug", false, "Write each generated patch to .kai/prprepare and print git apply commands (implies dry-run if used alone)")
}

func init() {
	addCommonLLMFlags(prprepareCmd, &prprepareFlags.Provider, &prprepareFlags.Model)
	prprepareAddFlags(prprepareCmd)

	rootCmd.AddCommand(prprepareCmd)
}

type prprepareOptions struct {
	Provider    ProviderType
	Model       string
	BaseBranch  string
	MaxDiffSize int
	AutoApply   bool
	DryRun      bool
	Debug       bool
}

// prprepareSetupCommandClackIntro sets up clack intro and injects into command context
func prprepareSetupCommandClackIntro(cmd *cobra.Command) {
	prompts.Intro(picocolors.BgCyan(picocolors.Black(fmt.Sprintf(" %s ", AppName))))
	// in order to show custom error
	injectIntoCommandContextWithKey(cmd, ctxKeyClackPromptStarted{}, true)
}

func prprepareSetup(cmd *cobra.Command) (string, error) {
	prprepareSetupCommandClackIntro(cmd)
	if prprepareFlags.Debug && !prprepareFlags.DryRun {
		prprepareFlags.DryRun = true
	}
	return setupGitWorkDir()
}

// prprepareApplyCommitPlan executes the AI-generated commit plan
func prprepareApplyCommitPlan(
	workDir string,
	commitPlan *llm.CommitPlan,
	hunks []*gitdiff.Hunk,
	baseBranch string,
	dryRun bool,
) error {
	// Create hunk lookup map
	hunkMap := make(map[string]*gitdiff.Hunk)
	for _, hunk := range hunks {
		hunkMap[hunk.ID] = hunk
	}

	if dryRun {
		promptsx.Note(fmt.Sprintf("Would reset to base branch: %s", baseBranch))
	} else {
		// Reset to base branch to start clean
		resetSpinner := prompts.Spinner(prompts.SpinnerOptions{})
		resetSpinner.Start("Resetting to base branch")

		err := gitResetHard(workDir, baseBranch)
		if err != nil {
			resetSpinner.Stop("Failed to reset to base branch", 1)
			return fmt.Errorf("failed to reset to base branch: %w", err)
		}

		resetSpinner.Stop("Reset to base branch", 0)
	}

	// Get the original diff for patch creation
	originalDiff, err := gitGetDiffBetweenBranches(workDir, baseBranch)
	if err != nil {
		return fmt.Errorf("failed to get original diff: %w", err)
	}

	// Apply each commit in the plan
	for i, plannedCommit := range commitPlan.Commits {
		if dryRun {
			// Validate that all hunks exist
			for _, hunkID := range plannedCommit.HunkIDs {
				if _, exists := hunkMap[hunkID]; !exists {
					return fmt.Errorf("hunk %s not found in parsed hunks", hunkID)
				}
			}

			if prprepareFlags.Debug {
				hunksForCommit := getHunksForCommit(plannedCommit.HunkIDs, hunkMap)
				patchContent := gitdiff.CreateHunkPatch(hunksForCommit, originalDiff)
				debugDir := filepath.Join(workDir, ".kai", "prprepare")
				_ = os.MkdirAll(debugDir, 0o755)
				patchPath := filepath.Join(debugDir, fmt.Sprintf("%03d.patch", i+1))
				_ = os.WriteFile(patchPath, []byte(patchContent), 0o644)
				promptsx.Note(fmt.Sprintf("[debug] Patch written to %s\n       git apply --check --cached %s", patchPath, patchPath))
			}

			promptsx.InfoWithLastLine(fmt.Sprintf(
				"Would create commit %d/%d:\n   Message: %s\n   Hunks: %s",
				i+1,
				len(commitPlan.Commits),
				plannedCommit.Message,
				strings.Join(plannedCommit.HunkIDs, ", "),
			))
		} else {
			commitSpinner := prompts.Spinner(prompts.SpinnerOptions{})
			commitSpinner.Start(fmt.Sprintf("Applying commit %d/%d", i+1, len(commitPlan.Commits)))
			commitSpinner.Message(fmt.Sprintf("Message: %s", plannedCommit.Message))

			// Reset staging area
			err := gitUnstageAll(workDir)
			if err != nil {
				commitSpinner.Stop("Failed to unstage files", 1)
				return fmt.Errorf("failed to reset staging area: %w", err)
			}

			// Validate that all hunks exist
			for _, hunkID := range plannedCommit.HunkIDs {
				if _, exists := hunkMap[hunkID]; !exists {
					commitSpinner.Stop("Invalid hunk reference", 1)
					return fmt.Errorf("hunk %s not found in parsed hunks", hunkID)
				}
			}

			// Apply specific hunks for this commit using gitdiff package
			hunksForCommit := getHunksForCommit(plannedCommit.HunkIDs, hunkMap)
			err = gitdiff.ApplyHunks(plannedCommit.HunkIDs, hunkMap, originalDiff)
			if err != nil {
				commitSpinner.Stop("Failed to apply hunks", 1)
				return fmt.Errorf("failed to apply hunks for commit %d: %w", i+1, err)
			}

			// Create the commit
			err = gitCommit(workDir, plannedCommit.Message)
			if err != nil {
				commitSpinner.Stop("Failed to create commit", 1)
				return fmt.Errorf("failed to create commit: %w", err)
			}

			// If debug flag is set, write the patch file for manual inspection
			if prprepareFlags.Debug {
				patchContent := gitdiff.CreateHunkPatch(hunksForCommit, originalDiff)

				debugDir := filepath.Join(workDir, ".kai", "prprepare")
				_ = os.MkdirAll(debugDir, 0o755)
				patchPath := filepath.Join(debugDir, fmt.Sprintf("%03d.patch", i+1))
				_ = os.WriteFile(patchPath, []byte(patchContent), 0o644)

				promptsx.Note(fmt.Sprintf("[debug] Patch written to %s\n       git apply --check --cached %s", patchPath, patchPath))
			}

			commitSpinner.Stop(fmt.Sprintf("Applied: %s", plannedCommit.Message), 0)
		}
	}

	return nil
}

// getHunksForCommit returns the hunks for a given set of hunk IDs
func getHunksForCommit(hunkIDs []string, hunkMap map[string]*gitdiff.Hunk) []*gitdiff.Hunk {
	var hunks []*gitdiff.Hunk
	for _, hunkID := range hunkIDs {
		if hunk, exists := hunkMap[hunkID]; exists {
			hunks = append(hunks, hunk)
		}
	}
	return hunks
}

// prprepareCreateBackupIfNeeded creates a backup branch if needed, checking for existing ones first
func prprepareCreateBackupIfNeeded(workDir string) (string, error) {
	backupBranch, err := createBackupBranchIfNeeded(workDir, true)
	if err != nil {
		return "", err
	}

	if backupBranch != "" {
		prompts.Info(fmt.Sprintf("Using existing backup branch: %s", backupBranch))
	}

	return backupBranch, nil
}

func runPrPrepareE(cmd *cobra.Command, args []string) error {
	// Enable debug logging if debug flag is set
	if prprepareFlags.Debug {
		gitdiff.EnableDebug()
	}

	workDir, err := prprepareSetup(cmd)
	if err != nil {
		return err
	}

	currentBranch, err := gitCurrentBranch(workDir)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if we're not on the base branch
	if currentBranch == prprepareFlags.BaseBranch {
		return fmt.Errorf("you are currently on the %s branch - please switch to your feature branch", prprepareFlags.BaseBranch)
	}

	// Check if base branch exists
	baseBranchExists := gitBranchExists(workDir, prprepareFlags.BaseBranch)
	if !baseBranchExists {
		// Check if it exists as a remote branch
		if gitRemoteBranchExists(workDir, prprepareFlags.BaseBranch) {
			prprepareFlags.BaseBranch = "origin/" + prprepareFlags.BaseBranch
		} else {
			return fmt.Errorf("base branch '%s' does not exist locally or remotely", prprepareFlags.BaseBranch)
		}
	}

	if prprepareFlags.DryRun {
		promptsx.Note("Running in dry-run mode. No changes will be made.")
	}

	prompts.Info(fmt.Sprintf("Analyzing commits on branch %s compared to %s", picocolors.Cyan(currentBranch), picocolors.Cyan(prprepareFlags.BaseBranch)))

	// Create backup branch first (unless dry-run)
	var backupBranch string
	if prprepareFlags.DryRun {
		promptsx.Note("Would create backup branch")
	} else {
		backupSpinner := prompts.Spinner(prompts.SpinnerOptions{})
		backupSpinner.Start("Creating backup branch")

		backupBranch, err = prprepareCreateBackupIfNeeded(workDir)
		if err != nil {
			backupSpinner.Stop("Failed to create backup", 1)
			return err
		}

		backupSpinner.Stop(fmt.Sprintf("Created backup branch: %s", picocolors.Green(backupBranch)), 0)
	}

	// Get the diff between branches
	fetchingSpinner := prompts.Spinner(prompts.SpinnerOptions{})
	fetchingSpinner.Start("Fetching code changes")

	diff, err := gitGetDiffBetweenBranches(workDir, prprepareFlags.BaseBranch)
	if err != nil {
		fetchingSpinner.Stop("Failed to get diff", 1)
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if strings.TrimSpace(diff) == "" {
		fetchingSpinner.Stop("No changes found", 1)
		return fmt.Errorf("no changes found between %s and %s", prprepareFlags.BaseBranch, currentBranch)
	}

	fetchingSpinner.Stop(fmt.Sprintf("Found %d bytes of changes", len(diff)), 0)

	// Parse diff into hunks
	parseSpinner := prompts.Spinner(prompts.SpinnerOptions{})
	parseSpinner.Start("Parsing code changes")

	hunks, err := gitdiff.ParseDiff(diff)
	if err != nil {
		parseSpinner.Stop("Failed to parse changes", 1)
		return fmt.Errorf("failed to parse diff into hunks: %w", err)
	}

	if len(hunks) == 0 {
		parseSpinner.Stop("No code hunks found", 1)
		return fmt.Errorf("no code hunks found in diff")
	}

	parseSpinner.Stop(fmt.Sprintf("Parsed %d code hunks", len(hunks)), 0)

	// Initialize LLM provider
	providerSpinner := prompts.Spinner(prompts.SpinnerOptions{})
	providerSpinner.Start("Initializing LLM provider")

	aip, err := initializeLLMProvider(cmd.Flags().Changed("provider"), prprepareFlags.Provider, prprepareFlags.Model)
	if err != nil {
		providerSpinner.Stop("Failed to initialize LLM provider", 1)
		return err
	}

	providerSpinner.Stop(fmt.Sprintf("Using %s", aip.String()), 0)

	// Generate commit plan
	spinner := prompts.Spinner(prompts.SpinnerOptions{})
	spinner.Start("Analyzing code changes")
	spinner.Message(fmt.Sprintf("Using %s to generate commit plan", aip.String()))

	commitPlan, err := llm.GenerateCommitPlan(
		cmd.Context(),
		aip,
		hunks,
		currentBranch,
		prprepareFlags.BaseBranch,
	)
	if err != nil {
		spinner.Stop("Failed to generate commit plan", 1)
		return err
	}

	spinner.Stop("Commit plan generated", 0)

	// Display the commit plan
	fmt.Println("")
	fmt.Printf("%s\n", picocolors.Bold("Commit Reorganization Plan:"))
	fmt.Println("")

	for i, commit := range commitPlan.Commits {
		fmt.Printf("%s %s\n", picocolors.Bold(fmt.Sprintf("%d.", i+1)), picocolors.Cyan(commit.Message))
		fmt.Printf("   %s %s\n", picocolors.Dim("Rationale:"), commit.Rationale)
		fmt.Printf("   %s %s\n", picocolors.Dim("Hunks:"), strings.Join(commit.HunkIDs, ", "))
		fmt.Println("")
	}

	// Ask for confirmation unless auto-apply or dry-run is enabled
	if !prprepareFlags.AutoApply && !prprepareFlags.DryRun {
		confirmed, err := prompts.Confirm(prompts.ConfirmParams{
			Message: "Apply this commit reorganization plan?",
		})
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}

		if !confirmed {
			prompts.Info("Reorganization cancelled")
			if backupBranch != "" {
				prompts.Info(fmt.Sprintf("Your backup branch %s is available if needed", picocolors.Green(backupBranch)))
			}
			return nil
		}
	}

	// Apply the commit plan
	err = prprepareApplyCommitPlan(workDir, commitPlan, hunks, prprepareFlags.BaseBranch, prprepareFlags.DryRun)
	if err != nil {
		if !prprepareFlags.DryRun {
			prompts.Error("Failed to apply commit plan")
			if backupBranch != "" {
				prompts.Info(fmt.Sprintf("Your original commits are safely backed up in: %s", picocolors.Green(backupBranch)))
			}
		}
		return err
	}

	if prprepareFlags.DryRun {
		promptsx.Note("Dry-run completed. No actual changes were made.")
	} else {
		prompts.Success("Commit reorganization completed successfully!")
		if backupBranch != "" {
			prompts.Info(fmt.Sprintf("Your original commits are backed up in: %s", picocolors.Green(backupBranch)))
		}
	}

	return nil
}
