package config

import (
	"fmt"
	"os"

	"github.com/zbiljic/vconfig-go"
)

// loadCreateMigrate loads existing config or creates new one, handling migrations
func loadCreateMigrate() (*Config, error) {
	configPath, err := FindFile()
	if err != nil {
		if os.IsNotExist(err) {
			// no config file found, return default configuration
			config := NewDefault()
			return config, nil
		}
		return nil, fmt.Errorf("error searching for config file: %w", err)
	}

	version, err := vconfig.GetVersion(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// fallback create new config
			config := NewDefault()
			return config, nil
		}
		return nil, err
	}

	switch version {
	case configVersionV0:
		config, err := vconfig.LoadConfig[configV0](configPath)
		if err != nil {
			return nil, errLoadVersion(version, err)
		}
		return config, nil
	default:
		return nil, errUnknownVersion(version)
	}
}
