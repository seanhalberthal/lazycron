package gui

import (
	"fmt"
	"strings"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/gui/style"
	"github.com/bssmnt/lazycron/internal/mail"
)

// Mail view name constants.
const (
	mailListView    = "mailList"
	mailOverlayView = "mailOverlay"
)

// mailSource represents the origin of the currently displayed mail.
type mailSource int

const (
	mailSourceLocal  mailSource = iota
	mailSourceRemote            // remote server via SSH
)

// Mail table column widths.
type mailColumnWidths struct {
	statusWidth  int
	fromWidth    int
	subjectWidth int
	dateWidth    int
}

// loadMailbox reads the local mailbox and updates the message list.
func (gui *Gui) loadMailbox() error {
	mb, err := mail.DefaultMailbox()
	if err != nil {
		return fmt.Errorf("failed to get mailbox: %w", err)
	}

	gui.mailbox = mb
	messages, err := mb.Read()
	if err != nil {
		return fmt.Errorf("failed to read mailbox: %w", err)
	}

	// Reverse so most recent messages appear first.
	reverseMessages(messages)
	gui.mailMessages = messages
	if gui.mailSelected >= len(gui.mailMessages) {
		gui.mailSelected = max(0, len(gui.mailMessages)-1)
	}

	return nil
}

// mailSourceLabel returns the display label for the current mail source.
func (gui *Gui) mailSourceLabel() string {
	switch gui.mailActiveSource {
	case mailSourceRemote:
		if gui.activeClient != nil {
			return fmt.Sprintf("%s (remote)", gui.activeClient.ServerName())
		}
		return "Remote"
	default:
		return "Local"
	}
}

// createMailListView creates the full-width mail list view.
func (gui *Gui) createMailListView(g *gocui.Gui, maxX, maxY int) error {
	v, err := g.SetView(mailListView, 0, 2, maxX-1, maxY-2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		v.Frame = true
		v.Highlight = true
		v.SelBgColor = style.SelectedBgColour
		v.SelFgColor = style.SelectedFgColour
		if _, err := g.SetCurrentView(mailListView); err != nil {
			return err
		}
		gui.renderMailList()
	}

	v.Title = fmt.Sprintf(" Mailbox: %s ", gui.mailSourceLabel())
	v.FrameColor = style.ActiveBorderColour
	return nil
}

// renderMailList renders the mail message list.
func (gui *Gui) renderMailList() {
	v, err := gui.g.View(mailListView)
	if err != nil {
		return
	}
	v.Clear()
	_ = v.SetOrigin(0, 0)

	if len(gui.mailMessages) == 0 {
		fmt.Fprintln(v, "")
		fmt.Fprintln(v, style.Coloured(style.Dim, "  No mail messages. Press [r] to refresh."))
		return
	}

	maxX, _ := gui.g.Size()
	cols := calculateMailColumns(maxX - 2)

	// Header line
	header := formatMailRow(cols, " ", "From", "Subject", "Date")
	fmt.Fprintln(v, style.Coloured(style.Bold, header))

	// Separator line — dim horizontal rule under header
	sepWidth := maxX - 6
	if sepWidth < 10 {
		sepWidth = 10
	}
	fmt.Fprintln(v, " "+style.Coloured(style.Dim, strings.Repeat("─", sepWidth)))

	// Message rows
	for _, msg := range gui.mailMessages {
		readIndicator := style.Coloured(style.FgCyan, "●") // unread
		if msg.IsRead() {
			readIndicator = style.Coloured(style.Dim, "○") // read
		}

		row := formatMailRow(cols, readIndicator, msg.ShortFrom(), msg.Subject, msg.DateString())
		fmt.Fprintln(v, row)
	}

	// Sync cursor with logical selection — v.Clear() resets cursor to (0,0),
	// so we must restore it after every render.
	if len(gui.mailMessages) > 0 {
		_ = v.SetCursor(0, gui.mailSelected+2) // +2 for header + separator
	}
}

// calculateMailColumns computes column widths for the mail list.
func calculateMailColumns(totalWidth int) mailColumnWidths {
	statusW := 2
	dateW := 16 // "06.01.2025 09:00"

	overhead := 1 + 6 + 2 // leading space + separators + frame borders
	remaining := totalWidth - statusW - dateW - overhead

	if remaining < 30 {
		dateW = 0
		remaining = totalWidth - statusW - 1 - 2 - 2
	}

	// Split remaining between from and subject (30/70)
	fromW := max(10, remaining*30/100)
	subjectW := max(15, remaining-fromW)

	return mailColumnWidths{
		statusWidth:  statusW,
		fromWidth:    fromW,
		subjectWidth: subjectW,
		dateWidth:    dateW,
	}
}

