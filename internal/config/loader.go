package config

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/samber/lo"

	"github.com/zbiljic/vconfig-go"
)

var (
	// Cached configuration to avoid loading multiple times
	cachedConfig *Config
	// Mutex for thread-safe access to config file
	configMutex = &sync.Mutex{}
)

// Load loads configuration using the migration system
func Load() (*Config, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	if cachedConfig != nil {
		return cachedConfig, nil
	}

	// use the migration system to load/create/migrate configuration
	// since Config is an alias for latest version, we can return it directly
	config, err := loadCreateMigrate()
	if err != nil {
		return nil, err
	}

	cachedConfig = config
	return config, nil
}

// Save saves configuration to a file
func Save(config *Config, filename string) error {
	if config == nil || filename == "" {
		return errInvalidArgument
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	// ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errFailedToCreateDirectory(dir, err)
	}

	if err := vconfig.SaveConfig(config, filename); err != nil {
		return errFailedToSaveConfig(filename, err)
	}

	// update the cached config so subsequent loads see saved state
	cachedConfig = config

	return nil
}

// FindFile searches for configuration file in hierarchical order
func FindFile() (string, error) {
	searchPaths := GetSearchPaths()

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", os.ErrNotExist
}

// GetSearchPaths returns the list of paths to search for configuration files
func GetSearchPaths() []string {
	var paths []string

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// 1. ./.kai.json (current directory)
	paths = append(paths, filepath.Join(cwd, ".kai.json"))

	// 2. ./kai.json (current directory)
	paths = append(paths, filepath.Join(cwd, "kai.json"))

	// 3. Walk up directories looking for kai.json
	dir := cwd
	homeDir := lo.Must(os.UserHomeDir())
	for {
		parent := filepath.Dir(dir)
		if parent == dir || parent == homeDir {
			break // reached root or home directory
		}
		dir = parent
		paths = append(paths, filepath.Join(dir, "kai.json"))
	}

	// 4. ~/.config/kai/kai.json (user config)
	configDir := filepath.Join(homeDir, ".config", "kai")
	paths = append(paths, filepath.Join(configDir, "kai.json"))

	// 5. ~/.kai.json (user home fallback)
	paths = append(paths, filepath.Join(homeDir, ".kai.json"))

	return paths
}

// GetPath returns the path where configuration would be loaded from
func GetPath() (string, bool) {
	path, err := FindFile()
	return path, err == nil
}

// GetDefaultPath returns the default path for user configuration
func GetDefaultPath() string {
	homeDir := lo.Must(os.UserHomeDir())

	// Prefer ~/.config/kai/kai.json
	configDir := filepath.Join(homeDir, ".config", "kai")
	return filepath.Join(configDir, "kai.json")
}

// ResetCache clears the cached configuration (useful for testing)
func ResetCache() {
	configMutex.Lock()
	defer configMutex.Unlock()

	cachedConfig = nil
}
