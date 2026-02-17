package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds specflow project configuration.
type Config struct {
	Mode             string            `yaml:"mode"`
	ConventionsFile  string            `yaml:"conventions_file"`
	AgentsFile       string            `yaml:"agents_file,omitempty"`
	PatternExemplars map[string]string `yaml:"pattern_exemplars,omitempty"`
	DefaultPriority  string            `yaml:"default_priority"`
	DefaultLabels    []string          `yaml:"default_labels,omitempty"`
}

// GlobalConfig holds user-level configuration from ~/.specflow/config.yaml.
type GlobalConfig struct {
	Editor      string `yaml:"editor,omitempty"`
	DefaultMode string `yaml:"default_mode,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Mode:            "careful",
		ConventionsFile: "CLAUDE.md",
		DefaultPriority: "medium",
	}
}

// Load reads project config from the given .specflow/config.yaml path.
// Returns default config if file doesn't exist.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}

// Save writes project config to the given path.
func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadGlobal reads user-level config from ~/.specflow/config.yaml.
func LoadGlobal() (GlobalConfig, error) {
	var gc GlobalConfig

	home, err := os.UserHomeDir()
	if err != nil {
		return gc, nil
	}

	path := filepath.Join(home, ".specflow", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return gc, nil
		}
		return gc, fmt.Errorf("reading global config: %w", err)
	}

	if err := yaml.Unmarshal(data, &gc); err != nil {
		return gc, fmt.Errorf("parsing global config: %w", err)
	}

	return gc, nil
}

// Get retrieves a config value by key name.
func (c Config) Get(key string) (string, error) {
	switch key {
	case "mode":
		return c.Mode, nil
	case "conventions_file":
		return c.ConventionsFile, nil
	case "agents_file":
		return c.AgentsFile, nil
	case "default_priority":
		return c.DefaultPriority, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Set updates a config value by key name.
func (c *Config) Set(key, value string) error {
	switch key {
	case "mode":
		if value != "careful" && value != "fast" {
			return fmt.Errorf("mode must be 'careful' or 'fast', got %q", value)
		}
		c.Mode = value
	case "conventions_file":
		c.ConventionsFile = value
	case "agents_file":
		c.AgentsFile = value
	case "default_priority":
		c.DefaultPriority = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}
