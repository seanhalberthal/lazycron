package gui

import (
	"fmt"
	"strings"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/gui/style"
	"github.com/bssmnt/lazycron/internal/types"
)

// Panel split ratio for left/right panels.
const jobsPanelRatio = 0.3

// columnWidths holds the calculated widths for each table column.
type columnWidths struct {
	idWidth   int
	exprWidth int
	cmdWidth  int
	lastWidth int
	nextWidth int
	statWidth int
}

// createStatusView creates the top status bar (2 rows: version + tabs).
func (gui *Gui) createStatusView(g *gocui.Gui, maxX, _ int) error {
	v, err := g.SetView(statusView, -1, -1, maxX, 2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		v.Frame = false
	}
	gui.renderStatus()
	return nil
}

// createDetailView creates the right detail panel.
func (gui *Gui) createDetailView(g *gocui.Gui, maxX, maxY int) error {
	splitX := max(int(float64(maxX)*jobsPanelRatio), 20)

	v, err := g.SetView(detailView, splitX, 2, maxX-1, maxY-2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		v.Title = " Detail "
		v.Wrap = true
		gui.renderDetail()
	}

	// Update border colour based on focus
	if gui.currentPanel() == detailView {
		v.FrameColor = style.ActiveBorderColour
	} else {
		v.FrameColor = style.InactiveBorderColour
	}

	return nil
}

// createHintsView creates the bottom keybinding hints bar.
func (gui *Gui) createHintsView(g *gocui.Gui, maxX, maxY int) error {
	v, err := g.SetView(hintsView, -1, maxY-2, maxX, maxY, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		v.Frame = false
		gui.renderHints()
	}
	return nil
}

// createServersView creates the left server list panel.
func (gui *Gui) createServersView(g *gocui.Gui, maxX, maxY int) error {
	splitX := max(int(float64(maxX)*jobsPanelRatio), 20)

	v, err := g.SetView(serversView, 0, 2, splitX, maxY-2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		v.Title = " Servers "
		v.Highlight = true
		v.SelBgColor = style.SelectedBgColour
		v.SelFgColor = style.SelectedFgColour
		if _, setErr := g.SetCurrentView(serversView); setErr != nil {
			return setErr
		}
		gui.renderServerList()
	}

	// Update border colour based on focus
	if gui.currentPanel() == serversView {
		v.FrameColor = style.ActiveBorderColour
	} else {
		v.FrameColor = style.InactiveBorderColour
	}

	return nil
}

// createTableView creates the full-width table view for the Local tab.
func (gui *Gui) createTableView(g *gocui.Gui, maxX, maxY int) error {
	v, err := g.SetView(tableView, 0, 2, maxX-1, maxY-2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		v.Frame = true
		v.Title = " Jobs "
		v.Highlight = true
		v.SelBgColor = style.SelectedBgColour
		v.SelFgColor = style.SelectedFgColour
		if _, err := g.SetCurrentView(tableView); err != nil {
			return err
		}
		gui.renderTable()
	}

	v.FrameColor = style.ActiveBorderColour
	return nil
}

// renderTable renders the full-width table of cron jobs.
func (gui *Gui) renderTable() {
	v, err := gui.g.View(tableView)
	if err != nil {
		return
	}
	v.Clear()

	if len(gui.jobs) == 0 {
		fmt.Fprintln(v, "")
		fmt.Fprintln(v, style.Coloured(style.Dim, "  No cron jobs found. Press [c] to create one."))
		return
	}

	maxX, _ := gui.g.Size()
	cols := calculateColumns(maxX - 2)

	// Header line
	header := formatTableRow(cols, "ID", "Expression", "Command", "Last Run", "Next Run", "Status")
	fmt.Fprintln(v, style.Coloured(style.Bold, header))

	// Separator line — dim horizontal rule under header
	sepWidth := maxX - 6
	if sepWidth < 10 {
		sepWidth = 10
	}
	fmt.Fprintln(v, " "+style.Coloured(style.Dim, strings.Repeat("─", sepWidth)))

	// Job rows
	for _, job := range gui.jobs {
		status := style.Coloured(style.FgGreen, "Active")
		if !job.Enabled {
			status = style.Coloured(style.FgRed, "Paused")
		}

		lastRun := ""
		if t, err := job.PrevRun(); err == nil {
			lastRun = t.Format("02.01.2006 at 15:04")
		}

		nextRun := ""
		if t, err := job.NextRun(); err == nil {
			nextRun = t.Format("02.01.2006 at 15:04")
		}

		row := formatTableRow(cols,
			job.DisplayName(),
			job.Expression,
			job.Command,
			lastRun,
			nextRun,
			status,
		)
		fmt.Fprintln(v, row)
	}

	// Sync cursor with logical selection — v.Clear() resets cursor to (0,0),
	// so we must restore it after every render.
	if len(gui.jobs) > 0 {
		_ = v.SetCursor(0, gui.selected+2) // +2 for header + separator
	}
}

