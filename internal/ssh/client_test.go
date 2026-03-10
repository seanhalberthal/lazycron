package ssh

import (
	"testing"
)

func TestBuildSSHConfigRejectsInvalidServer(t *testing.T) {
	server := Server{Host: "", Port: 22, User: ""}
	_, err := buildSSHConfig(server)
	if err == nil {
		t.Error("expected error for invalid server, got nil")
	}
}

func TestBuildSSHConfigRejectsUndecryptablePassword(t *testing.T) {
	server := Server{
		Host:     "example.com",
		Port:     22,
		User:     "root",
		AuthType: "password",
		Password: "not-valid-hex-ciphertext",
	}
	_, err := buildSSHConfig(server)
	if err == nil {
		t.Error("expected error for undecryptable password, got nil")
	}
}

func TestLoadPrivateKeyRejectsPathOutsideHome(t *testing.T) {
	_, err := loadPrivateKey("/etc/passwd")
	if err == nil {
		t.Error("expected error for path outside home directory, got nil")
	}
}
