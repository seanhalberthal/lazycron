package gui

import (
	"unicode"

	"github.com/awesome-gocui/gocui"
)

// inputEditor is a custom editor for input fields that handles macOS
// modifier key combinations (Cmd+Arrow, Opt+Arrow) instead of writing
// garbage escape sequences to the buffer.
var inputEditor gocui.Editor = gocui.EditorFunc(inputEditorFn)

func inputEditorFn(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	// Regular printable characters (no modifier).
	if ch != 0 && mod == 0 {
		v.EditWrite(ch)
		return
	}

	// Alt/Opt modifier — word-level navigation and deletion.
	// macOS terminals typically send ESC+b / ESC+f for Opt+Left/Right,
	// which tcell reports as ModAlt + rune 'b'/'f' rather than
	// ModAlt + KeyArrowLeft/Right. Handle both forms.
	if mod == gocui.ModAlt {
		switch {
		case key == gocui.KeyArrowLeft, ch == 'b':
			moveWordLeft(v)
		case key == gocui.KeyArrowRight, ch == 'f':
			moveWordRight(v)
		case key == gocui.KeyBackspace, key == gocui.KeyBackspace2:
			deleteWordBack(v)
		case ch == 'd':
			deleteWordForward(v)
		}
		// Drop all other Alt combos silently.
		return
	}

	// No modifier — handle special keys.
	switch key {
	case gocui.KeySpace:
		v.EditWrite(' ')
	case gocui.KeyBackspace, gocui.KeyBackspace2:
		v.EditDelete(true)
	case gocui.KeyDelete:
		v.EditDelete(false)
	case gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0)
	case gocui.KeyArrowRight:
		v.MoveCursor(1, 0)
	case gocui.KeyHome:
		v.EditGotoToStartOfLine()
	case gocui.KeyEnd:
		v.EditGotoToEndOfLine()

	// Emacs-style shortcuts.
	case gocui.KeyCtrlA:
		v.EditGotoToStartOfLine()
	case gocui.KeyCtrlE:
		v.EditGotoToEndOfLine()
	case gocui.KeyCtrlU:
		v.EditDeleteToStartOfLine()
	case gocui.KeyCtrlK:
		deleteToEndOfLine(v)
	case gocui.KeyCtrlW:
		deleteWordBack(v)

	// Enter, Tab, Esc are handled by keybindings — ignore here.
	case gocui.KeyEnter, gocui.KeyTab, gocui.KeyEsc:
		// no-op
	default:
		// Only write printable runes; silently drop anything else
		// (e.g. unrecognised modifier combos, function keys).
		if ch != 0 && mod == 0 {
			v.EditWrite(ch)
		}
	}
}

// moveWordLeft moves the cursor to the start of the previous word.
func moveWordLeft(v *gocui.View) {
	buf := v.Buffer()
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	pos := ox + cx

	if pos <= 0 || len(buf) == 0 {
		return
	}
	if pos > len(buf) {
		pos = len(buf)
	}

	runes := []rune(buf)
	// Trim trailing newline that gocui appends.
	for len(runes) > 0 && runes[len(runes)-1] == '\n' {
		runes = runes[:len(runes)-1]
	}
	if pos > len(runes) {
		pos = len(runes)
	}

	// Skip whitespace/non-word chars backwards.
	for pos > 0 && !isWordChar(runes[pos-1]) {
		pos--
	}
	// Skip word chars backwards.
	for pos > 0 && isWordChar(runes[pos-1]) {
		pos--
	}

	moveCursorTo(v, pos)
}

// moveWordRight moves the cursor to the end of the next word.
func moveWordRight(v *gocui.View) {
	buf := v.Buffer()
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	pos := ox + cx

	runes := []rune(buf)
	// Trim trailing newline.
	for len(runes) > 0 && runes[len(runes)-1] == '\n' {
		runes = runes[:len(runes)-1]
	}

	if pos >= len(runes) {
		return
	}

	// Skip word chars forwards.
	for pos < len(runes) && isWordChar(runes[pos]) {
		pos++
	}
	// Skip whitespace/non-word chars forwards.
	for pos < len(runes) && !isWordChar(runes[pos]) {
		pos++
	}

	moveCursorTo(v, pos)
}

// deleteWordBack deletes the word before the cursor (like Opt+Backspace or Ctrl+W).
func deleteWordBack(v *gocui.View) {
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	startPos := ox + cx

	if startPos <= 0 {
		return
	}

	buf := v.Buffer()
	runes := []rune(buf)
	for len(runes) > 0 && runes[len(runes)-1] == '\n' {
		runes = runes[:len(runes)-1]
	}

	pos := startPos
	if pos > len(runes) {
		pos = len(runes)
	}

	// Skip whitespace backwards.
	for pos > 0 && !isWordChar(runes[pos-1]) {
		pos--
	}
	// Skip word chars backwards.
	for pos > 0 && isWordChar(runes[pos-1]) {
		pos--
	}

	// Delete characters from pos to startPos.
	count := startPos - pos
	for range count {
		v.EditDelete(true)
	}
}

// deleteWordForward deletes the word after the cursor (Alt+d).
func deleteWordForward(v *gocui.View) {
	buf := v.Buffer()
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	startPos := ox + cx

	runes := []rune(buf)
	for len(runes) > 0 && runes[len(runes)-1] == '\n' {
		runes = runes[:len(runes)-1]
	}

	if startPos >= len(runes) {
		return
	}

	pos := startPos
	// Skip word chars forwards.
	for pos < len(runes) && isWordChar(runes[pos]) {
		pos++
	}
	// Skip whitespace/non-word chars forwards.
	for pos < len(runes) && !isWordChar(runes[pos]) {
		pos++
	}

	count := pos - startPos
	for range count {
		v.EditDelete(false)
	}
}

// deleteToEndOfLine deletes from cursor to end of line (Ctrl+K).
func deleteToEndOfLine(v *gocui.View) {
	buf := v.Buffer()
	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	pos := ox + cx

	runes := []rune(buf)
	for len(runes) > 0 && runes[len(runes)-1] == '\n' {
		runes = runes[:len(runes)-1]
	}

	count := len(runes) - pos
	for range count {
		v.EditDelete(false)
	}
}

// moveCursorTo places the cursor at the given rune position within a single-line view.
func moveCursorTo(v *gocui.View, pos int) {
	maxX, _ := v.Size()
	if pos < maxX {
		_ = v.SetOrigin(0, 0)
		_ = v.SetCursor(pos, 0)
	} else {
		origin := pos - maxX + 1
		_ = v.SetOrigin(origin, 0)
		_ = v.SetCursor(min(pos, maxX-1), 0)
	}
}

// isWordChar returns true for characters that are part of a "word" (letters, digits, underscore).
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
