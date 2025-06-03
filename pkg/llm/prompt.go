package llm

import (
	"context"
)

// AIPrompt is an interface for generating prompts from input data.
type AIPrompt interface {
	// IsAvailable checks if the provider has all required configuration (e.g. API keys)
	// to be used. Returns true if the provider can be used, false otherwise.
	IsAvailable() bool

	// Generate creates a new output string using the context,
	// system prompt, and user prompt. Returns the generated
	// string and an error if it fails.
	Generate(ctx context.Context, systemPrompt, userPrompt string) ([]string, error)
}
