package gui

import (
	"fmt"
	"strings"

	"github.com/awesome-gocui/gocui"
)

const searchInputView = "searchInput"

// searchState holds the state of the search feature.
type searchState struct {
	query   string
	matches []int // indices of matching jobs
	current int   // index into matches
}

// openSearch opens the search input at the bottom of the screen.
func (gui *Gui) openSearch(_ *gocui.Gui, _ *gocui.View) error {
	maxX, maxY := gui.g.Size()

	v, err := gui.g.SetView(searchInputView, 0, maxY-3, maxX-1, maxY-1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}

	v.Title = " Search "
	v.Editable = true
	v.Frame = true
	v.Editor = gocui.DefaultEditor

	gui.g.Cursor = true
	gui.search = &searchState{}

	if _, err := gui.g.SetCurrentView(searchInputView); err != nil {
		return err
	}

	// Keybindings for search
	if err := gui.g.SetKeybinding(searchInputView, gocui.KeyEnter, gocui.ModNone, gui.executeSearch); err != nil {
		return err
	}
	if err := gui.g.SetKeybinding(searchInputView, gocui.KeyEsc, gocui.ModNone, gui.closeSearch); err != nil {
		return err
	}

	return nil
}

// executeSearch runs the search and highlights matches.
func (gui *Gui) executeSearch(_ *gocui.Gui, _ *gocui.View) error {
	query := gui.getViewContent(searchInputView)
	if query == "" {
		return gui.closeSearch(nil, nil)
	}

	gui.search.query = strings.ToLower(query)
	gui.search.matches = nil
	gui.search.current = 0

	// Find matching jobs
	for i, job := range gui.jobs {
		name := strings.ToLower(job.DisplayName())
		expr := strings.ToLower(job.Expression)
		cmd := strings.ToLower(job.Command)

		if strings.Contains(name, gui.search.query) ||
			strings.Contains(expr, gui.search.query) ||
			strings.Contains(cmd, gui.search.query) {
			gui.search.matches = append(gui.search.matches, i)
		}
	}

	gui.closeSearchInput()

	if len(gui.search.matches) > 0 {
		gui.jumpToMatch(0)
		gui.setStatusMessage(fmt.Sprintf("Found %d match(es) for %q", len(gui.search.matches), query))
	} else {
		gui.setStatusMessage(fmt.Sprintf("No matches for %q", query))
		gui.search = nil
	}

	return nil
}

// nextMatch jumps to the next search match.
func (gui *Gui) nextMatch(_ *gocui.Gui, _ *gocui.View) error {
	if gui.search == nil || len(gui.search.matches) == 0 {
		return nil
	}

	gui.search.current = (gui.search.current + 1) % len(gui.search.matches)
	gui.jumpToMatch(gui.search.current)
	gui.setStatusMessage(fmt.Sprintf("Match %d/%d", gui.search.current+1, len(gui.search.matches)))
	return nil
}

// prevMatch jumps to the previous search match.
func (gui *Gui) prevMatch(_ *gocui.Gui, _ *gocui.View) error {
	if gui.search == nil || len(gui.search.matches) == 0 {
		return nil
	}

	gui.search.current--
	if gui.search.current < 0 {
		gui.search.current = len(gui.search.matches) - 1
	}
	gui.jumpToMatch(gui.search.current)
	gui.setStatusMessage(fmt.Sprintf("Match %d/%d", gui.search.current+1, len(gui.search.matches)))
	return nil
}

// jumpToMatch moves the cursor to the specified match index.
func (gui *Gui) jumpToMatch(matchIdx int) {
	if matchIdx >= len(gui.search.matches) {
		return
	}

	jobIdx := gui.search.matches[matchIdx]
	gui.selected = jobIdx

	// Account for header row offset in table view
	cursorIdx := jobIdx
	viewName := gui.currentPanel()
	if viewName == tableView {
		cursorIdx = jobIdx + 1 // +1 for header row
	}

	v, err := gui.g.View(viewName)
	if err != nil {
		return
	}

	_ = v.SetCursor(0, cursorIdx)
	_ = v.SetOrigin(0, 0)
}

// closeSearch closes the search input and clears search state.
func (gui *Gui) closeSearch(_ *gocui.Gui, _ *gocui.View) error {
	gui.search = nil
	gui.closeSearchInput()
	gui.refreshViews()
	return nil
}

// closeSearchInput removes the search input view.
func (gui *Gui) closeSearchInput() {
	gui.g.Cursor = false
	gui.g.DeleteView(searchInputView)
	gui.g.DeleteKeybindings(searchInputView)
	if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
		_ = err
	}
}
