package gui

import (
	"fmt"
	"os"
	"testing"

	"github.com/awesome-gocui/gocui"
)

// requireEditorTTY skips if no TTY is available.
func requireEditorTTY(t *testing.T) *gocui.Gui {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping editor test in short mode")
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		t.Skip("skipping editor test: no TTY available")
	}
	_ = f.Close()

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		t.Fatalf("failed to create gocui: %v", err)
	}
	t.Cleanup(g.Close)
	return g
}

// setupEditorView creates a single-line editable view with the given content.
func setupEditorView(t *testing.T, g *gocui.Gui, content string) *gocui.View {
	t.Helper()
	v, err := g.SetView("testInput", 0, 0, 80, 2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		t.Fatalf("failed to create view: %v", err)
	}
	v.Editable = true
	for _, ch := range content {
		v.EditWrite(ch)
	}
	return v
}

func TestMoveWordLeftFromEnd(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello world")
	// Cursor should be at end (pos 11)
	moveWordLeft(v)
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	pos := ox + cx
	if pos != 6 {
		t.Errorf("expected pos 6 (start of 'world'), got %d", pos)
	}
}

func TestMoveWordLeftTwice(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello world")
	moveWordLeft(v)
	moveWordLeft(v)
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	pos := ox + cx
	if pos != 0 {
		t.Errorf("expected pos 0 (start of 'hello'), got %d", pos)
	}
}

func TestMoveWordLeftAtStart(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello")
	// Move to start first
	moveCursorTo(v, 0)
	moveWordLeft(v)
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	if ox+cx != 0 {
		t.Errorf("expected pos 0 (already at start), got %d", ox+cx)
	}
}

func TestMoveWordRightFromStart(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello world")
	moveCursorTo(v, 0)
	moveWordRight(v)
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	pos := ox + cx
	if pos != 6 {
		t.Errorf("expected pos 6 (start of 'world'), got %d", pos)
	}
}

func TestMoveWordRightAtEnd(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello")
	// Cursor already at end
	moveWordRight(v)
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	pos := ox + cx
	// Should stay at end (5)
	if pos != 5 {
		t.Errorf("expected pos 5 (already at end), got %d", pos)
	}
}

func TestMoveWordLeftEmptyBuffer(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "")
	moveWordLeft(v) // should not panic
	cx, _ := v.Cursor()
	if cx != 0 {
		t.Errorf("expected cursor at 0, got %d", cx)
	}
}

func TestMoveWordRightEmptyBuffer(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "")
	moveWordRight(v) // should not panic
	cx, _ := v.Cursor()
	if cx != 0 {
		t.Errorf("expected cursor at 0, got %d", cx)
	}
}

func TestDeleteWordBack(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello world")
	deleteWordBack(v)
	got := v.Buffer()
	// Should have deleted "world" leaving "hello "
	expected := "hello "
	// gocui appends a newline
	if len(got) > 0 && got[len(got)-1] == '\n' {
		got = got[:len(got)-1]
	}
	if got != expected {
		t.Errorf("after deleteWordBack: buffer = %q, want %q", got, expected)
	}
}

func TestDeleteWordBackAtStart(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello")
	moveCursorTo(v, 0)
	deleteWordBack(v) // should be no-op
	got := v.Buffer()
	if len(got) > 0 && got[len(got)-1] == '\n' {
		got = got[:len(got)-1]
	}
	if got != "hello" {
		t.Errorf("expected buffer unchanged, got %q", got)
	}
}

func TestDeleteWordForward(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello world")
	moveCursorTo(v, 0)
	deleteWordForward(v)
	got := v.Buffer()
	if len(got) > 0 && got[len(got)-1] == '\n' {
		got = got[:len(got)-1]
	}
	if got != "world" {
		t.Errorf("after deleteWordForward: buffer = %q, want %q", got, "world")
	}
}

func TestDeleteWordForwardAtEnd(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello")
	// Cursor at end
	deleteWordForward(v) // should be no-op
	got := v.Buffer()
	if len(got) > 0 && got[len(got)-1] == '\n' {
		got = got[:len(got)-1]
	}
	if got != "hello" {
		t.Errorf("expected buffer unchanged, got %q", got)
	}
}

func TestDeleteToEndOfLine(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello world")
	moveCursorTo(v, 5)
	deleteToEndOfLine(v)
	got := v.Buffer()
	if len(got) > 0 && got[len(got)-1] == '\n' {
		got = got[:len(got)-1]
	}
	if got != "hello" {
		t.Errorf("after deleteToEndOfLine: buffer = %q, want %q", got, "hello")
	}
}

func TestMoveCursorToWithinView(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello world")
	moveCursorTo(v, 3)
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	if ox+cx != 3 {
		t.Errorf("expected pos 3, got %d", ox+cx)
	}
}

func TestMoveWordWithMultipleSpaces(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello   world")
	// Cursor at end (13)
	moveWordLeft(v)
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	pos := ox + cx
	if pos != 8 {
		t.Errorf("expected pos 8 (start of 'world'), got %d", pos)
	}
}

func TestMoveWordWithSpecialChars(t *testing.T) {
	g := requireEditorTTY(t)
	v := setupEditorView(t, g, "hello/world.txt")
	// Cursor at end (15)
	moveWordLeft(v)
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	pos := ox + cx
	// "txt" is a word, so should land at start of "txt" = pos 12
	if pos != 12 {
		t.Errorf("expected pos 12 (start of 'txt'), got %d", pos)
	}
}

func TestInputEditorFnPrintableChar(t *testing.T) {
	g := requireEditorTTY(t)
	v, err := g.SetView("editorFnTest", 0, 0, 80, 2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		t.Fatalf("failed to create view: %v", err)
	}
	v.Editable = true

	// Write 'a' via the editor function
	inputEditorFn(v, 0, 'a', 0)
	got := v.Buffer()
	if len(got) > 0 && got[len(got)-1] == '\n' {
		got = got[:len(got)-1]
	}
	if got != "a" {
		t.Errorf("expected 'a', got %q", got)
	}
}

func TestInputEditorFnSpaceKey(t *testing.T) {
	g := requireEditorTTY(t)
	v, err := g.SetView("editorSpaceTest", 0, 0, 80, 2, 0)
	if err != nil && err != gocui.ErrUnknownView {
		t.Fatalf("failed to create view: %v", err)
	}
	v.Editable = true

	inputEditorFn(v, 0, 'h', 0)
	inputEditorFn(v, gocui.KeySpace, 0, 0)
	inputEditorFn(v, 0, 'i', 0)

	got := v.Buffer()
	if len(got) > 0 && got[len(got)-1] == '\n' {
		got = got[:len(got)-1]
	}
	if got != "h i" {
		t.Errorf("expected 'h i', got %q", got)
	}
}

func TestIsWordCharTable(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', true}, {'Z', true}, {'0', true}, {'9', true},
		{'_', true}, {'é', true}, {'日', true},
		{' ', false}, {'-', false}, {'.', false}, {'/', false},
		{'*', false}, {'\t', false}, {'@', false}, {'!', false},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%c", tc.r), func(t *testing.T) {
			if got := isWordChar(tc.r); got != tc.want {
				t.Errorf("isWordChar(%q) = %v, want %v", tc.r, got, tc.want)
			}
		})
	}
}
