package gui

import (
	"fmt"
	"strings"

	"github.com/awesome-gocui/gocui"
	"github.com/bssmnt/lazycron/internal/cron"
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
}

// openCreateModal opens the create/edit modal.
func (gui *Gui) openCreateModal(editing bool) error {
	maxX, maxY := gui.g.Size()
	width := 50
	height := 12

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

	// Modal frame
	frame, err := gui.g.SetView(createModalView, x0, y0, x1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if editing {
		frame.Title = " Edit Cron Job "
	} else {
		frame.Title = " Create Cron Job "
	}
	frame.Clear()

	// Labels
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  Name:")
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  Expression:")
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  Command:")
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "")
	fmt.Fprintln(frame, "  [Tab] next  [Enter] save  [Esc] cancel")

	// Name input
	nameV, err := gui.g.SetView(nameInputView, x0+14, y0+1, x1-2, y0+3, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	nameV.Editable = true
	nameV.Frame = true
	nameV.Editor = gocui.DefaultEditor

	// Expression input
	exprV, err := gui.g.SetView(expressionInputView, x0+14, y0+3, x1-2, y0+5, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	exprV.Editable = true
	exprV.Frame = true
	exprV.Editor = gocui.DefaultEditor

	// Validation label (below expression)
	valV, err := gui.g.SetView(validationView, x0+14, y0+5, x1-2, y0+6, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	valV.Frame = false

	// Command input
	cmdV, err := gui.g.SetView(commandInputView, x0+14, y0+6, x1-2, y0+8, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	cmdV.Editable = true
	cmdV.Frame = true
	cmdV.Editor = gocui.DefaultEditor

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

	// Validate expression
	testJob := &cron.CronJob{Expression: expr}
	if _, err := testJob.NextRun(); err != nil {
		gui.setStatusMessage(fmt.Sprintf("Invalid expression: %s", expr))
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

	views := []string{createModalView, nameInputView, expressionInputView, commandInputView, validationView}
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

// validateExpression checks the expression and updates the validation label.
func (gui *Gui) validateExpression() {
	v, err := gui.g.View(validationView)
	if err != nil {
		return
	}
	v.Clear()

	expr := gui.getViewContent(expressionInputView)
	if expr == "" {
		return
	}

	testJob := &cron.CronJob{Expression: expr}
	desc := testJob.Describe()
	fmt.Fprint(v, desc)
}

// openDeleteModal opens the delete confirmation modal.
func (gui *Gui) openDeleteModal() error {
	if len(gui.jobs) == 0 || gui.selected >= len(gui.jobs) {
		return nil
	}

	job := gui.jobs[gui.selected]
	maxX, maxY := gui.g.Size()
	width := 44
	height := 8

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
	fmt.Fprintf(v, "  Delete %q?\n", job.DisplayName())
	fmt.Fprintf(v, "  Expression: %s\n", job.Expression)
	fmt.Fprintf(v, "  Command: %s\n", job.Command)
	fmt.Fprintln(v, "")
	fmt.Fprintln(v, "  [y] confirm  [n/Esc] cancel")

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
	height := 12

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
	fmt.Fprintf(v, "  Expression:  %s\n", job.Expression)
	fmt.Fprintf(v, "  Schedule:    %s\n", job.Describe())
	fmt.Fprintf(v, "  Command:     %s\n", job.Command)
	fmt.Fprintf(v, "  Status:      %s\n", status)
	if t, err := job.NextRun(); err == nil {
		fmt.Fprintf(v, "  Next run:    %s\n", t.Format("2006-01-02 15:04"))
	}
	if t, err := job.PrevRun(); err == nil {
		fmt.Fprintf(v, "  Prev run:    %s\n", t.Format("2006-01-02 15:04"))
	}
	fmt.Fprintln(v)
	fmt.Fprintln(v, "  [Esc] close")

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
