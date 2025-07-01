package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Root     string   `yaml:"root"`
	LogLevel string   `yaml:"log_level"`
	Exclude  []string `yaml:"exclude"`
	DryRun   bool     `yaml:"dry_run"`
}

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
	return &cfg, nil
}
