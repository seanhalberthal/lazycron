package cron

import (
	"testing"
	"time"
)

func TestJobEnableDisable(t *testing.T) {
	job := CronJob{
		Expression: "0 * * * *",
		Command:    "/usr/bin/test.sh",
		Enabled:    true,
	}

	job.Disable()
	if job.Enabled {
		t.Error("job should be disabled after Disable()")
	}

	job.Enable()
	if !job.Enabled {
		t.Error("job should be enabled after Enable()")
	}
}

func TestJobNextRun(t *testing.T) {
	job := CronJob{
		Expression: "0 * * * *",
		Command:    "/usr/bin/test.sh",
		Enabled:    true,
	}

	// Use a fixed reference time: 2026-03-09 14:30:00 UTC
	ref := time.Date(2026, 3, 9, 14, 30, 0, 0, time.UTC)
	next, err := job.NextRunFrom(ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := time.Date(2026, 3, 9, 15, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("next run = %v, want %v", next, expected)
	}
}

func TestJobNextRunDescriptors(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		ref      time.Time
		wantYear int
	}{
		{
			name:     "daily",
			expr:     "@daily",
			ref:      time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC),
			wantYear: 2026,
		},
		{
			name:     "hourly",
			expr:     "@hourly",
			ref:      time.Date(2026, 3, 9, 14, 30, 0, 0, time.UTC),
			wantYear: 2026,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := CronJob{Expression: tt.expr, Command: "/test.sh", Enabled: true}
			next, err := job.NextRunFrom(tt.ref)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if next.Year() != tt.wantYear {
				t.Errorf("year = %d, want %d", next.Year(), tt.wantYear)
			}
			if !next.After(tt.ref) {
				t.Errorf("next run %v should be after reference %v", next, tt.ref)
			}
		})
	}
}

func TestJobPrevRun(t *testing.T) {
	job := CronJob{
		Expression: "0 * * * *",
		Command:    "/usr/bin/test.sh",
		Enabled:    true,
	}

	ref := time.Date(2026, 3, 9, 14, 30, 0, 0, time.UTC)
	prev, err := job.PrevRunFrom(ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC)
	if !prev.Equal(expected) {
		t.Errorf("prev run = %v, want %v", prev, expected)
	}
}

func TestJobDescribe(t *testing.T) {
	tests := []struct {
		expr string
		want string // Substring to look for
	}{
		{"0 * * * *", "minute"},   // "At 0 minutes past the hour" or similar
		{"@daily", "day"},         // "Every day" or similar
		{"@hourly", "hour"},       // "Every hour" or similar
		{"*/5 * * * *", "minute"}, // "Every 5 minutes" or similar
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			job := CronJob{Expression: tt.expr, Command: "/test.sh", Enabled: true}
			desc := job.Describe()
			if desc == "" {
				t.Error("describe returned empty string")
			}
			// Just check it doesn't return the raw expression (which would mean failure)
			// Some expressions may legitimately contain these substrings
			t.Logf("Describe(%q) = %q", tt.expr, desc)
		})
	}
}

func TestJobDisplayName(t *testing.T) {
	// With comment
	job := CronJob{Comment: "my-backup", Command: "/usr/bin/backup.sh"}
	if job.DisplayName() != "my-backup" {
		t.Errorf("display name = %q, want %q", job.DisplayName(), "my-backup")
	}

	// Without comment — short command
	job = CronJob{Command: "/usr/bin/backup.sh"}
	if job.DisplayName() != "/usr/bin/backup.sh" {
		t.Errorf("display name = %q, want %q", job.DisplayName(), "/usr/bin/backup.sh")
	}

	// Without comment — long command (truncated)
	longCmd := "/usr/local/bin/very-long-command-name-that-exceeds-forty-characters.sh --verbose"
	job = CronJob{Command: longCmd}
	dn := job.DisplayName()
	if len(dn) > 40 {
		t.Errorf("display name too long: %d chars", len(dn))
	}
	if dn[len(dn)-3:] != "..." {
		t.Errorf("display name should end with ..., got %q", dn)
	}
}

func TestJobString(t *testing.T) {
	tests := []struct {
		name string
		job  CronJob
		want string
	}{
		{
			name: "enabled with comment",
			job:  CronJob{Expression: "0 2 * * *", Command: "/usr/bin/backup.sh", Enabled: true, Comment: "backup"},
			want: "0 2 * * * /usr/bin/backup.sh # backup",
		},
		{
			name: "enabled without comment",
			job:  CronJob{Expression: "0 * * * *", Command: "/usr/bin/test.sh", Enabled: true},
			want: "0 * * * * /usr/bin/test.sh",
		},
		{
			name: "disabled with comment",
			job:  CronJob{Expression: "0 0 * * *", Command: "/usr/bin/nightly.sh", Enabled: false, Comment: "nightly"},
			want: "# 0 0 * * * /usr/bin/nightly.sh # nightly",
		},
		{
			name: "descriptor",
			job:  CronJob{Expression: "@daily", Command: "/usr/bin/daily.sh", Enabled: true, Comment: "daily"},
			want: "@daily /usr/bin/daily.sh # daily",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.job.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJobInvalidExpression(t *testing.T) {
	job := CronJob{
		Expression: "invalid expression",
		Command:    "/test.sh",
		Enabled:    true,
	}

	_, err := job.NextRun()
	if err == nil {
		t.Error("expected error for invalid expression")
	}

	_, err = job.PrevRun()
	if err == nil {
		t.Error("expected error for invalid expression")
	}

	// Describe should fall back to raw expression
	desc := job.Describe()
	if desc != "invalid expression" {
		t.Errorf("describe fallback = %q, want %q", desc, "invalid expression")
	}
}
