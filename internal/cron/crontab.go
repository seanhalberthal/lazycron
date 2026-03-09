package cron

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ReadCrontab reads and parses the current user's crontab.
func ReadCrontab() (*Crontab, error) {
	output, err := runCrontabList()
	if err != nil {
		return nil, err
	}
	return Parse(output)
}

// WriteCrontab writes the crontab back via `crontab -` stdin pipe.
func WriteCrontab(ct *Crontab) error {
	content := ct.String()
	// Ensure trailing newline
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write crontab: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

// runCrontabList executes `crontab -l` and returns the output.
func runCrontabList() (string, error) {
	cmd := exec.Command("crontab", "-l")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		// Handle "no crontab for user" gracefully
		if strings.Contains(errMsg, "no crontab for") {
			return "", nil
		}
		return "", fmt.Errorf("failed to read crontab: %s: %w", errMsg, err)
	}

	return stdout.String(), nil
}
