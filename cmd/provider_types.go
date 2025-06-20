package cmd

import (
	"github.com/thediveo/enumflag/v2"
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
