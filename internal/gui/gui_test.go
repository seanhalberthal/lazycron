package gui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/cron"
	"github.com/bssmnt/lazycron/internal/gui/style"
	"github.com/bssmnt/lazycron/internal/ssh"
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

	if len(g.panels) != 1 {
		t.Errorf("expected 1 panel, got %d", len(g.panels))
	}

	if g.panels[0] != tableView {
		t.Errorf("expected first panel to be %q, got %q", tableView, g.panels[0])
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
		exprGuideView:       true,
		srvLblName:          true,
		srvLblHost:          true,
		srvLblPort:          true,
		srvLblUser:          true,
		srvLblAuth:          true,
		srvLblHints:         true,
	}

	if len(names) != 24 {
		t.Errorf("expected 24 unique view names, got %d", len(names))
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

// testJobs returns a slice of cron jobs for testing.
func testJobs(n int) []*cron.CronJob {
	jobs := make([]*cron.CronJob, n)
	for i := range n {
		jobs[i] = &cron.CronJob{
			Expression: "* * * * *",
			Command:    "echo test",
			Enabled:    true,
		}
	}
	return jobs
}

// testServers returns a slice of servers for testing.
func testServers(n int) []ssh.Server {
	servers := make([]ssh.Server, n)
	for i := range n {
		servers[i] = ssh.Server{
			Name: "server",
			Host: "host",
			Port: 22,
			User: "user",
		}
	}
	return servers
}

// requireTTY skips the test if no terminal is available.
func requireTTY(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping TUI test in short mode")
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		t.Skip("skipping TUI test: no TTY available")
	}
	_ = f.Close()
}

func TestSwitchTabRoundTripState(t *testing.T) {
	gui := &Gui{
		panels:        []string{tableView},
		panelIdx:      0,
		activeTab:     tabLocal,
		selected:      3,
		serversConfig: &ssh.ServersConfig{},
	}

	// Local → Servers
	_ = gui.switchTabRight(nil, nil)
	if gui.activeTab != tabServers {
		t.Fatalf("expected tabServers, got %d", gui.activeTab)
	}
	if gui.panels[0] != serversView {
		t.Errorf("expected panels[0] = %q, got %q", serversView, gui.panels[0])
	}
	if gui.panelIdx != 0 {
		t.Errorf("expected panelIdx 0, got %d", gui.panelIdx)
	}

	// Servers → Local
	_ = gui.switchTabLeft(nil, nil)
	if gui.activeTab != tabLocal {
		t.Fatalf("expected tabLocal, got %d", gui.activeTab)
	}
	if gui.panels[0] != tableView {
		t.Errorf("expected panels[0] = %q, got %q", tableView, gui.panels[0])
	}
	// selected must survive the round-trip (cursor sync is the view's job)
	if gui.selected != 3 {
		t.Errorf("expected selected to be preserved at 3, got %d", gui.selected)
	}
}

func TestSwitchTabNoOpWhenAlreadyOnTab(t *testing.T) {
	gui := &Gui{
		panels:        []string{tableView},
		panelIdx:      0,
		activeTab:     tabLocal,
		serversConfig: &ssh.ServersConfig{},
	}

	_ = gui.switchTabLeft(nil, nil)
	if gui.activeTab != tabLocal {
		t.Error("switchTabLeft from Local should be a no-op")
	}

	gui.activeTab = tabServers
	gui.panels = []string{serversView, detailView}

	_ = gui.switchTabRight(nil, nil)
	if gui.activeTab != tabServers {
		t.Error("switchTabRight from Servers should be a no-op")
	}
}

func TestTableCursorSyncAfterTabSwitch(t *testing.T) {
	requireTTY(t)

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	defer g.Close()

	gui := &Gui{
		g:             g,
		panels:        []string{tableView},
		panelIdx:      0,
		activeTab:     tabLocal,
		jobs:          testJobs(5),
		selected:      3,
		serversConfig: &ssh.ServersConfig{Servers: testServers(2)},
	}

	// Simulate the full tab round-trip via layout:
	// 1. Initial layout creates the table view (Local tab)
	if err := gui.layout(g); err != nil {
		t.Fatalf("initial layout failed: %v", err)
	}

	// Verify cursor matches selected after initial creation
	v, err := g.View(tableView)
	if err != nil {
		t.Fatalf("table view not found: %v", err)
	}
	_, cy := v.Cursor()
	if cy != gui.selected+2 {
		t.Errorf("after initial layout: cursor Y = %d, want %d (selected=%d)", cy, gui.selected+2, gui.selected)
	}

	// 2. Switch to Servers tab and run layout (destroys table view)
	_ = gui.switchTabRight(nil, nil)
	if err := gui.layout(g); err != nil {
		t.Fatalf("servers layout failed: %v", err)
	}
	if _, err := g.View(tableView); err == nil {
		t.Fatal("table view should not exist on Servers tab")
	}

	// 3. Switch back to Local and run layout (recreates table view)
	_ = gui.switchTabLeft(nil, nil)
	if err := gui.layout(g); err != nil {
		t.Fatalf("local layout (return) failed: %v", err)
	}

	v, err = g.View(tableView)
	if err != nil {
		t.Fatalf("table view not found after return: %v", err)
	}
	_, cy = v.Cursor()
	if cy != gui.selected+2 {
		t.Errorf("after tab round-trip: cursor Y = %d, want %d (selected=%d)", cy, gui.selected+2, gui.selected)
	}
}

