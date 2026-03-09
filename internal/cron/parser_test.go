package cron

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", name)
}

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(testdataPath(name))
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return string(data)
}

func TestParseEmpty(t *testing.T) {
	ct, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ct.Lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(ct.Lines))
	}
	if len(ct.Jobs()) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(ct.Jobs()))
	}
}

func TestParseBasicCrontab(t *testing.T) {
	text := loadFixture(t, "basic.crontab")
	ct, err := Parse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jobs := ct.Jobs()
	if len(jobs) != 4 {
		t.Fatalf("expected 4 jobs, got %d", len(jobs))
	}

	// First job: backup-db (enabled)
	if jobs[0].Expression != "0 2 * * *" {
		t.Errorf("job 0 expression = %q, want %q", jobs[0].Expression, "0 2 * * *")
	}
	if jobs[0].Command != "/usr/bin/backup.sh" {
		t.Errorf("job 0 command = %q, want %q", jobs[0].Command, "/usr/bin/backup.sh")
	}
	if jobs[0].Comment != "backup-db" {
		t.Errorf("job 0 comment = %q, want %q", jobs[0].Comment, "backup-db")
	}
	if !jobs[0].Enabled {
		t.Error("job 0 should be enabled")
	}

	// Second job: weekly-cleanup (enabled)
	if jobs[1].Expression != "30 3 * * 0" {
		t.Errorf("job 1 expression = %q, want %q", jobs[1].Expression, "30 3 * * 0")
	}
	if jobs[1].Comment != "weekly-cleanup" {
		t.Errorf("job 1 comment = %q, want %q", jobs[1].Comment, "weekly-cleanup")
	}

	// Third job: logrotate (disabled)
	if jobs[2].Enabled {
		t.Error("job 2 (logrotate) should be disabled")
	}
	if jobs[2].Comment != "logrotate" {
		t.Errorf("job 2 comment = %q, want %q", jobs[2].Comment, "logrotate")
	}

	// Fourth job: healthcheck (enabled)
	if jobs[3].Expression != "*/5 * * * *" {
		t.Errorf("job 3 expression = %q, want %q", jobs[3].Expression, "*/5 * * * *")
	}
	if jobs[3].Comment != "healthcheck" {
		t.Errorf("job 3 comment = %q, want %q", jobs[3].Comment, "healthcheck")
	}
}

func TestParseDescriptors(t *testing.T) {
	text := loadFixture(t, "descriptors.crontab")
	ct, err := Parse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jobs := ct.Jobs()
	if len(jobs) != 6 {
		t.Fatalf("expected 6 jobs, got %d", len(jobs))
	}

	expected := []struct {
		expr    string
		comment string
	}{
		{"@daily", "daily-task"},
		{"@hourly", "hourly-check"},
		{"@weekly", "weekly-report"},
		{"@monthly", "monthly-audit"},
		{"@yearly", "annual-review"},
		{"@every 5m", "frequent-check"},
	}

	for i, exp := range expected {
		if jobs[i].Expression != exp.expr {
			t.Errorf("job %d expression = %q, want %q", i, jobs[i].Expression, exp.expr)
		}
		if jobs[i].Comment != exp.comment {
			t.Errorf("job %d comment = %q, want %q", i, jobs[i].Comment, exp.comment)
		}
		if !jobs[i].Enabled {
			t.Errorf("job %d should be enabled", i)
		}
	}
}

func TestParseEnvVars(t *testing.T) {
	text := loadFixture(t, "envvars.crontab")
	ct, err := Parse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count env lines
	var envLines []*EnvLine
	for _, line := range ct.Lines {
		if el, ok := line.(*EnvLine); ok {
			envLines = append(envLines, el)
		}
	}

	if len(envLines) != 4 {
		t.Fatalf("expected 4 env lines, got %d", len(envLines))
	}

	expectedEnv := []struct{ key, value string }{
		{"SHELL", "/bin/bash"},
		{"PATH", "/usr/local/bin:/usr/bin:/bin"},
		{"MAILTO", "admin@example.com"},
		{"HOME", "/home/deploy"},
	}

	for i, exp := range expectedEnv {
		if envLines[i].Key != exp.key || envLines[i].Value != exp.value {
			t.Errorf("env %d = %s=%s, want %s=%s", i, envLines[i].Key, envLines[i].Value, exp.key, exp.value)
		}
	}

	jobs := ct.Jobs()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
}

