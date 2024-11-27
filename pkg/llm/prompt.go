package llm

import (
	"context"
)

// AIPrompt is an interface for generating prompts from input data.
type AIPrompt interface {
	// Generate creates a new output string using the context,
	// system prompt, and user prompt. Returns the generated
	// string and an error if it fails.
	Generate(ctx context.Context, systemPrompt, userPrompt string) ([]string, error)
}