func TestTableCursorSyncAfterRefresh(t *testing.T) {
	requireTTY(t)

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	defer g.Close()

	gui := &Gui{
		g:             g,
		panels:        []string{tableView},
		panelIdx:      0,
		activeTab:     tabLocal,
		jobs:          testJobs(5),
		selected:      2,
		serversConfig: &ssh.ServersConfig{},
	}

	if err := gui.layout(g); err != nil {
		t.Fatalf("layout failed: %v", err)
	}

	// renderTable is called during refresh — verify cursor survives Clear()
	gui.renderTable()

	v, err := g.View(tableView)
	if err != nil {
		t.Fatalf("table view not found: %v", err)
	}
	_, cy := v.Cursor()
	if cy != gui.selected+2 {
		t.Errorf("after renderTable: cursor Y = %d, want %d (selected=%d)", cy, gui.selected+2, gui.selected)
	}
}

func TestServerCursorSyncAfterTabSwitch(t *testing.T) {
	requireTTY(t)

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	defer g.Close()

	gui := &Gui{
		g:              g,
		panels:         []string{serversView, detailView},
		panelIdx:       0,
		activeTab:      tabServers,
		jobs:           testJobs(3),
		serversConfig:  &ssh.ServersConfig{Servers: testServers(4)},
		serverSelected: 2,
	}

	// 1. Initial layout on Servers tab
	if err := gui.layout(g); err != nil {
		t.Fatalf("initial layout failed: %v", err)
	}

	v, err := g.View(serversView)
	if err != nil {
		t.Fatalf("servers view not found: %v", err)
	}
	_, cy := v.Cursor()
	if cy != gui.serverSelected {
		t.Errorf("after initial layout: cursor Y = %d, want %d (serverSelected=%d)", cy, gui.serverSelected, gui.serverSelected)
	}

	// 2. Switch to Local tab and back (destroys and recreates servers view)
	_ = gui.switchTabLeft(nil, nil)
	if err := gui.layout(g); err != nil {
		t.Fatalf("local layout failed: %v", err)
	}

	_ = gui.switchTabRight(nil, nil)
	if err := gui.layout(g); err != nil {
		t.Fatalf("servers layout (return) failed: %v", err)
	}

	v, err = g.View(serversView)
	if err != nil {
		t.Fatalf("servers view not found after return: %v", err)
	}
	_, cy = v.Cursor()
	if cy != gui.serverSelected {
		t.Errorf("after tab round-trip: cursor Y = %d, want %d (serverSelected=%d)", cy, gui.serverSelected, gui.serverSelected)
	}
}

func TestTableCursorAtZeroSelection(t *testing.T) {
	requireTTY(t)

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	defer g.Close()

	gui := &Gui{
		g:             g,
		panels:        []string{tableView},
		panelIdx:      0,
		activeTab:     tabLocal,
		jobs:          testJobs(3),
		selected:      0,
		serversConfig: &ssh.ServersConfig{},
	}

	if err := gui.layout(g); err != nil {
		t.Fatalf("layout failed: %v", err)
	}

	v, err := g.View(tableView)
	if err != nil {
		t.Fatalf("table view not found: %v", err)
	}
	_, cy := v.Cursor()
	// Row 0 is the header, row 1 is separator; first data row is at Y=2
	if cy != 2 {
		t.Errorf("cursor Y = %d, want 2 (first data row, selected=0)", cy)
	}
}

