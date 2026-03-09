package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user preferences.
type Config struct {
	Theme      string `json:"theme"`
	TabWidth   int    `json:"tab_width"`
	ExpandTabs bool   `json:"expand_tabs"`
}

func Default() *Config {
	return &Config{Theme: "default", TabWidth: 4, ExpandTabs: true}
}

// TabWidthOrDefault returns TabWidth, falling back to 4 if not set.
func (c *Config) TabWidthOrDefault() int {
	if c.TabWidth < 1 {
		return 4
	}
	return c.TabWidth
}

func Load() (*Config, error) {
	cfg := Default()
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "eden", "config.json"))
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	_ = json.Unmarshal(data, cfg)
	return cfg, nil
}

func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "eden")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}
