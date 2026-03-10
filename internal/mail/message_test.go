package mail

import (
	"strings"
	"testing"
	"time"
)

func TestShortFromUnchanged(t *testing.T) {
	msg := &Message{From: "root@host"}
	if msg.ShortFrom() != "root@host" {
		t.Errorf("short from should be unchanged, got %q", msg.ShortFrom())
	}
}

func TestShortFromExactly30(t *testing.T) {
	from := strings.Repeat("a", 30)
	msg := &Message{From: from}
	if msg.ShortFrom() != from {
		t.Errorf("expected no truncation at exactly 30 chars")
	}
}

func TestShortFromTruncation(t *testing.T) {
	from := "A Very Long Email Address That Is Way Too Long <long@example.com>"
	msg := &Message{From: from}
	short := msg.ShortFrom()

	if len(short) > 30 {
		t.Errorf("ShortFrom should be <= 30 chars, got %d", len(short))
	}
	if !strings.HasSuffix(short, "...") {
		t.Errorf("expected '...' suffix, got %q", short)
	}
}

func TestShortFromEmpty(t *testing.T) {
	msg := &Message{From: ""}
	if msg.ShortFrom() != "" {
		t.Errorf("expected empty string, got %q", msg.ShortFrom())
	}
}

func TestDateStringZero(t *testing.T) {
	msg := &Message{}
	if msg.DateString() != "unknown" {
		t.Errorf("zero date should return %q, got %q", "unknown", msg.DateString())
	}
}

func TestDateStringValid(t *testing.T) {
	msg := &Message{Date: time.Date(2025, 1, 7, 14, 30, 0, 0, time.UTC)}
	got := msg.DateString()
	expected := "07.01.2025 14:30"
	if got != expected {
		t.Errorf("DateString() = %q, want %q", got, expected)
	}
}

func TestIsReadVariants(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"RO", "RO", true},
		{"R only", "R", true},
		{"O only", "O", false},
		{"empty", "", false},
		{"N", "N", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &Message{Status: tc.status}
			if got := msg.IsRead(); got != tc.want {
				t.Errorf("IsRead() = %v, want %v", got, tc.want)
			}
		})
	}
}
