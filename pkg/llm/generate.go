package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/zbiljic/kai/pkg/commit"
)

// Default values for commit message generation
const (
	DefaultMaxCommitsPerFile = 3
	DefaultMaxTotalCommits   = 10
)

// commitType returns a descriptive string for the given commit Type.
// It utilizes a predefined map to fetch the corresponding description.
// This helps in providing guidance or context about the chosen commit type.
func commitType(t commit.Type) string {
	return commitTypes[t]
}

// formatPreviousCommits formats a list of previous commit messages for display in the prompt
func formatPreviousCommits(messages []string) string {
	if len(messages) == 0 {
		return ""
	}

	// Format each message with a bullet point
	var formattedMessages []string
	for _, msg := range messages {
		formattedMessages = append(formattedMessages, "- "+msg)
	}

	return strings.Join(formattedMessages, "\n")
}

func GenerateSystemPrompt(t commit.Type) string {
	var content []string
	content = append(content, fmt.Sprintf(PromptSystemFormat, t.CommitFormat()))
	return strings.Join(content, "\n")
}

func GenerateUserPrompt(t commit.Type, maxLength int, diff string) string {
	return GenerateUserPromptWithPreviousCommits(t, maxLength, diff, nil)
}

func GenerateUserPromptWithPreviousCommits(t commit.Type, maxLength int, diff string, previousCommits []string) string {
	var content []string
	content = append(content, PromptIntro)
	content = append(content, "")
	if ct := commitType(t); ct != "" {
		content = append(content, commitType(t))
		content = append(content, "")
	}
	content = append(content, PromptDetails)
	content = append(content, fmt.Sprintf(PromptMaxLengthFormat, maxLength))
	content = append(content, "")

	// Add previous commit messages if available
	if len(previousCommits) > 0 {
		content = append(content, fmt.Sprintf(PromptPreviousCommitsFormat, formatPreviousCommits(previousCommits)))
		content = append(content, "")
	}

	content = append(content, fmt.Sprintf(PromptCodeDiffFormat, diff))
	return strings.Join(content, "\n")
}

func GenerateCommitMessage(ctx context.Context, provider AIPrompt, commitType commit.Type, diff string) ([]string, error) {
	systemPrompt := GenerateSystemPrompt(commitType)
	userPrompt := GenerateUserPrompt(commitType, commit.DefaultMaxLength, diff)
	return provider.Generate(ctx, systemPrompt, userPrompt)
}

func GenerateCommitMessageWithPreviousCommits(
	ctx context.Context,
	provider AIPrompt,
	commitType commit.Type,
	workDir,
	diff string,
	previousCommits []string,
) ([]string, error) {
	systemPrompt := GenerateSystemPrompt(commitType)
	userPrompt := GenerateUserPromptWithPreviousCommits(commitType, commit.DefaultMaxLength, diff, previousCommits)
	return provider.Generate(ctx, systemPrompt, userPrompt)
}
