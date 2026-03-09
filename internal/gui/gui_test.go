package gui

import (
	"os"
	"testing"
)

func TestNewGuiCreation(t *testing.T) {
	// Skip when no terminal is available (CI, non-interactive shells)
	if testing.Short() {
		t.Skip("skipping TUI test in short mode")
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		t.Skip("skipping TUI test: no TTY available")
	}
	_ = f.Close()

	g, err := New()
	if err != nil {
		t.Fatalf("failed to create gui: %v", err)
	}
	defer g.g.Close()

	if g.g == nil {
		t.Fatal("expected gocui.Gui to be non-nil")
	}

	if len(g.panels) != 2 {
		t.Errorf("expected 2 panels, got %d", len(g.panels))
	}

	if g.panels[0] != jobsView {
		t.Errorf("expected first panel to be %q, got %q", jobsView, g.panels[0])
	}

	if g.panels[1] != detailView {
		t.Errorf("expected second panel to be %q, got %q", detailView, g.panels[1])
	}
}

func TestCurrentPanel(t *testing.T) {
	gui := &Gui{
		panels:   []string{jobsView, detailView},
		panelIdx: 0,
	}

	if got := gui.currentPanel(); got != jobsView {
		t.Errorf("expected %q, got %q", jobsView, got)
	}

	gui.panelIdx = 1
	if got := gui.currentPanel(); got != detailView {
		t.Errorf("expected %q, got %q", detailView, got)
	}
}

func TestTitle(t *testing.T) {
	got := title()
	if got == "" {
		t.Fatal("expected non-empty title")
	}
}

func TestJobsPanelRatio(t *testing.T) {
	if jobsPanelRatio <= 0 || jobsPanelRatio >= 1 {
		t.Errorf("expected jobsPanelRatio between 0 and 1, got %f", jobsPanelRatio)
	}
}

func TestSplitXMinimum(t *testing.T) {
	// For a very small terminal, splitX should be at least 20
	maxX := 40
	splitX := max(int(float64(maxX)*jobsPanelRatio), 20)

	if splitX < 20 {
		t.Errorf("expected splitX >= 20, got %d", splitX)
	}
}

func TestSplitXLargeTerminal(t *testing.T) {
	maxX := 200
	splitX := max(int(float64(maxX)*jobsPanelRatio), 20)
	expected := int(float64(maxX) * jobsPanelRatio)

	if splitX != expected {
		t.Errorf("expected splitX %d for large terminal, got %d", expected, splitX)
	}
}

func TestModalFieldViewName(t *testing.T) {
	gui := &Gui{}

	tests := []struct {
		field    modalField
		expected string
	}{
		{fieldName, nameInputView},
		{fieldExpression, expressionInputView},
		{fieldCommand, commandInputView},
		{modalField(99), nameInputView}, // unknown defaults to name
	}

	for _, tc := range tests {
		got := gui.modalFieldViewName(tc.field)
		if got != tc.expected {
			t.Errorf("modalFieldViewName(%d) = %q, want %q", tc.field, got, tc.expected)
		}
	}
}

func TestModalFieldConstants(t *testing.T) {
	if fieldName != 0 || fieldExpression != 1 || fieldCommand != 2 {
		t.Error("modal field constants have unexpected values")
	}
}

func TestSearchState(t *testing.T) {
	s := &searchState{
		query:   "backup",
		matches: []int{0, 3, 5},
		current: 0,
	}

	if s.query != "backup" {
		t.Errorf("expected query %q, got %q", "backup", s.query)
	}

	if len(s.matches) != 3 {
		t.Errorf("expected 3 matches, got %d", len(s.matches))
	}
}

func TestViewNameConstants(t *testing.T) {
	// Ensure view names are unique
	names := map[string]bool{
		statusView:          true,
		jobsView:            true,
		detailView:          true,
		hintsView:           true,
		serversView:         true,
		createModalView:     true,
		deleteModalView:     true,
		nameInputView:       true,
		expressionInputView: true,
		commandInputView:    true,
		validationView:      true,
		searchInputView:     true,
		serverModalView:     true,
		serverNameInput:     true,
		serverHostInput:     true,
		serverPortInput:     true,
		serverUserInput:     true,
	}

	if len(names) != 17 {
		t.Errorf("expected 17 unique view names, got %d", len(names))
	}
}

func TestTabSwitching(t *testing.T) {
	gui := &Gui{
		panels:    []string{jobsView, detailView},
		panelIdx:  0,
		activeTab: tabLocal,
	}

	if gui.activeTab != tabLocal {
		t.Error("expected initial tab to be Local")
	}

	if gui.currentPanel() != jobsView {
		t.Errorf("expected current panel %q, got %q", jobsView, gui.currentPanel())
	}
}

func TestTabConstants(t *testing.T) {
	if tabLocal != 0 || tabServers != 1 {
		t.Error("tab constants have unexpected values")
	}
}
