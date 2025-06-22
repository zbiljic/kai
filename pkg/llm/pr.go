package llm

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

//go:embed templates/pr/*
var prTemplatesFS embed.FS

type Templates struct {
	stringTemplates map[string]string             // String templates (from .md files)
	goTemplates     map[string]*template.Template // Go templates (from .tmpl files)
}

// Default values for PR generation
const (
	DefaultMaxDiffSize = 10000
	PRTitlePrefix      = "PR Title"
	PRDescPrefix       = "PR Description"
)

func loadTemplates() (*Templates, error) {
	templates := &Templates{
		stringTemplates: make(map[string]string),
		goTemplates:     make(map[string]*template.Template),
	}

	entries, err := prTemplatesFS.ReadDir("templates/pr")
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		filename := entry.Name()
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)

		content, err := fs.ReadFile(prTemplatesFS, filepath.Join("templates/pr", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to load template %s: %w", filename, err)
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
				return nil, fmt.Errorf("failed to parse template %s: %w", filename, err)
			}
			templates.goTemplates[name] = tmpl
		}
	}

	return templates, nil
}

// prGenSystemPrompt generates system prompt for PR generation.
func prGenSystemPrompt(withContext, withTemplate bool) (string, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to load templates: %w", err)
	}

	var prompt strings.Builder
	prompt.WriteString(tmpl.stringTemplates["system_prompt_base"])
	prompt.WriteString("\n")

	if withContext {
		prompt.WriteString(tmpl.stringTemplates["system_prompt_with_context"])
	} else {
		prompt.WriteString(tmpl.stringTemplates["system_prompt_no_context"])
	}

	prompt.WriteString("\n")

	if withTemplate {
		prompt.WriteString(tmpl.stringTemplates["system_prompt_with_template"])
	} else {
		prompt.WriteString(tmpl.stringTemplates["system_prompt_no_template"])
	}

	return prompt.String(), nil
}

func prGenUserPrompt(diff, prTemplate string) (string, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to load templates: %w", err)
	}

	var userPromptBuf bytes.Buffer
	userPromptTmpl, ok := tmpl.goTemplates["user_prompt"]
	if !ok {
		return "", fmt.Errorf("user_prompt template not found")
	}

	err = userPromptTmpl.Execute(&userPromptBuf, map[string]any{
		"PRGuidelines": tmpl.stringTemplates["pr_guidelines"],
		"Diff":         diff,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute user prompt template: %w", err)
	}

	// Add output format
	userPromptBuf.WriteString("\n")
	userPromptBuf.WriteString(tmpl.stringTemplates["output_format"])

	// If template is provided, add layout template
	if prTemplate != "" {
		layoutTmpl, ok := tmpl.goTemplates["layout"]
		if !ok {
			return "", fmt.Errorf("layout template not found")
		}

		var layoutBuf bytes.Buffer
		err = layoutTmpl.Execute(&layoutBuf, map[string]any{
			"PRTemplate": prTemplate,
		})
		if err != nil {
			return "", fmt.Errorf("failed to execute layout template: %w", err)
		}

		userPromptBuf.WriteString("\n")
		userPromptBuf.Write(layoutBuf.Bytes())
	}

	return userPromptBuf.String(), nil
}

// GeneratePRContent generates PR title and description based on branch changes
func GeneratePRContent(
	ctx context.Context,
	aip AIPrompt,
	currentBranch,
	baseBranch,
	commits,
	diff,
	context,
	prTemplate string,
	maxDiffSize int,
) (string, string, error) {
	// Create system prompt
	systemPrompt, err := prGenSystemPrompt(context != "", prTemplate != "")
	if err != nil {
		return "", "", fmt.Errorf("failed to generate system prompt: %w", err)
	}

	// Create user prompt
	userPrompt, err := prGenUserPrompt(diff, prTemplate)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate user prompt: %w", err)
	}

	// Generate PR content
	responses, err := aip.Generate(ctx, systemPrompt, userPrompt, 1)
	if err != nil {
		return "", "", err
	}

	if len(responses) == 0 {
		return "", "", errors.New("no PR content was generated")
	}

	// Parse the response to extract title and description
	response := responses[0]

	// Extract title and description from response
	var (
		title       = ""
		description = ""
		titleIndex  = -1
	)

	var (
		leadingNonAlphanumericRegex = regexp.MustCompile(`^[^a-zA-Z0-9]+`)
		lines                       = strings.Split(response, "\n")
	)

	// First pass: find title and description
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Clean only the start of the line for prefix matching
		cleanedPrefix := leadingNonAlphanumericRegex.ReplaceAllString(line, "")

		if strings.HasPrefix(cleanedPrefix, PRTitlePrefix) {
			// Clean the title again after removing the prefix
			title = strings.TrimSpace(strings.TrimPrefix(cleanedPrefix, PRTitlePrefix))
			title = leadingNonAlphanumericRegex.ReplaceAllString(title, "")
			// handle case when title is in the next line
			if title == "" {
				i++
				title = strings.TrimSpace(lines[i])
			}
			titleIndex = i
		} else if strings.HasPrefix(cleanedPrefix, PRDescPrefix) {
			description = strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
			break
		}
	}

	if description == "" {
		// If we found a title, use everything after the title line
		if title != "" && titleIndex+1 < len(lines) {
			description = strings.TrimSpace(strings.Join(lines[titleIndex+1:], "\n"))
		} else {
			description = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}

	return title, description, nil
}
