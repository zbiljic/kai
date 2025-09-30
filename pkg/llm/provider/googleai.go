package provider

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/duke-git/lancet/v2/slice"
	"google.golang.org/genai"

	"github.com/zbiljic/kai/pkg/llm"
)

const (
	googleAIModel = "gemini-2.5-flash-preview-09-2025"
)

// Compile-time proof of interface implementation.
var _ llm.AIPrompt = (*GoogleAI)(nil)

// GoogleAIOptions holds configuration for the GoogleAI provider.
type GoogleAIOptions struct {
	ApiKey string
	Model  string
}

// GoogleAI is the provider implementation for Google AI Studio using genai library.
type GoogleAI struct {
	options GoogleAIOptions
	client  *genai.Client
}

// NewGoogleAIProvider creates a new GoogleAI provider instance.
func NewGoogleAIProvider(opts ...GoogleAIOptions) (llm.AIPrompt, error) {
	o := GoogleAIOptions{}

	if len(opts) > 0 {
		o = opts[0]
	}

	if o.Model == "" {
		o.Model = googleAIModel
	}

	if o.ApiKey == "" {
		o.ApiKey = os.Getenv("GEMINI_API_KEY")
	}

	if o.ApiKey == "" {
		return nil, fmt.Errorf("google AI API Key is not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  o.ApiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Google AI client: %w", err)
	}

	return &GoogleAI{
		options: o,
		client:  client,
	}, nil
}

func (o *GoogleAI) String() string {
	return fmt.Sprintf("GoogleAI (%s)", o.options.Model)
}

func (o *GoogleAI) IsAvailable() bool {
	return os.Getenv("GEMINI_API_KEY") != ""
}

// Generate sends a prompt to the Google AI API and returns the generated text.
func (p *GoogleAI) Generate(ctx context.Context, systemPrompt, userPrompt string, candidateCount int) ([]string, error) {
	if p.client == nil {
		return nil, errors.New("client is not initialized")
	}

	resp, err := p.client.Models.GenerateContent(
		ctx,
		p.options.Model,
		genai.Text(userPrompt),
		&genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: systemPrompt},
				},
			},
			CandidateCount: int32(candidateCount),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 {
		if resp != nil && resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != genai.BlockedReasonUnspecified {
			return nil, fmt.Errorf("prompt blocked due to: %s. Safety Ratings: %+v", resp.PromptFeedback.BlockReason, resp.PromptFeedback.SafetyRatings)
		}
		return nil, errors.New("returned no candidates")
	}

	var results []string
	for _, cand := range resp.Candidates {
		if cand.Content != nil && len(cand.Content.Parts) > 0 {
			var fullText string
			for _, part := range cand.Content.Parts {
				if txt := part.Text; txt != "" {
					fullText += txt
				}
			}
			if fullText != "" {
				results = append(results, fullText)
			}
		}
	}

	// remove duplicates
	results = slice.Unique(results)

	if len(results) == 0 {
		var finishReasons []string
		for _, cand := range resp.Candidates {
			if cand.FinishReason != genai.FinishReasonUnspecified {
				finishReasons = append(finishReasons, string(cand.FinishReason))
			}
		}
		if len(finishReasons) > 0 {
			return nil, fmt.Errorf("returned no text content; finish reasons: %v", finishReasons)
		}
		return nil, errors.New("returned no text content from candidates")
	}

	return results, nil
}
