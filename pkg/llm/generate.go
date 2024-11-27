package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/zbiljic/kai/pkg/commit"
)

// commitType returns a descriptive string for the given commit Type.
// It utilizes a predefined map to fetch the corresponding description.
// This helps in providing guidance or context about the chosen commit type.
func commitType(t commit.Type) string {
	return commitTypes[t]
}

func GenerateSystemPrompt(t commit.Type) string {
	var content []string
	content = append(content, fmt.Sprintf(PromptSystemFormat, t.CommitFormat()))
	return strings.Join(content, "\n")
}

func GenerateUserPrompt(t commit.Type, maxLength int, diff string) string {
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
	content = append(content, fmt.Sprintf(PromptCodeDiffFormat, diff))
	return strings.Join(content, "\n")
}

func GenerateCommitMessage(ctx context.Context, provider AIPrompt, commitType commit.Type, diff string) ([]string, error) {
	systemPrompt := GenerateSystemPrompt(commitType)
	userPrompt := GenerateUserPrompt(commitType, commit.DefaultMaxLength, diff)
	return provider.Generate(ctx, systemPrompt, userPrompt)
}
