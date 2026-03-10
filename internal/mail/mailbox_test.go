package mail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestMailbox(t *testing.T, fixture string) (*Mailbox, string) {
	t.Helper()
	tmpDir := t.TempDir()
	mailPath := filepath.Join(tmpDir, "testmail")

	if fixture != "" {
		data, err := os.ReadFile(testdataPath(fixture))
		if err != nil {
			t.Fatalf("failed to load fixture %s: %v", fixture, err)
		}
		if err := os.WriteFile(mailPath, data, 0600); err != nil {
			t.Fatalf("failed to write test mailbox: %v", err)
		}
	}

	return NewMailbox(mailPath), tmpDir
}

func TestMailboxReadEmpty(t *testing.T) {
	mb, _ := setupTestMailbox(t, "")
	// File doesn't exist — should return nil, nil
	messages, err := mb.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestMailboxReadMultiple(t *testing.T) {
	mb, _ := setupTestMailbox(t, "multiple.mbox")
	messages, err := mb.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}
}

func TestMailboxCount(t *testing.T) {
	mb, _ := setupTestMailbox(t, "multiple.mbox")
	count, err := mb.Count()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestMailboxUnreadCount(t *testing.T) {
	mb, _ := setupTestMailbox(t, "multiple.mbox")
	count, err := mb.UnreadCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// multiple.mbox: 1 read (RO), 1 unread (no status), 1 old (O) = 2 unread
	if count != 2 {
		t.Errorf("expected unread count 2, got %d", count)
	}
}

func TestMailboxDeleteMessage(t *testing.T) {
	mb, _ := setupTestMailbox(t, "multiple.mbox")

	// Delete the second message (index 1)
	if err := mb.Delete(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages, err := mb.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages after delete, got %d", len(messages))
	}

	// First message should still be the backup one
	if messages[0].Subject != "Cron <user@server> /usr/bin/backup.sh" {
		t.Errorf("message 0 subject = %q, want backup", messages[0].Subject)
	}
	// Second should now be the healthcheck one (was third)
	if messages[1].Subject != "Cron <user@server> /usr/bin/healthcheck.sh" {
		t.Errorf("message 1 subject = %q, want healthcheck", messages[1].Subject)
	}
}

func TestMailboxDeleteOutOfRange(t *testing.T) {
	mb, _ := setupTestMailbox(t, "single.mbox")
	err := mb.Delete(5)
	if err == nil {
		t.Error("expected error for out-of-range delete")
	}
}

func TestMailboxDeleteAll(t *testing.T) {
	mb, _ := setupTestMailbox(t, "multiple.mbox")

	if err := mb.DeleteAll(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages, err := mb.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages after delete all, got %d", len(messages))
	}
}

func TestMailboxDeleteNonExistent(t *testing.T) {
	mb, _ := setupTestMailbox(t, "")
	// DeleteAll on non-existent file — should not panic
	_ = mb.DeleteAll() // may error on non-existent file; we just check it doesn't panic
}

func TestMailboxMarkReadAlreadyRead(t *testing.T) {
	mb, _ := setupTestMailbox(t, "single.mbox")
	// single.mbox has Status: RO — already read
	if err := mb.MarkRead(0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	messages, err := mb.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !messages[0].IsRead() {
		t.Error("message should still be read")
	}
}

func TestMailboxMarkReadUnreadNoStatusHeader(t *testing.T) {
	mb, _ := setupTestMailbox(t, "multiple.mbox")
	// multiple.mbox message[1] has no Status header
	if err := mb.MarkRead(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	messages, err := mb.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !messages[1].IsRead() {
		t.Error("expected message to be marked read")
	}
	if messages[1].Status != "RO" {
		t.Errorf("expected status RO, got %q", messages[1].Status)
	}
}

func TestMailboxMarkReadExistingStatusO(t *testing.T) {
	mb, _ := setupTestMailbox(t, "status-o.mbox")
	// status-o.mbox has Status: O — should be replaced, not duplicated
	if err := mb.MarkRead(0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	messages, err := mb.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !messages[0].IsRead() {
		t.Error("expected message to be marked read")
	}
	// Verify no duplicate Status header
	count := strings.Count(messages[0].Raw, "Status:")
	if count != 1 {
		t.Errorf("expected exactly 1 Status header, found %d", count)
	}
}

func TestMailboxMarkReadOutOfRange(t *testing.T) {
	mb, _ := setupTestMailbox(t, "single.mbox")
	err := mb.MarkRead(99)
	if err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestMailboxMarkReadNegativeIndex(t *testing.T) {
	mb, _ := setupTestMailbox(t, "single.mbox")
	err := mb.MarkRead(-1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestMailboxReadPermissionDenied(t *testing.T) {
	mb, _ := setupTestMailbox(t, "single.mbox")
	if err := os.Chmod(mb.Path, 0000); err != nil {
		t.Skip("cannot change file permissions")
	}
	t.Cleanup(func() { _ = os.Chmod(mb.Path, 0600) })

	_, err := mb.Read()
	if err == nil {
		t.Error("expected error for unreadable file")
	}
}

func TestMailboxDeleteReadError(t *testing.T) {
	mb, _ := setupTestMailbox(t, "single.mbox")
	if err := os.Chmod(mb.Path, 0000); err != nil {
		t.Skip("cannot change file permissions")
	}
	t.Cleanup(func() { _ = os.Chmod(mb.Path, 0600) })

	err := mb.Delete(0)
	if err == nil {
		t.Error("expected error when file is unreadable")
	}
}
