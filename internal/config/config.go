package config

import "fmt"

// Config represents the current version of configuration
type Config = configV1

// Type aliases for external packages
type (
	ProviderConfig = providerConfigV1
	AgentConfig    = agentConfigV1
	ModelConfig    = modelConfigV1
)

// NewDefault creates a new configuration
func NewDefault() *Config {
	return newConfigV1()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	return c.validateV1()
}

// ParseModelReference parses a model reference in "provider/model-id" format
// from a string
func ParseModelReference(modelRef string) (provider, modelID string, err error) {
	if modelRef == "" {
		return "", "", fmt.Errorf("no model specified")
	}

	// split on the first "/" to separate provider and model
	for i, r := range modelRef {
		if r == '/' {
			if i == 0 {
				return "", "", fmt.Errorf("invalid model format: %s", modelRef)
			}
			return modelRef[:i], modelRef[i+1:], nil
		}
	}

	return "", "", fmt.Errorf("invalid model format (expected provider/model): %s", modelRef)
}
