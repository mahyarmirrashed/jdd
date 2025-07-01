package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the YAML configuration for the daemon.
type Config struct {
	Root      string   `yaml:"root"`      // Root directory to watch
	LogLevel  string   `yaml:"log_level"` // Logging level: debug, info, warn, error
	Exclude   []string `yaml:"exclude"`   // Glob patterns to exclude
	DryRun    bool     `yaml:"dry_run"`   // If true, don't move files
	Daemonize bool     `yaml:"daemonize"` // If true, run as daemon; if false, run in foreground
}

// LoadConfig loads the configuration from the given YAML file.
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.Exclude == nil {
		cfg.Exclude = []string{}
	}
	if cfg.Root == "" {
		cfg.Root = "."
	}

	// Resolve root path relative to config file location if not absolute
	if !filepath.IsAbs(cfg.Root) {
		cfg.Root = filepath.Join(filepath.Dir(path), cfg.Root)
	}

	cfg.Root, err = filepath.Abs(cfg.Root)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