// formatMailRow formats a single mail list row with aligned columns.
func formatMailRow(cols mailColumnWidths, status, from, subject, date string) string {
	row := fmt.Sprintf(" %s %-*s  %-*s",
		status,
		cols.fromWidth, truncate(from, cols.fromWidth),
		cols.subjectWidth, truncate(subject, cols.subjectWidth),
	)

	if cols.dateWidth > 0 {
		row += fmt.Sprintf("  %-*s", cols.dateWidth, truncate(date, cols.dateWidth))
	}

	return row
}

// reverseMessages reverses a slice of messages in place.
func reverseMessages(msgs []*mail.Message) {
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}

// mailOriginalIndex converts a display index (newest-first) back to the
// original file index (oldest-first) needed by mailbox operations.
func (gui *Gui) mailOriginalIndex(displayIndex int) int {
	return gui.mailMessages[displayIndex].Index
}

// mailCursorDown moves the mail list cursor down.
func (gui *Gui) mailCursorDown(_ *gocui.Gui, v *gocui.View) error {
	if len(gui.mailMessages) == 0 || gui.mailSelected >= len(gui.mailMessages)-1 {
		return nil
	}

	cx, cy := v.Cursor()
	gui.mailSelected++
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	return nil
}

// mailCursorUp moves the mail list cursor up.
func (gui *Gui) mailCursorUp(_ *gocui.Gui, v *gocui.View) error {
	if len(gui.mailMessages) == 0 || gui.mailSelected <= 0 {
		return nil
	}

	cx, cy := v.Cursor()
	gui.mailSelected--
	if err := v.SetCursor(cx, cy-1); err != nil {
		ox, oy := v.Origin()
		if oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}

	return nil
}

// openMailDetail opens an overlay showing the full message body.
// Styled to match the detail overlay and create/edit modal patterns.
func (gui *Gui) openMailDetail(_ *gocui.Gui, _ *gocui.View) error {
	if len(gui.mailMessages) == 0 || gui.mailSelected >= len(gui.mailMessages) {
		return nil
	}

	msg := gui.mailMessages[gui.mailSelected]

	// Mark as read (local only)
	if gui.mailActiveSource == mailSourceLocal && gui.mailbox != nil && !msg.IsRead() {
		_ = gui.mailbox.MarkRead(gui.mailOriginalIndex(gui.mailSelected))
		msg.Status = "RO"
		gui.renderMailList()
	}

	// Count body lines to calculate height
	bodyLines := strings.Split(msg.Body, "\n")
	// Header (4 lines) + separator + blank + body + blank + hints + frame
	contentHeight := 4 + 1 + 1 + len(bodyLines) + 1 + 1 + 2
	maxX, maxY := gui.g.Size()
	width := min(maxX-6, maxX*85/100)
	height := min(contentHeight, maxY-4)
	if height < 14 {
		height = 14
	}

	x0 := maxX/2 - width/2
	y0 := maxY/2 - height/2

	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}

	v, err := gui.g.SetView(mailOverlayView, x0, y0, x0+width, y0+height, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}

	v.Title = fmt.Sprintf(" %s ", truncate(msg.Subject, width-4))
	v.Wrap = false
	v.Clear()

	// Available content width inside the frame, minus indentation.
	// Header values: 3 (indent) + 8 (label) + 2 (gap) = 13 chars overhead.
	valWidth := width - 2 - 13
	// Body lines: 3 (indent) chars overhead.
	bodyWidth := width - 2 - 3

	fmt.Fprintln(v)
	fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "From:   "), style.Coloured(style.Dim, truncate(msg.From, valWidth)))
	fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "To:     "), style.Coloured(style.Dim, truncate(msg.To, valWidth)))
	fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "Date:   "), style.Coloured(style.Dim, truncate(msg.DateString(), valWidth)))
	fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "Subject:"), style.Coloured(style.Dim, truncate(msg.Subject, valWidth)))
	fmt.Fprintln(v)

	// Body section — truncate each line to fit without wrapping
	for _, line := range bodyLines {
		fmt.Fprintf(v, "   %s\n", truncate(line, bodyWidth))
	}

	fmt.Fprintln(v)
	if gui.mailActiveSource == mailSourceLocal {
		fmt.Fprintln(v, style.Coloured(style.Dim, "   [Esc] close   [d] delete   [j/k] scroll"))
	} else {
		fmt.Fprintln(v, style.Coloured(style.Dim, "   [Esc] close   [j/k] scroll"))
	}

	if _, err := gui.g.SetCurrentView(mailOverlayView); err != nil {
		return err
	}

	// Keybindings for mail detail overlay
	if err := gui.g.SetKeybinding(mailOverlayView, gocui.KeyEsc, gocui.ModNone, gui.closeMailDetail); err != nil {
		return err
	}
	if err := gui.g.SetKeybinding(mailOverlayView, gocui.KeyEnter, gocui.ModNone, gui.closeMailDetail); err != nil {
		return err
	}
	if gui.mailActiveSource == mailSourceLocal {
		if err := gui.g.SetKeybinding(mailOverlayView, 'd', gocui.ModNone, gui.deleteMailFromOverlay); err != nil {
			return err
		}
	}
	// Allow scrolling within the overlay
	if err := gui.g.SetKeybinding(mailOverlayView, 'j', gocui.ModNone, gui.scrollMailDown); err != nil {
		return err
	}
	if err := gui.g.SetKeybinding(mailOverlayView, 'k', gocui.ModNone, gui.scrollMailUp); err != nil {
		return err
	}

	return nil
}

