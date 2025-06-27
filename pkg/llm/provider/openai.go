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

	if o.ApiKey == "" {
		o.ApiKey = os.Getenv("OPENAI_API_KEY")
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

func (o *OpenAI) String() string {
	return fmt.Sprintf("OpenAI (%s)", o.options.Model)
}

func (o *OpenAI) IsAvailable() bool {
	return os.Getenv("OPENAI_API_KEY") != ""
}

func (p *OpenAI) Generate(ctx context.Context, systemPrompt, userPrompt string, candidateCount int) ([]string, error) {
	if p.options.ApiKey == "" {
		return nil, errors.New("OpenAI API Key is not set")
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
		MaxTokens:        256,
		Stream:           false,
		N:                candidateCount,
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

	messages := slice.Map(respContent.Choices, func(_ int, s openai.ChatCompletionChoice) string {
		return s.Message.Content
	})

	messages = slice.Unique(messages)

	return messages, nil
}
