package llm

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/zbiljic/kai/pkg/commit"
	"github.com/zbiljic/kai/pkg/gitdiff"
)

//go:embed templates/prp/*
var prpTemplatesFS embed.FS

// CommitPlan represents the AI-generated commit reorganization plan
type CommitPlan struct {
	Commits []PlannedCommit `json:"commits"`
}

// PlannedCommit represents a single commit in the reorganization plan
type PlannedCommit struct {
	Message   string   `json:"message"`
	HunkIDs   []string `json:"hunk_ids"`
	Rationale string   `json:"rationale"`
}

type PrpTemplates struct {
	stringTemplates map[string]string             // String templates (from .md files)
	goTemplates     map[string]*template.Template // Go templates (from .tmpl files)
}

func loadPrpTemplates() (*PrpTemplates, error) {
	templates := &PrpTemplates{
		stringTemplates: make(map[string]string),
		goTemplates:     make(map[string]*template.Template),
	}

	entries, err := prpTemplatesFS.ReadDir("templates/prp")
	if err != nil {
		return nil, fmt.Errorf("failed to read prp templates directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		filename := entry.Name()
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)

		content, err := fs.ReadFile(prpTemplatesFS, filepath.Join("templates/prp", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to load prp template %s: %w", filename, err)
		}

		// Categorize by extension
		switch ext {
		case ".md":
			// Markdown files go to stringTemplates
			templates.stringTemplates[name] = string(content)
		case ".tmpl":
			// Template files get parsed and go to goTemplates
			tmpl, err := template.New(name).Parse(string(content))
			if err != nil {
				return nil, fmt.Errorf("failed to parse prp template %s: %w", filename, err)
			}
			templates.goTemplates[name] = tmpl
		}
	}

	return templates, nil
}

// prpGenSystemPrompt generates system prompt for commit reorganization
func prpGenSystemPrompt() (string, error) {
	tmpl, err := loadPrpTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to load prp templates: %w", err)
	}

	systemPrompt, ok := tmpl.stringTemplates["system_prompt"]
	if !ok {
		return "", fmt.Errorf("system_prompt template not found")
	}

	return systemPrompt, nil
}

// prpGenUserPrompt generates user prompt for commit reorganization
func prpGenUserPrompt(hunks []*gitdiff.Hunk, currentBranch, baseBranch string) (string, error) {
	tmpl, err := loadPrpTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to load prp templates: %w", err)
	}

	var userPromptBuf bytes.Buffer
	userPromptTmpl, ok := tmpl.goTemplates["user_prompt"]
	if !ok {
		return "", fmt.Errorf("user_prompt template not found")
	}

	err = userPromptTmpl.Execute(&userPromptBuf, map[string]any{
		"CurrentBranch": currentBranch,
		"BaseBranch":    baseBranch,
		"Hunks":         hunks,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute prp user prompt template: %w", err)
	}

	return userPromptBuf.String(), nil
}

// extractJSONFromResponse extracts JSON content from AI responses that may be wrapped in markdown
func extractJSONFromResponse(response string) string {
	// Remove leading/trailing whitespace
	response = strings.TrimSpace(response)

	// Check if response is wrapped in markdown code block
	if strings.HasPrefix(response, "```") {
		// Find the first ``` and the closing ```
		lines := strings.Split(response, "\n")
		var jsonLines []string
		inCodeBlock := false

		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				if inCodeBlock {
					// End of code block, stop collecting
					break
				} else {
					// Start of code block, start collecting from next line
					inCodeBlock = true
					continue
				}
			}

			if inCodeBlock {
				jsonLines = append(jsonLines, line)
			}
		}

		if len(jsonLines) > 0 {
			return strings.Join(jsonLines, "\n")
		}
	}

	// If no markdown wrapper found, return original response
	return response
}

// GenerateCommitPlan uses AI to analyze hunks and generate a structured commit
// plan.
func GenerateCommitPlan(
	ctx context.Context,
	aip AIPrompt,
	hunks []*gitdiff.Hunk,
	currentBranch,
	baseBranch string,
) (*CommitPlan, error) {
	// Build system prompt
	systemPrompt, err := prpGenSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate system prompt: %w", err)
	}

	// Build user prompt with hunk information
	userPrompt, err := prpGenUserPrompt(hunks, currentBranch, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate user prompt: %w", err)
	}

	responses, err := aip.Generate(ctx, systemPrompt, userPrompt, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate commit plan: %w", err)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("no commit plan was generated")
	}

	response := responses[0]

	// Extract JSON from response (handles markdown-wrapped JSON)
	jsonContent := extractJSONFromResponse(response)

	// Parse JSON response
	var commitPlan CommitPlan
	if err := json.Unmarshal([]byte(jsonContent), &commitPlan); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w\nOriginal Response: %s\nExtracted JSON: %s", err, response, jsonContent)
	}

	// Ensure commit messages follow lowercase convention after colon
	commitPlan.Commits = slice.Map(commitPlan.Commits, func(_ int, plannedCommit PlannedCommit) PlannedCommit {
		m := commit.ParseMessage(plannedCommit.Message)
		m.CommitMessage = strutil.LowerFirst(m.CommitMessage)
		plannedCommit.Message = m.ToString()
		return plannedCommit
	})

	return &commitPlan, nil
}
