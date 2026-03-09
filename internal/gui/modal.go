package gui

import (
	"fmt"
	"strings"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/cron"
	"github.com/bssmnt/lazycron/internal/gui/style"
)

// Modal view names.
const (
	createModalView     = "createModal"
	deleteModalView     = "deleteModal"
	detailOverlayView   = "detailOverlay"
	nameInputView       = "nameInput"
	expressionInputView = "expressionInput"
	commandInputView    = "commandInput"
	validationView      = "validation"
	exprGuideView       = "exprGuide"
)

// modalField tracks which input field is active in the create/edit modal.
type modalField int

const (
	fieldName modalField = iota
	fieldExpression
	fieldCommand
)

// modalState holds the state of the create/edit modal.
type modalState struct {
	editing     bool // true = editing existing job, false = creating new
	editIndex   int  // index of the job being edited (if editing)
	activeField modalField
	exprValid   bool // true when the expression field contains a valid cron expression
}

// guideWidth is the fixed width of the expression guide panel.
const guideWidth = 36

// openCreateModal opens the create/edit modal.
func (gui *Gui) openCreateModal(editing bool) error {
	maxX, maxY := gui.g.Size()
	// Use 80% of terminal width, clamped between 60 and maxX-4.
	// Add guide panel width when the terminal is wide enough.
	formWidth := max(60, min(maxX*60/100, maxX-4))
	showGuide := maxX >= formWidth+guideWidth+6
	width := formWidth
	if showGuide {
		width = formWidth + guideWidth + 1 // +1 for shared border
	}
	height := 17

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

	gui.modal = &modalState{
		editing:     editing,
		editIndex:   gui.selected,
		activeField: fieldName,
	}

	// Form area: left portion of the modal.
	formX1 := x1
	if showGuide {
		formX1 = x0 + formWidth
	}
	labelCol := x0 + 3
	inputX0 := x0 + 16
	inputX1 := formX1 - 3

	// Modal frame (covers just the form area).
	frame, err := gui.g.SetView(createModalView, x0, y0, formX1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if editing {
		frame.Title = " Edit Cron Job "
	} else {
		frame.Title = " Create Cron Job "
	}
	frame.Clear()

	// Expression guide panel (right side).
	if showGuide {
		gui.createExprGuide(formX1, y0, x1, y1)
	}

	// Row layout:  y0+0  = frame border
	//              y0+1  = blank
	//              y0+2  – y0+4  = Name label + input
	//              y0+5  – y0+7  = Expression label + input
	//              y0+7  – y0+9  = Validation (frameless, 1 content row)
	//              y0+9  – y0+11 = Command label + input
	//              y0+13 – y0+15 = Hints
	//              y0+16 = frame border

	// Name label
	lbl, _ := gui.g.SetView("lblName", labelCol, y0+2, inputX0-1, y0+4, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.FgGreen, " Name:"))
	}

	// Expression label
	lbl, _ = gui.g.SetView("lblExpr", labelCol, y0+5, inputX0-1, y0+7, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.FgGreen, " Expression:"))
	}

	// Command label
	lbl, _ = gui.g.SetView("lblCmd", labelCol, y0+9, inputX0-1, y0+11, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.FgGreen, " Command:"))
	}

	// Hints
	lbl, _ = gui.g.SetView("lblHints", labelCol, y0+13, inputX1+1, y0+15, 0)
	if lbl != nil {
		lbl.Frame = false
		lbl.Clear()
		fmt.Fprint(lbl, style.Coloured(style.Dim, " [Tab/S-Tab] next/prev   [Enter] save   [Esc] cancel"))
	}

	// Name input
	nameV, err := gui.g.SetView(nameInputView, inputX0, y0+2, inputX1, y0+4, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	nameV.Editable = true
	nameV.Frame = true
	nameV.Editor = inputEditor

	// Expression input — uses a wrapping editor that validates on every keystroke.
	exprV, err := gui.g.SetView(expressionInputView, inputX0, y0+5, inputX1, y0+7, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	exprV.Editable = true
	exprV.Frame = true
	exprV.Editor = gui.expressionEditor()

	// Validation label (below expression, frameless, 1 content row)
	valV, err := gui.g.SetView(validationView, inputX0, y0+7, inputX1, y0+9, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	valV.Frame = false

	// Command input
	cmdV, err := gui.g.SetView(commandInputView, inputX0, y0+9, inputX1, y0+11, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	cmdV.Editable = true
	cmdV.Frame = true
	cmdV.Editor = inputEditor

	// Pre-fill if editing
	if editing && gui.selected < len(gui.jobs) {
		job := gui.jobs[gui.selected]
		fmt.Fprint(nameV, job.Comment)
		fmt.Fprint(exprV, job.Expression)
		fmt.Fprint(cmdV, job.Command)
		gui.validateExpression()
	}

	gui.g.Cursor = true

	// Set keybindings for modal
	if err := gui.setupModalKeybindings(); err != nil {
		return err
	}

	// Focus name input
	if _, err := gui.g.SetCurrentView(nameInputView); err != nil {
		return err
	}

	return nil
}

// createExprGuide renders the expression reference guide panel.
func (gui *Gui) createExprGuide(x0, y0, x1, y1 int) {
	v, err := gui.g.SetView(exprGuideView, x0, y0, x1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return
	}
	v.Title = " Expression Guide "
	v.Wrap = true
	v.Clear()

	g := style.Coloured
	dim := style.Dim
	green := style.FgGreen
	cyan := style.FgCyan

	fmt.Fprintln(v)
	fmt.Fprintf(v, " %s\n", g(cyan, "┌───┬───┬───┬────┬─────┐"))
	fmt.Fprintf(v, " %s\n", g(cyan, "│min│ hr│day│ mon│ wday│"))
	fmt.Fprintf(v, " %s\n", g(cyan, "└─┬─┴─┬─┴─┬─┴──┬─┴──┬──┘"))
	fmt.Fprintf(v, " %s\n", g(dim, "  *   *   *    *    *"))
	fmt.Fprintln(v)
	fmt.Fprintf(v, " %s  %s\n", g(green, "0 9 * * *"), g(dim, "Daily at 09:00"))
	fmt.Fprintf(v, " %s  %s\n", g(green, "*/15 * * * *"), g(dim, "Every 15 mins"))
	fmt.Fprintf(v, " %s  %s\n", g(green, "0 0 * * 1"), g(dim, "Mondays at 00:00"))
	fmt.Fprintf(v, " %s  %s\n", g(green, "0 8 1 * *"), g(dim, "1st of month, 08:00"))
	fmt.Fprintln(v)
	fmt.Fprintf(v, " %s\n", g(cyan, "Shortcuts:"))
	fmt.Fprintf(v, " %s %s %s\n",
		g(green, "@daily"), g(green, "@weekly"), g(green, "@monthly"))
	fmt.Fprintf(v, " %s %s %s\n",
		g(green, "@yearly"), g(green, "@hourly"), g(green, "@reboot"))
}

// setupModalKeybindings registers keybindings for the create/edit modal inputs.
func (gui *Gui) setupModalKeybindings() error {
	inputViews := []string{nameInputView, expressionInputView, commandInputView}

	for _, viewName := range inputViews {
		vn := viewName
		if err := gui.g.SetKeybinding(vn, gocui.KeyEsc, gocui.ModNone, gui.closeCreateModal); err != nil {
			return err
		}
		if err := gui.g.SetKeybinding(vn, gocui.KeyEnter, gocui.ModNone, gui.saveModal); err != nil {
			return err
		}
		if err := gui.g.SetKeybinding(vn, gocui.KeyTab, gocui.ModNone, gui.nextModalField); err != nil {
			return err
		}
		if err := gui.g.SetKeybinding(vn, gocui.KeyBacktab, gocui.ModNone, gui.prevModalField); err != nil {
			return err
		}
	}

	return nil
}

// nextModalField cycles to the next input field.
func (gui *Gui) nextModalField(_ *gocui.Gui, _ *gocui.View) error {
	if gui.modal == nil {
		return nil
	}

	gui.modal.activeField = (gui.modal.activeField + 1) % 3
	viewName := gui.modalFieldViewName(gui.modal.activeField)
	if _, err := gui.g.SetCurrentView(viewName); err != nil {
		return err
	}

	return nil
}

// prevModalField cycles to the previous input field.
func (gui *Gui) prevModalField(_ *gocui.Gui, _ *gocui.View) error {
	if gui.modal == nil {
		return nil
	}

	gui.modal.activeField = (gui.modal.activeField + 2) % 3 // +2 ≡ -1 mod 3
	viewName := gui.modalFieldViewName(gui.modal.activeField)
	if _, err := gui.g.SetCurrentView(viewName); err != nil {
		return err
	}

	return nil
}

// modalFieldViewName returns the view name for a modal field.
func (gui *Gui) modalFieldViewName(field modalField) string {
	switch field {
	case fieldName:
		return nameInputView
	case fieldExpression:
		return expressionInputView
	case fieldCommand:
		return commandInputView
	default:
		return nameInputView
	}
}

// saveModal saves the modal form data.
func (gui *Gui) saveModal(_ *gocui.Gui, _ *gocui.View) error {
	if gui.modal == nil {
		return nil
	}

	name := gui.getViewContent(nameInputView)
	expr := gui.getViewContent(expressionInputView)
	command := gui.getViewContent(commandInputView)

	if expr == "" || command == "" {
		gui.setStatusMessage("Expression and command are required")
		return nil
	}

	if !gui.modal.exprValid {
		gui.setStatusMessage("Cannot save: invalid cron expression")
		return nil
	}

	job := cron.CronJob{
		Expression: expr,
		Command:    command,
		Enabled:    true,
		Comment:    name,
	}

	if gui.modal.editing && gui.modal.editIndex < len(gui.jobs) {
		// Update existing job
		existing := gui.jobs[gui.modal.editIndex]
		existing.Expression = job.Expression
		existing.Command = job.Command
		existing.Comment = job.Comment
	} else {
		// Add new job
		gui.crontab.AddJob(job)
	}

	if err := cron.WriteCrontab(gui.crontab); err != nil {
		gui.setStatusMessage(fmt.Sprintf("Error saving: %v", err))
		return nil
	}

	// Reload to get fresh state
	if err := gui.loadCrontab(); err != nil {
		gui.setStatusMessage(fmt.Sprintf("Error reloading: %v", err))
	}

	gui.closeCreateModalViews()
	gui.refreshViews()
	return nil
}

// closeCreateModal closes the create/edit modal without saving.
func (gui *Gui) closeCreateModal(_ *gocui.Gui, _ *gocui.View) error {
	gui.closeCreateModalViews()
	return nil
}

// closeCreateModalViews removes all modal views.
func (gui *Gui) closeCreateModalViews() {
	gui.modal = nil
	gui.g.Cursor = false

	views := []string{
		createModalView, nameInputView, expressionInputView,
		commandInputView, validationView, exprGuideView,
		"lblName", "lblExpr", "lblCmd", "lblHints",
	}
	for _, name := range views {
		gui.g.DeleteView(name)
		gui.g.DeleteKeybindings(name)
	}

	if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
		// Best effort
		_ = err
	}
}

// getViewContent returns the trimmed text content of a view.
func (gui *Gui) getViewContent(viewName string) string {
	v, err := gui.g.View(viewName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v.Buffer())
}

// expressionEditor returns an Editor that delegates to inputEditor and
// triggers real-time validation after every keystroke.
func (gui *Gui) expressionEditor() gocui.Editor {
	return gocui.EditorFunc(func(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
		inputEditorFn(v, key, ch, mod)
		gui.g.Update(func(_ *gocui.Gui) error {
			gui.validateExpression()
			return nil
		})
	})
}

// validateExpression checks the expression and updates the validation label.
func (gui *Gui) validateExpression() {
	v, err := gui.g.View(validationView)
	if err != nil {
		return
	}
	v.Clear()

	if gui.modal == nil {
		return
	}

	expr := gui.getViewContent(expressionInputView)
	if expr == "" {
		gui.modal.exprValid = false
		return
	}

	testJob := &cron.CronJob{Expression: expr}
	if _, err := testJob.NextRun(); err != nil {
		gui.modal.exprValid = false
		fmt.Fprint(v, style.Coloured(style.FgRed, "✗ invalid expression"))
		return
	}

	gui.modal.exprValid = true
	fmt.Fprint(v, style.Coloured(style.FgGreen, "✓ ")+style.Coloured(style.Dim, testJob.Describe()))
}

// openDeleteModal opens the delete confirmation modal.
func (gui *Gui) openDeleteModal() error {
	if len(gui.jobs) == 0 || gui.selected >= len(gui.jobs) {
		return nil
	}

	job := gui.jobs[gui.selected]
	maxX, maxY := gui.g.Size()
	width := 50
	height := 10

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

	v, err := gui.g.SetView(deleteModalView, x0, y0, x1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}

	v.Title = " Delete Job "
	v.Clear()

	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "  Delete %s?\n", style.Coloured(style.FgGreen+style.Bold, job.DisplayName()))
	fmt.Fprintln(v, "")
	fmt.Fprintf(v, "  %s  %s\n", style.Coloured(style.Dim, "Expression:"), job.Expression)
	fmt.Fprintf(v, "  %s     %s\n", style.Coloured(style.Dim, "Command:"), job.Command)
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, style.Coloured(style.Dim, "  [y] confirm   [n/Esc] cancel"))

	if _, err := gui.g.SetCurrentView(deleteModalView); err != nil {
		return err
	}

	// Keybindings for delete confirmation
	if err := gui.g.SetKeybinding(deleteModalView, 'y', gocui.ModNone, gui.confirmDelete); err != nil {
		return err
	}
	if err := gui.g.SetKeybinding(deleteModalView, 'n', gocui.ModNone, gui.cancelDelete); err != nil {
		return err
	}
	if err := gui.g.SetKeybinding(deleteModalView, gocui.KeyEsc, gocui.ModNone, gui.cancelDelete); err != nil {
		return err
	}

	return nil
}

