package gui

import (
	"strings"
	"testing"
)

func TestFormatTableRowAligned(t *testing.T) {
	cols := columnWidths{idWidth: 12, exprWidth: 11, cmdWidth: 20, lastWidth: 19, nextWidth: 19, statWidth: 6}
	row := formatTableRow(cols, "backup", "0 2 * * *", "/usr/bin/backup.sh", "07.01.2025 at 09:00", "08.01.2025 at 09:00", "Active")

	if !strings.Contains(row, "backup") {
		t.Error("expected id in row")
	}
	if !strings.Contains(row, "0 2 * * *") {
		t.Error("expected expression in row")
	}
	if !strings.Contains(row, "/usr/bin/backup.sh") {
		t.Error("expected command in row")
	}
}

func TestFormatTableRowNoLastNext(t *testing.T) {
	cols := columnWidths{idWidth: 12, exprWidth: 11, cmdWidth: 20, lastWidth: 0, nextWidth: 0, statWidth: 6}
	row := formatTableRow(cols, "backup", "0 2 * * *", "/usr/bin/backup.sh", "07.01.2025 at 09:00", "08.01.2025 at 09:00", "Active")

	if strings.Contains(row, "07.01.2025") {
		t.Error("last run should not appear when lastWidth is 0")
	}
	if strings.Contains(row, "08.01.2025") {
		t.Error("next run should not appear when nextWidth is 0")
	}
}

func TestFormatTableRowTruncation(t *testing.T) {
	cols := columnWidths{idWidth: 8, exprWidth: 11, cmdWidth: 10, lastWidth: 0, nextWidth: 0, statWidth: 6}
	row := formatTableRow(cols, "a-very-long-job-name", "0 2 * * *", "/usr/local/bin/very-long-script.sh", "", "", "Active")

	if strings.Contains(row, "a-very-long-job-name") {
		t.Error("id should be truncated")
	}
}
