package provider

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/carlmjohnson/requests"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/sashabaranov/go-openai"

	"github.com/zbiljic/kai/pkg/llm"
)

const (
	openRouterBaseURL = "https://openrouter.ai/api/v1/chat/completions"
	openRouterModel   = "mistralai/devstral-small:free"
)

// Compile-time proof of interface implementation.
var _ llm.AIPrompt = (*OpenRouter)(nil)

type OpenRouterOptions struct {
	ApiKey  string
	BaseURL string
	Model   string
}

type OpenRouter struct {
	options OpenRouterOptions
}

func NewOpenRouterProvider(opts ...OpenRouterOptions) llm.AIPrompt {
	o := OpenRouterOptions{}

	if len(opts) > 0 {
		o = opts[0]
	}

	if o.ApiKey == "" {
		o.ApiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	if o.BaseURL == "" {
		o.BaseURL = openRouterBaseURL
	}
	if o.Model == "" {
		o.Model = openRouterModel
	}

	return &OpenRouter{
		options: o,
	}
}

func (o *OpenRouter) String() string {
	return fmt.Sprintf("OpenRouter (%s)", o.options.Model)
}

func (o *OpenRouter) IsAvailable() bool {
	return os.Getenv("OPENROUTER_API_KEY") != ""
}

func (p *OpenRouter) Generate(ctx context.Context, systemPrompt, userPrompt string, candidateCount int) ([]string, error) {
	if p.options.ApiKey == "" {
		return nil, errors.New("OpenRouter API Key is not set")
	}

	payload := openai.ChatCompletionRequest{
		Model: p.options.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature:      0.7,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
		MaxTokens:        1024,
		Stream:           false,
		N:                candidateCount,
	}

	var (
		respContent openai.ChatCompletionResponse
		respError   openai.ErrorResponse
	)

	// OpenRouter API requires 'HTTP-Referer' and 'X-Title' headers.
	httpReferer := os.Getenv("OPENROUTER_HTTP_REFERER")
	if httpReferer == "" {
		httpReferer = "https://github.com/zbiljic/kai"
	}
	xTitle := os.Getenv("OPENROUTER_X_TITLE")
	if xTitle == "" {
		xTitle = "kai"
	}

	err := requests.
		URL(p.options.BaseURL).
		Post().
		Headers(map[string][]string{
			"Authorization": {fmt.Sprintf("Bearer %s", p.options.ApiKey)},
			"HTTP-Referer":  {httpReferer}, // Required by OpenRouter
			"X-Title":       {xTitle},      // Recommended by OpenRouter
		}).
		BodyJSON(payload).
		ToJSON(&respContent).
		ErrorJSON(&respError).
		Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to OpenRouter failed: %w", err)
	}

	if respError.Error != nil && respError.Error.Message != "" {
		return nil, fmt.Errorf("OpenRouter API error: %s", respError.Error.Message)
	}

	if len(respContent.Choices) == 0 {
		return nil, errors.New("no completion choice available from OpenRouter")
	}

	messages := slice.Map(respContent.Choices, func(_ int, s openai.ChatCompletionChoice) string {
		return s.Message.Content
	})

	messages = slice.Unique(messages)

	return messages, nil
}
