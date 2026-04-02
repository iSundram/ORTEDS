// Package config provides configuration management for ORTEDS.
package config

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Diagnostics holds diagnostic system configuration.
type Diagnostics struct {
	Enabled         bool  `yaml:"enabled"`
	ShowInRead      bool  `yaml:"show_in_read"`
	BlockOnError    bool  `yaml:"block_on_error"`
	BlockOnWarning  bool  `yaml:"block_on_warning"`
	MaxFileSizeBytes int64 `yaml:"max_file_size"`
	CacheDurationSec int   `yaml:"cache_duration"`
}

// Config holds the global application configuration.
type Config struct {
	Diagnostics Diagnostics `yaml:"diagnostics"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Diagnostics: Diagnostics{
			Enabled:          true,
			ShowInRead:       true,
			BlockOnError:     true,
			BlockOnWarning:   false,
			MaxFileSizeBytes: 1 << 20, // 1 MB
			CacheDurationSec: 30,
		},
	}
}

var (
	once   sync.Once
	global *Config
)

// Load reads the config file at ~/.owecode/config.yaml, falling back to
// defaults when the file does not exist or cannot be parsed.
func Load() *Config {
	once.Do(func() {
		global = loadOnce()
	})
	return global
}

// Reset clears the cached config (useful for tests).
func Reset() {
	once = sync.Once{}
	global = nil
}

func loadOnce() *Config {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	data, err := os.ReadFile(filepath.Join(home, ".owecode", "config.yaml"))
	if err != nil {
		return cfg
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return cfg
	}
	return cfg
}
