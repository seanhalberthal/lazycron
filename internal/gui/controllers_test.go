package gui

import (
	"testing"

	"github.com/bssmnt/lazycron/internal/ssh"
)

func TestSwitchTabRightLocalToServers(t *testing.T) {
	gui := &Gui{
		panels:        []string{tableView},
		panelIdx:      0,
		activeTab:     tabLocal,
		serversConfig: &ssh.ServersConfig{},
	}

	_ = gui.switchTabRight(nil, nil)
	if gui.activeTab != tabServers {
		t.Errorf("expected tabServers, got %d", gui.activeTab)
	}
	if gui.panels[0] != serversView {
		t.Errorf("expected panels[0] = %q, got %q", serversView, gui.panels[0])
	}
}

func TestSwitchTabRightServersToMail(t *testing.T) {
	gui := &Gui{
		panels:        []string{serversView, detailView},
		panelIdx:      0,
		activeTab:     tabServers,
		serversConfig: &ssh.ServersConfig{},
	}

	_ = gui.switchTabRight(nil, nil)
	if gui.activeTab != tabMail {
		t.Errorf("expected tabMail, got %d", gui.activeTab)
	}
	if gui.panels[0] != mailListView {
		t.Errorf("expected panels[0] = %q, got %q", mailListView, gui.panels[0])
	}
}

func TestSwitchTabRightMailToLocal(t *testing.T) {
	gui := &Gui{
		panels:        []string{mailListView},
		panelIdx:      0,
		activeTab:     tabMail,
		serversConfig: &ssh.ServersConfig{},
	}

	_ = gui.switchTabRight(nil, nil)
	if gui.activeTab != tabLocal {
		t.Errorf("expected tabLocal, got %d", gui.activeTab)
	}
	if gui.panels[0] != tableView {
		t.Errorf("expected panels[0] = %q, got %q", tableView, gui.panels[0])
	}
}

func TestSwitchTabLeftLocalToMail(t *testing.T) {
	gui := &Gui{
		panels:        []string{tableView},
		panelIdx:      0,
		activeTab:     tabLocal,
		serversConfig: &ssh.ServersConfig{},
	}

	_ = gui.switchTabLeft(nil, nil)
	if gui.activeTab != tabMail {
		t.Errorf("expected tabMail, got %d", gui.activeTab)
	}
}

func TestSwitchTabLeftServersToLocal(t *testing.T) {
	gui := &Gui{
		panels:        []string{serversView, detailView},
		panelIdx:      0,
		activeTab:     tabServers,
		serversConfig: &ssh.ServersConfig{},
	}

	_ = gui.switchTabLeft(nil, nil)
	if gui.activeTab != tabLocal {
		t.Errorf("expected tabLocal, got %d", gui.activeTab)
	}
}

func TestSwitchTabLeftMailToServers(t *testing.T) {
	gui := &Gui{
		panels:        []string{mailListView},
		panelIdx:      0,
		activeTab:     tabMail,
		serversConfig: &ssh.ServersConfig{},
	}

	_ = gui.switchTabLeft(nil, nil)
	if gui.activeTab != tabServers {
		t.Errorf("expected tabServers, got %d", gui.activeTab)
	}
}

func TestSwitchTabFullCycleRight(t *testing.T) {
	gui := &Gui{
		panels:        []string{tableView},
		panelIdx:      0,
		activeTab:     tabLocal,
		serversConfig: &ssh.ServersConfig{},
	}

	// Local -> Servers -> Mail -> Local (full cycle)
	_ = gui.switchTabRight(nil, nil)
	_ = gui.switchTabRight(nil, nil)
	_ = gui.switchTabRight(nil, nil)

	if gui.activeTab != tabLocal {
		t.Errorf("expected tabLocal after full cycle, got %d", gui.activeTab)
	}
}

func TestSwitchTabPanelIdxResets(t *testing.T) {
	gui := &Gui{
		panels:        []string{serversView, detailView},
		panelIdx:      1, // focused on detail
		activeTab:     tabServers,
		serversConfig: &ssh.ServersConfig{},
	}

	_ = gui.switchTabRight(nil, nil)
	if gui.panelIdx != 0 {
		t.Errorf("expected panelIdx 0 after tab switch, got %d", gui.panelIdx)
	}
}
