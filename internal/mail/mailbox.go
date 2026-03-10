package mail

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

// Mailbox represents a local mbox-format mail file.
type Mailbox struct {
	Path string
}

// DefaultMailbox returns the mailbox for the current user.
func DefaultMailbox() (*Mailbox, error) {
	path, err := DefaultMailPath()
	if err != nil {
		return nil, err
	}
	return &Mailbox{Path: path}, nil
}

// NewMailbox creates a Mailbox for a given path (useful for testing).
func NewMailbox(path string) *Mailbox {
	return &Mailbox{Path: path}
}

// DefaultMailPath returns the default mail spool path for the current user.
func DefaultMailPath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	var mailDir string
	switch runtime.GOOS {
	case "darwin":
		mailDir = "/var/mail"
	default:
		mailDir = "/var/mail"
	}

	return filepath.Join(mailDir, u.Username), nil
}

// Read reads and parses all messages from the mailbox file.
func (mb *Mailbox) Read() ([]*Message, error) {
	data, err := os.ReadFile(mb.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No mail
		}
		return nil, fmt.Errorf("failed to read mailbox %s: %w", mb.Path, err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	return Parse(string(data))
}

// Count returns the number of messages without fully parsing bodies.
func (mb *Mailbox) Count() (int, error) {
	messages, err := mb.Read()
	if err != nil {
		return 0, err
	}
	return len(messages), nil
}

// UnreadCount returns the number of unread messages.
func (mb *Mailbox) UnreadCount() (int, error) {
	messages, err := mb.Read()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, msg := range messages {
		if !msg.IsRead() {
			count++
		}
	}
	return count, nil
}

// Delete removes a single message by index and rewrites the mailbox file.
func (mb *Mailbox) Delete(index int) error {
	messages, err := mb.Read()
	if err != nil {
		return err
	}

	if index < 0 || index >= len(messages) {
		return fmt.Errorf("message index %d out of range (have %d messages)", index, len(messages))
	}

	// Rebuild the file without the deleted message
	messages = append(messages[:index], messages[index+1:]...)
	return mb.writeMessages(messages)
}

// DeleteAll truncates the mailbox file (removes all messages).
func (mb *Mailbox) DeleteAll() error {
	return os.Truncate(mb.Path, 0)
}

// MarkRead updates the Status header of a message to "RO" (read, old).
func (mb *Mailbox) MarkRead(index int) error {
	messages, err := mb.Read()
	if err != nil {
		return err
	}

	if index < 0 || index >= len(messages) {
		return fmt.Errorf("message index %d out of range (have %d messages)", index, len(messages))
	}

	if messages[index].IsRead() {
		return nil // Already read
	}

	// Update the Status in the raw text — capture old status before overwriting.
	oldStatus := messages[index].Status
	raw := messages[index].Raw

	if oldStatus != "" && strings.Contains(raw, "Status: "+oldStatus) {
		raw = strings.Replace(raw, "Status: "+oldStatus, "Status: RO", 1)
	} else {
		// Insert Status header before the blank line separating headers from body
		idx := strings.Index(raw, "\n")
		if idx > 0 {
			headerEnd := strings.Index(raw[idx+1:], "\n\n")
			if headerEnd > 0 {
				insertAt := idx + 1 + headerEnd
				raw = raw[:insertAt] + "\nStatus: RO" + raw[insertAt:]
			}
		}
	}

	messages[index].Status = "RO"
	messages[index].Raw = raw

	return mb.writeMessages(messages)
}

// writeMessages serialises messages back to mbox format and writes the file.
func (mb *Mailbox) writeMessages(messages []*Message) error {
	var sb strings.Builder
	for _, msg := range messages {
		sb.WriteString(msg.Raw)
		// Ensure each message ends with a newline
		if !strings.HasSuffix(msg.Raw, "\n") {
			sb.WriteString("\n")
		}
	}

	if err := os.WriteFile(mb.Path, []byte(sb.String()), 0600); err != nil {
		return fmt.Errorf("failed to write mailbox %s: %w", mb.Path, err)
	}
	return nil
}
