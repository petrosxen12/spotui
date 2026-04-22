package spotifyd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

func (m *Manager) verifyStartup(ctx context.Context, pid int, binaryPath string, startedAt time.Time) (Status, error) {
	// Give spotifyd a brief window to reject an invalid config before we report success.
	time.Sleep(250 * time.Millisecond)

	status, err := m.Status(ctx)
	if err != nil {
		return Status{}, err
	}
	if status.ProcessState == ProcessStateStopped {
		return Status{}, m.startupFailure()
	}
	if status.ProcessState == ProcessStateStarting {
		status.PID = pid
		status.StartedAt = startedAt
		status.BinaryPath = binaryPath
		status.BinaryAvailable = true
		status.RuntimeFiles = m.files
	}
	return status, nil
}

func (m *Manager) startupFailure() error {
	excerpt := tailLog(m.files.LogPath, 12)
	if excerpt == "" {
		return fmt.Errorf("spotifyd exited during startup; check %s", m.files.LogPath)
	}
	return fmt.Errorf("spotifyd exited during startup; check %s\n%s", m.files.LogPath, excerpt)
}

func tailLog(path string, maxLines int) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	lines := make([]string, 0, maxLines)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
