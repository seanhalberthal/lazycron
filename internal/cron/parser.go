package cron

import (
	"fmt"
	"strings"
)

// Line represents a single line in a crontab file.
type Line interface {
	String() string
}

// JobLine wraps a CronJob as a crontab line.
type JobLine struct {
	Job CronJob
}

func (l *JobLine) String() string {
	return l.Job.String()
}

// EnvLine represents an environment variable assignment (e.g. SHELL=/bin/bash).
type EnvLine struct {
	Key   string
	Value string
}

func (l *EnvLine) String() string {
	return fmt.Sprintf("%s=%s", l.Key, l.Value)
}

// CommentLine represents a standalone comment (not a disabled job).
type CommentLine struct {
	Text string
}

func (l *CommentLine) String() string {
	return l.Text
}

// BlankLine represents an empty line.
type BlankLine struct{}

func (l *BlankLine) String() string {
	return ""
}

// Crontab represents the full contents of a crontab file, preserving structure.
type Crontab struct {
	Lines []Line
}

// Jobs returns all cron jobs (both enabled and disabled).
func (ct *Crontab) Jobs() []*CronJob {
	var jobs []*CronJob
	for _, line := range ct.Lines {
		if jl, ok := line.(*JobLine); ok {
			jobs = append(jobs, &jl.Job)
		}
	}
	return jobs
}

// AddJob appends a new job to the crontab.
func (ct *Crontab) AddJob(job CronJob) {
	ct.Lines = append(ct.Lines, &JobLine{Job: job})
}

// RemoveJob removes the job at the given index (among JobLines only).
func (ct *Crontab) RemoveJob(jobIndex int) error {
	count := 0
	for i, line := range ct.Lines {
		if _, ok := line.(*JobLine); ok {
			if count == jobIndex {
				ct.Lines = append(ct.Lines[:i], ct.Lines[i+1:]...)
				return nil
			}
			count++
		}
	}
	return fmt.Errorf("job index %d out of range (have %d jobs)", jobIndex, count)
}

// String serialises the entire crontab back to its text representation.
func (ct *Crontab) String() string {
	var sb strings.Builder
	for i, line := range ct.Lines {
		sb.WriteString(line.String())
		if i < len(ct.Lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// Parse parses raw crontab text into a Crontab, preserving all line types.
func Parse(text string) (*Crontab, error) {
	ct := &Crontab{}

	if text == "" {
		return ct, nil
	}

	lines := strings.Split(text, "\n")
	for _, raw := range lines {
		line, err := parseLine(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse line %q: %w", raw, err)
		}
		ct.Lines = append(ct.Lines, line)
	}

	return ct, nil
}

// parseLine classifies and parses a single crontab line.
func parseLine(raw string) (Line, error) {
	trimmed := strings.TrimSpace(raw)

	// Blank line
	if trimmed == "" {
		return &BlankLine{}, nil
	}

	// Comment line — could be a disabled job or a standalone comment
	if strings.HasPrefix(trimmed, "#") {
		return parseCommentOrDisabledJob(trimmed)
	}

	// Environment variable line (KEY=value)
	if isEnvLine(trimmed) {
		return parseEnvLine(trimmed)
	}

	// Otherwise, try to parse as a job line
	return parseJobLine(trimmed, true)
}

// parseCommentOrDisabledJob determines if a comment is a disabled job or a standalone comment.
func parseCommentOrDisabledJob(line string) (Line, error) {
	// Strip leading "# " or "#" to see if the remainder is a valid job
	content := strings.TrimPrefix(line, "# ")
	if content == line {
		content = strings.TrimPrefix(line, "#")
	}
	content = strings.TrimSpace(content)

	// Try to parse as a disabled job
	if content != "" {
		jobLine, err := parseJobLine(content, false)
		if err == nil {
			jl := jobLine.(*JobLine)
			jl.Job.Enabled = false
			return jl, nil
		}
	}

	// It's a standalone comment
	return &CommentLine{Text: line}, nil
}

// isEnvLine checks if a line looks like an environment variable assignment.
func isEnvLine(line string) bool {
	// Must contain = and the part before = must be a valid env var name
	eqIdx := strings.Index(line, "=")
	if eqIdx <= 0 {
		return false
	}

	key := strings.TrimSpace(line[:eqIdx])
	// Env var names: letters, digits, underscores, starting with letter or underscore
	for i, ch := range key {
		if ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			continue
		}
		if i > 0 && ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}

	return true
}

// parseEnvLine parses a KEY=value line.
func parseEnvLine(line string) (*EnvLine, error) {
	eqIdx := strings.Index(line, "=")
	if eqIdx <= 0 {
		return nil, fmt.Errorf("invalid environment line: %s", line)
	}

	key := strings.TrimSpace(line[:eqIdx])
	value := strings.TrimSpace(line[eqIdx+1:])

	return &EnvLine{Key: key, Value: value}, nil
}

// parseJobLine parses a cron job line. The enabled parameter sets the initial enabled state.
func parseJobLine(line string, enabled bool) (Line, error) {
	expr, rest, err := extractExpression(line)
	if err != nil {
		return nil, err
	}

	command, comment := splitCommandAndComment(rest)
	if command == "" {
		return nil, fmt.Errorf("job line has no command: %s", line)
	}

	return &JobLine{
		Job: CronJob{
			Expression: expr,
			Command:    command,
			Enabled:    enabled,
			Comment:    comment,
		},
	}, nil
}

// extractExpression extracts the cron expression from the beginning of a line.
// Handles both standard 5-field expressions and predefined descriptors (@daily, etc).
func extractExpression(line string) (expr string, rest string, err error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", "", fmt.Errorf("empty line")
	}

	// Check for predefined descriptors (@daily, @hourly, @every, etc.)
	if strings.HasPrefix(fields[0], "@") {
		descriptor := fields[0]
		// @every takes an additional duration argument
		if strings.EqualFold(descriptor, "@every") && len(fields) > 1 {
			expr = descriptor + " " + fields[1]
			rest = strings.Join(fields[2:], " ")
		} else {
			expr = descriptor
			rest = strings.Join(fields[1:], " ")
		}

		// Validate the expression
		if _, parseErr := parseSchedule(expr); parseErr != nil {
			return "", "", fmt.Errorf("invalid cron descriptor %q: %w", expr, parseErr)
		}
		return expr, strings.TrimSpace(rest), nil
	}

	// Standard 5-field expression: min hour dom month dow
	if len(fields) < 6 {
		return "", "", fmt.Errorf("not enough fields for cron expression (need 5 + command): %s", line)
	}

	expr = strings.Join(fields[:5], " ")
	rest = strings.Join(fields[5:], " ")

	// Validate the expression
	if _, parseErr := parseSchedule(expr); parseErr != nil {
		return "", "", fmt.Errorf("invalid cron expression %q: %w", expr, parseErr)
	}

	return expr, strings.TrimSpace(rest), nil
}

// splitCommandAndComment splits a command string from its inline comment.
// Inline comments start with " # " (space-hash-space).
func splitCommandAndComment(s string) (command, comment string) {
	// Look for " # " as the comment separator
	idx := strings.Index(s, " # ")
	if idx < 0 {
		return strings.TrimSpace(s), ""
	}

	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+3:])
}
