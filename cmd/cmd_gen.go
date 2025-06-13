package cmd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/orochaa/go-clack/prompts"
	"github.com/orochaa/go-clack/third_party/picocolors"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag/v2"

	"github.com/zbiljic/kai/pkg/commit"
	"github.com/zbiljic/kai/pkg/llm"
	"github.com/zbiljic/kai/pkg/llm/provider"
	"github.com/zbiljic/kai/pkg/promptsx"
)

// ProviderType represents the supported LLM providers.
type ProviderType enumflag.Flag

const (
	// PhindProvider represents the Phind provider.
	PhindProvider ProviderType = iota
	// OpenAIProvider represents the OpenAI provider.
	OpenAIProvider
	// GoogleAIProvider represents the GoogleAI provider.
	GoogleAIProvider
	// OpenRouterProvider represents the OpenRouter provider.
	OpenRouterProvider
)

// ProviderIds maps ProviderType to their string representations.
var ProviderIds = map[ProviderType][]string{
	PhindProvider:      {"phind"},
	OpenAIProvider:     {"openai"},
	GoogleAIProvider:   {"googleai"},
	OpenRouterProvider: {"openrouter"},
}

var genCmd = &cobra.Command{
	Use: "gen",
	Aliases: []string{
		"g",
		"generate",
	},
	Short:       "Generate commit message",
	Long:        `Generates commit message based on staged changes. Can optionally include previous commit messages for similar files as examples to maintain consistent style.`,
	Annotations: map[string]string{"group": "main"},
	Args:        cobra.ArbitraryArgs,
	RunE:        runGenE,
}

var genFlags = genOptions{
	Type:           commit.ConventionalType,
	Provider:       PhindProvider,
	All:            false,
	IncludeHistory: true,
	CandidateCount: 2,
	Yes:            false,
}

func genAddFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(enumflag.New(&genFlags.Type, "type", commit.TypeIds, enumflag.EnumCaseInsensitive), "type", "t", "Type of commit message to generate")
	cmd.Flags().VarP(enumflag.New(&genFlags.Provider, "provider", ProviderIds, enumflag.EnumCaseInsensitive), "provider", "p", "LLM provider to use for generating commit messages (phind, openai, googleai, openrouter)")
	cmd.Flags().BoolVarP(&genFlags.All, "all", "a", false, "Automatically stage all changes in tracked files")
	cmd.Flags().BoolVar(&genFlags.IncludeHistory, "history", true, "Include previous commit messages as examples")
	cmd.Flags().IntVarP(&genFlags.CandidateCount, "count", "n", 2, "Number of commit message suggestions to generate")
	cmd.Flags().BoolVarP(&genFlags.Yes, "yes", "y", false, "Run in non-interactive mode, automatically using the first generated commit message")
}

func init() {
	genAddFlags(genCmd)

	rootCmd.AddCommand(genCmd)
}

type genOptions struct {
	Type           commit.Type
	Provider       ProviderType
	All            bool
	IncludeHistory bool
	CandidateCount int
	Yes            bool
}

func genSetup(cmd *cobra.Command) (string, error) {
	if !genFlags.Yes {
		prompts.Intro(picocolors.BgCyan(picocolors.Black(fmt.Sprintf(" %s ", AppName))))
		// in order to show custom error
		injectIntoCommandContextWithKey(cmd, ctxKeyClackPromptStarted{}, true)
	}

	workDir, err := gitWorkingTreeDir(getWd())
	if err != nil {
		return "", errors.New("The current directory must be a Git repository") //nolint:staticcheck
	}
	return workDir, nil
}

func genDetectAndStageFiles(workDir string, all bool) ([]string, string, error) {
	var detectingFilesSpinner *prompts.SpinnerController
	if !genFlags.Yes {
		detectingFilesSpinner = prompts.Spinner(prompts.SpinnerOptions{})
		detectingFilesSpinner.Start("Detecting staged files")
	}

	// Check for staged files first
	files, diff, err := gitDiffStaged(workDir)
	if err != nil {
		if !genFlags.Yes && detectingFilesSpinner != nil {
			detectingFilesSpinner.Stop("Error detecting staged files", 1)
		}
		return nil, "", err
	}

	// If no files are staged, automatically set All flag to true
	if len(files) == 0 {
		all = true
	}

	if all {
		if err := gitAddAll(workDir); err != nil {
			if !genFlags.Yes && detectingFilesSpinner != nil {
				detectingFilesSpinner.Stop("Error staging files", 1)
			}
			return nil, "", err
		}

		// Get updated list of staged files after adding all
		files, diff, err = gitDiffStaged(workDir)
		if err != nil {
			if !genFlags.Yes && detectingFilesSpinner != nil {
				detectingFilesSpinner.Stop("Error detecting staged files", 1)
			}
			return nil, "", err
		}

		if len(files) == 0 {
			if !genFlags.Yes && detectingFilesSpinner != nil {
				detectingFilesSpinner.Stop("No changes detected to stage", 0)
			}
			return nil, "", errors.New("No changes detected to stage") //nolint:staticcheck
		}
	}

	detectedMessage := fmt.Sprintf(
		"Detected %d staged file(s):\n     %s",
		len(files),
		strings.Join(files, "\n     "),
	)

	if !genFlags.Yes && detectingFilesSpinner != nil {
		detectingFilesSpinner.Stop(detectedMessage, 0)
	}
	return files, diff, nil
}

