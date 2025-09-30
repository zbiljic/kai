package config

import "fmt"

const configVersionV1 = "1"

type configV1 struct {
	Version   string                      `json:"version"`         // required by vconfig-go
	Model     string                      `json:"model,omitempty"` // global default model
	Providers map[string]providerConfigV1 `json:"providers"`
	Agents    map[string]agentConfigV1    `json:"agents,omitempty"`
}

// providerConfigV1 represents a single provider configuration
type providerConfigV1 struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"` // "openai", "anthropic", etc.
	BaseURL      string            `json:"base_url,omitempty"`
	APIKey       string            `json:"api_key,omitempty"`
	Models       []modelConfigV1   `json:"models,omitempty"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
	Disable      bool              `json:"disable,omitempty"`
}

// modelConfigV1 represents a model definition
type modelConfigV1 struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// agentConfigV1 represents command-specific model configuration
type agentConfigV1 struct {
	Model       string `json:"model,omitempty"` // "provider/model-id" format
	Description string `json:"description,omitempty"`
}

// newConfigV1 creates a new v1 configuration
func newConfigV1() *configV1 {
	return &configV1{
		Version: configVersionV1,
		Model:   "phind/phind-70b",
		Providers: map[string]providerConfigV1{
			"phind": {
				Name: "Phind",
				Type: "openai",
				Models: []modelConfigV1{
					{ID: "phind-70b", Name: "Phind-70B"},
				},
			},
		},
		Agents: map[string]agentConfigV1{
			"gen": {
				Model:       "phind/phind-70b",
				Description: "Fast commit message generation",
			},
			"prgen": {
				Model:       "phind/phind-70b",
				Description: "PR title and description generation",
			},
			"prprepare": {
				Model:       "phind/phind-70b",
				Description: "Commit history reorganization",
			},
		},
	}
}

func (c *configV1) validateV1() error {
	if c.Providers == nil {
		return fmt.Errorf("providers section is required")
	}

	// validate that all provider references in global models exist
	if c.Model != "" {
		provider, _, err := ParseModelReference(c.Model)
		if err != nil {
			return fmt.Errorf("invalid global model reference: %w", err)
		}
		if _, exists := c.Providers[provider]; !exists {
			return fmt.Errorf("provider '%s' referenced in model '%s' does not exist", provider, c.Model)
		}
	}

	// validate provider configurations
	for providerName, provider := range c.Providers {
		if provider.Name == "" {
			return fmt.Errorf("provider '%s' must have a name", providerName)
		}
		if provider.Type == "" {
			return fmt.Errorf("provider '%s' must have a type", providerName)
		}
	}

	// Validate agent configurations
	for agentName, agent := range c.Agents {
		if agent.Model != "" {
			provider, _, err := ParseModelReference(agent.Model)
			if err != nil {
				return fmt.Errorf("invalid model reference in agent '%s': %w", agentName, err)
			}
			if _, exists := c.Providers[provider]; !exists {
				return fmt.Errorf("provider '%s' referenced in agent '%s' does not exist", provider, agentName)
			}
		}
	}

	return nil
}
