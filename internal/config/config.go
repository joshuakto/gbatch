package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds gBatch configuration from .gbatchrc or ~/.gbatch/config.yaml.
type Config struct {
	Project     string   `yaml:"project"`
	Region      string   `yaml:"region"`
	DefaultCPUs int      `yaml:"default_cpus"`
	DefaultMem  string   `yaml:"default_mem"`
	Mounts      []string `yaml:"mounts"`
}

// Load reads config with layered resolution:
// project .gbatchrc > user ~/.gbatch/config.yaml > defaults.
func Load() *Config {
	cfg := &Config{
		Region:      "us-central1",
		DefaultCPUs: 4,
		DefaultMem:  "16G",
	}

	// Layer 1: user config (~/.gbatch/config.yaml)
	if home, err := os.UserHomeDir(); err == nil {
		userPath := filepath.Join(home, ".gbatch", "config.yaml")
		loadFile(userPath, cfg)
	}

	// Layer 2: project config (.gbatchrc) — overrides user
	loadFile(".gbatchrc", cfg)

	return cfg
}

func loadFile(path string, cfg *Config) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Parse into a temp struct, merge non-zero values
	var overlay Config
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return
	}
	if overlay.Project != "" {
		cfg.Project = overlay.Project
	}
	if overlay.Region != "" {
		cfg.Region = overlay.Region
	}
	if overlay.DefaultCPUs != 0 {
		cfg.DefaultCPUs = overlay.DefaultCPUs
	}
	if overlay.DefaultMem != "" {
		cfg.DefaultMem = overlay.DefaultMem
	}
	if len(overlay.Mounts) > 0 {
		cfg.Mounts = overlay.Mounts
	}
}
