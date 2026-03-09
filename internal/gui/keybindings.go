package gui

import "github.com/awesome-gocui/gocui"

// setupKeybindings registers all global and view-specific keybindings.
func (gui *Gui) setupKeybindings() error {
	bindings := []struct {
		viewName string
		key      any
		mod      gocui.Modifier
		handler  func(*gocui.Gui, *gocui.View) error
	}{
		// Global
		{"", gocui.KeyCtrlC, gocui.ModNone, gui.quit},
		{"", 'q', gocui.ModNone, gui.quit},
		{"", 'r', gocui.ModNone, gui.refresh},
		{"", gocui.KeyTab, gocui.ModNone, gui.switchPanel},
		{"", '?', gocui.ModNone, gui.toggleHelp},

		// Table navigation (Local tab)
		{tableView, 'j', gocui.ModNone, gui.cursorDown},
		{tableView, 'k', gocui.ModNone, gui.cursorUp},
		{tableView, gocui.KeyArrowDown, gocui.ModNone, gui.cursorDown},
		{tableView, gocui.KeyArrowUp, gocui.ModNone, gui.cursorUp},

		// Job actions (Local tab)
		{tableView, 'p', gocui.ModNone, gui.togglePause},
		{tableView, 'c', gocui.ModNone, gui.createJob},
		{tableView, 'e', gocui.ModNone, gui.editJob},
		{tableView, 'D', gocui.ModNone, gui.deleteJob},
		{tableView, gocui.KeyEnter, gocui.ModNone, gui.openDetailOverlay},

		// Search
		{tableView, '/', gocui.ModNone, gui.openSearch},
		{tableView, 'n', gocui.ModNone, gui.nextMatch},
		{tableView, 'N', gocui.ModNone, gui.prevMatch},

		// Tab switching (from table view)
		{tableView, 'h', gocui.ModNone, gui.switchTabLeft},
		{tableView, 'l', gocui.ModNone, gui.switchTabRight},

		// Server list navigation
		{serversView, 'j', gocui.ModNone, gui.serverCursorDown},
		{serversView, 'k', gocui.ModNone, gui.serverCursorUp},
		{serversView, gocui.KeyArrowDown, gocui.ModNone, gui.serverCursorDown},
		{serversView, gocui.KeyArrowUp, gocui.ModNone, gui.serverCursorUp},

		// Server actions
		{serversView, 'a', gocui.ModNone, gui.addServer},
		{serversView, 'c', gocui.ModNone, gui.connectServer},
		{serversView, 'd', gocui.ModNone, gui.disconnectServer},
		{serversView, 'D', gocui.ModNone, gui.deleteServer},

		// Tab switching (from servers view)
		{serversView, 'h', gocui.ModNone, gui.switchTabLeft},
		{serversView, 'l', gocui.ModNone, gui.switchTabRight},

		// Mail list navigation
		{mailListView, 'j', gocui.ModNone, gui.mailCursorDown},
		{mailListView, 'k', gocui.ModNone, gui.mailCursorUp},
		{mailListView, gocui.KeyArrowDown, gocui.ModNone, gui.mailCursorDown},
		{mailListView, gocui.KeyArrowUp, gocui.ModNone, gui.mailCursorUp},

		// Mail actions
		{mailListView, gocui.KeyEnter, gocui.ModNone, gui.openMailDetail},
		{mailListView, 'd', gocui.ModNone, gui.deleteMail},
		{mailListView, 'D', gocui.ModNone, gui.deleteAllMail},
		{mailListView, 'r', gocui.ModNone, gui.refreshMail},
		{mailListView, 'S', gocui.ModNone, gui.switchMailSource},

		// Tab switching (from mail view)
		{mailListView, 'h', gocui.ModNone, gui.switchTabLeft},
		{mailListView, 'l', gocui.ModNone, gui.switchTabRight},
	}

	for _, b := range bindings {
		if err := gui.g.SetKeybinding(b.viewName, b.key, b.mod, b.handler); err != nil {
			return err
		}
	}

	return nil
}
