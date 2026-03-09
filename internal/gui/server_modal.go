package gui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/gui/style"
	"github.com/bssmnt/lazycron/internal/ssh"
)

// Server modal view names.
const (
	serverModalView = "serverModal"
	serverNameInput = "serverNameInput"
	serverHostInput = "serverHostInput"
	serverPortInput = "serverPortInput"
	serverUserInput = "serverUserInput"
)

// Server modal label view names.
const (
	srvLblName  = "srvLblName"
	srvLblHost  = "srvLblHost"
	srvLblPort  = "srvLblPort"
	srvLblUser  = "srvLblUser"
	srvLblAuth  = "srvLblAuth"
	srvLblHints = "srvLblHints"
)

// addServer opens the add server modal.
func (gui *Gui) addServer(_ *gocui.Gui, _ *gocui.View) error {
	maxX, maxY := gui.g.Size()
	width := 60
	height := 20

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

	labelCol := x0 + 3
	inputX0 := x0 + 12
	inputX1 := x1 - 3

	// Modal frame
	frame, err := gui.g.SetView(serverModalView, x0, y0, x1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	frame.Title = " Add Server "
	frame.Clear()

	// Row layout:  y0+0 = frame border
	//              y0+1 = blank
	//              y0+2 = Name (rows 2–4)
	//              y0+5 = Host (rows 5–7)
	//              y0+8 = Port (rows 8–10)
	//              y0+11 = User (rows 11–13)
	//              y0+14 = blank
	//              y0+15 = Auth info
	//              y0+16 = blank
	//              y0+17 = Hints
	//              y0+18 = blank
	//              y0+19 = frame border

	// Name label
	lbl, _ := gui.g.SetView(srvLblName, labelCol, y0+2, inputX0-1, y0+4, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.FgGreen, " Name:"))
	}

	// Host label
	lbl, _ = gui.g.SetView(srvLblHost, labelCol, y0+5, inputX0-1, y0+7, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.FgGreen, " Host:"))
	}

	// Port label
	lbl, _ = gui.g.SetView(srvLblPort, labelCol, y0+8, inputX0-1, y0+10, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.FgGreen, " Port:"))
	}

	// User label
	lbl, _ = gui.g.SetView(srvLblUser, labelCol, y0+11, inputX0-1, y0+13, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.FgGreen, " User:"))
	}

	// Auth info
	lbl, _ = gui.g.SetView(srvLblAuth, labelCol, y0+15, inputX1+1, y0+17, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.Dim, " Auth: key (uses ~/.ssh/id_ed25519)"))
	}

	// Hints
	lbl, _ = gui.g.SetView(srvLblHints, labelCol, y0+17, inputX1+1, y0+19, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.Dim, " [Tab/S-Tab] next/prev   [Enter] save   [Esc] cancel"))
	}

	// Input fields
	nameV, err := gui.g.SetView(serverNameInput, inputX0, y0+2, inputX1, y0+4, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	nameV.Editable = true
	nameV.Frame = true
	nameV.Editor = inputEditor

	hostV, err := gui.g.SetView(serverHostInput, inputX0, y0+5, inputX1, y0+7, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	hostV.Editable = true
	hostV.Frame = true
	hostV.Editor = inputEditor

	portV, err := gui.g.SetView(serverPortInput, inputX0, y0+8, inputX1, y0+10, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	portV.Editable = true
	portV.Frame = true
	portV.Editor = inputEditor
	fmt.Fprint(portV, "22")

	userV, err := gui.g.SetView(serverUserInput, inputX0, y0+11, inputX1, y0+13, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	userV.Editable = true
	userV.Frame = true
	userV.Editor = inputEditor

	gui.g.Cursor = true

	// Set keybindings for modal inputs
	inputViews := []string{serverNameInput, serverHostInput, serverPortInput, serverUserInput}
	for _, vn := range inputViews {
		name := vn
		if err := gui.g.SetKeybinding(name, gocui.KeyEsc, gocui.ModNone, gui.closeServerModal); err != nil {
			return err
		}
		if err := gui.g.SetKeybinding(name, gocui.KeyEnter, gocui.ModNone, gui.saveServer); err != nil {
			return err
		}
		if err := gui.g.SetKeybinding(name, gocui.KeyTab, gocui.ModNone, gui.nextServerField); err != nil {
			return err
		}
		if err := gui.g.SetKeybinding(name, gocui.KeyBacktab, gocui.ModNone, gui.prevServerField); err != nil {
			return err
		}
	}

	if _, err := gui.g.SetCurrentView(serverNameInput); err != nil {
		return err
	}

	return nil
}

// serverFieldOrder defines the tab order for server modal fields.
var serverFieldOrder = []string{serverNameInput, serverHostInput, serverPortInput, serverUserInput}

// nextServerField cycles to the next input field.
func (gui *Gui) nextServerField(_ *gocui.Gui, v *gocui.View) error {
	current := v.Name()
	for i, name := range serverFieldOrder {
		if name == current {
			next := serverFieldOrder[(i+1)%len(serverFieldOrder)]
			if _, err := gui.g.SetCurrentView(next); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

// prevServerField cycles to the previous input field.
func (gui *Gui) prevServerField(_ *gocui.Gui, v *gocui.View) error {
	current := v.Name()
	for i, name := range serverFieldOrder {
		if name == current {
			prev := serverFieldOrder[(i+len(serverFieldOrder)-1)%len(serverFieldOrder)]
			if _, err := gui.g.SetCurrentView(prev); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

// saveServer saves the new server from the modal form.
func (gui *Gui) saveServer(_ *gocui.Gui, _ *gocui.View) error {
	name := gui.getViewContent(serverNameInput)
	host := gui.getViewContent(serverHostInput)
	portStr := gui.getViewContent(serverPortInput)
	user := gui.getViewContent(serverUserInput)

	if name == "" || host == "" || user == "" {
		gui.setStatusMessage("Name, host, and user are required")
		return nil
	}

	port := 22
	if portStr != "" {
		p, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil || p <= 0 || p > 65535 {
			gui.setStatusMessage("Invalid port number")
			return nil
		}
		port = p
	}

	server := ssh.Server{
		Name:     name,
		Host:     host,
		Port:     port,
		User:     user,
		AuthType: "key",
	}

	gui.serversConfig.Servers = append(gui.serversConfig.Servers, server)

	// Persist
	if path, err := ssh.DefaultServersPath(); err == nil {
		if err := ssh.SaveServers(path, gui.serversConfig); err != nil {
			gui.setStatusMessage(fmt.Sprintf("Saved in memory but failed to persist: %v", err))
		}
	}

	gui.closeServerModalViews()
	gui.refreshViews()
	gui.setStatusMessage(fmt.Sprintf("Added server %q", name))
	return nil
}

// closeServerModal closes the server modal without saving.
func (gui *Gui) closeServerModal(_ *gocui.Gui, _ *gocui.View) error {
	gui.closeServerModalViews()
	return nil
}

// closeServerModalViews removes all server modal views.
func (gui *Gui) closeServerModalViews() {
	gui.g.Cursor = false

	views := []string{
		serverModalView, serverNameInput, serverHostInput,
		serverPortInput, serverUserInput,
		srvLblName, srvLblHost, srvLblPort, srvLblUser,
		srvLblAuth, srvLblHints,
	}
	for _, name := range views {
		gui.g.DeleteView(name)
		gui.g.DeleteKeybindings(name)
	}

	if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
		_ = err
	}
}
