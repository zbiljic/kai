package config

// Config represents the current version of configuration
type Config = configV0

// NewDefault creates a new configuration
func NewDefault() *Config {
	return newConfigV0()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	return c.validateV0()
}
