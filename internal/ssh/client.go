package ssh

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client wraps an SSH connection for remote crontab management.
type Client struct {
	server Server
	client *ssh.Client
}

// ConnectTimeout is the maximum time to wait for an SSH connection.
const ConnectTimeout = 10 * time.Second

// NewClient creates a new SSH client and connects to the server.
func NewClient(server Server) (*Client, error) {
	config, err := buildSSHConfig(server)
	if err != nil {
		return nil, fmt.Errorf("failed to build ssh config for %s: %w", server.Name, err)
	}

	addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Printf("ssh: connection failed to %s (%s): %v", server.Name, addr, err)
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	log.Printf("ssh: connected to %s (%s@%s)", server.Name, server.User, addr)

	return &Client{
		server: server,
		client: conn,
	}, nil
}

// Close closes the SSH connection.
func (c *Client) Close() error {
	if c.client != nil {
		log.Printf("ssh: disconnected from %s", c.server.Name)
		return c.client.Close()
	}
	return nil
}

// ReadCrontab reads the remote crontab via `crontab -l`.
func (c *Client) ReadCrontab() (string, error) {
	output, err := c.runCommand("crontab -l")
	if err != nil {
		// Handle "no crontab for user" gracefully
		if strings.Contains(err.Error(), "no crontab for") {
			return "", nil
		}
		return "", fmt.Errorf("failed to read remote crontab: %w", err)
	}
	return output, nil
}

// ReadMail reads the remote user's mail file via `cat /var/mail/$USER`.
// Returns empty string if no mail file exists.
func (c *Client) ReadMail() (string, error) {
	output, err := c.runCommand("cat /var/mail/$(whoami) 2>/dev/null || true")
	if err != nil {
		return "", fmt.Errorf("failed to read remote mail: %w", err)
	}
	return output, nil
}

// WriteCrontab writes crontab content via `crontab -` on the remote server.
func (c *Client) WriteCrontab(content string) error {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdin = strings.NewReader(content)

	var stderr bytes.Buffer
	session.Stderr = &stderr

	if err := session.Run("crontab -"); err != nil {
		return fmt.Errorf("failed to write remote crontab: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	return nil
}

// ServerName returns the configured name of the server.
func (c *Client) ServerName() string {
	return c.server.Name
}

// IsConnected returns whether the client has an active connection.
func (c *Client) IsConnected() bool {
	if c.client == nil {
		return false
	}
	// Send a keep-alive request to check the connection
	_, _, err := c.client.SendRequest("keepalive@lazycron", true, nil)
	return err == nil
}

// runCommand executes a command on the remote server and returns stdout.
func (c *Client) runCommand(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("%s: %w", errMsg, err)
		}
		return "", err
	}

	return stdout.String(), nil
}

// knownHostsPath returns the path to the SSH known_hosts file.
// Uses the system-standard ~/.ssh/known_hosts.
func knownHostsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

// hostKeyCallback returns a host key callback that verifies against known_hosts.
// If the known_hosts file doesn't exist, it creates an empty one.
func hostKeyCallback() (ssh.HostKeyCallback, error) {
	path, err := knownHostsPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine known_hosts path: %w", err)
	}

	// Ensure ~/.ssh directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Create known_hosts if it doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			return nil, fmt.Errorf("failed to create known_hosts: %w", err)
		}
	}

	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load known_hosts: %w", err)
	}

	return callback, nil
}

// buildSSHConfig creates an ssh.ClientConfig from a Server configuration.
func buildSSHConfig(server Server) (*ssh.ClientConfig, error) {
	if err := server.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	var authMethods []ssh.AuthMethod

	switch server.AuthType {
	case "key":
		signer, err := loadPrivateKey(server.KeyPath)
		if err != nil {
			return nil, err
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	case "password":
		decrypted, err := DecryptPassword(server.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password for server %q: %w (re-add the server to re-encrypt)", server.Name, err)
		}
		authMethods = append(authMethods, ssh.Password(decrypted))

	default:
		// Try default key locations
		signer, err := loadDefaultKey()
		if err != nil {
			return nil, fmt.Errorf("no auth method configured and no default key found: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	hkCallback, err := hostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("failed to setup host key verification: %w", err)
	}

	return &ssh.ClientConfig{
		User:            server.User,
		Auth:            authMethods,
		Timeout:         ConnectTimeout,
		HostKeyCallback: hkCallback,
	}, nil
}

// loadPrivateKey reads and parses a private key file.
// The path must resolve to within the user's home directory.
func loadPrivateKey(path string) (ssh.Signer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(home, path[2:])
	}

	// Canonicalise and validate
	clean := filepath.Clean(path)
	if !strings.HasPrefix(clean, home+string(filepath.Separator)) && clean != home {
		return nil, fmt.Errorf("key path %q must be within home directory", path)
	}

	key, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("failed to read key %s: %w", clean, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key %s: %w", clean, err)
	}

	return signer, nil
}

// loadDefaultKey tries to load the default SSH key (~/.ssh/id_ed25519 or ~/.ssh/id_rsa).
func loadDefaultKey() (ssh.Signer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	keyPaths := []string{
		home + "/.ssh/id_ed25519",
		home + "/.ssh/id_rsa",
	}

	for _, path := range keyPaths {
		if signer, err := loadPrivateKey(path); err == nil {
			return signer, nil
		}
	}

	return nil, fmt.Errorf("no default SSH key found")
}

// TestConnection tests connectivity to the server without running any commands.
func TestConnection(server Server) error {
	config, err := buildSSHConfig(server)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))
	conn, err := net.DialTimeout("tcp", addr, ConnectTimeout)
	if err != nil {
		return fmt.Errorf("cannot reach %s: %w", addr, err)
	}
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return fmt.Errorf("ssh handshake failed: %w", err)
	}

	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close() //nolint:errcheck // best-effort cleanup

	return nil
}
