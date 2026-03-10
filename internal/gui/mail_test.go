package gui

import (
	"strings"
	"testing"

	"github.com/bssmnt/lazycron/internal/mail"
)

func TestCalculateMailColumnsWide(t *testing.T) {
	cols := calculateMailColumns(120)
	total := cols.statusWidth + cols.fromWidth + cols.subjectWidth + cols.dateWidth
	if total > 120 {
		t.Errorf("column total %d exceeds width 120", total)
	}
	if cols.dateWidth == 0 {
		t.Error("expected date column at width 120")
	}
}

func TestCalculateMailColumnsNarrow(t *testing.T) {
	cols := calculateMailColumns(40)
	if cols.dateWidth != 0 {
		t.Errorf("expected dateWidth 0 for narrow terminal, got %d", cols.dateWidth)
	}
}

func TestCalculateMailColumnsMinWidths(t *testing.T) {
	cols := calculateMailColumns(30)
	if cols.fromWidth < 10 {
		t.Errorf("fromWidth %d below minimum 10", cols.fromWidth)
	}
	if cols.subjectWidth < 15 {
		t.Errorf("subjectWidth %d below minimum 15", cols.subjectWidth)
	}
}

func TestFormatMailRowWithDate(t *testing.T) {
	cols := mailColumnWidths{statusWidth: 2, fromWidth: 15, subjectWidth: 20, dateWidth: 16}
	row := formatMailRow(cols, "●", "root@host", "Test subject", "07.01.2025 14:30")

	if !strings.Contains(row, "root@host") {
		t.Error("expected from field in output")
	}
	if !strings.Contains(row, "Test subject") {
		t.Error("expected subject field in output")
	}
	if !strings.Contains(row, "07.01.2025 14:30") {
		t.Error("expected date field in output")
	}
}

func TestFormatMailRowWithoutDate(t *testing.T) {
	cols := mailColumnWidths{statusWidth: 2, fromWidth: 15, subjectWidth: 20, dateWidth: 0}
	row := formatMailRow(cols, "○", "root@host", "Test subject", "07.01.2025 14:30")

	if strings.Contains(row, "07.01.2025") {
		t.Error("date should not appear when dateWidth is 0")
	}
}

func TestFormatMailRowTruncation(t *testing.T) {
	cols := mailColumnWidths{statusWidth: 2, fromWidth: 10, subjectWidth: 10, dateWidth: 0}
	row := formatMailRow(cols, "●", "a-very-long-from-address", "a very long subject line", "")
	stripped := stripANSI(row)

	if strings.Contains(stripped, "a-very-long-from-address") {
		t.Error("from should be truncated")
	}
}

func TestMailCountNilMessages(t *testing.T) {
	gui := &Gui{mailMessages: nil}
	if gui.mailCount() != "" {
		t.Errorf("expected empty string for nil messages, got %q", gui.mailCount())
	}
}

func TestMailCountEmpty(t *testing.T) {
	gui := &Gui{mailMessages: []*mail.Message{}}
	if gui.mailCount() != "" {
		t.Errorf("expected empty string for 0 messages, got %q", gui.mailCount())
	}
}

func TestMailCountWithUnread(t *testing.T) {
	gui := &Gui{
		mailMessages: []*mail.Message{
			{Status: ""},   // unread
			{Status: "RO"}, // read
			{Status: "O"},  // unread (old but not read)
		},
	}
	got := gui.mailCount()
	if !strings.Contains(got, "2/3") {
		t.Errorf("expected '2/3' in mailCount, got %q", got)
	}
}

func TestMailCountAllRead(t *testing.T) {
	gui := &Gui{
		mailMessages: []*mail.Message{
			{Status: "RO"},
			{Status: "R"},
		},
	}
	got := gui.mailCount()
	// When all read, should just show total
	if strings.Contains(got, "/") {
		t.Errorf("expected no unread count, got %q", got)
	}
	if !strings.Contains(got, "2") {
		t.Errorf("expected total 2, got %q", got)
	}
}

func TestMailSourceLabelLocal(t *testing.T) {
	gui := &Gui{mailActiveSource: mailSourceLocal}
	if gui.mailSourceLabel() != "Local" {
		t.Errorf("expected 'Local', got %q", gui.mailSourceLabel())
	}
}

func TestMailSourceLabelRemoteNoClient(t *testing.T) {
	gui := &Gui{mailActiveSource: mailSourceRemote, activeClient: nil}
	if gui.mailSourceLabel() != "Remote" {
		t.Errorf("expected 'Remote', got %q", gui.mailSourceLabel())
	}
}
