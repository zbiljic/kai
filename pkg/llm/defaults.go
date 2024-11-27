package llm

import (
	"fmt"

	"github.com/zbiljic/kai/pkg/commit"
)

const (
	PromptIntro   = "Generate a concise git commit message written in present tense for the following code diff with the given specifications below:"
	PromptDetails = `Exclude anything unnecessary such as translation.
Your entire response will be passed directly into git commit.
`
)

var (
	PromptSystemFormat = `You are a commit message generator that follows these rules:
1. Write in present tense
2. Be concise and direct
3. Output only the commit message without any explanations
4. Follow the format: %s
`
	PromptMaxLengthFormat = "Commit message must be a maximum of %d characters."
	PromptCodeDiffFormat  = "Code diff:\n```diff\n%s\n```\n"
)

var commitTypes = map[commit.Type]string{
	commit.SimpleType: "",
	commit.ConventionalType: fmt.Sprintf("%s\n%s",
		"Choose a type from the type-to-description map below that best describes the git diff:",
		mapToString(commit.ConventionalCommitTypes),
	),
}
