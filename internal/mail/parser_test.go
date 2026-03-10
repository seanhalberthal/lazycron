package mail

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", name)
}

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(testdataPath(name))
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return string(data)
}

func TestParseEmpty(t *testing.T) {
	messages, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestParseEmptyFile(t *testing.T) {
	text := loadFixture(t, "empty.mbox")
	messages, err := Parse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestParseSingleMessage(t *testing.T) {
	text := loadFixture(t, "single.mbox")
	messages, err := Parse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.Subject != "Cron <user@server> /usr/bin/backup.sh" {
		t.Errorf("subject = %q, want %q", msg.Subject, "Cron <user@server> /usr/bin/backup.sh")
	}
	if msg.From != "Cron Daemon <root@server.local>" {
		t.Errorf("from = %q, want %q", msg.From, "Cron Daemon <root@server.local>")
	}
	if !msg.IsRead() {
		t.Error("message should be marked as read (Status: RO)")
	}
	if msg.Body == "" {
		t.Error("body should not be empty")
	}
}

func TestParseMultipleMessages(t *testing.T) {
	text := loadFixture(t, "multiple.mbox")
	messages, err := Parse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// First message: read
	if !messages[0].IsRead() {
		t.Error("message 0 should be read (Status: RO)")
	}

	// Second message: unread (no Status header)
	if messages[1].IsRead() {
		t.Error("message 1 should be unread (no Status header)")
	}

	// Third message: old but not explicitly read
	if messages[2].IsRead() {
		t.Error("message 2 should not be marked as read (Status: O)")
	}

	// Check subjects
	expectedSubjects := []string{
		"Cron <user@server> /usr/bin/backup.sh",
		"Cron <user@server> /usr/bin/cleanup.sh",
		"Cron <user@server> /usr/bin/healthcheck.sh",
	}
	for i, exp := range expectedSubjects {
		if messages[i].Subject != exp {
			t.Errorf("message %d subject = %q, want %q", i, messages[i].Subject, exp)
		}
	}
}

func TestParseMalformed(t *testing.T) {
	text := loadFixture(t, "malformed.mbox")
	messages, err := Parse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should parse at least the well-formed message; malformed ones may be skipped
	if len(messages) < 1 {
		t.Errorf("expected at least 1 message from malformed fixture, got %d", len(messages))
	}

	// First message should be valid
	if messages[0].Subject != "Cron <user@server> /usr/bin/report.sh" {
		t.Errorf("message 0 subject = %q, want %q", messages[0].Subject, "Cron <user@server> /usr/bin/report.sh")
	}
}

func TestIsEnvelopeLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid envelope", "From user@host Mon Jan 06 09:00:00 2025", true},
		{"minimal envelope", "From user@host date", true},
		{"not envelope — too short", "From x", false},
		{"not envelope — From header", "From: user@host", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEnvelopeLine(tt.input)
			if got != tt.want {
				t.Errorf("isEnvelopeLine(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMessageIsRead(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"read and old", "RO", true},
		{"read only", "R", true},
		{"old only", "O", false},
		{"empty", "", false},
		{"new", "N", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{Status: tt.status}
			if got := msg.IsRead(); got != tt.want {
				t.Errorf("IsRead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageDateString(t *testing.T) {
	msg := &Message{}
	if msg.DateString() != "unknown" {
		t.Errorf("zero date should return %q, got %q", "unknown", msg.DateString())
	}
}

func TestParseBadDate(t *testing.T) {
	text := loadFixture(t, "bad-date.mbox")
	messages, err := Parse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if !messages[0].Date.IsZero() {
		t.Error("expected zero date for unparseable Date header")
	}
	if messages[0].Subject != "Test with bad date" {
		t.Errorf("subject = %q, want %q", messages[0].Subject, "Test with bad date")
	}
}
