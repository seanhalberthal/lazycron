package style

import (
	"fmt"

	"github.com/awesome-gocui/gocui"
)

// Colour constants for the TUI.
var (
	// Panel borders
	ActiveBorderColour   = gocui.ColorGreen
	InactiveBorderColour = gocui.ColorDefault

	// Job list
	SelectedBgColour = gocui.ColorGreen
	SelectedFgColour = gocui.ColorBlack

	// Status indicators
	EnabledColour  = gocui.ColorGreen
	DisabledColour = gocui.ColorRed

	// Title bar
	TitleFgColour = gocui.ColorWhite
	TitleBgColour = gocui.ColorDefault

	// Table header
	HeaderFgColour = gocui.ColorWhite
)

// ANSI escape code constants for inline text colouring.
const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Dim   = "\033[2m"

	FgBlack   = "\033[30m"
	FgRed     = "\033[31m"
	FgGreen   = "\033[32m"
	FgYellow  = "\033[33m"
	FgBlue    = "\033[34m"
	FgMagenta = "\033[35m"
	FgCyan    = "\033[36m"
	FgWhite   = "\033[37m"

	BgGreen = "\033[42m"
)

// Indicator characters for job status.
const (
	EnabledIndicator  = "●"
	DisabledIndicator = "○"
)

// Coloured returns text wrapped in the given ANSI colour code.
func Coloured(colour, text string) string {
	return fmt.Sprintf("%s%s%s", colour, text, Reset)
}
