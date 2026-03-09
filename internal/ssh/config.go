package ssh

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Server represents a configured remote server.
type Server struct {
	Name     string `toml:"name"`
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	AuthType string `toml:"auth_type"` // "key" or "password"
	KeyPath  string `toml:"key_path"`
	Password string `toml:"password"` // encrypted at rest
}

// ServersConfig holds the list of configured servers.
type ServersConfig struct {
	Servers []Server `toml:"servers"`
}

// DefaultConfigDir returns the default configuration directory.
func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "lazycron"), nil
}

// DefaultServersPath returns the default path for the servers config file.
func DefaultServersPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "servers.toml"), nil
}

// LoadServers reads the servers config from disk.
func LoadServers(path string) (*ServersConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ServersConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read servers config: %w", err)
	}

	config := &ServersConfig{}
	if err := parseServersToml(string(data), config); err != nil {
		return nil, fmt.Errorf("failed to parse servers config: %w", err)
	}

	return config, nil
}

// SaveServers writes the servers config to disk.
func SaveServers(path string, config *ServersConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	content := serialiseServersToml(config)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write servers config: %w", err)
	}

	return nil
}

// parseServersToml is a minimal TOML parser for the servers config.
// We avoid the full TOML dependency until Phase 5 by handling just what we need.
func parseServersToml(data string, config *ServersConfig) error {
	var current *Server
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "[[servers]]" {
			config.Servers = append(config.Servers, Server{Port: 22})
			current = &config.Servers[len(config.Servers)-1]
			continue
		}

		if current == nil {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")

		switch key {
		case "name":
			current.Name = value
		case "host":
			current.Host = value
		case "port":
			fmt.Sscanf(value, "%d", &current.Port)
		case "user":
			current.User = value
		case "auth_type":
			current.AuthType = value
		case "key_path":
			current.KeyPath = value
		case "password":
			current.Password = value
		}
	}

	return nil
}

// serialiseServersToml converts the config to TOML format.
func serialiseServersToml(config *ServersConfig) string {
	var sb strings.Builder
	sb.WriteString("# lazycron server configuration\n\n")

	for _, s := range config.Servers {
		sb.WriteString("[[servers]]\n")
		fmt.Fprintf(&sb, "name = %q\n", s.Name)
		fmt.Fprintf(&sb, "host = %q\n", s.Host)
		fmt.Fprintf(&sb, "port = %d\n", s.Port)
		fmt.Fprintf(&sb, "user = %q\n", s.User)
		fmt.Fprintf(&sb, "auth_type = %q\n", s.AuthType)
		if s.KeyPath != "" {
			fmt.Fprintf(&sb, "key_path = %q\n", s.KeyPath)
		}
		if s.Password != "" {
			fmt.Fprintf(&sb, "password = %q\n", s.Password)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// encryptionKey derives a simple key from the machine. In production, this
// should use a proper key derivation function, but for v0.0.4 this suffices.
func encryptionKey() ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	// Use home dir path as seed — not cryptographically ideal but
	// prevents plaintext storage. Phase 5 should improve this.
	key := make([]byte, 32)
	seed := []byte(home + "lazycron-server-config")
	for i := range key {
		key[i] = seed[i%len(seed)]
	}
	return key, nil
}

// EncryptPassword encrypts a password for storage.
func EncryptPassword(plaintext string) (string, error) {
	key, err := encryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// DecryptPassword decrypts a stored password.
func DecryptPassword(encrypted string) (string, error) {
	key, err := encryptionKey()
	if err != nil {
		return "", err
	}

	data, err := hex.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
