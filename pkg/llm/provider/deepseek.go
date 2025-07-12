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
	deepseekBaseURL = "https://api.deepseek.com/v1/chat/completions"
	deepseekModel   = "deepseek-chat"
)

// Compile-time proof of interface implementation.
var _ llm.AIPrompt = (*DeepSeek)(nil)

type DeepSeekOptions struct {
	ApiKey  string
	BaseURL string
	Model   string
}

type DeepSeek struct {
	options DeepSeekOptions
}

func NewDeepSeekProvider(opts ...DeepSeekOptions) llm.AIPrompt {
	o := DeepSeekOptions{}

	if len(opts) > 0 {
		o = opts[0]
	}

	if o.ApiKey == "" {
		o.ApiKey = os.Getenv("DEEPSEEK_API_KEY")
	}

	if o.BaseURL == "" {
		o.BaseURL = deepseekBaseURL
	}
	if o.Model == "" {
		o.Model = deepseekModel
	}

	return &DeepSeek{
		options: o,
	}
}

func (d *DeepSeek) String() string {
	return fmt.Sprintf("DeepSeek (%s)", d.options.Model)
}

func (d *DeepSeek) IsAvailable() bool {
	return os.Getenv("DEEPSEEK_API_KEY") != ""
}

func (d *DeepSeek) Generate(ctx context.Context, systemPrompt, userPrompt string, candidateCount int) ([]string, error) {
	if d.options.ApiKey == "" {
		return nil, errors.New("DeepSeek API Key is not set")
	}

	payload := openai.ChatCompletionRequest{
		Model: d.options.Model,
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
		URL(d.options.BaseURL).
		Post().
		Headers(map[string][]string{
			"Authorization": {fmt.Sprintf("Bearer %s", d.options.ApiKey)},
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
