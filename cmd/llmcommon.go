package cmd

import (
	"errors"

	"github.com/zbiljic/kai/pkg/llm"
	"github.com/zbiljic/kai/pkg/llm/provider"
)

// initializeLLMProvider initializes an LLM provider based on provider type and model.
func initializeLLMProvider(cmdChanged bool, providerType ProviderType, model string) (llm.AIPrompt, error) {
	if cmdChanged {
		switch providerType {
		case OpenAIProvider:
			return provider.NewOpenAIProvider(provider.OpenAIOptions{
				Model: model,
			}), nil
		case GoogleAIProvider:
			return provider.NewGoogleAIProvider(provider.GoogleAIOptions{
				Model: model,
			})
		case OpenRouterProvider:
			return provider.NewOpenRouterProvider(provider.OpenRouterOptions{
				Model: model,
			}), nil
		case PhindProvider:
			return provider.NewPhindProvider(provider.PhindOptions{
				Model: model,
			}), nil
		}
	}

	// Try providers in preferred order
	providers := []struct {
		create func() (llm.AIPrompt, error)
	}{
		{create: func() (llm.AIPrompt, error) {
			return provider.NewGoogleAIProvider(provider.GoogleAIOptions{
				Model: model,
			})
		}},
		{create: func() (llm.AIPrompt, error) {
			return provider.NewOpenRouterProvider(provider.OpenRouterOptions{
				Model: model,
			}), nil
		}},
		{create: func() (llm.AIPrompt, error) {
			return provider.NewOpenAIProvider(provider.OpenAIOptions{
				Model: model,
			}), nil
		}},
		{create: func() (llm.AIPrompt, error) {
			return provider.NewPhindProvider(provider.PhindOptions{
				Model: model,
			}), nil
		}},
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
