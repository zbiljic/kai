package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/tidwall/gjson"

	"github.com/zbiljic/kai/pkg/llm"
)

const (
	phindBaseURL = "https://https.extension.phind.com/agent/"
	phindModel   = "Phind-70B"
)

// Compile-time proof of interface implementation.
var _ llm.AIPrompt = (*Phind)(nil)

type PhindOptions struct {
	BaseURL string
	Model   string
}

type Phind struct {
	options PhindOptions
}

func NewPhindProvider(opts ...PhindOptions) llm.AIPrompt {
	o := PhindOptions{}

	if len(opts) > 0 {
		o = opts[0]
	}

	if o.BaseURL == "" {
		o.BaseURL = phindBaseURL
	}
	if o.Model == "" {
		o.Model = phindModel
	}

	return &Phind{
		options: o,
	}
}

func (p *Phind) Generate(ctx context.Context, systemPrompt, userPrompt string) ([]string, error) {
	prompt := ""
	prompt += systemPrompt
	prompt += "\n"
	prompt += userPrompt

	payload := map[string]any{
		"additional_extension_context": "",
		"allow_magic_buttons":          true,
		"is_vscode_extension":          true,
		"message_history": []any{
			map[string]any{
				"role":    "user",
				"content": prompt,
			},
		},
		"requested_model": p.options.Model,
		"user_input":      prompt,
	}

	var responseText string

	err := requests.
		URL(p.options.BaseURL).
		Post().
		Headers(map[string][]string{
			"User-Agent":      {""},
			"Accept":          {"*/*"},
			"Accept-Encoding": {"Identity"},
		}).
		BodyJSON(payload).
		ToString(&responseText).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}

	if responseText == "" {
		return nil, errors.New("no completion choice available")
	}

	fullText, err := p.parseStreamResponse(responseText)
	if err != nil {
		return nil, err
	}

	return []string{fullText}, nil
}

func (p *Phind) parseLine(line string) (string, error) {
	data := strings.TrimPrefix(line, "data: ")

	if val := gjson.Get(data, "choices.0.delta.content"); val.Exists() && val.Type == gjson.String {
		return val.String(), nil
	}

	return "", nil
}

func (p *Phind) parseStreamResponse(responseText string) (string, error) {
	text := ""
	for _, line := range strings.Split(responseText, "\n") {
		if parsedLine, err := p.parseLine(line); err != nil {
			return "", err
		} else {
			text += parsedLine
		}
	}
	return text, nil
}