func genInitializeLLMProvider(cmd *cobra.Command, providerType ProviderType) (llm.AIPrompt, error) {
	if cmd.Flags().Changed("provider") {
		switch providerType {
		case OpenAIProvider:
			return provider.NewOpenAIProvider(), nil
		case GoogleAIProvider:
			return provider.NewGoogleAIProvider()
		case OpenRouterProvider:
			return provider.NewOpenRouterProvider(), nil
		case PhindProvider:
			return provider.NewPhindProvider(), nil
		}
	}

	// Try providers in preferred order
	providers := []struct {
		create func() (llm.AIPrompt, error)
	}{
		{create: func() (llm.AIPrompt, error) { return provider.NewGoogleAIProvider() }},
		{create: func() (llm.AIPrompt, error) { return provider.NewOpenRouterProvider(), nil }},
		{create: func() (llm.AIPrompt, error) { return provider.NewOpenAIProvider(), nil }},
		{create: func() (llm.AIPrompt, error) { return provider.NewPhindProvider(), nil }},
	}

	for _, p := range providers {
		provider, err := p.create()
		if err != nil {
			continue
		}
		if provider.IsAvailable() {
			return provider, nil
		}
	}

	return nil, errors.New("no available LLM providers found - please configure at least one provider's API key")
}

// genGetPreviousCommitsForStagedFiles returns previous commit messages for all staged files.
func genGetPreviousCommitsForStagedFiles(workDir string) ([]string, error) {
	// Get staged files
	stagedFiles, err := gitStagedFiles(workDir)
	if err != nil {
		return nil, err
	}

	// If no staged files, return empty slice
	if len(stagedFiles) == 0 {
		return []string{}, nil
	}

	// Get previous commit messages for staged files
	allMessages := make(map[string]struct{})
	for _, file := range stagedFiles {
		fileMessages, err := gitPreviousCommitMessages(workDir, []string{file}, llm.DefaultMaxCommitsPerFile)
		if err != nil {
			continue // Skip files with errors
		}

		// Add messages to the deduplicated set
		for _, msg := range fileMessages {
			allMessages[msg] = struct{}{}
		}

		// Limit the total number of messages
		if len(allMessages) >= llm.DefaultMaxTotalCommits {
			break
		}
	}

	// Convert map to slice
	var result []string
	for msg := range allMessages {
		result = append(result, msg)
		if len(result) >= llm.DefaultMaxTotalCommits {
			break
		}
	}

	return result, nil
}

func genMessages(ctx context.Context, aip llm.AIPrompt, commitType commit.Type, workDir, diff string) ([]string, error) {
	var generateMessageSpinner *prompts.SpinnerController
	if !genFlags.Yes {
		generateMessageSpinner = prompts.Spinner(prompts.SpinnerOptions{})
		generateMessageSpinner.Start("Generating commit message")
		generateMessageSpinner.Message(fmt.Sprintf("Generating commit message with %s", aip.String()))
	}

	var messages []string
	var err error

	// Decide whether to include commit history based on the flag
	if genFlags.IncludeHistory {
		// Get previous commit messages for staged files
		var previousCommits []string
		previousCommits, err = genGetPreviousCommitsForStagedFiles(workDir)
		if err == nil {
			messages, err = llm.GenerateCommitMessageWithPreviousCommits(ctx, aip, commitType, workDir, diff, previousCommits, genFlags.CandidateCount)
		}
	} else {
		messages, err = llm.GenerateCommitMessage(ctx, aip, commitType, diff, genFlags.CandidateCount)
	}

	if err != nil {
		return nil, err
	}

	if !genFlags.Yes && generateMessageSpinner != nil {
		generateMessageSpinner.Stop("Changes analyzed", 0)
	}

	return filterAndProcessMessages(messages)
}

// filterAndProcessMessages removes empty messages and formats them properly
func filterAndProcessMessages(messages []string) ([]string, error) {
	// remove empty messages
	messages = slice.Filter(messages, func(_ int, s string) bool {
		return strutil.IsNotBlank(s)
	})

	if len(messages) == 0 {
		return nil, errors.New("No commit messages were generated. Try again.") //nolint:staticcheck
	}

	// lowercase the first letter of commit message
	return slice.Map(messages, func(_ int, s string) string {
		m := commit.ParseMessage(s)
		m.CommitMessage = strutil.LowerFirst(m.CommitMessage)
		return m.ToString()
	}), nil
}

