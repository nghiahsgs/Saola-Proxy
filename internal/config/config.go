// Package config loads and merges Saola Proxy configuration from global and
// project-level YAML files.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for Saola Proxy.
type Config struct {
	Version      int           `yaml:"version"`
	LogLevel     string        `yaml:"log_level"`
	AuditEnabled bool          `yaml:"audit_enabled"`
	Patterns     PatternConfig `yaml:"patterns"`
	Whitelist    []string      `yaml:"whitelist"`
}

// PatternConfig controls which patterns are active and allows custom ones.
type PatternConfig struct {
	Disabled []string        `yaml:"disabled"`
	Custom   []CustomPattern `yaml:"custom"`
}

// CustomPattern defines a user-provided PII detection rule.
type CustomPattern struct {
	Name        string `yaml:"name"`
	Category    string `yaml:"category"`
	Regex       string `yaml:"regex"`
	Description string `yaml:"description"`
}

// ConfigDir returns the saola config directory: $XDG_CONFIG_HOME/saola or ~/.saola.
// Returns an error if the home directory cannot be determined.
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "saola"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ".saola"), nil
}

// Load builds a merged Config:
//  1. Start with defaults.
//  2. Overlay global config ($XDG_CONFIG_HOME/saola/config.yaml or ~/.saola/config.yaml).
//  3. Overlay project config (.saola.yaml in cwd).
//
// Missing config files are silently ignored (no error).
func Load() (*Config, error) {
	cfg := DefaultConfig()

	configDir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	globalPath := filepath.Join(configDir, "config.yaml")
	if err := mergeFile(cfg, globalPath); err != nil {
		return nil, err
	}

	projectPath := ".saola.yaml"
	if err := mergeFile(cfg, projectPath); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromPath parses a single YAML file into a Config, starting from defaults.
func LoadFromPath(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// WriteToFile serialises cfg as YAML and writes it to path (mode 0600).
func (c *Config) WriteToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// mergeFile reads YAML from path and overlays it onto dst.
// A missing file is silently ignored.
func mergeFile(dst *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return yaml.Unmarshal(data, dst)
}
