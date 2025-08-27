package config

import (
	"errors"
	"fmt"
)

var errInvalidArgument = errors.New("invalid arguments provided")

var (
	errLoadVersion = func(version string, err error) error {
		return fmt.Errorf("unable to load config version '%s': %w", version, err)
	}

	errUnknownVersion = func(version string) error {
		return fmt.Errorf("unknown version: '%s'", version)
	}

	errFailedToCreateDirectory = func(dir string, err error) error {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	errFailedToSaveConfig = func(filename string, err error) error {
		return fmt.Errorf("failed to save config to %s: %w", filename, err)
	}
)
