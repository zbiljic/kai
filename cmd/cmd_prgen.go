package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orochaa/go-clack/prompts"
	"github.com/orochaa/go-clack/third_party/picocolors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/zbiljic/kai/pkg/llm"
)

var prgenCmd = &cobra.Command{
	Use: "prgen",
	Aliases: []string{
		"pr",
	},
	Short:       "Generate PR title and description",
	Long:        `Generates PR title and description based on commits and changes between the current branch and main branch`,
	Annotations: map[string]string{"group": "main"},
	Args:        cobra.ArbitraryArgs,
	RunE:        runPrGenE,
}

var prgenFlags = prgenOptions{
	Provider:    PhindProvider,
	Model:       "",
	BaseBranch:  "main",
	MaxDiffSize: llm.DefaultMaxDiffSize,
}

func prgenAddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&prgenFlags.BaseBranch, "base", "b", "main", "Base branch to compare against")
	cmd.Flags().IntVar(&prgenFlags.MaxDiffSize, "max-diff", 10000, "Maximum size of diff to send to LLM (in characters)")
	cmd.Flags().BoolVar(&prgenFlags.NoContext, "no-context", false, "Skip prompting for additional context about changes")
}

func init() {
	addCommonLLMFlags(prgenCmd, &prgenFlags.Provider, &prgenFlags.Model)
	prgenAddFlags(prgenCmd)

	rootCmd.AddCommand(prgenCmd)
}

type prgenOptions struct {
	Provider    ProviderType
	Model       string
	BaseBranch  string
	MaxDiffSize int
	NoContext   bool
}

// prgenSetupCommandClackIntro sets up clack intro and injects into command context
func prgenSetupCommandClackIntro(cmd *cobra.Command) {
	prompts.Intro(picocolors.BgCyan(picocolors.Black(fmt.Sprintf(" %s ", AppName))))
	// in order to show custom error
	injectIntoCommandContextWithKey(cmd, ctxKeyClackPromptStarted{}, true)
}

func prgenSetup(cmd *cobra.Command) (string, error) {
	prgenSetupCommandClackIntro(cmd)
	return setupGitWorkDir()
}

// getPRTemplate tries to find a PR template file in the repo
func getPRTemplate(workDir string) (string, string, error) {
	templatePaths := []string{
		filepath.Join(workDir, ".github", "pull_request_template.md"),
		filepath.Join(workDir, ".github", "PULL_REQUEST_TEMPLATE.md"),
		filepath.Join(workDir, "docs", "pull_request_template.md"),
		filepath.Join(workDir, ".gitlab", "merge_request_templates", "default.md"),
	}

	for _, path := range templatePaths {
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content), path, nil
		}
	}

	return "", "", errors.New("no PR template found")
}

func prgenGeneratePRContent(
	ctx context.Context,
	aip llm.AIPrompt,
	workDir,
	currentBranch,
	baseBranch,
	commits,
	diff,
	prContext,
	prTemplate string,
) (string, string, error) {
	generatePRSpinner := prompts.Spinner(prompts.SpinnerOptions{})
	generatePRSpinner.Start("Generating PR content")
	generatePRSpinner.Message(fmt.Sprintf("Analyzing changes with %s", aip.String()))

	title, description, err := llm.GeneratePRContent(
		ctx,
		aip,
		currentBranch,
		baseBranch,
		commits,
		diff,
		prContext,
		prTemplate,
		prgenFlags.MaxDiffSize,
	)
	if err != nil {
		generatePRSpinner.Stop("Failed to generate PR content", 1)
		return "", "", err
	}

	generatePRSpinner.Stop("PR content generated successfully", 0)
	return title, description, nil
}