// confirmDelete deletes the selected job and closes the modal.
func (gui *Gui) confirmDelete(_ *gocui.Gui, _ *gocui.View) error {
	if err := gui.crontab.RemoveJob(gui.selected); err != nil {
		gui.setStatusMessage(fmt.Sprintf("Error: %v", err))
		gui.closeDeleteModal()
		return nil
	}

	if err := cron.WriteCrontab(gui.crontab); err != nil {
		gui.setStatusMessage(fmt.Sprintf("Error saving: %v", err))
		gui.closeDeleteModal()
		return nil
	}

	if err := gui.loadCrontab(); err != nil {
		gui.setStatusMessage(fmt.Sprintf("Error reloading: %v", err))
	}

	gui.closeDeleteModal()
	gui.refreshViews()
	return nil
}

// cancelDelete closes the delete modal without deleting.
func (gui *Gui) cancelDelete(_ *gocui.Gui, _ *gocui.View) error {
	gui.closeDeleteModal()
	return nil
}

// closeDeleteModal removes the delete confirmation modal.
func (gui *Gui) closeDeleteModal() {
	gui.g.DeleteView(deleteModalView)
	gui.g.DeleteKeybindings(deleteModalView)
	if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
		_ = err
	}
}

// openDetailOverlay opens a centred detail modal for the selected job.
func (gui *Gui) openDetailOverlay(_ *gocui.Gui, _ *gocui.View) error {
	if len(gui.jobs) == 0 || gui.selected >= len(gui.jobs) {
		return nil
	}

	job := gui.jobs[gui.selected]
	maxX, maxY := gui.g.Size()
	width := min(70, maxX-4)
	height := 14

	x0 := maxX/2 - width/2
	y0 := maxY/2 - height/2

	v, err := gui.g.SetView(detailOverlayView, x0, y0, x0+width, y0+height, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}

	v.Title = fmt.Sprintf(" %s ", job.DisplayName())
	v.Clear()

	status := "Enabled"
	if !job.Enabled {
		status = "Disabled"
	}

	fmt.Fprintln(v)
	fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "Expression:"), job.Expression)
	fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "Schedule:  "), job.Describe())
	fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "Command:   "), job.Command)
	fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "Status:    "), status)
	fmt.Fprintln(v)
	if t, err := job.NextRun(); err == nil {
		fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "Next run:  "), style.Coloured(style.Dim, t.Format("2006-01-02 15:04")))
	}
	if t, err := job.PrevRun(); err == nil {
		fmt.Fprintf(v, "   %s  %s\n", style.Coloured(style.FgGreen, "Prev run:  "), style.Coloured(style.Dim, t.Format("2006-01-02 15:04")))
	}
	fmt.Fprintln(v)
	fmt.Fprintln(v, style.Coloured(style.Dim, "   [Esc] close"))

	if _, err := gui.g.SetCurrentView(detailOverlayView); err != nil {
		return err
	}

	if err := gui.g.SetKeybinding(detailOverlayView, gocui.KeyEsc, gocui.ModNone, gui.closeDetailOverlay); err != nil {
		return err
	}
	if err := gui.g.SetKeybinding(detailOverlayView, gocui.KeyEnter, gocui.ModNone, gui.closeDetailOverlay); err != nil {
		return err
	}

	return nil
}

// closeDetailOverlay removes the detail overlay and returns focus to the table.
func (gui *Gui) closeDetailOverlay(_ *gocui.Gui, _ *gocui.View) error {
	gui.g.DeleteView(detailOverlayView)
	gui.g.DeleteKeybindings(detailOverlayView)
	if _, err := gui.g.SetCurrentView(gui.currentPanel()); err != nil {
		_ = err
	}
	return nil
}
