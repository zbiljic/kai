package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/carlmjohnson/requests"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/sashabaranov/go-openai"

	"github.com/zbiljic/kai/pkg/llm"
)

const (
	openaiBaseURL = "https://api.openai.com/v1/chat/completions"
	openaiModel   = openai.GPT4oMini
)

// Compile-time proof of interface implementation.
var _ llm.AIPrompt = (*OpenAI)(nil)

type OpenAIOptions struct {
	ApiKey  string
	BaseURL string
	Model   string
}

type OpenAI struct {
	options OpenAIOptions
}

func NewOpenAIProvider(opts ...OpenAIOptions) llm.AIPrompt {
	o := OpenAIOptions{}

	if len(opts) > 0 {
		o = opts[0]
	}

	if o.BaseURL == "" {
		o.BaseURL = openaiBaseURL
	}
	if o.Model == "" {
		o.Model = openaiModel
	}

	return &OpenAI{
		options: o,
	}
}

func (p *OpenAI) Generate(ctx context.Context, systemPrompt, userPrompt string) ([]string, error) {
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
		MaxTokens:        256,
		Stream:           false,
		N:                1,
	}

	var (
		respContent openai.ChatCompletionResponse
		respError   openai.ErrorResponse
	)

	err := requests.
		URL(p.options.BaseURL).
		Post().
		Headers(map[string][]string{
			"Authorization": {fmt.Sprintf("Bearer %s", p.options.ApiKey)},
		}).
		BodyJSON(payload).
		ToJSON(&respContent).
		ErrorJSON(&respError).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}

	if respError.Error != nil && respError.Error.Message != "" {
		return nil, errors.New(respError.Error.Message)
	}

	if len(respContent.Choices) == 0 {
		return nil, errors.New("no completion choice available")
	}

	// extract messages
	messages := slice.Map(respContent.Choices, func(_ int, s openai.ChatCompletionChoice) string {
		return s.Message.Content
	})

	// remove duplicates
	messages = slice.Unique(messages)

	return messages, nil
}