func TestParseMixedCrontab(t *testing.T) {
	text := loadFixture(t, "mixed.crontab")
	ct, err := Parse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jobs := ct.Jobs()

	// Count enabled and disabled
	var enabled, disabled int
	for _, j := range jobs {
		if j.Enabled {
			enabled++
		} else {
			disabled++
		}
	}

	if enabled != 6 {
		t.Errorf("expected 6 enabled jobs, got %d", enabled)
	}
	if disabled != 2 {
		t.Errorf("expected 2 disabled jobs, got %d", disabled)
	}
}

func TestRoundTrip(t *testing.T) {
	fixtures := []string{"basic.crontab", "descriptors.crontab", "envvars.crontab", "mixed.crontab"}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			original := loadFixture(t, fixture)
			ct, err := Parse(original)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			serialised := ct.String()

			// Re-parse the serialised output
			ct2, err := Parse(serialised)
			if err != nil {
				t.Fatalf("re-parse error: %v", err)
			}

			// Compare job counts
			jobs1 := ct.Jobs()
			jobs2 := ct2.Jobs()
			if len(jobs1) != len(jobs2) {
				t.Errorf("job count mismatch: %d vs %d", len(jobs1), len(jobs2))
			}

			// Compare individual jobs
			for i := range jobs1 {
				if i >= len(jobs2) {
					break
				}
				if jobs1[i].Expression != jobs2[i].Expression {
					t.Errorf("job %d expression: %q vs %q", i, jobs1[i].Expression, jobs2[i].Expression)
				}
				if jobs1[i].Command != jobs2[i].Command {
					t.Errorf("job %d command: %q vs %q", i, jobs1[i].Command, jobs2[i].Command)
				}
				if jobs1[i].Enabled != jobs2[i].Enabled {
					t.Errorf("job %d enabled: %v vs %v", i, jobs1[i].Enabled, jobs2[i].Enabled)
				}
				if jobs1[i].Comment != jobs2[i].Comment {
					t.Errorf("job %d comment: %q vs %q", i, jobs1[i].Comment, jobs2[i].Comment)
				}
			}
		})
	}
}

func TestRoundTripPreservesNonJobLines(t *testing.T) {
	text := loadFixture(t, "envvars.crontab")
	ct, err := Parse(text)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	serialised := ct.String()

	// Env vars should be preserved
	if !strings.Contains(serialised, "SHELL=/bin/bash") {
		t.Error("SHELL env var not preserved")
	}
	if !strings.Contains(serialised, "PATH=/usr/local/bin:/usr/bin:/bin") {
		t.Error("PATH env var not preserved")
	}
	if !strings.Contains(serialised, "MAILTO=admin@example.com") {
		t.Error("MAILTO env var not preserved")
	}
}

func TestAddAndRemoveJob(t *testing.T) {
	ct, err := Parse("")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Add a job
	ct.AddJob(CronJob{
		Expression: "0 * * * *",
		Command:    "/usr/bin/test.sh",
		Enabled:    true,
		Comment:    "test-job",
	})

	jobs := ct.Jobs()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Comment != "test-job" {
		t.Errorf("comment = %q, want %q", jobs[0].Comment, "test-job")
	}

	// Remove the job
	if err := ct.RemoveJob(0); err != nil {
		t.Fatalf("remove error: %v", err)
	}

	if len(ct.Jobs()) != 0 {
		t.Errorf("expected 0 jobs after removal, got %d", len(ct.Jobs()))
	}
}

func TestRemoveJobOutOfRange(t *testing.T) {
	ct, err := Parse("")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	err = ct.RemoveJob(0)
	if err == nil {
		t.Error("expected error for out-of-range removal")
	}
}

func TestParseLineTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{"blank", "", "blank"},
		{"comment", "# This is a comment", "comment"},
		{"env var", "SHELL=/bin/bash", "env"},
		{"standard job", "0 * * * * /usr/bin/test.sh", "job"},
		{"descriptor job", "@daily /usr/bin/test.sh", "job"},
		{"disabled job", "# 0 * * * * /usr/bin/test.sh", "job"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := parseLine(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			switch tt.wantType {
			case "blank":
				if _, ok := line.(*BlankLine); !ok {
					t.Errorf("expected BlankLine, got %T", line)
				}
			case "comment":
				if _, ok := line.(*CommentLine); !ok {
					t.Errorf("expected CommentLine, got %T", line)
				}
			case "env":
				if _, ok := line.(*EnvLine); !ok {
					t.Errorf("expected EnvLine, got %T", line)
				}
			case "job":
				if _, ok := line.(*JobLine); !ok {
					t.Errorf("expected JobLine, got %T", line)
				}
			}
		})
	}
}
