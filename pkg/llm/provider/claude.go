package provider

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/duke-git/lancet/v2/slice"

	"github.com/zbiljic/kai/pkg/llm"
)

const (
	claudeModel = string(anthropic.ModelClaude3_5HaikuLatest)
)

// Compile-time proof of interface implementation.
var _ llm.AIPrompt = (*Claude)(nil)

type ClaudeOptions struct {
	ApiKey  string
	BaseURL string
	Model   string
}

type Claude struct {
	options ClaudeOptions
	client  *anthropic.Client
}

func NewClaudeProvider(opts ...ClaudeOptions) (llm.AIPrompt, error) {
	o := ClaudeOptions{}

	if len(opts) > 0 {
		o = opts[0]
	}

	if o.ApiKey == "" {
		o.ApiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	if o.Model == "" {
		o.Model = claudeModel
	}

	if o.ApiKey == "" {
		return nil, errors.New("anthropic API Key is not set")
	}

	clientOpts := []option.RequestOption{
		option.WithAPIKey(o.ApiKey),
	}

	if o.BaseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(o.BaseURL))
	}

	client := anthropic.NewClient(clientOpts...)

	return &Claude{
		options: o,
		client:  &client,
	}, nil
}

func (c *Claude) String() string {
	return fmt.Sprintf("Claude (%s)", c.options.Model)
}

func (c *Claude) IsAvailable() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != ""
}

func (c *Claude) Generate(ctx context.Context, systemPrompt, userPrompt string, candidateCount int) ([]string, error) {
	if c.client == nil {
		return nil, errors.New("client is not initialized")
	}

	if candidateCount < 1 {
		candidateCount = 1
	}

	var messages []string

	// Make multiple requests since Claude only supports N=1
	for i := 0; i < candidateCount; i++ {
		resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(c.options.Model),
			System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
			Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt))},
			MaxTokens: 1024,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate content: %w", err)
		}

		if len(resp.Content) == 0 {
			return nil, errors.New("no completion choice available")
		}

		if len(resp.Content) > 0 {
			if textBlock, ok := resp.Content[0].AsAny().(anthropic.TextBlock); ok {
				messages = append(messages, textBlock.Text)
			} else {
				return nil, fmt.Errorf("unexpected content type in response: %T", resp.Content[0].AsAny())
			}
		}
	}

	if len(messages) == 0 {
		return nil, errors.New("returned no candidates")
	}

	messages = slice.Unique(messages)

	if len(messages) == 0 {
		return nil, errors.New("returned no text content from candidates")
	}

	return messages, nil
}
