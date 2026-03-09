package gui

import (
	"fmt"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/cron"
	"github.com/bssmnt/lazycron/internal/gui/style"
	"github.com/bssmnt/lazycron/internal/ssh"
)

// quit exits the application.
func (gui *Gui) quit(_ *gocui.Gui, _ *gocui.View) error {
	return gocui.ErrQuit
}

// refresh reloads the crontab and updates all views.
func (gui *Gui) refresh(_ *gocui.Gui, _ *gocui.View) error {
	if err := gui.loadCrontab(); err != nil {
		return nil // Silently ignore — keep showing stale data
	}
	gui.refreshViews()
	return nil
}

// switchPanel cycles focus between the jobs and detail panels.
func (gui *Gui) switchPanel(_ *gocui.Gui, _ *gocui.View) error {
	gui.panelIdx = (gui.panelIdx + 1) % len(gui.panels)
	if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
		return err
	}
	return nil
}

// cursorDown moves the job list cursor down by one.
func (gui *Gui) cursorDown(_ *gocui.Gui, v *gocui.View) error {
	if len(gui.jobs) == 0 {
		return nil
	}

	cx, cy := v.Cursor()
	if gui.selected >= len(gui.jobs)-1 {
		return nil
	}

	gui.selected++
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	return nil
}

// cursorUp moves the job list cursor up by one.
func (gui *Gui) cursorUp(_ *gocui.Gui, v *gocui.View) error {
	if len(gui.jobs) == 0 || gui.selected <= 0 {
		return nil
	}

	cx, cy := v.Cursor()
	gui.selected--
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

// togglePause toggles the enabled/disabled state of the selected job.
func (gui *Gui) togglePause(_ *gocui.Gui, _ *gocui.View) error {
	if len(gui.jobs) == 0 || gui.selected >= len(gui.jobs) {
		return nil
	}

	job := gui.jobs[gui.selected]
	if job.Enabled {
		job.Disable()
	} else {
		job.Enable()
	}

	if err := cron.WriteCrontab(gui.crontab); err != nil {
		gui.setStatusMessage(fmt.Sprintf("Error: %v", err))
		// Revert the toggle
		if job.Enabled {
			job.Disable()
		} else {
			job.Enable()
		}
		return nil
	}

	gui.refreshViews()
	return nil
}

// toggleHelp shows or hides the help overlay.
func (gui *Gui) toggleHelp(_ *gocui.Gui, _ *gocui.View) error {
	const helpViewName = "help"

	// If help is already visible, close it
	if v, err := gui.g.View(helpViewName); err == nil {
		if v != nil {
			gui.g.DeleteView(helpViewName)
			if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
				return err
			}
			return nil
		}
	}

	maxX, maxY := gui.g.Size()
	x0 := maxX/2 - 30
	y0 := maxY/2 - 10
	x1 := maxX/2 + 30
	y1 := maxY/2 + 10

	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}

	v, err := gui.g.SetView(helpViewName, x0, y0, x1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}

	v.Title = " Help "
	v.Wrap = true
	v.Clear()

	fmt.Fprintln(v, "")
	fmt.Fprintln(v, style.Coloured(style.FgCyan+style.Bold, "  Navigation"))
	fmt.Fprintln(v, style.Coloured(style.Dim, "  ──────────"))
	fmt.Fprintf(v, "  %s    %s\n", style.Coloured(style.FgGreen, "j/k, ↑/↓"), style.Coloured(style.Dim, "Navigate list"))
	fmt.Fprintf(v, "  %s         %s\n", style.Coloured(style.FgGreen, "h/l"), style.Coloured(style.Dim, "Switch tab (Local/Servers/Mail)"))
	fmt.Fprintf(v, "  %s         %s\n", style.Coloured(style.FgGreen, "Tab"), style.Coloured(style.Dim, "Switch panel"))
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, style.Coloured(style.FgCyan+style.Bold, "  Local Tab"))
	fmt.Fprintln(v, style.Coloured(style.Dim, "  ─────────"))
	fmt.Fprintf(v, "  %s     %s\n", style.Coloured(style.FgGreen, "c/e/D/p"), style.Coloured(style.Dim, "Create/Edit/Delete/Pause"))
	fmt.Fprintf(v, "  %s           %s\n", style.Coloured(style.FgGreen, "/"), style.Coloured(style.Dim, "Search  n/N next/prev match"))
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, style.Coloured(style.FgCyan+style.Bold, "  Servers Tab"))
	fmt.Fprintln(v, style.Coloured(style.Dim, "  ───────────"))
	fmt.Fprintf(v, "  %s           %s\n", style.Coloured(style.FgGreen, "a"), style.Coloured(style.Dim, "Add server"))
	fmt.Fprintf(v, "  %s       %s\n", style.Coloured(style.FgGreen, "c/d/D"), style.Coloured(style.Dim, "Connect/Disconnect/Delete"))
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, style.Coloured(style.FgCyan+style.Bold, "  Mail Tab"))
	fmt.Fprintln(v, style.Coloured(style.Dim, "  ────────"))
	fmt.Fprintf(v, "  %s      %s\n", style.Coloured(style.FgGreen, "Enter"), style.Coloured(style.Dim, "Read message"))
	fmt.Fprintf(v, "  %s        %s\n", style.Coloured(style.FgGreen, "d/D"), style.Coloured(style.Dim, "Delete / Delete all (local only)"))
	fmt.Fprintf(v, "  %s          %s\n", style.Coloured(style.FgGreen, "S"), style.Coloured(style.Dim, "Switch source (Local / Remote)"))
	fmt.Fprintf(v, "  %s          %s\n", style.Coloured(style.FgGreen, "r"), style.Coloured(style.Dim, "Refresh mailbox"))
	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "  %s           %s     %s %s\n", style.Coloured(style.FgGreen, "r"), style.Coloured(style.Dim, "Refresh"), style.Coloured(style.FgGreen, "?"), style.Coloured(style.Dim, "Help"))
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.FgGreen, "q / Ctrl+C"), style.Coloured(style.Dim, "Quit"))

	if _, err := gui.g.SetCurrentView(helpViewName); err != nil {
		return err
	}

	// Add keybindings for closing help
	for _, key := range []any{'?', 'q', gocui.KeyEsc} {
		if err := gui.g.SetKeybinding(helpViewName, key, gocui.ModNone, gui.closeHelp); err != nil {
			return err
		}
	}

	return nil
}

