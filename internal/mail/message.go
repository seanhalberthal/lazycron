package mail

import (
	"time"
)

// Message represents a single parsed mail message from an mbox file.
type Message struct {
	From    string
	To      string
	Subject string
	Date    time.Time
	Body    string
	Status  string // mbox Status header: "R" = read, "O" = old, "" = new
	Raw     string // original raw text including headers
}

// IsRead returns whether the message has been read.
func (m *Message) IsRead() bool {
	return m.Status == "RO" || m.Status == "R"
}

// DateString returns the date formatted for display.
func (m *Message) DateString() string {
	if m.Date.IsZero() {
		return "unknown"
	}
	return m.Date.Format("02.01.2006 15:04")
}

// ShortFrom returns just the email address or name, truncated for display.
func (m *Message) ShortFrom() string {
	from := m.From
	if len(from) > 30 {
		from = from[:27] + "..."
	}
	return from
}
