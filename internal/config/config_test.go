package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()

	if c.DefaultTab != "local" {
		t.Errorf("expected default_tab %q, got %q", "local", c.DefaultTab)
	}
	if c.Theme.ActiveBorder != "green" {
		t.Errorf("expected active_border %q, got %q", "green", c.Theme.ActiveBorder)
	}
	if c.Theme.SelectedBg != "green" {
		t.Errorf("expected selected_bg %q, got %q", "green", c.Theme.SelectedBg)
	}
}

func TestParseConfigToml(t *testing.T) {
	data := `
# Configuration
default_tab = "servers"

[theme]
active_border = "blue"
inactive_border = "white"
selected_bg = "cyan"
selected_fg = "white"
`

	config := DefaultConfig()
	if err := parseConfigToml(data, config); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if config.DefaultTab != "servers" {
		t.Errorf("expected default_tab %q, got %q", "servers", config.DefaultTab)
	}
	if config.Theme.ActiveBorder != "blue" {
		t.Errorf("expected active_border %q, got %q", "blue", config.Theme.ActiveBorder)
	}
	if config.Theme.SelectedFg != "white" {
		t.Errorf("expected selected_fg %q, got %q", "white", config.Theme.SelectedFg)
	}
}

func TestSerialiseRoundTrip(t *testing.T) {
	original := &Config{
		DefaultTab: "servers",
		Theme: Theme{
			ActiveBorder:   "blue",
			InactiveBorder: "white",
			SelectedBg:     "cyan",
			SelectedFg:     "black",
		},
	}

	toml := serialiseConfigToml(original)
	parsed := DefaultConfig()
	if err := parseConfigToml(toml, parsed); err != nil {
		t.Fatalf("failed to parse serialised TOML: %v", err)
	}

	if parsed.DefaultTab != original.DefaultTab {
		t.Errorf("default_tab mismatch: %q vs %q", parsed.DefaultTab, original.DefaultTab)
	}
	if parsed.Theme.ActiveBorder != original.Theme.ActiveBorder {
		t.Errorf("active_border mismatch: %q vs %q", parsed.Theme.ActiveBorder, original.Theme.ActiveBorder)
	}
}

func TestLoadSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	config := &Config{
		DefaultTab: "servers",
		Theme: Theme{
			ActiveBorder: "red",
		},
	}

	if err := Save(path, config); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.DefaultTab != "servers" {
		t.Errorf("expected default_tab %q, got %q", "servers", loaded.DefaultTab)
	}
	if loaded.Theme.ActiveBorder != "red" {
		t.Errorf("expected active_border %q, got %q", "red", loaded.Theme.ActiveBorder)
	}
}

func TestLoadNotExist(t *testing.T) {
	config, err := Load("/nonexistent/config.toml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if config.DefaultTab != "local" {
		t.Errorf("expected default config, got default_tab %q", config.DefaultTab)
	}
}

func TestParseEmptyConfig(t *testing.T) {
	config := DefaultConfig()
	if err := parseConfigToml("", config); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should retain defaults
	if config.DefaultTab != "local" {
		t.Errorf("expected default_tab %q, got %q", "local", config.DefaultTab)
	}
}
