package llm

import (
	"context"
)

// AIPrompt is an interface for generating prompts from input data.
type AIPrompt interface {
	// String returns the name of the provider.
	String() string

	// IsAvailable checks if the provider has all required configuration (e.g. API keys)
	// to be used. Returns true if the provider can be used, false otherwise.
	IsAvailable() bool

	// Generate creates a new output string using the context,
	// system prompt, and user prompt. Returns the generated
	// string and an error if it fails.
	// The candidateCount parameter determines how many message candidates to generate.
	Generate(ctx context.Context, systemPrompt, userPrompt string, candidateCount int) ([]string, error)
}
