package cron

import (
	"fmt"
	"strconv"
	"strings"
)

// dayNames maps day name tokens to cron day-of-week numbers (0=Sun, 6=Sat).
var dayNames = map[string]int{
	"sunday": 0, "sun": 0,
	"monday": 1, "mon": 1,
	"tuesday": 2, "tue": 2,
	"wednesday": 3, "wed": 3,
	"thursday": 4, "thu": 4,
	"friday": 5, "fri": 5,
	"saturday": 6, "sat": 6,
}

// friendlyBases are the frequency keywords that support modifiers.
var friendlyBases = map[string]struct{}{
	"@hourly": {}, "@daily": {}, "@weekly": {},
	"@monthly": {}, "@yearly": {}, "@annually": {},
}

// ExpandFriendly expands a human-friendly cron expression into a standard
// 5-field cron expression. If the input has no modifiers or is not a
// friendly expression, it is returned unchanged.
//
// Supported syntax:
//
//	@daily @2pm           -> 0 14 * * *
//	@weekly @tue @9:30am  -> 30 9 * * 2
//	@monthly @3am         -> 0 3 1 * *
//	@yearly @6:15pm       -> 15 18 1 1 *
//	@hourly @:45          -> 45 * * * *
func ExpandFriendly(expr string) (string, error) {
	tokens := strings.Fields(strings.TrimSpace(expr))
	if len(tokens) == 0 {
		return expr, nil
	}

	freq := strings.ToLower(tokens[0])
	if _, ok := friendlyBases[freq]; !ok {
		return expr, nil // not a friendly base — return as-is
	}

	// No modifiers — pass through for robfig/cron to handle natively.
	if len(tokens) == 1 {
		return expr, nil
	}

	// Set defaults per frequency.
	minute, hour, dom, month, dow := "0", "0", "*", "*", "*"
	switch freq {
	case "@hourly":
		hour = "*"
	case "@weekly":
		dow = "0"
	case "@monthly":
		dom = "1"
	case "@yearly", "@annually":
		dom, month = "1", "1"
	}

	hourSet := false

	for _, token := range tokens[1:] {
		lower := strings.ToLower(token)
		if !strings.HasPrefix(lower, "@") {
			return "", fmt.Errorf("unexpected token %q — modifiers must start with @", token)
		}
		modifier := lower[1:] // strip leading @

		// Day name?
		if dayNum, ok := dayNames[modifier]; ok {
			if freq != "@weekly" {
				return "", fmt.Errorf("day modifier @%s is only valid with @weekly", modifier)
			}
			dow = strconv.Itoa(dayNum)
			continue
		}

		// Special time names?
		switch modifier {
		case "noon":
			hour, minute = "12", "0"
			hourSet = true
			continue
		case "midnight":
			hour, minute = "0", "0"
			hourSet = true
			continue
		}

		// Time modifier?
		h, m, err := parseTimeModifier(modifier)
		if err != nil {
			return "", fmt.Errorf("unknown modifier @%s", modifier)
		}
		if h >= 0 {
			hour = strconv.Itoa(h)
			hourSet = true
		}
		minute = strconv.Itoa(m)
	}

	// @hourly should not have an hour component — only @:MM is valid.
	if freq == "@hourly" && hourSet {
		return "", fmt.Errorf("@hourly only accepts minute modifiers (e.g. @:30)")
	}

	return fmt.Sprintf("%s %s %s %s %s", minute, hour, dom, month, dow), nil
}

// parseTimeModifier parses a time token like "12pm", "1:30am", "14:30", ":30".
// Returns (hour, minute, error). Hour is -1 if only a minute was specified.
func parseTimeModifier(s string) (hour, minute int, err error) {
	// Minute-only: ":30"
	if strings.HasPrefix(s, ":") {
		m, err := strconv.Atoi(s[1:])
		if err != nil || m < 0 || m > 59 {
			return 0, 0, fmt.Errorf("invalid minute %q", s)
		}
		return -1, m, nil
	}

	// Detect am/pm suffix.
	isPM, isAM := false, false
	if strings.HasSuffix(s, "pm") {
		isPM = true
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "am") {
		isAM = true
		s = s[:len(s)-2]
	}

	parts := strings.SplitN(s, ":", 2)

	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour %q", parts[0])
	}

	m := 0
	if len(parts) == 2 {
		m, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid minute %q", parts[1])
		}
	}

	// Convert 12-hour to 24-hour.
	if isPM || isAM {
		if h < 1 || h > 12 {
			return 0, 0, fmt.Errorf("hour %d out of range for 12-hour clock", h)
		}
		if isPM && h != 12 {
			h += 12
		}
		if isAM && h == 12 {
			h = 0
		}
	}

	if h < 0 || h > 23 {
		return 0, 0, fmt.Errorf("hour %d out of range", h)
	}
	if m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("minute %d out of range", m)
	}

	return h, m, nil
}
