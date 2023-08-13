package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type OpenPR struct {
	Repo         string `yaml:"repository"`
	TargetFile   string `yaml:"target_file"`
	TargetBranch string `yaml:"target_branch"`
}

type Backend struct {
	URL      string `yaml:"url"`
	Schedule string `yaml:"schedule"`
	OpenPR   OpenPR `yaml:"open_pull_request"`
}

type Config struct {
	Backends []Backend `yaml:"backends"`
}

// Parse a yaml given file into a *config.Config struct.
func Parse(path string) (*Config, error) {
	yamlFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dir.Open: %w", err)
	}

	defer yamlFile.Close()

	var config Config

	if err := yaml.NewDecoder(yamlFile).Decode(&config); err != nil {
		return nil, fmt.Errorf("yaml.NewDecoder.Decode: %w", err)
	}

	return &config, nil
}