// calculateColumns computes column widths based on terminal width.
func calculateColumns(totalWidth int) columnWidths {
	// Fixed column widths (content only, no padding)
	exprW := 11 // "0 11 * * *"
	lastW := 19 // "09.03.2026 at 11:00"
	nextW := 19 // "09.03.2026 at 11:00"
	statW := 6  // "Active"

	// Total overhead: 1 leading space + 5×2 separators between 6 cols + 2 frame borders
	overhead := 1 + 10 + 2
	fixed := exprW + lastW + nextW + statW
	remaining := totalWidth - fixed - overhead

	if remaining < 40 {
		// Very narrow: hide Last Run and Next Run
		lastW = 0
		nextW = 0
		remaining = totalWidth - exprW - statW - 1 - 6 - 2
	}

	// Split remaining between ID and Command (30/70 ratio)
	idW := max(12, remaining*30/100)
	cmdW := max(15, remaining-idW)

	return columnWidths{
		idWidth: idW, exprWidth: exprW, cmdWidth: cmdW,
		lastWidth: lastW, nextWidth: nextW, statWidth: statW,
	}
}

// formatTableRow formats a single table row with aligned columns.
func formatTableRow(cols columnWidths, id, expr, cmd, last, next, status string) string {
	row := fmt.Sprintf(" %-*s  %-*s  %-*s",
		cols.idWidth, truncate(id, cols.idWidth),
		cols.exprWidth, truncate(expr, cols.exprWidth),
		cols.cmdWidth, truncate(cmd, cols.cmdWidth),
	)

	if cols.lastWidth > 0 {
		row += fmt.Sprintf("  %-*s", cols.lastWidth, truncate(last, cols.lastWidth))
	}
	if cols.nextWidth > 0 {
		row += fmt.Sprintf("  %-*s", cols.nextWidth, truncate(next, cols.nextWidth))
	}

	row += fmt.Sprintf("  %s", status)
	return row
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// renderStatus writes the status bar content (2-line: version + tabs).
func (gui *Gui) renderStatus() {
	v, err := gui.g.View(statusView)
	if err != nil {
		return
	}
	v.Clear()

	version := style.Coloured(style.FgGreen, fmt.Sprintf("lazycron %s", types.Version))

	var localTab, serversTab string
	if gui.activeTab == tabLocal {
		localTab = style.Coloured(style.FgCyan+style.Bold, "Local")
		serversTab = style.Coloured(style.Dim, "Servers")
	} else {
		localTab = style.Coloured(style.Dim, "Local")
		serversTab = style.Coloured(style.FgCyan+style.Bold, "Servers")
	}

	jobCount := ""
	switch gui.activeTab {
	case tabLocal:
		jobCount = fmt.Sprintf("  %d jobs", len(gui.jobs))
	case tabServers:
		jobCount = fmt.Sprintf("  %d servers", len(gui.serversConfig.Servers))
		if gui.activeClient != nil {
			jobCount += fmt.Sprintf("  Connected: %s", gui.activeClient.ServerName())
		}
	}

	fmt.Fprintf(v, " %s\n %s  %s%s", version, localTab, serversTab, jobCount)
}

// renderServerList writes the server list content.
func (gui *Gui) renderServerList() {
	v, err := gui.g.View(serversView)
	if err != nil {
		return
	}
	v.Clear()

	if len(gui.serversConfig.Servers) == 0 {
		fmt.Fprintln(v, style.Coloured(style.Dim, "  No servers configured"))
		fmt.Fprintln(v, style.Coloured(style.Dim, "  Press [a] to add one"))
		return
	}

	for _, server := range gui.serversConfig.Servers {
		connIndicator := style.Coloured(style.Dim, "○")
		if gui.activeClient != nil && gui.activeClient.ServerName() == server.Name {
			connIndicator = style.Coloured(style.FgGreen, "●")
		}
		fmt.Fprintf(v, " %s %s\n", connIndicator, server.Name)
	}

	// Sync cursor with logical selection — v.Clear() resets cursor to (0,0).
	if len(gui.serversConfig.Servers) > 0 {
		_ = v.SetCursor(0, gui.serverSelected)
	}
}

// renderDetail writes the detail panel content for the selected item.
func (gui *Gui) renderDetail() {
	v, err := gui.g.View(detailView)
	if err != nil {
		return
	}
	v.Clear()

	switch gui.activeTab {
	case tabLocal:
		gui.renderJobDetail(v)
	case tabServers:
		gui.renderServerDetail(v)
	}
}

// renderJobDetail shows detail for the selected local job.
func (gui *Gui) renderJobDetail(v *gocui.View) {
	if len(gui.jobs) == 0 || gui.selected >= len(gui.jobs) {
		fmt.Fprintln(v, "")
		fmt.Fprintln(v, style.Coloured(style.Dim, "  No job selected"))
		return
	}

	job := gui.jobs[gui.selected]

	status := style.Coloured(style.FgGreen, "Enabled")
	if !job.Enabled {
		status = style.Coloured(style.FgRed, "Disabled")
	}

	fmt.Fprintln(v)
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Name:      "), job.DisplayName())
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Expression:"), job.Expression)
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Schedule:  "), job.Describe())
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Command:   "), job.Command)
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Status:    "), status)

	if nextRun, err := job.NextRun(); err == nil {
		fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Next run:  "), style.Coloured(style.Dim, nextRun.Format("2006-01-02 15:04")))
	}

	if prevRun, err := job.PrevRun(); err == nil {
		fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Prev run:  "), style.Coloured(style.Dim, prevRun.Format("2006-01-02 15:04")))
	}
}

// renderServerDetail shows detail for the selected server.
func (gui *Gui) renderServerDetail(v *gocui.View) {
	servers := gui.serversConfig.Servers
	if len(servers) == 0 || gui.serverSelected >= len(servers) {
		fmt.Fprintln(v, "")
		fmt.Fprintln(v, style.Coloured(style.Dim, "  No server selected"))
		return
	}

	server := servers[gui.serverSelected]

	connected := style.Coloured(style.Dim, "Disconnected")
	if gui.activeClient != nil && gui.activeClient.ServerName() == server.Name {
		connected = style.Coloured(style.FgGreen, "Connected")
	}

	fmt.Fprintln(v)
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Name:  "), server.Name)
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Host:  "), server.Host)
	fmt.Fprintf(v, "  %s  %d\n", style.Coloured(style.FgGreen, "Port:  "), server.Port)
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "User:  "), server.User)
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Auth:  "), server.AuthType)
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "Status:"), connected)

	// Remote jobs section
	if gui.activeClient != nil && gui.activeClient.ServerName() == server.Name && gui.remoteJobs != nil {
		fmt.Fprintln(v)
		fmt.Fprintf(v, "  %s %d\n", style.Coloured(style.FgGreen, "Remote jobs:"), len(gui.remoteJobs))
		fmt.Fprintln(v, style.Coloured(style.Dim, "  ────────────────"))
		for _, job := range gui.remoteJobs {
			indicator := style.EnabledIndicator
			if !job.Enabled {
				indicator = style.DisabledIndicator
			}
			fmt.Fprintf(v, "  %s %s  %s\n", indicator, job.DisplayName(), job.Expression)
		}
	}
}

