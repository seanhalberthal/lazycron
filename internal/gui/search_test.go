package gui

import (
	"testing"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/cron"
)

func newTestGocui() (*gocui.Gui, error) {
	return gocui.NewGui(gocui.OutputSimulator, true)
}

func TestNextMatchNilSearch(t *testing.T) {
	gui := &Gui{search: nil}
	err := gui.nextMatch(nil, nil)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestPrevMatchNilSearch(t *testing.T) {
	gui := &Gui{search: nil}
	err := gui.prevMatch(nil, nil)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNextMatchEmptyMatches(t *testing.T) {
	gui := &Gui{search: &searchState{matches: nil}}
	err := gui.nextMatch(nil, nil)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestPrevMatchEmptyMatches(t *testing.T) {
	gui := &Gui{search: &searchState{matches: nil}}
	err := gui.prevMatch(nil, nil)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNextMatchWrapsAround(t *testing.T) {
	g, err := newTestGocui()
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}

	gui := &Gui{
		g:        g,
		panels:   []string{tableView},
		panelIdx: 0,
		search:   &searchState{matches: []int{0, 2, 4}, current: 2},
		jobs: []*cron.CronJob{
			{Command: "a"}, {Command: "b"}, {Command: "c"},
			{Command: "d"}, {Command: "e"},
		},
	}

	// Create the view so jumpToMatch doesn't fail
	if _, err := g.SetView(tableView, 0, 0, 80, 20, 0); err != nil && err != gocui.ErrUnknownView {
		t.Fatal(err)
	}
	if _, err := g.SetView(statusView, 0, 0, 80, 2, 0); err != nil && err != gocui.ErrUnknownView {
		t.Fatal(err)
	}

	_ = gui.nextMatch(nil, nil)
	if gui.search.current != 0 {
		t.Errorf("expected wrap to 0, got %d", gui.search.current)
	}
}

func TestPrevMatchWrapsAround(t *testing.T) {
	g, err := newTestGocui()
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}

	gui := &Gui{
		g:        g,
		panels:   []string{tableView},
		panelIdx: 0,
		search:   &searchState{matches: []int{0, 2, 4}, current: 0},
		jobs: []*cron.CronJob{
			{Command: "a"}, {Command: "b"}, {Command: "c"},
			{Command: "d"}, {Command: "e"},
		},
	}

	if _, err := g.SetView(tableView, 0, 0, 80, 20, 0); err != nil && err != gocui.ErrUnknownView {
		t.Fatal(err)
	}
	if _, err := g.SetView(statusView, 0, 0, 80, 2, 0); err != nil && err != gocui.ErrUnknownView {
		t.Fatal(err)
	}

	_ = gui.prevMatch(nil, nil)
	if gui.search.current != 2 {
		t.Errorf("expected wrap to 2 (last match), got %d", gui.search.current)
	}
}

func TestJumpToMatchOutOfRange(t *testing.T) {
	gui := &Gui{
		search: &searchState{matches: []int{0}},
	}
	// Should not panic
	gui.jumpToMatch(5)
}

func TestSearchStateQueryStorage(t *testing.T) {
	s := &searchState{query: "backup", matches: []int{0, 3}, current: 1}
	if s.query != "backup" {
		t.Errorf("expected query 'backup', got %q", s.query)
	}
	if len(s.matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(s.matches))
	}
}