func TestFormatHintsOutput(t *testing.T) {
	hints := []hintPair{
		{"c", "reate"},
		{"q", "uit"},
	}

	result := formatHints(hints)

	// Should contain green-coloured keys
	if !strings.Contains(result, style.FgGreen) {
		t.Error("expected green colour codes in hints output")
	}

	// Should contain dim descriptions
	if !strings.Contains(result, style.Dim) {
		t.Error("expected dim colour codes in hints output")
	}

	// Should contain the key text in brackets
	if !strings.Contains(result, "[c]") {
		t.Error("expected [c] in hints output")
	}
	if !strings.Contains(result, "[q]") {
		t.Error("expected [q] in hints output")
	}

	// Should contain description text
	if !strings.Contains(result, "reate") {
		t.Error("expected 'reate' in hints output")
	}
	if !strings.Contains(result, "uit") {
		t.Error("expected 'uit' in hints output")
	}
}

func TestFormatHintsLeadingSpace(t *testing.T) {
	result := formatHints([]hintPair{{"a", "dd"}})
	if !strings.HasPrefix(result, " ") {
		t.Error("expected leading space in hints output")
	}
}

func TestFormatHintsSeparation(t *testing.T) {
	hints := []hintPair{
		{"a", "dd"},
		{"b", "uild"},
	}

	result := formatHints(hints)

	// The two hints should be separated by a space
	// Strip ANSI codes to check spacing
	stripped := stripANSI(result)
	if !strings.Contains(stripped, "]dd [") {
		t.Errorf("expected space between hints, got stripped: %q", stripped)
	}
}

func TestCalculateColumnsWithInset(t *testing.T) {
	// With inset, calculateColumns receives maxX-2
	// Verify it produces reasonable widths for common terminal sizes
	for _, width := range []int{78, 118, 158, 198} {
		cols := calculateColumns(width)
		total := cols.idWidth + cols.exprWidth + cols.cmdWidth + cols.lastWidth + cols.nextWidth + cols.statWidth
		if total > width {
			t.Errorf("column total %d exceeds available width %d", total, width)
		}
		if cols.idWidth < 12 {
			t.Errorf("idWidth %d too narrow at width %d", cols.idWidth, width)
		}
		if cols.cmdWidth < 15 {
			t.Errorf("cmdWidth %d too narrow at width %d", cols.cmdWidth, width)
		}
	}
}

func TestCalculateColumnsNarrowTerminal(t *testing.T) {
	// Very narrow terminal should hide Last Run and Next Run
	cols := calculateColumns(50)
	if cols.lastWidth != 0 {
		t.Errorf("expected lastWidth 0 for narrow terminal, got %d", cols.lastWidth)
	}
	if cols.nextWidth != 0 {
		t.Errorf("expected nextWidth 0 for narrow terminal, got %d", cols.nextWidth)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 3, "hel"},
		{"hello", 4, "h..."},
	}

	for _, tc := range tests {
		got := truncate(tc.input, tc.maxLen)
		if got != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.expected)
		}
	}
}

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'m', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{'é', true},
		{'日', true},
		{' ', false},
		{'-', false},
		{'.', false},
		{'/', false},
		{'*', false},
		{'\t', false},
		{'\n', false},
		{'@', false},
		{'!', false},
	}

	for _, tc := range tests {
		got := isWordChar(tc.r)
		if got != tc.want {
			t.Errorf("isWordChar(%q) = %v, want %v", tc.r, got, tc.want)
		}
	}
}

func TestPrevModalFieldNilModal(t *testing.T) {
	gui := &Gui{}
	err := gui.prevModalField(nil, nil)
	if err != nil {
		t.Errorf("expected nil error for nil modal, got %v", err)
	}
}

func TestPrevModalFieldCycling(t *testing.T) {
	requireTTY(t)

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	defer g.Close()

	// Create the three input views so SetCurrentView works.
	for _, name := range []string{nameInputView, expressionInputView, commandInputView} {
		if _, err := g.SetView(name, 0, 0, 10, 2, 0); err != nil && err != gocui.ErrUnknownView {
			t.Fatalf("failed to create view %s: %v", name, err)
		}
	}

	gui := &Gui{
		g:     g,
		modal: &modalState{activeField: fieldName},
	}

	// Name (0) → prev → Command (2)
	if err := gui.prevModalField(nil, nil); err != nil {
		t.Fatalf("prevModalField from Name failed: %v", err)
	}
	if gui.modal.activeField != fieldCommand {
		t.Errorf("expected fieldCommand (2), got %d", gui.modal.activeField)
	}

	// Command (2) → prev → Expression (1)
	if err := gui.prevModalField(nil, nil); err != nil {
		t.Fatalf("prevModalField from Command failed: %v", err)
	}
	if gui.modal.activeField != fieldExpression {
		t.Errorf("expected fieldExpression (1), got %d", gui.modal.activeField)
	}

	// Expression (1) → prev → Name (0)
	if err := gui.prevModalField(nil, nil); err != nil {
		t.Fatalf("prevModalField from Expression failed: %v", err)
	}
	if gui.modal.activeField != fieldName {
		t.Errorf("expected fieldName (0), got %d", gui.modal.activeField)
	}
}

