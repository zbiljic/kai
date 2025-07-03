package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/orochaa/go-clack/prompts"
	"github.com/orochaa/go-clack/third_party/picocolors"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag/v2"

	"github.com/zbiljic/kai/pkg/llm"
)

var prprepareCmd = &cobra.Command{
	Use: "prprepare",
	Aliases: []string{
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
}

func prprepareAddFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(enumflag.New(&prprepareFlags.Provider, "provider", ProviderIds, enumflag.EnumCaseInsensitive), "provider", "p", "LLM provider to use for generating reorganized commits (phind, openai, googleai, openrouter)")
	cmd.Flags().StringVarP(&prprepareFlags.Model, "model", "m", "", "Specific model to use for the selected provider")
	cmd.Flags().StringVarP(&prprepareFlags.BaseBranch, "base", "b", "main", "Base branch to compare against")
	cmd.Flags().IntVar(&prprepareFlags.MaxDiffSize, "max-diff", 10000, "Maximum size of diff to send to LLM (in characters)")
	cmd.Flags().BoolVar(&prprepareFlags.AutoApply, "auto-apply", false, "Automatically apply the reorganization without confirmation")
}

func init() {
	prprepareAddFlags(prprepareCmd)

	rootCmd.AddCommand(prprepareCmd)
}

type prprepareOptions struct {
	Provider    ProviderType
	Model       string
	BaseBranch  string
	MaxDiffSize int
	AutoApply   bool
}

// prprepareSetupCommandClackIntro sets up clack intro and injects into command context
func prprepareSetupCommandClackIntro(cmd *cobra.Command) {
	prompts.Intro(picocolors.BgCyan(picocolors.Black(fmt.Sprintf(" %s ", AppName))))
	// in order to show custom error
	injectIntoCommandContextWithKey(cmd, ctxKeyClackPromptStarted{}, true)
}

func prprepareSetup(cmd *cobra.Command) (string, error) {
	prprepareSetupCommandClackIntro(cmd)
	return setupGitWorkDir()
}

// generateReorganizationPlan uses AI to analyze commits and generate a reorganization plan
func generateReorganizationPlan(
	ctx context.Context,
	aip llm.AIPrompt,
	currentBranch,
	baseBranch,
	commits,
	diff string,
) (string, error) {
	spinner := prompts.Spinner(prompts.SpinnerOptions{})
	spinner.Start("Analyzing commit history")
	spinner.Message(fmt.Sprintf("Using %s to generate reorganization plan", aip.String()))

	// Create a prompt for reorganizing commits
	prompt := fmt.Sprintf(`You are a Git expert. Analyze the following commit history and diff to create a reorganization plan that groups related changes into logical, clean commits.

Current branch: %s
Base branch: %s

Commit history:
%s

Code diff:
%s

Please provide a reorganization plan that:
1. Groups related changes together
2. Creates logical, atomic commits
3. Follows conventional commit message format
4. Maintains chronological order where possible
5. Removes unnecessary "fix", "typo", "WIP" commits by incorporating them into main feature commits

Return your response in this format:
REORGANIZATION PLAN:
1. [commit message] - [description of what changes go in this commit]
2. [commit message] - [description of what changes go in this commit]
...

JUSTIFICATION:
[Brief explanation of why this reorganization makes sense]`, currentBranch, baseBranch, commits, diff)

	// Create system prompt for reorganization
	systemPrompt := "You are a Git expert specializing in commit history reorganization. Your task is to analyze messy commit histories and create clean, logical reorganization plans."

	responses, err := aip.Generate(ctx, systemPrompt, prompt, 1)
	if err != nil {
		spinner.Stop("Failed to generate reorganization plan", 1)
		return "", fmt.Errorf("failed to generate reorganization plan: %w", err)
	}

	if len(responses) == 0 {
		spinner.Stop("No reorganization plan generated", 1)
		return "", fmt.Errorf("no reorganization plan was generated")
	}

	plan := responses[0]

	spinner.Stop("Reorganization plan generated", 0)
	return plan, nil
}