func genHandleMessageSelection(messages []string) (string, error) {
	for {
		selected, err := promptsx.SelectEdit(promptsx.SelectEditParams[string]{
			Message: fmt.Sprintf("Pick a commit message to use: %s", picocolors.Gray("(Ctrl+c to exit)")),
			Options: slice.FlatMap(messages, func(i int, s string) []promptsx.SelectEditOption[string] {
				return []promptsx.SelectEditOption[string]{{Label: s, Key: fmt.Sprintf("%d", i+1)}}
			}),
			EditHint: "e to edit",
		})
		if err != nil {
			if prompts.IsCancel(err) {
				prompts.Outro("Commit cancelled")
				return "", nil
			}
			return "", err
		}

		message := selected.Value

		if !selected.Edit {
			return message, nil
		}

		editedMessage, err := genEditCommitMessage(message, genFlags.Type)
		if err != nil {
			if prompts.IsCancel(err) {
				prompts.Outro("Commit cancelled")
				return "", nil
			}
			return "", err
		}

		messages = []string{editedMessage}
	}
}

func genEditCommitMessage(message string, commitType commit.Type) (string, error) {
	commitMessage := commit.ParseMessage(message)

	err := prompts.Workflow(&commitMessage).
		ConditionalStep("Type",
			func() bool {
				return commitMessage.Type != "" || commitType == commit.ConventionalType
			},
			func() (any, error) {
				var options []*prompts.SelectOption[string]

				// in case of unknown type
				if _, ok := commit.ConventionalCommitTypes[commitMessage.Type]; !ok {
					options = append(options, &prompts.SelectOption[string]{
						Label: commitMessage.Type,
						Value: commitMessage.Type,
					})
				}

				// add rest of the conventional commit types
				options = append(options, slice.FlatMap(
					maputil.Keys(commit.ConventionalCommitTypes),
					func(_ int, item string) []*prompts.SelectOption[string] {
						return []*prompts.SelectOption[string]{
							{Label: item, Value: item},
						}
					})...)

				// sort options
				sort.Slice(options, func(i, j int) bool {
					return options[i].Label < options[j].Label
				})

				return prompts.Select(prompts.SelectParams[string]{
					Message:      "Select a type",
					InitialValue: commitMessage.Type,
					Options:      options,
				})
			}).
		ConditionalStep("Scope",
			func() bool {
				return commitMessage.Type != ""
			},
			func() (any, error) {
				initialValue := commitMessage.Scope
				if commitMessage.Breaking {
					initialValue += "!"
				}
				return prompts.Text(prompts.TextParams{
					Message:      "Enter a scope",
					Placeholder:  "<optional scope>",
					InitialValue: initialValue,
					Validate: func(value string) error {
						return nil
					},
				})
			}).
		Step("CommitMessage", func() (any, error) {
			return prompts.Text(prompts.TextParams{
				Message:      "Enter a message",
				Placeholder:  "<message>",
				InitialValue: commitMessage.CommitMessage,
				Validate: func(value string) error {
					if value == "" {
						return errors.New("please enter a message")
					}
					return nil
				},
			})
		}).
		Run()
	if err != nil {
		return "", err
	}

	return commitMessage.ToString(), nil
}

func runGenE(cmd *cobra.Command, args []string) error {
	workDir, err := genSetup(cmd)
	if err != nil {
		return err
	}

	_, diff, err := genDetectAndStageFiles(workDir, genFlags.All)
	if err != nil {
		return err
	}

	aip, err := genInitializeLLMProvider(cmd, genFlags.Provider)
	if err != nil {
		return err
	}

	messages, err := genMessages(cmd.Context(), aip, genFlags.Type, workDir, diff)
	if err != nil {
		return err
	}

	var message string
	if genFlags.Yes {
		// In automatic mode, use the first message
		if len(messages) > 0 {
			message = messages[0]
		}
	} else {
		// In interactive mode, let the user select a message
		message, err = genHandleMessageSelection(messages)
		if err != nil {
			return err
		}
	}

	if message == "" {
		return errors.New("no commit message selected") //nolint:staticcheck
	}

	if err := gitCommit(workDir, message); err != nil {
		return err
	}

	if !genFlags.Yes {
		prompts.Outro(fmt.Sprintf("%s Successfully committed", picocolors.Green("✔")))
	} else {
		fmt.Printf("Successfully committed: %s\n", message)
	}

	return nil
}

func isGenCmd() bool {
	if workDir, err := gitWorkingTreeDir(getWd()); err != nil || workDir == "" {
		return false
	}
	return true
}
