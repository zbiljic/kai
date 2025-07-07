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
	groqBaseURL = "https://api.groq.com/openai/v1/chat/completions"
	groqModel   = "meta-llama/llama-4-scout-17b-16e-instruct"
)

// Compile-time proof of interface implementation.
var _ llm.AIPrompt = (*Groq)(nil)

type GroqOptions struct {
	ApiKey  string
	BaseURL string
	Model   string
}

type Groq struct {
	options GroqOptions
}

func NewGroqProvider(opts ...GroqOptions) llm.AIPrompt {
	o := GroqOptions{}

	if len(opts) > 0 {
		o = opts[0]
	}

	if o.ApiKey == "" {
		o.ApiKey = os.Getenv("GROQ_API_KEY")
	}

	if o.BaseURL == "" {
		o.BaseURL = groqBaseURL
	}
	if o.Model == "" {
		o.Model = groqModel
	}

	return &Groq{
		options: o,
	}
}

func (g *Groq) String() string {
	return fmt.Sprintf("Groq (%s)", g.options.Model)
}

func (g *Groq) IsAvailable() bool {
	return os.Getenv("GROQ_API_KEY") != ""
}

func (g *Groq) Generate(ctx context.Context, systemPrompt, userPrompt string, candidateCount int) ([]string, error) {
	if g.options.ApiKey == "" {
		return nil, errors.New("Groq API Key is not set")
	}

	if candidateCount < 1 {
		candidateCount = 1
	}

	var messages []string

	// Make multiple requests since Groq only supports N=1
	for i := 0; i < candidateCount; i++ {
		payload := openai.ChatCompletionRequest{
			Model: g.options.Model,
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
			N:                1, // Groq API only supports N=1
		}

		var (
			respContent openai.ChatCompletionResponse
			respError   openai.ErrorResponse
		)

		err := requests.
			URL(g.options.BaseURL).
			Post().
			Headers(map[string][]string{
				"Authorization": {fmt.Sprintf("Bearer %s", g.options.ApiKey)},
				"Content-Type":  {"application/json"},
			}).
			BodyJSON(payload).
			ToJSON(&respContent).
			ErrorJSON(&respError).
			Fetch(ctx)
		if err != nil {
			return nil, fmt.Errorf("request to Groq failed: %w", err)
		}

		if respError.Error != nil && respError.Error.Message != "" {
			return nil, fmt.Errorf("Groq API error: %s", respError.Error.Message)
		}

		if len(respContent.Choices) == 0 {
			return nil, errors.New("no completion choice available from Groq")
		}

		// Extract the content from the first (and only) choice
		if len(respContent.Choices) > 0 {
			content := respContent.Choices[0].Message.Content
			if content != "" {
				messages = append(messages, content)
			}
		}
	}

	messages = slice.Unique(messages)

	if len(messages) == 0 {
		return nil, errors.New("no valid completion content received from Groq")
	}

	return messages, nil
}
