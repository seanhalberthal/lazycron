package gui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/awesome-gocui/gocui"
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
	// When all read, badge should be hidden
	if got != "" {
		t.Errorf("expected empty string when all read, got %q", got)
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

func testMailMessages(n int) []*mail.Message {
	msgs := make([]*mail.Message, n)
	for i := range n {
		msgs[i] = &mail.Message{
			From:    fmt.Sprintf("user%d@example.com", i),
			Subject: fmt.Sprintf("Subject %d", i),
			Index:   i,
		}
	}
	return msgs
}

func TestMailCursorFirst(t *testing.T) {
	g, err := gocui.NewGui(gocui.OutputSimulator, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}

	gui := &Gui{
		g:            g,
		mailMessages: testMailMessages(5),
		mailSelected: 3,
	}

	v, err := g.SetView(mailListView, 0, 0, 80, 24, 0)
	if err != nil && err != gocui.ErrUnknownView {
		t.Fatalf("failed to create view: %v", err)
	}
	// Write content so the cursor has lines to land on.
	for i := range 7 { // header + separator + 5 messages
		fmt.Fprintf(v, "line %d\n", i)
	}
	_ = v.SetCursor(0, 5)

	if err := gui.mailCursorFirst(nil, v); err != nil {
		t.Fatalf("mailCursorFirst failed: %v", err)
	}

	if gui.mailSelected != 0 {
		t.Errorf("expected mailSelected 0, got %d", gui.mailSelected)
	}
	_, cy := v.Cursor()
	if cy != 2 { // header + separator
		t.Errorf("expected cursor Y = 2, got %d", cy)
	}
	_, oy := v.Origin()
	if oy != 0 {
		t.Errorf("expected origin Y = 0, got %d", oy)
	}
}

func TestMailCursorFirstEmpty(t *testing.T) {
	gui := &Gui{mailMessages: nil}
	// Should not panic with nil view arg when messages are empty.
	if err := gui.mailCursorFirst(nil, nil); err != nil {
		t.Fatalf("mailCursorFirst with empty messages failed: %v", err)
	}
}

func TestMailCursorLast(t *testing.T) {
	g, err := gocui.NewGui(gocui.OutputSimulator, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}

	msgs := testMailMessages(5)
	gui := &Gui{
		g:            g,
		mailMessages: msgs,
		mailSelected: 0,
	}

	v, err := g.SetView(mailListView, 0, 0, 80, 24, 0)
	if err != nil && err != gocui.ErrUnknownView {
		t.Fatalf("failed to create view: %v", err)
	}

	if err := gui.mailCursorLast(nil, v); err != nil {
		t.Fatalf("mailCursorLast failed: %v", err)
	}

	if gui.mailSelected != len(msgs)-1 {
		t.Errorf("expected mailSelected %d, got %d", len(msgs)-1, gui.mailSelected)
	}
}

func TestMailCursorLastEmpty(t *testing.T) {
	gui := &Gui{mailMessages: nil}
	if err := gui.mailCursorLast(nil, nil); err != nil {
		t.Fatalf("mailCursorLast with empty messages failed: %v", err)
	}
}

func TestScrollMailTop(t *testing.T) {
	g, err := gocui.NewGui(gocui.OutputSimulator, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}

	gui := &Gui{g: g}

	v, err := g.SetView(mailOverlayView, 0, 0, 60, 10, 0)
	if err != nil && err != gocui.ErrUnknownView {
		t.Fatalf("failed to create view: %v", err)
	}

	// Scroll down first.
	_ = v.SetOrigin(0, 5)

	if err := gui.scrollMailTop(nil, v); err != nil {
		t.Fatalf("scrollMailTop failed: %v", err)
	}

	_, oy := v.Origin()
	if oy != 0 {
		t.Errorf("expected origin Y = 0 after scrollMailTop, got %d", oy)
	}
}

func TestScrollMailBottom(t *testing.T) {
	g, err := gocui.NewGui(gocui.OutputSimulator, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}

	gui := &Gui{g: g}

	v, err := g.SetView(mailOverlayView, 0, 0, 60, 10, 0)
	if err != nil && err != gocui.ErrUnknownView {
		t.Fatalf("failed to create view: %v", err)
	}

	// Write enough lines to exceed the view height.
	for i := range 30 {
		fmt.Fprintf(v, "Line %d\n", i)
	}

	if err := gui.scrollMailBottom(nil, v); err != nil {
		t.Fatalf("scrollMailBottom failed: %v", err)
	}

	_, oy := v.Origin()
	if oy <= 0 {
		t.Errorf("expected origin Y > 0 after scrollMailBottom, got %d", oy)
	}
}

func TestScrollMailBottomShortContent(t *testing.T) {
	g, err := gocui.NewGui(gocui.OutputSimulator, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}

	gui := &Gui{g: g}

	v, err := g.SetView(mailOverlayView, 0, 0, 60, 20, 0)
	if err != nil && err != gocui.ErrUnknownView {
		t.Fatalf("failed to create view: %v", err)
	}

	// Write fewer lines than the view height.
	fmt.Fprintln(v, "Short content")

	if err := gui.scrollMailBottom(nil, v); err != nil {
		t.Fatalf("scrollMailBottom failed: %v", err)
	}

	// Origin should stay at 0 when content fits in view.
	_, oy := v.Origin()
	if oy != 0 {
		t.Errorf("expected origin Y = 0 for short content, got %d", oy)
	}
}
