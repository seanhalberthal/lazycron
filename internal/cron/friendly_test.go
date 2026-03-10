package cron

import "testing"

func TestExpandFriendly(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Daily with time.
		{"daily at 12pm", "@daily @12pm", "0 12 * * *", false},
		{"daily at 9am", "@daily @9am", "0 9 * * *", false},
		{"daily at 1:30am", "@daily @1:30am", "30 1 * * *", false},
		{"daily at 14:30 (24h)", "@daily @14:30", "30 14 * * *", false},
		{"daily at noon", "@daily @noon", "0 12 * * *", false},
		{"daily at midnight", "@daily @midnight", "0 0 * * *", false},

		// 12am/12pm edge cases.
		{"daily at 12am (midnight)", "@daily @12am", "0 0 * * *", false},
		{"daily at 12pm (noon)", "@daily @12pm", "0 12 * * *", false},
		{"daily at 12:30am", "@daily @12:30am", "30 0 * * *", false},
		{"daily at 12:30pm", "@daily @12:30pm", "30 12 * * *", false},

		// Weekly with day and time.
		{"weekly tue 1:30am", "@weekly @tuesday @1:30am", "30 1 * * 2", false},
		{"weekly fri noon", "@weekly @friday @noon", "0 12 * * 5", false},
		{"weekly sun (day only)", "@weekly @sun", "0 0 * * 0", false},
		{"weekly mon 9am", "@weekly @mon @9am", "0 9 * * 1", false},
		{"weekly time only", "@weekly @9am", "0 9 * * 0", false},
		{"weekly time before day", "@weekly @1:30am @tuesday", "30 1 * * 2", false},

		// Monthly with time.
		{"monthly at 8am", "@monthly @8am", "0 8 1 * *", false},
		{"monthly at 3:45pm", "@monthly @3:45pm", "45 15 1 * *", false},

		// Yearly with time.
		{"yearly at 6:15pm", "@yearly @6:15pm", "15 18 1 1 *", false},
		{"annually at 6am", "@annually @6am", "0 6 1 1 *", false},

		// Hourly with minute.
		{"hourly at :30", "@hourly @:30", "30 * * * *", false},
		{"hourly at :00", "@hourly @:00", "0 * * * *", false},
		{"hourly at :45", "@hourly @:45", "45 * * * *", false},

		// Case insensitivity.
		{"case insensitive freq", "@Daily @12PM", "0 12 * * *", false},
		{"case insensitive day", "@WEEKLY @Tuesday", "0 0 * * 2", false},
		{"case insensitive mixed", "@Weekly @FRI @9AM", "0 9 * * 5", false},

		// Pass-through (no modifiers).
		{"plain @daily", "@daily", "@daily", false},
		{"plain @hourly", "@hourly", "@hourly", false},
		{"plain @weekly", "@weekly", "@weekly", false},
		{"plain @reboot", "@reboot", "@reboot", false},
		{"standard 5-field", "*/5 * * * *", "*/5 * * * *", false},
		{"standard with time", "0 9 * * *", "0 9 * * *", false},
		{"empty string", "", "", false},

		// Errors.
		{"day with daily", "@daily @tuesday", "", true},
		{"day with monthly", "@monthly @monday", "", true},
		{"hour with hourly", "@hourly @2pm", "", true},
		{"unknown modifier", "@daily @bogus", "", true},
		{"missing @ prefix", "@daily noon", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpandFriendly(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ExpandFriendly(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("ExpandFriendly(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseTimeModifier(t *testing.T) {
	tests := []struct {
		input      string
		wantHour   int
		wantMinute int
		wantErr    bool
	}{
		{"9am", 9, 0, false},
		{"12pm", 12, 0, false},
		{"12am", 0, 0, false},
		{"1:30am", 1, 30, false},
		{"1:30pm", 13, 30, false},
		{"14:30", 14, 30, false},
		{"0:00", 0, 0, false},
		{"23:59", 23, 59, false},
		{":30", -1, 30, false},
		{":00", -1, 0, false},
		{":59", -1, 59, false},

		// Errors.
		{":60", 0, 0, true},
		{"13pm", 0, 0, true},
		{"0am", 0, 0, true},
		{"25:00", 0, 0, true},
		{"abc", 0, 0, true},
		{":abc", 0, 0, true},
		{"24:00", 0, 0, true},
		{"0:60", 0, 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			h, m, err := parseTimeModifier(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseTimeModifier(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if h != tc.wantHour || m != tc.wantMinute {
				t.Errorf("parseTimeModifier(%q) = (%d, %d), want (%d, %d)", tc.input, h, m, tc.wantHour, tc.wantMinute)
			}
		})
	}
}
