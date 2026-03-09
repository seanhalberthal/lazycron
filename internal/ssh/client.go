package ssh

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
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
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &Client{
		server: server,
		client: conn,
	}, nil
}

// Close closes the SSH connection.
func (c *Client) Close() error {
	if c.client != nil {
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

// buildSSHConfig creates an ssh.ClientConfig from a Server configuration.
func buildSSHConfig(server Server) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	switch server.AuthType {
	case "key":
		signer, err := loadPrivateKey(server.KeyPath)
		if err != nil {
			return nil, err
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	case "password":
		password := server.Password
		// Try to decrypt if it looks encrypted (hex-encoded)
		if decrypted, err := DecryptPassword(password); err == nil {
			password = decrypted
		}
		authMethods = append(authMethods, ssh.Password(password))

	default:
		// Try default key locations
		signer, err := loadDefaultKey()
		if err != nil {
			return nil, fmt.Errorf("no auth method configured and no default key found: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	return &ssh.ClientConfig{
		User:            server.User,
		Auth:            authMethods,
		Timeout:         ConnectTimeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // User-configured servers
	}, nil
}

// loadPrivateKey reads and parses a private key file.
func loadPrivateKey(path string) (ssh.Signer, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = home + path[1:]
	}

	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key %s: %w", path, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key %s: %w", path, err)
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

// DialFunc allows overriding the SSH dial function for testing.
var DialFunc = func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	return ssh.Dial(network, addr, config)
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
