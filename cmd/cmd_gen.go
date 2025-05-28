package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Mist3rBru/go-clack/prompts"
	"github.com/Mist3rBru/go-clack/third_party/picocolors"
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
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
	Long:        `Generates commit message based on staged changes`,
	Annotations: map[string]string{"group": "main"},
	Args:        cobra.ArbitraryArgs,
	RunE:        runGenE,
}

var genFlags = genOptions{
	Type:     commit.ConventionalType,
	Provider: PhindProvider, // Default provider
}

func genAddFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(enumflag.New(&genFlags.Type, "type", commit.TypeIds, enumflag.EnumCaseInsensitive), "type", "t", "Type of commit message to generate")
	cmd.Flags().VarP(enumflag.New(&genFlags.Provider, "provider", ProviderIds, enumflag.EnumCaseInsensitive), "provider", "p", "LLM provider to use for generating commit messages (phind, openai, googleai, openrouter)")
}

func init() {
	genAddFlags(genCmd)

	rootCmd.AddCommand(genCmd)
}

type genOptions struct {
	Type     commit.Type
	Provider ProviderType
}

func runGenE(cmd *cobra.Command, args []string) error {
	prompts.Intro(picocolors.BgCyan(picocolors.Black(fmt.Sprintf(" %s ", AppName))))
	// in order to show custom error
	injectIntoCommandContextWithKey(cmd, ctxKeyClackPromptStarted{}, true)

	workDir, err := gitWorkingTreeDir(getWd())
	if err != nil {
		return errors.New("The current directory must be a Git repository") //nolint:staticcheck
	}

	detectingFilesSpinner := prompts.Spinner(prompts.SpinnerOptions{})

	detectingFilesSpinner.Start("Detecting staged files")

	files, diff, err := gitDiffStaged(workDir)
	if err != nil {
		detectingFilesSpinner.Stop("Error detecting staged files", 1)
		return err
	}

	if len(files) == 0 {
		detectingFilesSpinner.Stop("Detecting staged files", 0)
		return errors.New("No staged files detected") //nolint:staticcheck
	}

	detectedMessage := fmt.Sprintf(
		"Detected %d staged file(s):\n     %s",
		len(files),
		strings.Join(files, "\n     "),
	)

	detectingFilesSpinner.Stop(detectedMessage, 0)

	generateMessageSpinner := prompts.Spinner(prompts.SpinnerOptions{})

	generateMessageSpinner.Start("Generating commit message")

	var aip llm.AIPrompt
	switch genFlags.Provider {
	case OpenAIProvider:
		aip = provider.NewOpenAIProvider()
	case GoogleAIProvider:
		aip, err = provider.NewGoogleAIProvider()
		if err != nil {
			return err
		}
	case OpenRouterProvider:
		aip = provider.NewOpenRouterProvider()
	case PhindProvider:
		fallthrough
	default:
		aip = provider.NewPhindProvider()
	}

	messages, err := llm.GenerateCommitMessage(cmd.Context(), aip, genFlags.Type, diff)
	if err != nil {
		return err
	}

	generateMessageSpinner.Stop("Changes analyzed", 0)

	// remove empty messages
	messages = slice.Filter(messages, func(_ int, s string) bool {
		return strutil.IsNotBlank(s)
	})

	if len(messages) == 0 {
		return errors.New("No commit messages were generated. Try again.") //nolint:staticcheck
	}

	// lowercase the first letter of commit message
	messages = slice.Map(messages, func(_ int, s string) string {
		m := commit.ParseMessage(s)
		m.CommitMessage = strutil.LowerFirst(m.CommitMessage)
		return m.ToString()
	})

	var message string

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
				return nil
			}
			return err
		}

		message = selected.Value

		// if we need to edit the message
		if selected.Edit {
			commitMessage := commit.ParseMessage(message)

			err := prompts.Workflow(&commitMessage).
				ConditionalStep("Type",
					func() bool {
						return commitMessage.Type != "" || genFlags.Type == commit.ConventionalType
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
				if prompts.IsCancel(err) {
					prompts.Outro("Commit cancelled")
					return nil
				}
				return err
			}

			message = commitMessage.ToString()

			messages = []string{message}
		} else {
			break
		}
	}

	if err := gitCommit(workDir, message); err != nil {
		return err
	}

	prompts.Outro(fmt.Sprintf("%s Successfully committed", picocolors.Green("✔")))

	return nil
}

func isGenCmd() bool {
	if workDir, err := gitWorkingTreeDir(getWd()); err != nil || workDir == "" {
		return false
	}
	return true
}