// renderHints writes the keybinding hints bar.
func (gui *Gui) renderHints() {
	v, err := gui.g.View(hintsView)
	if err != nil {
		return
	}
	v.Clear()

	switch gui.activeTab {
	case tabLocal:
		fmt.Fprint(v, formatHints([]hintPair{
			{"c", "reate"}, {"e", "dit"}, {"D", "elete"}, {"p", "ause"},
			{"/", "search"}, {"Enter", "detail"}, {"h/l", "tab"}, {"?", "help"}, {"q", "uit"},
		}))
	case tabServers:
		fmt.Fprint(v, formatHints([]hintPair{
			{"a", "dd"}, {"c", "onnect"}, {"d", "isconnect"}, {"D", "elete"},
			{"h/l", "tab"}, {"?", "help"}, {"q", "uit"},
		}))
	}
}

// hintPair represents a key-description pair for the hints bar.
type hintPair struct {
	key  string
	desc string
}

// formatHints formats hint pairs with green keys and dim descriptions.
func formatHints(hints []hintPair) string {
	result := " "
	for i, h := range hints {
		if i > 0 {
			result += " "
		}
		result += style.Coloured(style.FgGreen, "["+h.key+"]")
		result += style.Coloured(style.Dim, h.desc)
	}
	return result
}