func TestNextAndPrevModalFieldRoundTrip(t *testing.T) {
	requireTTY(t)

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	defer g.Close()

	for _, name := range []string{nameInputView, expressionInputView, commandInputView} {
		if _, err := g.SetView(name, 0, 0, 10, 2, 0); err != nil && err != gocui.ErrUnknownView {
			t.Fatalf("failed to create view %s: %v", name, err)
		}
	}

	gui := &Gui{
		g:     g,
		modal: &modalState{activeField: fieldName},
	}

	// Forward then backward should return to the same field.
	_ = gui.nextModalField(nil, nil)
	_ = gui.prevModalField(nil, nil)
	if gui.modal.activeField != fieldName {
		t.Errorf("expected fieldName after next+prev round-trip, got %d", gui.modal.activeField)
	}

	// Three prevs from Name should return to Name (full cycle).
	for range 3 {
		_ = gui.prevModalField(nil, nil)
	}
	if gui.modal.activeField != fieldName {
		t.Errorf("expected fieldName after 3 prevs, got %d", gui.modal.activeField)
	}
}

func TestPrevServerFieldCycling(t *testing.T) {
	requireTTY(t)

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	defer g.Close()

	// Create all server input views.
	for _, name := range serverFieldOrder {
		if _, err := g.SetView(name, 0, 0, 10, 2, 0); err != nil && err != gocui.ErrUnknownView {
			t.Fatalf("failed to create view %s: %v", name, err)
		}
	}

	gui := &Gui{g: g}

	// Set current view to the first field (serverNameInput).
	if _, err := g.SetCurrentView(serverFieldOrder[0]); err != nil {
		t.Fatalf("failed to set current view: %v", err)
	}

	// From Name → prev → User (last field, wraps around).
	v, _ := g.View(serverFieldOrder[0])
	if err := gui.prevServerField(nil, v); err != nil {
		t.Fatalf("prevServerField from Name failed: %v", err)
	}
	cur := g.CurrentView()
	if cur.Name() != serverUserInput {
		t.Errorf("expected %s, got %s", serverUserInput, cur.Name())
	}

	// Walk backwards through all fields and verify full cycle.
	expected := []string{serverPortInput, serverHostInput, serverNameInput}
	for _, want := range expected {
		cur = g.CurrentView()
		if err := gui.prevServerField(nil, cur); err != nil {
			t.Fatalf("prevServerField failed: %v", err)
		}
		cur = g.CurrentView()
		if cur.Name() != want {
			t.Errorf("expected %s, got %s", want, cur.Name())
		}
	}
}

func TestValidateExpressionStates(t *testing.T) {
	requireTTY(t)

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	defer g.Close()

	// Create the views that validateExpression reads/writes.
	if _, err := g.SetView(validationView, 0, 0, 40, 2, 0); err != nil && err != gocui.ErrUnknownView {
		t.Fatal(err)
	}
	exprV, err := g.SetView(expressionInputView, 0, 3, 40, 5, 0)
	if err != nil && err != gocui.ErrUnknownView {
		t.Fatal(err)
	}
	exprV.Editable = true

	gui := &Gui{
		g:     g,
		modal: &modalState{},
	}

	// Empty expression → exprValid=false.
	gui.validateExpression()
	if gui.modal.exprValid {
		t.Error("expected exprValid=false for empty expression")
	}

	// Invalid expression → exprValid=false.
	exprV.Clear()
	fmt.Fprint(exprV, "not a cron expr")
	gui.validateExpression()
	if gui.modal.exprValid {
		t.Error("expected exprValid=false for invalid expression")
	}

	// Valid expression → exprValid=true.
	exprV.Clear()
	fmt.Fprint(exprV, "*/5 * * * *")
	gui.validateExpression()
	if !gui.modal.exprValid {
		t.Error("expected exprValid=true for valid expression '*/5 * * * *'")
	}

	// Back to invalid → exprValid=false.
	exprV.Clear()
	fmt.Fprint(exprV, "bad")
	gui.validateExpression()
	if gui.modal.exprValid {
		t.Error("expected exprValid=false after switching to invalid expression")
	}

	// Nil modal → no panic.
	gui.modal = nil
	gui.validateExpression() // should not panic
}

// stripANSI removes ANSI escape sequences from a string for testing.
func stripANSI(s string) string {
	result := strings.Builder{}
	i := 0
	for i < len(s) {
		if s[i] == '\033' {
			// Skip until 'm'
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip the 'm'
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}