// closeHelp removes the help overlay.
func (gui *Gui) closeHelp(_ *gocui.Gui, _ *gocui.View) error {
	gui.g.DeleteView("help")
	gui.g.DeleteKeybindings("help")
	if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
		return err
	}
	return nil
}

// switchTabLeft switches to the next tab to the left.
func (gui *Gui) switchTabLeft(_ *gocui.Gui, _ *gocui.View) error {
	switch gui.activeTab {
	case tabLocal:
		// Wrap to Mail
		gui.activeTab = tabMail
		gui.panels = []string{mailListView}
		gui.panelIdx = 0
		if gui.mailbox == nil {
			_ = gui.loadMailbox()
		}
	case tabServers:
		gui.activeTab = tabLocal
		gui.panels = []string{tableView}
		gui.panelIdx = 0
	case tabMail:
		gui.activeTab = tabServers
		gui.panels = []string{serversView, detailView}
		gui.panelIdx = 0
	}
	return nil
}

// switchTabRight switches to the next tab to the right.
func (gui *Gui) switchTabRight(_ *gocui.Gui, _ *gocui.View) error {
	switch gui.activeTab {
	case tabLocal:
		gui.activeTab = tabServers
		gui.panels = []string{serversView, detailView}
		gui.panelIdx = 0
	case tabServers:
		gui.activeTab = tabMail
		gui.panels = []string{mailListView}
		gui.panelIdx = 0
		if gui.mailbox == nil {
			_ = gui.loadMailbox()
		}
	case tabMail:
		// Wrap to Local
		gui.activeTab = tabLocal
		gui.panels = []string{tableView}
		gui.panelIdx = 0
	}
	return nil
}

// serverCursorDown moves the server list cursor down.
func (gui *Gui) serverCursorDown(_ *gocui.Gui, v *gocui.View) error {
	servers := gui.serversConfig.Servers
	if len(servers) == 0 || gui.serverSelected >= len(servers)-1 {
		return nil
	}

	cx, cy := v.Cursor()
	gui.serverSelected++
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	gui.renderDetail()
	return nil
}

