package ssh

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
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

// Validate checks that the server configuration has required fields and sane values.
func (s Server) Validate() error {
	if s.Host == "" {
		return fmt.Errorf("host is required")
	}
	if s.User == "" {
		return fmt.Errorf("user is required")
	}
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", s.Port)
	}
	if len(s.Host) > 253 {
		return fmt.Errorf("host exceeds maximum length of 253 characters")
	}
	if s.AuthType != "" && s.AuthType != "key" && s.AuthType != "password" {
		return fmt.Errorf("auth_type must be \"key\" or \"password\", got %q", s.AuthType)
	}
	return nil
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

// parseServersToml decodes TOML data into the servers config.
func parseServersToml(data string, config *ServersConfig) error {
	if _, err := toml.Decode(data, config); err != nil {
		return fmt.Errorf("failed to decode TOML: %w", err)
	}
	// Apply default port for servers that don't specify one
	for i := range config.Servers {
		if config.Servers[i].Port == 0 {
			config.Servers[i].Port = 22
		}
	}
	return nil
}

// serialiseServersToml encodes the config as TOML.
func serialiseServersToml(config *ServersConfig) string {
	var buf bytes.Buffer
	buf.WriteString("# lazycron server configuration\n\n")
	encoder := toml.NewEncoder(&buf)
	_ = encoder.Encode(config)
	return buf.String()
}

// keyPath returns the path to the encryption keyfile.
func keyPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".key"), nil
}

// encryptionKey loads or generates the encryption key.
// On first run, a random 32-byte key is generated and stored at
// ~/.config/lazycron/.key with 0600 permissions.
func encryptionKey() ([]byte, error) {
	path, err := keyPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err == nil && len(data) == 32 {
		return data, nil
	}

	// Generate new key
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, key, 0600); err != nil {
		return nil, fmt.Errorf("failed to persist encryption key: %w", err)
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
