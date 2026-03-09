package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the lazycron application configuration.
type Config struct {
	DefaultTab string `toml:"default_tab"` // "local" or "servers"
	Theme      Theme  `toml:"theme"`
}

// Theme holds colour/style configuration.
type Theme struct {
	ActiveBorder   string `toml:"active_border"`   // colour name
	InactiveBorder string `toml:"inactive_border"` // colour name
	SelectedBg     string `toml:"selected_bg"`     // colour name
	SelectedFg     string `toml:"selected_fg"`     // colour name
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultTab: "local",
		Theme: Theme{
			ActiveBorder:   "green",
			InactiveBorder: "default",
			SelectedBg:     "green",
			SelectedFg:     "black",
		},
	}
}

// DefaultConfigPath returns the default path for the config file.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "lazycron", "config.toml"), nil
}

// Load reads the config from disk, returning defaults if the file doesn't exist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	config := DefaultConfig()
	if err := parseConfigToml(string(data), config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// Save writes the config to disk.
func Save(path string, config *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	content := serialiseConfigToml(config)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// parseConfigToml is a minimal TOML parser for the config file.
func parseConfigToml(data string, config *Config) error {
	section := ""
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "[theme]" {
			section = "theme"
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")

		switch section {
		case "":
			if key == "default_tab" {
				config.DefaultTab = value
			}
		case "theme":
			switch key {
			case "active_border":
				config.Theme.ActiveBorder = value
			case "inactive_border":
				config.Theme.InactiveBorder = value
			case "selected_bg":
				config.Theme.SelectedBg = value
			case "selected_fg":
				config.Theme.SelectedFg = value
			}
		}
	}

	return nil
}

// serialiseConfigToml converts the config to TOML format.
func serialiseConfigToml(config *Config) string {
	var sb strings.Builder
	sb.WriteString("# lazycron configuration\n\n")
	fmt.Fprintf(&sb, "default_tab = %q\n", config.DefaultTab)
	sb.WriteString("\n[theme]\n")
	fmt.Fprintf(&sb, "active_border = %q\n", config.Theme.ActiveBorder)
	fmt.Fprintf(&sb, "inactive_border = %q\n", config.Theme.InactiveBorder)
	fmt.Fprintf(&sb, "selected_bg = %q\n", config.Theme.SelectedBg)
	fmt.Fprintf(&sb, "selected_fg = %q\n", config.Theme.SelectedFg)
	return sb.String()
}
