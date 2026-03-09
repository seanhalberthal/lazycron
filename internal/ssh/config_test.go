package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseServersToml(t *testing.T) {
	data := `
# Server configuration

[[servers]]
name = "prod"
host = "192.168.1.100"
port = 22
user = "admin"
auth_type = "key"
key_path = "~/.ssh/id_ed25519"

[[servers]]
name = "staging"
host = "staging.example.com"
port = 2222
user = "deploy"
auth_type = "password"
password = "encrypted_value"
`

	config := &ServersConfig{}
	if err := parseServersToml(data, config); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(config.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(config.Servers))
	}

	s1 := config.Servers[0]
	if s1.Name != "prod" {
		t.Errorf("expected name %q, got %q", "prod", s1.Name)
	}
	if s1.Host != "192.168.1.100" {
		t.Errorf("expected host %q, got %q", "192.168.1.100", s1.Host)
	}
	if s1.Port != 22 {
		t.Errorf("expected port 22, got %d", s1.Port)
	}
	if s1.AuthType != "key" {
		t.Errorf("expected auth_type %q, got %q", "key", s1.AuthType)
	}

	s2 := config.Servers[1]
	if s2.Name != "staging" {
		t.Errorf("expected name %q, got %q", "staging", s2.Name)
	}
	if s2.Port != 2222 {
		t.Errorf("expected port 2222, got %d", s2.Port)
	}
}

func TestSerialiseRoundTrip(t *testing.T) {
	config := &ServersConfig{
		Servers: []Server{
			{
				Name:     "test-server",
				Host:     "10.0.0.1",
				Port:     22,
				User:     "root",
				AuthType: "key",
				KeyPath:  "~/.ssh/id_rsa",
			},
		},
	}

	toml := serialiseServersToml(config)
	parsed := &ServersConfig{}
	if err := parseServersToml(toml, parsed); err != nil {
		t.Fatalf("failed to parse serialised TOML: %v", err)
	}

	if len(parsed.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(parsed.Servers))
	}

	got := parsed.Servers[0]
	if got.Name != config.Servers[0].Name {
		t.Errorf("name mismatch: %q vs %q", got.Name, config.Servers[0].Name)
	}
	if got.Host != config.Servers[0].Host {
		t.Errorf("host mismatch: %q vs %q", got.Host, config.Servers[0].Host)
	}
}

func TestLoadSaveServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.toml")

	config := &ServersConfig{
		Servers: []Server{
			{
				Name:     "dev",
				Host:     "localhost",
				Port:     22,
				User:     "user",
				AuthType: "key",
			},
		},
	}

	if err := SaveServers(path, config); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := LoadServers(path)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if len(loaded.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(loaded.Servers))
	}

	if loaded.Servers[0].Name != "dev" {
		t.Errorf("expected name %q, got %q", "dev", loaded.Servers[0].Name)
	}
}

func TestLoadServersNotExist(t *testing.T) {
	config, err := LoadServers("/nonexistent/path/servers.toml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(config.Servers) != 0 {
		t.Errorf("expected empty config, got %d servers", len(config.Servers))
	}
}

func TestSaveServersCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "servers.toml")

	config := &ServersConfig{}
	if err := SaveServers(path, config); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}
}

func TestEncryptDecryptPassword(t *testing.T) {
	original := "mysecretpassword"

	encrypted, err := EncryptPassword(original)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	if encrypted == original {
		t.Error("encrypted password should differ from original")
	}

	decrypted, err := DecryptPassword(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if decrypted != original {
		t.Errorf("expected %q, got %q", original, decrypted)
	}
}

func TestParseEmptyConfig(t *testing.T) {
	config := &ServersConfig{}
	if err := parseServersToml("", config); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(config.Servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(config.Servers))
	}
}

func TestDefaultPort(t *testing.T) {
	data := `
[[servers]]
name = "minimal"
host = "example.com"
user = "user"
auth_type = "key"
`
	config := &ServersConfig{}
	if err := parseServersToml(data, config); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if config.Servers[0].Port != 22 {
		t.Errorf("expected default port 22, got %d", config.Servers[0].Port)
	}
}
