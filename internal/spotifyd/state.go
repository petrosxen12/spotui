package spotifyd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/petrosxen/spotui/internal/config"
)

func (m *Manager) loadRuntimeState() (*State, int, error) {
	pid, err := readPIDFile(m.files.PIDPath)
	if err != nil {
		return nil, 0, err
	}
	data, err := os.ReadFile(m.files.StatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, pid, nil
		}
		return nil, 0, fmt.Errorf("read spotifyd state: %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, 0, fmt.Errorf("parse spotifyd state: %w", err)
	}
	if state.PID == 0 {
		state.PID = pid
	}
	if pid != 0 && state.PID != pid {
		return nil, 0, fmt.Errorf("spotifyd runtime metadata pid mismatch: pid file=%d state=%d", pid, state.PID)
	}
	return &state, state.PID, nil
}

func (m *Manager) writeRuntimeState(state State) error {
	if err := config.WriteSecureFile(m.files.PIDPath, []byte(strconv.Itoa(state.PID)+"\n")); err != nil {
		return fmt.Errorf("write spotifyd pid: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode spotifyd state: %w", err)
	}
	if err := config.WriteSecureFile(m.files.StatePath, append(data, '\n')); err != nil {
		return fmt.Errorf("write spotifyd state: %w", err)
	}
	return nil
}

func (m *Manager) cleanupStaleRuntimeState() error {
	for _, path := range []string{m.files.PIDPath, m.files.StatePath} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale runtime file %s: %w", path, err)
		}
	}
	return nil
}

func readPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read spotifyd pid: %w", err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return 0, nil
	}
	pid, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse spotifyd pid: %w", err)
	}
	return pid, nil
}