func runPrGenE(cmd *cobra.Command, args []string) error {
	workDir, err := prgenSetup(cmd)
	if err != nil {
		return err
	}

	currentBranch, err := gitCurrentBranch(workDir)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if we're not on the base branch
	if currentBranch == prgenFlags.BaseBranch {
		return fmt.Errorf("you are currently on the %s branch - please switch to your feature branch", prgenFlags.BaseBranch)
	}

	// Check if base branch exists
	baseBranchExists := gitBranchExists(workDir, prgenFlags.BaseBranch)
	if !baseBranchExists {
		// Check if it exists as a remote branch
		if gitRemoteBranchExists(workDir, prgenFlags.BaseBranch) {
			prgenFlags.BaseBranch = "origin/" + prgenFlags.BaseBranch
		} else {
			return fmt.Errorf("base branch '%s' does not exist locally or remotely", prgenFlags.BaseBranch)
		}
	}

	// Ask for additional context about the changes if not disabled
	prContext := ""
	if !prgenFlags.NoContext {
		prContext, err = prompts.Text(prompts.TextParams{
			Message:     "Provide additional context about changes",
			Placeholder: "<optional context>",
		})
		if err != nil {
			return fmt.Errorf("failed to get additional context: %w", err)
		}
	}

	prompts.Info(fmt.Sprintf("Comparing branch %s with %s", picocolors.Cyan(currentBranch), picocolors.Cyan(prgenFlags.BaseBranch)))

	// Try to get PR template
	prTemplate, prTemplatePath, _ := getPRTemplate(workDir)
	// Note: we ignore the error since template is optional

	if prTemplate != "" {
		fileRelPath := lo.Must(filepath.Rel(workDir, prTemplatePath))
		prompts.Info(fmt.Sprintf("Using PR template: %s", picocolors.Cyan(fileRelPath)))
	}

	fetchingSpinner := prompts.Spinner(prompts.SpinnerOptions{})
	fetchingSpinner.Start("Fetching commits between branches")

	commits, err := gitGetCommitsBetweenBranches(workDir, prgenFlags.BaseBranch)
	if err != nil {
		fetchingSpinner.Stop("Failed to get commits", 1)
		return fmt.Errorf("failed to get commits: %w", err)
	}

	if commits == "" {
		fetchingSpinner.Stop("No commits found", 1)
		return fmt.Errorf("no commits found between %s and %s", prgenFlags.BaseBranch, currentBranch)
	}

	fetchingSpinner.Message("Fetching code diff between branches")

	diff, err := gitGetDiffBetweenBranches(workDir, prgenFlags.BaseBranch)
	if err != nil {
		fetchingSpinner.Stop("Failed to get diff", 1)
		return fmt.Errorf("failed to get diff: %w", err)
	}

	fetchingSpinner.Stop(fmt.Sprintf("Found %d commits and %d bytes of diff", strings.Count(commits, "---COMMIT---"), len(diff)), 0)

	providerSpinner := prompts.Spinner(prompts.SpinnerOptions{})
	providerSpinner.Start("Initializing LLM provider")

	aip, err := initializeLLMProvider(cmd.Flags().Changed("provider"), prgenFlags.Provider, prgenFlags.Model)
	if err != nil {
		providerSpinner.Stop("Failed to initialize LLM provider", 1)
		return err
	}

	providerSpinner.Stop(fmt.Sprintf("Using %s", aip.String()), 0)

	title, description, err := prgenGeneratePRContent(
		cmd.Context(),
		aip,
		workDir,
		currentBranch,
		prgenFlags.BaseBranch,
		commits,
		diff,
		prContext,
		prTemplate,
	)
	if err != nil {
		return err
	}

	// Display the generated PR content
	fmt.Println("")
	fmt.Printf("%s %s\n", picocolors.Bold("PR Title:"), picocolors.Cyan(title))

	fmt.Println("")
	fmt.Printf("%s\n\n", picocolors.Bold("PR Description:"))

	// Split description into lines and display with formatting
	descLines := strings.Split(description, "\n")
	for _, line := range descLines {
		fmt.Printf("%s\n", picocolors.Cyan(line))
	}
	fmt.Println("")

	return nil
}
