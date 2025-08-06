package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mahyarmirrashed/jdd/internal/config"
)

// loadConfig loads the existing configuration file.
func loadConfig(path string) (*config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &config.Config{}
	if err = yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return cfg, nil
}

// parseExcludePatterns parses exclude patterns from multiline text.
func parseExcludePatterns(text string) []string {
	lines := strings.Split(text, "\n")
	var patterns []string
	for _, line := range lines {
		p := strings.TrimSpace(line)
		if p != "" {
			patterns = append(patterns, p)
		}
	}
	return patterns
}

// saveConfig saves configuration back to YAML file.
func saveConfig(path string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