// serverCursorUp moves the server list cursor up.
func (gui *Gui) serverCursorUp(_ *gocui.Gui, v *gocui.View) error {
	if gui.serverSelected <= 0 {
		return nil
	}

	cx, cy := v.Cursor()
	gui.serverSelected--
	if err := v.SetCursor(cx, cy-1); err != nil && gui.serverSelected > 0 {
		ox, oy := v.Origin()
		if oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}

	gui.renderDetail()
	return nil
}

// connectServer connects to the selected server via SSH.
func (gui *Gui) connectServer(_ *gocui.Gui, _ *gocui.View) error {
	servers := gui.serversConfig.Servers
	if len(servers) == 0 || gui.serverSelected >= len(servers) {
		return nil
	}

	server := servers[gui.serverSelected]

	// Disconnect existing connection if any
	if gui.activeClient != nil {
		_ = gui.activeClient.Close()
		gui.activeClient = nil
		gui.remoteCrontab = nil
		gui.remoteJobs = nil
	}

	gui.setStatusMessage(fmt.Sprintf("Connecting to %s...", server.Name))

	client, err := ssh.NewClient(server)
	if err != nil {
		gui.setStatusMessage(fmt.Sprintf("Connection failed: %v", err))
		return nil
	}

	gui.activeClient = client

	// Load remote crontab
	output, err := client.ReadCrontab()
	if err != nil {
		gui.setStatusMessage(fmt.Sprintf("Connected but failed to read crontab: %v", err))
		gui.refreshViews()
		return nil
	}

	ct, err := cron.Parse(output)
	if err != nil {
		gui.setStatusMessage(fmt.Sprintf("Connected but failed to parse crontab: %v", err))
		gui.refreshViews()
		return nil
	}

	gui.remoteCrontab = ct
	gui.remoteJobs = ct.Jobs()
	gui.remoteSelected = 0
	gui.setStatusMessage(fmt.Sprintf("Connected to %s — %d remote jobs", server.Name, len(gui.remoteJobs)))
	gui.refreshViews()
	return nil
}

// disconnectServer disconnects from the current server.
func (gui *Gui) disconnectServer(_ *gocui.Gui, _ *gocui.View) error {
	if gui.activeClient == nil {
		gui.setStatusMessage("Not connected to any server")
		return nil
	}

	name := gui.activeClient.ServerName()
	_ = gui.activeClient.Close()
	gui.activeClient = nil
	gui.remoteCrontab = nil
	gui.remoteJobs = nil
	gui.remoteSelected = 0

	gui.setStatusMessage(fmt.Sprintf("Disconnected from %s", name))
	gui.refreshViews()
	return nil
}

// deleteServer removes the selected server from config.
func (gui *Gui) deleteServer(_ *gocui.Gui, _ *gocui.View) error {
	servers := gui.serversConfig.Servers
	if len(servers) == 0 || gui.serverSelected >= len(servers) {
		return nil
	}

	server := servers[gui.serverSelected]

	// Disconnect if we're connected to this server
	if gui.activeClient != nil && gui.activeClient.ServerName() == server.Name {
		_ = gui.activeClient.Close()
		gui.activeClient = nil
		gui.remoteCrontab = nil
		gui.remoteJobs = nil
	}

	gui.serversConfig.Servers = append(servers[:gui.serverSelected], servers[gui.serverSelected+1:]...)

	if gui.serverSelected >= len(gui.serversConfig.Servers) {
		gui.serverSelected = max(0, len(gui.serversConfig.Servers)-1)
	}

	// Persist
	if path, err := ssh.DefaultServersPath(); err == nil {
		_ = ssh.SaveServers(path, gui.serversConfig)
	}

	gui.refreshViews()
	return nil
}

// createJob opens the create modal.
func (gui *Gui) createJob(_ *gocui.Gui, _ *gocui.View) error {
	return gui.openCreateModal(false)
}

// editJob opens the edit modal for the selected job.
func (gui *Gui) editJob(_ *gocui.Gui, _ *gocui.View) error {
	if len(gui.jobs) == 0 {
		return nil
	}
	return gui.openCreateModal(true)
}

// deleteJob opens the delete confirmation modal.
func (gui *Gui) deleteJob(_ *gocui.Gui, _ *gocui.View) error {
	return gui.openDeleteModal()
}

// setStatusMessage temporarily displays a message in the status bar.
func (gui *Gui) setStatusMessage(msg string) {
	v, err := gui.g.View(statusView)
	if err != nil {
		return
	}
	v.Clear()
	fmt.Fprintf(v, "%s%s", title(), msg)
}
