package gui

import (
	"os"
	"testing"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/ssh"
)

func requireServerModalTTY(t *testing.T) *gocui.Gui {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping modal test in short mode")
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		t.Skip("skipping modal test: no TTY available")
	}
	_ = f.Close()

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	t.Cleanup(g.Close)
	return g
}

// writeToView writes text into a gocui view.
func writeToView(t *testing.T, g *gocui.Gui, name, text string) {
	t.Helper()
	v, err := g.View(name)
	if err != nil {
		t.Fatalf("view %s not found: %v", name, err)
	}
	v.Clear()
	for _, ch := range text {
		v.EditWrite(ch)
	}
}

func setupServerModal(t *testing.T, g *gocui.Gui) *Gui {
	t.Helper()

	// Create views for all server input fields
	for _, name := range []string{serverNameInput, serverHostInput, serverPortInput, serverUserInput} {
		v, err := g.SetView(name, 0, 0, 40, 2, 0)
		if err != nil && err != gocui.ErrUnknownView {
			t.Fatalf("failed to create view %s: %v", name, err)
		}
		v.Editable = true
	}

	// Create status view for setStatusMessage
	if _, err := g.SetView(statusView, 0, 0, 80, 2, 0); err != nil && err != gocui.ErrUnknownView {
		t.Fatal(err)
	}

	// Create a panel view for closeServerModalViews
	if _, err := g.SetView(tableView, 0, 3, 80, 20, 0); err != nil && err != gocui.ErrUnknownView {
		t.Fatal(err)
	}

	return &Gui{
		g:             g,
		panels:        []string{tableView},
		panelIdx:      0,
		serversConfig: &ssh.ServersConfig{},
	}
}

func TestSaveServerRequiredFields(t *testing.T) {
	g := requireServerModalTTY(t)
	gui := setupServerModal(t, g)

	// All fields empty — should not save
	err := gui.saveServer(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gui.serversConfig.Servers) != 0 {
		t.Error("expected no servers to be added with empty fields")
	}
}

func TestSaveServerMissingHost(t *testing.T) {
	g := requireServerModalTTY(t)
	gui := setupServerModal(t, g)

	writeToView(t, g, serverNameInput, "myserver")
	writeToView(t, g, serverUserInput, "root")
	// Host is empty

	_ = gui.saveServer(nil, nil)
	if len(gui.serversConfig.Servers) != 0 {
		t.Error("expected no servers without host")
	}
}

func TestSaveServerMissingUser(t *testing.T) {
	g := requireServerModalTTY(t)
	gui := setupServerModal(t, g)

	writeToView(t, g, serverNameInput, "myserver")
	writeToView(t, g, serverHostInput, "192.168.1.1")
	// User is empty

	_ = gui.saveServer(nil, nil)
	if len(gui.serversConfig.Servers) != 0 {
		t.Error("expected no servers without user")
	}
}

func TestSaveServerInvalidPort(t *testing.T) {
	g := requireServerModalTTY(t)
	gui := setupServerModal(t, g)

	writeToView(t, g, serverNameInput, "myserver")
	writeToView(t, g, serverHostInput, "192.168.1.1")
	writeToView(t, g, serverPortInput, "99999")
	writeToView(t, g, serverUserInput, "root")

	_ = gui.saveServer(nil, nil)
	if len(gui.serversConfig.Servers) != 0 {
		t.Error("expected no servers with invalid port 99999")
	}
}

func TestSaveServerPortZero(t *testing.T) {
	g := requireServerModalTTY(t)
	gui := setupServerModal(t, g)

	writeToView(t, g, serverNameInput, "myserver")
	writeToView(t, g, serverHostInput, "192.168.1.1")
	writeToView(t, g, serverPortInput, "0")
	writeToView(t, g, serverUserInput, "root")

	_ = gui.saveServer(nil, nil)
	if len(gui.serversConfig.Servers) != 0 {
		t.Error("expected no servers with port 0")
	}
}

func TestSaveServerNonNumericPort(t *testing.T) {
	g := requireServerModalTTY(t)
	gui := setupServerModal(t, g)

	writeToView(t, g, serverNameInput, "myserver")
	writeToView(t, g, serverHostInput, "192.168.1.1")
	writeToView(t, g, serverPortInput, "abc")
	writeToView(t, g, serverUserInput, "root")

	_ = gui.saveServer(nil, nil)
	if len(gui.serversConfig.Servers) != 0 {
		t.Error("expected no servers with non-numeric port")
	}
}

func TestSaveServerValidDefaultPort(t *testing.T) {
	g := requireServerModalTTY(t)
	gui := setupServerModal(t, g)

	writeToView(t, g, serverNameInput, "myserver")
	writeToView(t, g, serverHostInput, "192.168.1.1")
	// Port left empty — should default to 22
	writeToView(t, g, serverUserInput, "root")

	_ = gui.saveServer(nil, nil)
	if len(gui.serversConfig.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(gui.serversConfig.Servers))
	}
	if gui.serversConfig.Servers[0].Port != 22 {
		t.Errorf("expected default port 22, got %d", gui.serversConfig.Servers[0].Port)
	}
}

func TestSaveServerValidCustomPort(t *testing.T) {
	g := requireServerModalTTY(t)
	gui := setupServerModal(t, g)

	writeToView(t, g, serverNameInput, "myserver")
	writeToView(t, g, serverHostInput, "192.168.1.1")
	writeToView(t, g, serverPortInput, "2222")
	writeToView(t, g, serverUserInput, "root")

	_ = gui.saveServer(nil, nil)
	if len(gui.serversConfig.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(gui.serversConfig.Servers))
	}
	if gui.serversConfig.Servers[0].Port != 2222 {
		t.Errorf("expected port 2222, got %d", gui.serversConfig.Servers[0].Port)
	}
	if gui.serversConfig.Servers[0].Name != "myserver" {
		t.Errorf("expected name 'myserver', got %q", gui.serversConfig.Servers[0].Name)
	}
}
