package cron

import (
	"fmt"
	"strings"
	"time"

	crondesc "github.com/lnquy/cron"
	cronparser "github.com/robfig/cron/v3"
)

// CronJob represents a single cron job entry.
type CronJob struct {
	Expression string // "0 * * * *" or "@daily"
	Command    string
	Enabled    bool   // Commented-out = disabled
	Comment    string // Inline comment (used as display name)
}

// Enable marks the job as active.
func (j *CronJob) Enable() {
	j.Enabled = true
}

// Disable marks the job as inactive (will be commented out when serialised).
func (j *CronJob) Disable() {
	j.Enabled = false
}

// NextRun calculates the next scheduled execution time from now.
func (j *CronJob) NextRun() (time.Time, error) {
	return j.NextRunFrom(time.Now())
}

// NextRunFrom calculates the next scheduled execution time from the given time.
func (j *CronJob) NextRunFrom(from time.Time) (time.Time, error) {
	sched, err := parseSchedule(j.Expression)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse expression %q: %w", j.Expression, err)
	}
	return sched.Next(from), nil
}

// PrevRun estimates the previous scheduled time (not actual execution — cron has no history).
func (j *CronJob) PrevRun() (time.Time, error) {
	return j.PrevRunFrom(time.Now())
}

// PrevRunFrom estimates the previous scheduled time from the given time.
// It works by binary-searching backwards for the latest time whose Next() <= from.
func (j *CronJob) PrevRunFrom(from time.Time) (time.Time, error) {
	sched, err := parseSchedule(j.Expression)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse expression %q: %w", j.Expression, err)
	}

	// Search backwards: start from 2 years ago and step forward
	// to find the last execution time before `from`.
	candidate := from.Add(-2 * 365 * 24 * time.Hour)
	var prev time.Time
	for {
		next := sched.Next(candidate)
		if next.After(from) || next.Equal(from) {
			break
		}
		prev = next
		candidate = next
	}

	if prev.IsZero() {
		return time.Time{}, fmt.Errorf("could not determine previous run for %q", j.Expression)
	}
	return prev, nil
}

// Describe returns a human-readable description of the schedule.
// Falls back to the raw expression if description fails.
func (j *CronJob) Describe() string {
	descriptor, err := crondesc.NewDescriptor(
		crondesc.Use24HourTimeFormat(true),
	)
	if err != nil {
		return j.Expression
	}

	desc, err := descriptor.ToDescription(j.Expression, crondesc.Locale_en)
	if err != nil {
		return j.Expression
	}
	return desc
}

// DisplayName returns the comment (if set) or a truncated command as a label.
func (j *CronJob) DisplayName() string {
	if j.Comment != "" {
		return j.Comment
	}
	cmd := j.Command
	if len(cmd) > 40 {
		cmd = cmd[:37] + "..."
	}
	return cmd
}

// String serialises the job back to a crontab line.
func (j *CronJob) String() string {
	var sb strings.Builder

	if !j.Enabled {
		sb.WriteString("# ")
	}

	sb.WriteString(j.Expression)
	sb.WriteString(" ")
	sb.WriteString(j.Command)

	if j.Comment != "" {
		sb.WriteString(" # ")
		sb.WriteString(j.Comment)
	}

	return sb.String()
}

// parseSchedule parses a cron expression into a Schedule.
func parseSchedule(expr string) (cronparser.Schedule, error) {
	parser := cronparser.NewParser(
		cronparser.Minute | cronparser.Hour | cronparser.Dom | cronparser.Month | cronparser.Dow | cronparser.Descriptor,
	)
	return parser.Parse(expr)
}