func runPrPrepareE(cmd *cobra.Command, args []string) error {
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

	prompts.Info(fmt.Sprintf("Analyzing commits on branch %s compared to %s", picocolors.Cyan(currentBranch), picocolors.Cyan(prprepareFlags.BaseBranch)))

	// Create backup branch first
	backupSpinner := prompts.Spinner(prompts.SpinnerOptions{})
	backupSpinner.Start("Creating backup branch")

	backupBranch, err := gitCreateBackupBranch(workDir)
	if err != nil {
		backupSpinner.Stop("Failed to create backup", 1)
		return err
	}

	backupSpinner.Stop(fmt.Sprintf("Created backup branch: %s", picocolors.Green(backupBranch)), 0)

	fetchingSpinner := prompts.Spinner(prompts.SpinnerOptions{})
	fetchingSpinner.Start("Fetching commits and changes")

	commits, err := gitGetCommitsBetweenBranches(workDir, prprepareFlags.BaseBranch)
	if err != nil {
		fetchingSpinner.Stop("Failed to get commits", 1)
		return fmt.Errorf("failed to get commits: %w", err)
	}

	if commits == "" {
		fetchingSpinner.Stop("No commits found", 1)
		return fmt.Errorf("no commits found between %s and %s", prprepareFlags.BaseBranch, currentBranch)
	}

	fetchingSpinner.Message("Fetching code diff")

	diff, err := gitGetDiffBetweenBranches(workDir, prprepareFlags.BaseBranch)
	if err != nil {
		fetchingSpinner.Stop("Failed to get diff", 1)
		return fmt.Errorf("failed to get diff: %w", err)
	}

	commitCount := strings.Count(commits, "---COMMIT---")
	fetchingSpinner.Stop(fmt.Sprintf("Found %d commits and %d bytes of changes", commitCount, len(diff)), 0)

	// If only one commit, no need to reorganize
	if commitCount <= 1 {
		prompts.Info("Only one commit found - no reorganization needed")
		return nil
	}

	providerSpinner := prompts.Spinner(prompts.SpinnerOptions{})
	providerSpinner.Start("Initializing LLM provider")

	aip, err := initializeLLMProvider(cmd.Flags().Changed("provider"), prprepareFlags.Provider, prprepareFlags.Model)
	if err != nil {
		providerSpinner.Stop("Failed to initialize LLM provider", 1)
		return err
	}

	providerSpinner.Stop(fmt.Sprintf("Using %s", aip.String()), 0)

	plan, err := generateReorganizationPlan(
		cmd.Context(),
		aip,
		currentBranch,
		prprepareFlags.BaseBranch,
		commits,
		diff,
	)
	if err != nil {
		return err
	}

	// Display the reorganization plan
	fmt.Println("")
	fmt.Printf("%s\n", picocolors.Bold("Reorganization Plan:"))
	fmt.Println("")

	// Split plan into lines and display with formatting
	planLines := strings.Split(plan, "\n")
	for _, line := range planLines {
		if strings.TrimSpace(line) != "" {
			fmt.Printf("%s\n", picocolors.Cyan(line))
		}
	}
	fmt.Println("")

	// Ask for confirmation unless auto-apply is enabled
	if !prprepareFlags.AutoApply {
		confirmed, err := prompts.Confirm(prompts.ConfirmParams{
			Message: "Apply this reorganization plan?",
		})
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}

		if !confirmed {
			prompts.Info("Reorganization cancelled")
			prompts.Info(fmt.Sprintf("Your backup branch %s is available if needed", picocolors.Green(backupBranch)))
			return nil
		}
	}

	// For now, we'll just show the plan and inform the user about manual steps
	// In a full implementation, this would actually reorganize the commits
	prompts.Info("This is a preview implementation showing the AI-generated reorganization plan")
	prompts.Info("To implement the actual reorganization, you would need to:")
	prompts.Info("1. Reset to the base branch")
	prompts.Info("2. Apply changes according to the plan using git cherry-pick or manual commits")
	prompts.Info("3. Create new commit messages as suggested")
	prompts.Info("")
	prompts.Info(fmt.Sprintf("Your original commits are safely backed up in: %s", picocolors.Green(backupBranch)))

	return nil
}
