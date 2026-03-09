package gui

import (
	"fmt"
	"log"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/cron"
	"github.com/bssmnt/lazycron/internal/ssh"
	"github.com/bssmnt/lazycron/internal/types"
)

// View name constants.
const (
	statusView  = "status"
	jobsView    = "jobs"
	detailView  = "detail"
	hintsView   = "hints"
	serversView = "servers"
	tableView   = "table"
)

// tab represents the active top-level tab.
type tab int

const (
	tabLocal tab = iota
	tabServers
)

// Gui holds the application state and the gocui instance.
type Gui struct {
	g        *gocui.Gui
	crontab  *cron.Crontab
	jobs     []*cron.CronJob
	selected int
	panels   []string
	panelIdx int
	modal    *modalState
	search   *searchState

	// Tab state
	activeTab tab

	// Server state
	serversConfig  *ssh.ServersConfig
	serverSelected int
	activeClient   *ssh.Client
	remoteCrontab  *cron.Crontab
	remoteJobs     []*cron.CronJob
	remoteSelected int
}

// New creates and returns a new Gui instance.
func New() (*Gui, error) {
	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create gui: %w", err)
	}

	gui := &Gui{
		g:         g,
		panels:    []string{tableView},
		activeTab: tabLocal,
	}

	// Load servers config (best effort)
	if serversPath, err := ssh.DefaultServersPath(); err == nil {
		if cfg, err := ssh.LoadServers(serversPath); err == nil {
			gui.serversConfig = cfg
		}
	}
	if gui.serversConfig == nil {
		gui.serversConfig = &ssh.ServersConfig{}
	}

	g.SetManagerFunc(gui.layout)
	g.Highlight = true
	g.Cursor = false
	g.Mouse = false

	if err := gui.setupKeybindings(); err != nil {
		g.Close()
		return nil, fmt.Errorf("failed to setup keybindings: %w", err)
	}

	return gui, nil
}

// Run starts the main event loop.
func (gui *Gui) Run() error {
	defer gui.g.Close()

	if err := gui.loadCrontab(); err != nil {
		log.Printf("warning: %v", err)
	}

	if err := gui.g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return fmt.Errorf("main loop error: %w", err)
	}

	return nil
}

// loadCrontab reads the crontab and updates the job list.
func (gui *Gui) loadCrontab() error {
	ct, err := cron.ReadCrontab()
	if err != nil {
		return fmt.Errorf("failed to load crontab: %w", err)
	}

	gui.crontab = ct
	gui.jobs = ct.Jobs()

	if gui.selected >= len(gui.jobs) {
		gui.selected = max(0, len(gui.jobs)-1)
	}

	return nil
}

// refreshViews updates all view contents.
func (gui *Gui) refreshViews() {
	gui.g.Update(func(g *gocui.Gui) error {
		switch gui.activeTab {
		case tabLocal:
			gui.renderTable()
		case tabServers:
			gui.renderServerList()
			gui.renderDetail()
		}
		gui.renderStatus()
		gui.renderHints()
		return nil
	})
}

// layout is the gocui manager function that positions all views.
func (gui *Gui) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	// Status bar (top row)
	if err := gui.createStatusView(g, maxX, maxY); err != nil {
		return err
	}

	switch gui.activeTab {
	case tabLocal:
		// Hide servers/split views if they exist
		g.DeleteView(serversView)
		g.DeleteView(jobsView)
		g.DeleteView(detailView)

		// Full-width table
		if err := gui.createTableView(g, maxX, maxY); err != nil {
			return err
		}

	case tabServers:
		// Hide table view if it exists
		g.DeleteView(tableView)

		// Servers list (left panel)
		if err := gui.createServersView(g, maxX, maxY); err != nil {
			return err
		}

		// Detail panel (right panel — server detail or remote jobs)
		if err := gui.createDetailView(g, maxX, maxY); err != nil {
			return err
		}
	}

	// Hints bar (bottom row)
	if err := gui.createHintsView(g, maxX, maxY); err != nil {
		return err
	}

	return nil
}

// currentPanel returns the name of the currently focused panel.
func (gui *Gui) currentPanel() string {
	return gui.panels[gui.panelIdx]
}

// title returns the app title with version.
func title() string {
	return fmt.Sprintf(" lazycron %s ", types.Version)
}