// scrollMailDown scrolls the mail overlay content down.
func (gui *Gui) scrollMailDown(_ *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	if err := v.SetOrigin(ox, oy+1); err != nil {
		return err
	}
	return nil
}

// scrollMailUp scrolls the mail overlay content up.
func (gui *Gui) scrollMailUp(_ *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	if oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}
	return nil
}

// closeMailDetail closes the mail message overlay.
func (gui *Gui) closeMailDetail(_ *gocui.Gui, _ *gocui.View) error {
	gui.g.DeleteView(mailOverlayView)
	gui.g.DeleteKeybindings(mailOverlayView)
	if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
		_ = err
	}
	return nil
}

// deleteMail deletes the selected mail message (local source only).
func (gui *Gui) deleteMail(_ *gocui.Gui, _ *gocui.View) error {
	if gui.mailActiveSource != mailSourceLocal {
		gui.setStatusMessage("Cannot delete remote mail")
		return nil
	}

	if gui.mailbox == nil || len(gui.mailMessages) == 0 || gui.mailSelected >= len(gui.mailMessages) {
		return nil
	}

	if err := gui.mailbox.Delete(gui.mailOriginalIndex(gui.mailSelected)); err != nil {
		gui.setStatusMessage(fmt.Sprintf("Error deleting mail: %v", err))
		return nil
	}

	// Reload
	if err := gui.loadMailbox(); err != nil {
		gui.setStatusMessage(fmt.Sprintf("Error reloading mail: %v", err))
		return nil
	}

	gui.setStatusMessage("Message deleted")
	gui.refreshViews()
	return nil
}

// deleteMailFromOverlay deletes the currently viewed message and closes the overlay.
func (gui *Gui) deleteMailFromOverlay(g *gocui.Gui, v *gocui.View) error {
	if err := gui.closeMailDetail(g, v); err != nil {
		return err
	}
	return gui.deleteMail(g, v)
}

