// Package config resolves scan locations and loads the optional
// .agent-stack-audit.yml file. It is the only package that knows about
// environment variable overrides (CLAUDE_CONFIG_DIR, CLAUDE_MEM_DATA_DIR),
// so discover and memoryaudit can stay focused on parsing/reporting.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ScanPaths  ScanPaths  `yaml:"scan_paths"`
	Exclude    []string   `yaml:"exclude"`
	Output     Output     `yaml:"output"`
	Thresholds Thresholds `yaml:"thresholds"`
}

type ScanPaths struct {
	Extra []string `yaml:"extra"`
}

type Output struct {
	Format      []string `yaml:"format"`
	Destination string   `yaml:"destination"`
}

type Thresholds struct {
	TokenOverheadWarning int `yaml:"token_overhead_warning"`
}

const DefaultConfigFile = ".agent-stack-audit.yml"

// Load reads the optional config file. A missing file is not an error - the
// config file is opt-in per FASE 3 spec ("kalau ada").
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
