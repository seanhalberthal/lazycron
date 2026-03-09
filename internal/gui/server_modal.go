package gui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/awesome-gocui/gocui"
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

// addServer opens the add server modal.
func (gui *Gui) addServer(_ *gocui.Gui, _ *gocui.View) error {
	maxX, maxY := gui.g.Size()
	width := 50
	height := 14

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

	// Modal frame
	frame, err := gui.g.SetView(serverModalView, x0, y0, x1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	frame.Title = " Add Server "
	frame.Clear()

	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  Name:")
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  Host:")
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  Port:")
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  User:")
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  Auth: key (uses ~/.ssh/id_ed25519)")
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  [Tab] next  [Enter] save  [Esc] cancel")

	// Input fields
	nameV, err := gui.g.SetView(serverNameInput, x0+10, y0+1, x1-2, y0+3, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	nameV.Editable = true
	nameV.Frame = true
	nameV.Editor = gocui.DefaultEditor

	hostV, err := gui.g.SetView(serverHostInput, x0+10, y0+3, x1-2, y0+5, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	hostV.Editable = true
	hostV.Frame = true
	hostV.Editor = gocui.DefaultEditor

	portV, err := gui.g.SetView(serverPortInput, x0+10, y0+5, x1-2, y0+7, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	portV.Editable = true
	portV.Frame = true
	portV.Editor = gocui.DefaultEditor
	fmt.Fprint(portV, "22")

	userV, err := gui.g.SetView(serverUserInput, x0+10, y0+7, x1-2, y0+9, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	userV.Editable = true
	userV.Frame = true
	userV.Editor = gocui.DefaultEditor

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

	views := []string{serverModalView, serverNameInput, serverHostInput, serverPortInput, serverUserInput}
	for _, name := range views {
		gui.g.DeleteView(name)
		gui.g.DeleteKeybindings(name)
	}

	if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
		_ = err
	}
}