// deleteAllMail opens a confirmation modal to delete all mail messages.
func (gui *Gui) deleteAllMail(_ *gocui.Gui, _ *gocui.View) error {
	if gui.mailActiveSource != mailSourceLocal {
		gui.setStatusMessage("Cannot delete remote mail")
		return nil
	}

	if gui.mailbox == nil || len(gui.mailMessages) == 0 {
		gui.setStatusMessage("No mail to delete")
		return nil
	}

	maxX, maxY := gui.g.Size()
	width := 50
	height := 10

	x0 := maxX/2 - width/2
	y0 := maxY/2 - height/2
	x1 := x0 + width
	y1 := y0 + height

	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}

	const deleteAllModalView = "deleteAllMailModal"

	v, err := gui.g.SetView(deleteAllModalView, x0, y0, x1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}

	v.Title = " Delete All Mail "
	v.Clear()

	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "  Delete all %s?\n",
		style.Coloured(style.FgGreen+style.Bold, fmt.Sprintf("%d messages", len(gui.mailMessages))))
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, style.Coloured(style.Dim, "  This will truncate your local mailbox file."))
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, style.Coloured(style.Dim, "  [y] confirm   [n/Esc] cancel"))

	if _, err := gui.g.SetCurrentView(deleteAllModalView); err != nil {
		return err
	}

	// Confirm handler
	confirmHandler := func(_ *gocui.Gui, _ *gocui.View) error {
		gui.g.DeleteView(deleteAllModalView)
		gui.g.DeleteKeybindings(deleteAllModalView)

		if err := gui.mailbox.DeleteAll(); err != nil {
			gui.setStatusMessage(fmt.Sprintf("Error: %v", err))
		} else {
			gui.mailMessages = nil
			gui.mailSelected = 0
			gui.setStatusMessage("All mail deleted")
		}

		if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
			_ = err
		}
		gui.refreshViews()
		return nil
	}

	// Cancel handler
	cancelHandler := func(_ *gocui.Gui, _ *gocui.View) error {
		gui.g.DeleteView(deleteAllModalView)
		gui.g.DeleteKeybindings(deleteAllModalView)
		if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
			_ = err
		}
		return nil
	}

	if err := gui.g.SetKeybinding(deleteAllModalView, 'y', gocui.ModNone, confirmHandler); err != nil {
		return err
	}
	if err := gui.g.SetKeybinding(deleteAllModalView, 'n', gocui.ModNone, cancelHandler); err != nil {
		return err
	}
	if err := gui.g.SetKeybinding(deleteAllModalView, gocui.KeyEsc, gocui.ModNone, cancelHandler); err != nil {
		return err
	}

	return nil
}

// switchMailSource cycles the mail source between local and remote.
func (gui *Gui) switchMailSource(_ *gocui.Gui, _ *gocui.View) error {
	switch gui.mailActiveSource {
	case mailSourceLocal:
		// Switch to remote — only if connected
		if gui.activeClient == nil {
			gui.setStatusMessage("No server connected — connect via Servers tab")
			return nil
		}
		gui.mailActiveSource = mailSourceRemote
		gui.mailSelected = 0
		gui.loadRemoteMail()
	case mailSourceRemote:
		// Switch back to local
		gui.mailActiveSource = mailSourceLocal
		gui.mailSelected = 0
		_ = gui.loadMailbox()
	}

	gui.refreshViews()
	return nil
}

// refreshMail reloads the current mail source and updates views.
func (gui *Gui) refreshMail(_ *gocui.Gui, _ *gocui.View) error {
	switch gui.mailActiveSource {
	case mailSourceLocal:
		if err := gui.loadMailbox(); err != nil {
			gui.setStatusMessage(fmt.Sprintf("Error: %v", err))
			return nil
		}
	case mailSourceRemote:
		gui.loadRemoteMail()
	}

	gui.refreshViews()
	gui.setStatusMessage(fmt.Sprintf("Mail refreshed — %d messages", len(gui.mailMessages)))
	return nil
}

// loadRemoteMail fetches mail from the connected remote server via SSH.
func (gui *Gui) loadRemoteMail() {
	if gui.activeClient == nil {
		gui.mailMessages = nil
		gui.setStatusMessage("No server connected")
		return
	}

	raw, err := gui.activeClient.ReadMail()
	if err != nil {
		gui.mailMessages = nil
		gui.setStatusMessage(fmt.Sprintf("Error reading remote mail: %v", err))
		return
	}

	messages, err := mail.Parse(raw)
	if err != nil {
		gui.mailMessages = nil
		gui.setStatusMessage(fmt.Sprintf("Error parsing remote mail: %v", err))
		return
	}

	// Reverse so most recent messages appear first.
	reverseMessages(messages)
	gui.mailMessages = messages
	if gui.mailSelected >= len(gui.mailMessages) {
		gui.mailSelected = max(0, len(gui.mailMessages)-1)
	}
}

// mailCount returns a formatted mail count string for the status bar.
func (gui *Gui) mailCount() string {
	if gui.mailMessages == nil {
		return ""
	}

	total := len(gui.mailMessages)
	if total == 0 {
		return ""
	}

	unread := 0
	for _, msg := range gui.mailMessages {
		if !msg.IsRead() {
			unread++
		}
	}

	if unread == 0 {
		return ""
	}
	return fmt.Sprintf("  ✉ %d/%d", unread, total)
}
