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
	// ClaudeProvider represents the Claude provider.
	ClaudeProvider
	// GoogleAIProvider represents the GoogleAI provider.
	GoogleAIProvider
	// OpenRouterProvider represents the OpenRouter provider.
	OpenRouterProvider
	// GroqProvider represents the Groq provider.
	GroqProvider
	// DeepSeekProvider represents the DeepSeek provider.
	DeepSeekProvider
)

// ProviderIds maps ProviderType to their string representations.
var ProviderIds = map[ProviderType][]string{
	PhindProvider:      {"phind"},
	OpenAIProvider:     {"openai"},
	ClaudeProvider:     {"claude"},
	GoogleAIProvider:   {"googleai"},
	OpenRouterProvider: {"openrouter"},
	GroqProvider:       {"groq"},
	DeepSeekProvider:   {"deepseek"},
}
